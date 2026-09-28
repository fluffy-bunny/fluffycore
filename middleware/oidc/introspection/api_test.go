package introspection

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// introspectionServer answers with a fixed status/body and records what it received.
type introspectionServer struct {
	status int
	body   string
	delay  time.Duration

	gotContentType string
	gotUser        string
	gotPass        string
	gotBasic       bool
	gotForm        url.Values
}

func (s *introspectionServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.gotContentType = r.Header.Get("Content-Type")
		s.gotUser, s.gotPass, s.gotBasic = r.BasicAuth()
		b, _ := io.ReadAll(r.Body)
		s.gotForm, _ = url.ParseQuery(string(b))
		if s.delay > 0 {
			select {
			case <-time.After(s.delay):
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(s.status)
		_, _ = io.WriteString(w, s.body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func opts(url string) IntrospectionValidationOptions {
	return IntrospectionValidationOptions{
		IntrospectionURL: url,
		ClientID:         "client:1",
		ClientSecret:     "s3cr&t",
		Token:            "the-token",
		// Logger deliberately nil: this used to panic on the first log call.
	}
}

func TestValidateToken_ActiveTokenAccepted(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `{"active":true,"sub":"u1","scope":"read write","aud":["a","b"]}`}
	srv := s.start(t)

	resp, err := ValidateTokenWithContext(context.Background(), opts(srv.URL))
	require.NoError(t, err)
	require.True(t, resp.Active)
	require.Equal(t, "u1", resp.Sub)
	require.Equal(t, "read write", resp.Scope)
	require.NoError(t, ValidateToken(opts(srv.URL)))
}

// The core bug: RFC 7662 returns 200 for invalid/expired/revoked tokens, with
// active=false. ValidateToken used to accept all of them.
func TestValidateToken_InactiveTokenRejected(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `{"active":false}`}
	srv := s.start(t)
	require.Error(t, ValidateToken(opts(srv.URL)))
}

func TestValidateToken_MissingActiveRejected(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `{"sub":"u1"}`}
	srv := s.start(t)
	require.Error(t, ValidateToken(opts(srv.URL)))
}

func TestValidateToken_UnparseableBodyRejected(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `not json`}
	srv := s.start(t)
	require.Error(t, ValidateToken(opts(srv.URL)))
}

func TestValidateToken_NonOKRejected(t *testing.T) {
	s := &introspectionServer{status: http.StatusUnauthorized, body: `{"error":"invalid_client"}`}
	srv := s.start(t)
	require.Error(t, ValidateToken(opts(srv.URL)))
}

func TestValidateToken_SendsBasicAuthAndForm(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `{"active":true}`}
	srv := s.start(t)
	require.NoError(t, ValidateToken(opts(srv.URL)))

	require.Equal(t, "application/x-www-form-urlencoded", s.gotContentType)
	require.True(t, s.gotBasic, "client credentials go in HTTP Basic auth")
	// RFC 6749 §2.3.1: form-urlencoded before base64, so the server must decode them.
	user, _ := url.QueryUnescape(s.gotUser)
	pass, _ := url.QueryUnescape(s.gotPass)
	require.Equal(t, "client:1", user)
	require.Equal(t, "s3cr&t", pass)
	require.Equal(t, "the-token", s.gotForm.Get("token"))
	require.Empty(t, s.gotForm.Get("client_secret"), "secret must not also be sent in the body")
}

// A hung endpoint must not block forever: the old code used http.Client{} with no timeout.
func TestValidateToken_TimesOut(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `{"active":true}`, delay: 5 * time.Second}
	srv := s.start(t)
	o := opts(srv.URL)
	o.HTTPClient = &http.Client{Timeout: 100 * time.Millisecond}

	start := time.Now()
	require.Error(t, ValidateToken(o))
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestValidateToken_HonoursContextCancellation(t *testing.T) {
	s := &introspectionServer{status: http.StatusOK, body: `{"active":true}`, delay: 5 * time.Second}
	srv := s.start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := ValidateTokenWithContext(ctx, opts(srv.URL))
	require.Error(t, err)
}
