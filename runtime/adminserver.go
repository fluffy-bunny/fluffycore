package runtime

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// The control (CONTROL_PORT) and pprof (PPROF_PORT) servers are operator tools: /stop
// shuts the process down and /debug/pprof exposes heap contents and the command line.
// They are therefore locked down by default:
//
//   - They bind to loopback (127.0.0.1) unless ADMIN_BIND_ADDRESS says otherwise.
//     `kubectl port-forward` connects inside the pod, so loopback still works for it.
//   - Binding to anything other than loopback requires ADMIN_TOKEN to be set; without it
//     the server refuses to start rather than exposing unauthenticated endpoints.
//   - When ADMIN_TOKEN is set, every request must send `Authorization: Bearer <token>`.
const (
	envAdminBindAddress = "ADMIN_BIND_ADDRESS"
	envAdminToken       = "ADMIN_TOKEN"
	adminLoopback       = "127.0.0.1"
)

// adminListenConfig resolves where an admin server may listen and what token it requires.
// getenv is injected so this can be tested without touching the process environment.
func adminListenConfig(getenv func(string) string, port int) (addr string, token string, err error) {
	host := strings.TrimSpace(getenv(envAdminBindAddress))
	if host == "" {
		host = adminLoopback
	}
	token = getenv(envAdminToken)
	if !isLoopbackHost(host) && token == "" {
		return "", "", fmt.Errorf("%s=%q is not loopback; refusing to expose admin endpoints without %s", envAdminBindAddress, host, envAdminToken)
	}
	return net.JoinHostPort(host, fmt.Sprint(port)), token, nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// adminTokenMiddleware requires `Authorization: Bearer <token>` on every request when token
// is non-empty. The comparison is constant-time.
func adminTokenMiddleware(token string) echo.MiddlewareFunc {
	want := []byte("Bearer " + token)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if token == "" {
				return next(c)
			}
			got := []byte(c.Request().Header.Get("Authorization"))
			if subtle.ConstantTimeCompare(got, want) != 1 {
				return c.String(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}
