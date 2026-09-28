package runtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func env(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

func TestAdminListenConfig(t *testing.T) {
	cases := []struct {
		desc     string
		env      map[string]string
		wantAddr string
		wantErr  bool
	}{
		// The old default was "" (all interfaces) unless DEV_BIND_LOCALHOST was set.
		{"defaults to loopback", nil, "127.0.0.1:9000", false},
		{"explicit loopback ok without a token", map[string]string{envAdminBindAddress: "127.0.0.1"}, "127.0.0.1:9000", false},
		{"localhost ok without a token", map[string]string{envAdminBindAddress: "localhost"}, "localhost:9000", false},
		{"ipv6 loopback ok without a token", map[string]string{envAdminBindAddress: "::1"}, "[::1]:9000", false},
		{"all interfaces refused without a token", map[string]string{envAdminBindAddress: "0.0.0.0"}, "", true},
		{"pod IP refused without a token", map[string]string{envAdminBindAddress: "10.1.2.3"}, "", true},
		{"all interfaces allowed with a token", map[string]string{envAdminBindAddress: "0.0.0.0", envAdminToken: "s3cret"}, "0.0.0.0:9000", false},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			addr, _, err := adminListenConfig(env(tc.env), 9000)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAddr, addr)
		})
	}
}

func do(t *testing.T, h http.Handler, method, path, authorization string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestControlServer_StopRequiresPOST(t *testing.T) {
	rt := NewRuntime()
	e := NewControl(rt).newEcho("")

	// GET used to shut the process down; a link or prefetch could trigger it.
	require.Equal(t, http.StatusMethodNotAllowed, do(t, e, http.MethodGet, "/stop", ""))
	require.Len(t, rt.waitChannel, 0, "GET must not signal a stop")

	require.Equal(t, http.StatusOK, do(t, e, http.MethodPost, "/stop", ""))
	require.Len(t, rt.waitChannel, 1, "POST signals a stop")
}

func TestControlServer_TokenEnforced(t *testing.T) {
	rt := NewRuntime()
	e := NewControl(rt).newEcho("s3cret")

	require.Equal(t, http.StatusUnauthorized, do(t, e, http.MethodPost, "/stop", ""))
	require.Equal(t, http.StatusUnauthorized, do(t, e, http.MethodPost, "/stop", "Bearer wrong"))
	require.Equal(t, http.StatusUnauthorized, do(t, e, http.MethodPost, "/stop", "s3cret"), "scheme is required")
	require.Len(t, rt.waitChannel, 0, "rejected requests must not signal a stop")

	require.Equal(t, http.StatusOK, do(t, e, http.MethodPost, "/stop", "Bearer s3cret"))
	require.Len(t, rt.waitChannel, 1)
}

// Stop used to be a blocking send on a 1-slot channel that is read once, so a second
// /stop wedged its handler goroutine forever.
func TestRuntimeStop_NeverBlocks(t *testing.T) {
	rt := NewRuntime()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			rt.Stop()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Runtime.Stop blocked on repeated calls")
	}
	require.Len(t, rt.waitChannel, 1, "one pending stop is enough")
}
