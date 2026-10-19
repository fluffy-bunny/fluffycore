package storeopts

import (
	"net/http"
	"testing"
	"time"

	contracts_sessions "github.com/fluffy-bunny/fluffycore/echo/contracts/sessions"
	securecookie "github.com/gorilla/securecookie"
	gorilla_sessions "github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"
)

func TestApply_HardenedDefaults(t *testing.T) {
	store := gorilla_sessions.NewCookieStore([]byte("0123456789abcdef0123456789abcdef"))
	require.Equal(t, http.SameSiteNoneMode, store.Options.SameSite, "gorilla's own default, which we override")

	Apply(store.Options, store.Codecs, &contracts_sessions.SessionConfig{Domain: "example.com", MaxAge: 3600})
	require.True(t, store.Options.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, store.Options.SameSite)
	require.True(t, store.Options.Secure)
	require.Equal(t, "example.com", store.Options.Domain)
	require.Equal(t, 3600, store.Options.MaxAge)
}

func TestApply_InsecureAndSameSiteOverride(t *testing.T) {
	store := gorilla_sessions.NewCookieStore([]byte("0123456789abcdef0123456789abcdef"))
	Apply(store.Options, store.Codecs, &contracts_sessions.SessionConfig{Insecure: true, SameSite: http.SameSiteStrictMode})
	require.False(t, store.Options.Secure)
	require.True(t, store.Options.HttpOnly, "HttpOnly even in insecure (local dev) mode")
	require.Equal(t, http.SameSiteStrictMode, store.Options.SameSite)
}

// MaxAge 0 used to call securecookie.MaxAge(0), which disables the timestamp check: a signed
// session cookie captured once was accepted forever.
func TestSetMaxAge_ZeroStillExpiresSignedValue(t *testing.T) {
	prev := sessionCookieValidity
	sessionCookieValidity = 1
	t.Cleanup(func() { sessionCookieValidity = prev })

	codecs := securecookie.CodecsFromPairs([]byte("0123456789abcdef0123456789abcdef"))
	opts := &gorilla_sessions.Options{}
	SetMaxAge(opts, codecs, 0)
	require.Equal(t, 0, opts.MaxAge, "the cookie itself stays a browser-session cookie")

	encoded, err := securecookie.EncodeMulti("s", "session-id", codecs...)
	require.NoError(t, err)
	var id string
	require.NoError(t, securecookie.DecodeMulti("s", encoded, &id, codecs...))

	time.Sleep(2100 * time.Millisecond) // timestamps have 1s resolution
	require.Error(t, securecookie.DecodeMulti("s", encoded, &id, codecs...), "an old signed value must be rejected")
}
