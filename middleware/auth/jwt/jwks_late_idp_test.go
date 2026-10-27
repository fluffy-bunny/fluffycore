package jwt

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// lateIdP registers a JWKS URL while nothing listens on it, exactly as a server does when its own token
// service serves the JWKS (issuer http://localhost:9802, in the same process): Register runs at startup,
// before that listener is up. start brings the IdP up on the address that was registered.
type lateIdP struct {
	jwksURL string
	start   func()
	idp     *fakeIdP
}

func newLateIdP(t *testing.T, minRefresh time.Duration) *lateIdP {
	t.Helper()
	idp := newFakeIdP(t)
	key := newSigningKey(t, "kid-1")
	idp.publish(t, key)
	addr := idp.srv.Listener.Addr().String()
	handler := idp.srv.Config.Handler
	idp.srv.Close() // nothing listens now

	prevInterval := time.Duration(_minJWKSRefreshInterval.Load())
	SetMinJWKSRefreshInterval(minRefresh)
	t.Cleanup(func() {
		SetMinJWKSRefreshInterval(prevInterval)
		_validatorsOnce = sync.Once{}
		_validators = nil
	})
	_validatorsOnce = sync.Once{}
	_validators = nil

	jwksURL := "http://" + addr + "/jwks"
	b := di.Builder()
	fluffycore_services_common.AddCommonServices(b)
	AddValidators(b, &fluffycore_contracts_middleware_auth_jwt.IssuerConfigs{IssuerConfigs: []*fluffycore_contracts_middleware_auth_jwt.IssuerConfig{
		{OAuth2Config: &fluffycore_contracts_middleware_auth_jwt.OAuth2Config{Issuer: "http://" + addr, JWKSUrl: jwksURL}},
	}})
	b.Build()
	time.Sleep(300 * time.Millisecond) // the registration's own first fetch has failed by now

	return &lateIdP{jwksURL: jwksURL, idp: idp, start: func() {
		ln, err := net.Listen("tcp", addr)
		require.NoError(t, err)
		srv := &http.Server{Handler: handler}
		go func() { _ = srv.Serve(ln) }()
		t.Cleanup(func() { _ = srv.Close() })
	}}
}

// The keys were never fetched because the IdP was not up when the URL was registered. Once it is up the
// next request must get them: it used to wait 10s and fail with "context deadline exceeded", and every
// later request did the same, because the cache's own retry is minutes away.
func TestJWKS_IdPThatComesUpAfterRegistrationIsReachedByTheNextRequest(t *testing.T) {
	late := newLateIdP(t, 100*time.Millisecond)
	late.start()

	started := time.Now()
	set, ok, err := keySetForKid(context.Background(), late.jwksURL, "kid-1")
	require.NoError(t, err)
	require.True(t, ok, "the key must be found")
	require.NotNil(t, set)
	require.Less(t, time.Since(started), time.Second, "the first fetch had already failed, so there is nothing to wait for")

	// and it is cached now: the next request does not fetch again
	fetches := late.idp.fetches.Load()
	_, ok, err = keySetForKid(context.Background(), late.jwksURL, "kid-1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, fetches, late.idp.fetches.Load())
}

func TestJWKS_ABurstOfFirstRequestsAgainstALateIdPAllSucceedWithFewFetches(t *testing.T) {
	late := newLateIdP(t, time.Minute)
	late.start()

	const callers = 40
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := keySetForKid(context.Background(), late.jwksURL, "kid-1")
			if err == nil && !ok {
				err = context.DeadlineExceeded
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.LessOrEqual(t, late.idp.fetches.Load(), int64(3), "the burst must not turn into a fetch per request")
}

// An IdP that stays down: the first request pays the wait, the ones after it fail at once until the
// interval has passed, instead of each hanging.
func TestJWKS_IdPThatStaysDownFailsFastAfterTheFirstAttempt(t *testing.T) {
	late := newLateIdP(t, time.Minute) // never started

	_, _, err := keySetForKid(context.Background(), late.jwksURL, "kid-1")
	require.Error(t, err)

	started := time.Now()
	_, _, err = keySetForKid(context.Background(), late.jwksURL, "kid-1")
	require.Error(t, err)
	require.Less(t, time.Since(started), 500*time.Millisecond, "a later request must not wait again")
	require.Contains(t, err.Error(), "not available")
}

// How long a fetch took, and why it was needed, is logged: a slow or failing IdP is what makes a request slow.
func TestJWKS_FetchesAreLoggedWithTheirDuration(t *testing.T) {
	late := newLateIdP(t, 100*time.Millisecond)
	late.start()

	var buf bytes.Buffer
	ctx := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(context.Background())

	_, ok, err := keySetForKid(ctx, late.jwksURL, "kid-1")
	require.NoError(t, err)
	require.True(t, ok)

	var entry map[string]interface{}
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'}) {
		var e map[string]interface{}
		require.NoError(t, json.Unmarshal(line, &e))
		if e["message"] == "JWKS fetched" {
			entry = e
		}
	}
	require.NotNil(t, entry, "no 'JWKS fetched' line in: %s", buf.String())
	require.Equal(t, "first load", entry["reason"])
	require.Equal(t, late.jwksURL, entry["jwks_url"])
	require.EqualValues(t, 1, entry["keys"])
	require.Contains(t, entry, "took", "the duration must be logged")

	// a failing fetch is logged too, at warn, with its duration
	down := newLateIdP(t, time.Minute) // never started
	buf.Reset()
	_, _, err = keySetForKid(ctx, down.jwksURL, "kid-1")
	require.Error(t, err)
	require.Contains(t, buf.String(), "JWKS fetch failed")
	require.Contains(t, buf.String(), "took")
}
