package secure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	contracts_cookies "github.com/fluffy-bunny/fluffycore/echo/contracts/cookies"
	echo "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func schemeOf(r *http.Request) string {
	return echo.New().NewContext(r, httptest.NewRecorder()).Scheme()
}

// Behind a TLS-terminating proxy r.TLS is nil; cookies used to be set without Secure.
func TestSecureDefault_HonorsForwardedProto(t *testing.T) {
	plain := httptest.NewRequest(http.MethodGet, "http://app/", nil)
	require.False(t, secureDefault(schemeOf(plain), plain, nil))

	proxied := httptest.NewRequest(http.MethodGet, "http://app/", nil)
	proxied.Header.Set("X-Forwarded-Proto", "https")
	require.True(t, secureDefault(schemeOf(proxied), proxied, nil))

	no := false
	require.False(t, secureDefault(schemeOf(proxied), proxied, &no), "explicit override wins")
}

func TestDeleteCookie_DomainOptional(t *testing.T) {
	s := &service{}
	require.NoError(t, s.validateDeleteCookieRequest(&contracts_cookies.DeleteCookieRequest{Name: "c", Path: "/"}))
	require.Error(t, s.validateDeleteCookieRequest(&contracts_cookies.DeleteCookieRequest{Path: "/"}))
}
