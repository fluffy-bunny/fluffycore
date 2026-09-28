// Copyright 2012 The KidStuff Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Modified to use the official mongo driver
package mongostore

import (
	"context"
	"errors"
	"net/http"
	"time"

	storeopts "github.com/fluffy-bunny/fluffycore/echo/internal/storeopts"
	status "github.com/gogo/status"
	securecookie "github.com/gorilla/securecookie"
	sessions "github.com/gorilla/sessions"
	xid "github.com/rs/xid"
	bson "go.mongodb.org/mongo-driver/v2/bson"
	mongo_driver "go.mongodb.org/mongo-driver/v2/mongo"
	mongo_options "go.mongodb.org/mongo-driver/v2/mongo/options"
	codes "google.golang.org/grpc/codes"
)

var (
	ErrInvalidId = errors.New("mgostore: invalid session id")
)

// Session object store in MongoDB
type Session struct {
	ID       string `bson:"_id,omitempty"`
	Data     string
	Modified time.Time
}

// MongoStore stores sessions in MongoDB
type MongoStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options
	Token   ITokenGetSeter
	coll    *mongo_driver.Collection
	// loadFn overrides load in tests.
	loadFn func(*sessions.Session) error
}

// NewMongoStore returns a new MongoStore.
// Set ensureTTL to true let the database auto-remove expired object by maxAge.
func NewMongoStore(c *mongo_driver.Collection, maxAge int, ensureTTL bool,
	keyPairs ...[]byte) *MongoStore {
	store := &MongoStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:     "/",
			MaxAge:   maxAge,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		Token: &CookieToken{},
		coll:  c,
	}

	store.MaxAge(maxAge)

	// Your DEV ops team is responsible for creating the indexes

	return store
}

// Get registers and returns a session for the given name and session store.
// It returns a new session if there are no sessions registered for the name.
func (m *MongoStore) Get(r *http.Request, name string) (
	*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(m, name)
}

// New returns a session for the given name without adding it to the registry.
func (m *MongoStore) New(r *http.Request, name string) (
	*sessions.Session, error) {
	session := sessions.NewSession(m, name)
	opts := *m.Options // copy every field (SameSite, Partitioned, ...), not a hand-picked subset
	session.Options = &opts
	session.IsNew = true
	cook, errToken := m.Token.GetToken(r, name)
	if errToken != nil {
		return session, nil
	}
	if err := securecookie.DecodeMulti(name, cook, &session.ID, m.Codecs...); err != nil {
		session.ID = ""
		return session, err
	}
	return session, m.loadExisting(session)
}

// loadExisting loads session.ID's data. If the stored session is gone (expired, deleted) the
// session starts over with a fresh ID -- reusing the old one would let a stale (or planted)
// cookie name the new session. Any other failure (e.g. Mongo unavailable) is returned rather
// than silently handing back an empty session that Save would then overwrite the real one with.
func (m *MongoStore) loadExisting(session *sessions.Session) error {
	load := m.load
	if m.loadFn != nil {
		load = m.loadFn
	}
	err := load(session)
	switch {
	case err == nil:
		session.IsNew = false
		return nil
	case status.Code(err) == codes.NotFound || errors.Is(err, ErrInvalidId):
		session.ID = ""
		return nil
	default:
		session.ID = ""
		return err
	}
}

// Save saves all sessions registered for the current request.
func (m *MongoStore) Save(r *http.Request, w http.ResponseWriter,
	session *sessions.Session) error {
	if session.Options.MaxAge < 0 {
		if err := m.delete(session); err != nil {
			return err
		}
		m.Token.SetToken(w, session.Name(), "", session.Options)
		return nil
	}

	if session.ID == "" {
		session.ID = xid.New().String()
	}

	if err := m.upsert(session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID,
		m.Codecs...)
	if err != nil {
		return err
	}

	m.Token.SetToken(w, session.Name(), encoded, session.Options)
	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
// A 0 (browser-session cookie) age still bounds the signed value's validity; see
// storeopts.SetMaxAge.
func (m *MongoStore) MaxAge(age int) {
	storeopts.SetMaxAge(m.Options, m.Codecs, age)
}

// VerifyID checks if the given ID is a valid rs/xid ID.
func VerifyID(id string) bool {
	_, err := xid.FromString(id)
	return err == nil
}
func NewId() string {
	return xid.New().String()
}
func (m *MongoStore) load(session *sessions.Session) error {

	if !VerifyID(session.ID) {
		return ErrInvalidId
	}

	s := Session{}
	ctxTimeout, cancel := getMongoContext()
	defer cancel()
	result := m.coll.FindOne(ctxTimeout, bson.M{"_id": session.ID})
	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo_driver.ErrNoDocuments) {
			return status.Error(codes.NotFound, "not found")
		}
		return err
	}
	err = result.Decode(&s)
	if err != nil {
		return err
	}

	if err := securecookie.DecodeMulti(session.Name(), s.Data, &session.Values,
		m.Codecs...); err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) upsert(session *sessions.Session) error {
	if !VerifyID(session.ID) {
		return ErrInvalidId
	}

	var modified time.Time
	if val, ok := session.Values["modified"]; ok {
		modified, ok = val.(time.Time)
		if !ok {
			return errors.New("mongostore: invalid modified value")
		}
	} else {
		modified = time.Now()
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		m.Codecs...)
	if err != nil {
		return err
	}
	ctxTimeout, cancel := getMongoContext()
	defer cancel()

	s := Session{
		Data:     encoded,
		Modified: modified,
	}
	filter := bson.M{"_id": session.ID}
	_, err = m.coll.ReplaceOne(ctxTimeout, filter, s, mongo_options.Replace().SetUpsert(true))

	if err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) delete(session *sessions.Session) error {
	if !VerifyID(session.ID) {
		return ErrInvalidId
	}
	ctxTimeout, cancel := getMongoContext()
	defer cancel()

	_, err := m.coll.DeleteOne(ctxTimeout, bson.M{"_id": session.ID})
	if err != nil {
		return err
	}
	return nil

}

func getMongoContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
