package mongostore

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	status "github.com/gogo/status"
	securecookie "github.com/gorilla/securecookie"
	sessions "github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
)

// newTestStore builds a MongoStore with no collection; load is stubbed with loadFn.
func newTestStore(t *testing.T, load func(*sessions.Session) error) *MongoStore {
	t.Helper()
	m := NewMongoStore(nil, 3600, false, []byte("0123456789abcdef0123456789abcdef"))
	m.loadFn = load
	return m
}

func requestWithSessionID(t *testing.T, m *MongoStore, name, id string) *http.Request {
	t.Helper()
	encoded, err := securecookie.EncodeMulti(name, id, m.Codecs...)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: name, Value: encoded})
	return r
}

func TestNew_ExistingSessionLoaded(t *testing.T) {
	m := newTestStore(t, func(*sessions.Session) error { return nil })
	id := NewId()
	s, err := m.New(requestWithSessionID(t, m, "sess", id), "sess")
	require.NoError(t, err)
	require.False(t, s.IsNew)
	require.Equal(t, id, s.ID)
}

// The session is gone from Mongo (expired/deleted): the old, cookie-supplied ID used to be kept,
// so Save re-created the session under an ID the client chose earlier.
func TestNew_StoredSessionGone_StartsFreshID(t *testing.T) {
	m := newTestStore(t, func(*sessions.Session) error { return status.Error(codes.NotFound, "not found") })
	s, err := m.New(requestWithSessionID(t, m, "sess", NewId()), "sess")
	require.NoError(t, err)
	require.True(t, s.IsNew)
	require.Empty(t, s.ID, "Save must mint a new ID")
}

// A Mongo outage used to be swallowed: an empty session came back and Save overwrote the real one.
func TestNew_LoadFailure_ReturnsError(t *testing.T) {
	m := newTestStore(t, func(*sessions.Session) error { return errors.New("connection refused") })
	s, err := m.New(requestWithSessionID(t, m, "sess", NewId()), "sess")
	require.Error(t, err)
	require.True(t, s.IsNew)
	require.Empty(t, s.ID)
}

func TestNew_CopiesAllStoreOptions(t *testing.T) {
	m := newTestStore(t, nil)
	m.Options.SameSite = http.SameSiteStrictMode
	m.Options.Partitioned = true
	s, err := m.New(httptest.NewRequest(http.MethodGet, "/", nil), "sess")
	require.NoError(t, err)
	require.Equal(t, *m.Options, *s.Options)
	require.NotSame(t, m.Options, s.Options, "per-session options must be a copy")
}

func TestNewMongoStore_HardenedDefaults(t *testing.T) {
	m := newTestStore(t, nil)
	require.True(t, m.Options.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, m.Options.SameSite)
}
