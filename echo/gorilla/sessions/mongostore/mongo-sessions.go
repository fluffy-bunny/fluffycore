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

	status "github.com/gogo/status"
	securecookie "github.com/gorilla/securecookie"
	sessions "github.com/gorilla/sessions"
	xid "github.com/rs/xid"
	bson "go.mongodb.org/mongo-driver/bson"
	mongo_driver "go.mongodb.org/mongo-driver/mongo"
	mongo_options "go.mongodb.org/mongo-driver/mongo/options"
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
}

// NewMongoStore returns a new MongoStore.
// Set ensureTTL to true let the database auto-remove expired object by maxAge.
func NewMongoStore(c *mongo_driver.Collection, maxAge int, ensureTTL bool,
	keyPairs ...[]byte) *MongoStore {
	store := &MongoStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: maxAge,
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
	session.Options = &sessions.Options{
		Path:     m.Options.Path,
		MaxAge:   m.Options.MaxAge,
		Domain:   m.Options.Domain,
		Secure:   m.Options.Secure,
		HttpOnly: m.Options.HttpOnly,
	}
	session.IsNew = true
	var err error
	if cook, errToken := m.Token.GetToken(r, name); errToken == nil {
		err = securecookie.DecodeMulti(name, cook, &session.ID, m.Codecs...)
		if err == nil {
			err = m.load(session)
			if err == nil {
				session.IsNew = false
			} else {
				err = nil
			}
		}
	}
	return session, err
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
func (m *MongoStore) MaxAge(age int) {
	m.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range m.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
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
		// look for "mongo: no documents in result"
		if err.Error() == "mongo: no documents in result" {
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
