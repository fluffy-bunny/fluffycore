package cookie_session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	contracts_sessions "github.com/fluffy-bunny/fluffycore/echo/contracts/sessions"
	gorilla_sessions "github.com/gorilla/sessions"
	echo "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

type accessor struct{ c *echo.Context }

func (a accessor) GetContext() *echo.Context { return a.c }

func newAccessor(r *http.Request) (accessor, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	return accessor{c: echo.New().NewContext(r, rec)}, rec
}

// cookieFrom returns a session cookie written by store.
func cookieFrom(t *testing.T, store *gorilla_sessions.CookieStore, name string) *http.Cookie {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s, err := store.New(r, name)
	require.NoError(t, err)
	s.Values["user"] = "alice"
	require.NoError(t, store.Save(r, rec, s))
	return rec.Result().Cookies()[0]
}

// A cookie signed with a rotated-out key (or expired, or tampered) used to make Initialize
// fail on every request, so the user was locked out until they cleared cookies.
func TestInitialize_UnreadableCookieStartsFreshSession(t *testing.T) {
	oldKeys := gorilla_sessions.NewCookieStore([]byte("old-key-old-key-old-key-old-key!"))
	current := gorilla_sessions.NewCookieStore([]byte("new-key-new-key-new-key-new-key!"))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookieFrom(t, oldKeys, "sess"))
	acc, rec := newAccessor(r)

	s, err := stemService.Ctor(current)
	require.NoError(t, err)
	require.NoError(t, s.Initialize(&contracts_sessions.InitializeRequest{Name: "sess", EchoContextAccessor: acc}))
	_, err = s.Get("user")
	require.Error(t, err, "nothing from the unreadable cookie may leak through")

	// Saving replaces the bad cookie with one the current keys can read.
	require.NoError(t, s.Set("user", "bob"))
	require.NoError(t, s.Save())
	next := httptest.NewRequest(http.MethodGet, "/", nil)
	next.AddCookie(rec.Result().Cookies()[0])
	fresh, err := current.Get(next, "sess")
	require.NoError(t, err)
	require.Equal(t, "bob", fresh.Values["user"])
}

type failingStore struct{ *gorilla_sessions.CookieStore }

func (f failingStore) Get(r *http.Request, name string) (*gorilla_sessions.Session, error) {
	s, _ := f.CookieStore.New(r, name)
	return s, errors.New("backend unavailable")
}

func TestInitialize_OtherStoreErrorsStillFail(t *testing.T) {
	s, err := stemService.Ctor(failingStore{gorilla_sessions.NewCookieStore([]byte("k-k-k-k-k-k-k-k-k-k-k-k-k-k-k-k!"))})
	require.NoError(t, err)
	acc, _ := newAccessor(httptest.NewRequest(http.MethodGet, "/", nil))
	require.Error(t, s.Initialize(&contracts_sessions.InitializeRequest{Name: "sess", EchoContextAccessor: acc}))
}
