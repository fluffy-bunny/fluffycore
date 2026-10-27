package referencetoken

import (
	"sync/atomic"
	"testing"
	"time"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	servicescache "github.com/fluffy-bunny/fluffycore/services/common/cache"
	jwa "github.com/lestrrat-go/jwx/v4/jwa"
	jwxt "github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
)

func signedJWT(t *testing.T, exp *time.Time) string {
	t.Helper()
	b := jwxt.NewBuilder().Subject("u")
	if exp != nil {
		b = b.Expiration(*exp)
	}
	tok, err := b.Build()
	require.NoError(t, err)
	signed, err := jwxt.Sign(tok, jwxt.WithKey(jwa.HS256(), []byte("0123456789abcdef0123456789abcdef")))
	require.NoError(t, err)
	return string(signed)
}

func TestCapTTLToJWTExpiry(t *testing.T) {
	soon := time.Now().Add(2 * time.Second)
	far := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Minute)

	got, ok := capTTLToJWTExpiry(5*time.Minute, signedJWT(t, &soon))
	require.True(t, ok)
	require.Greater(t, got, time.Duration(0))
	require.LessOrEqual(t, got, 2*time.Second, "the cache must not outlive the JWT")

	got, ok = capTTLToJWTExpiry(5*time.Minute, signedJWT(t, &far))
	require.True(t, ok)
	require.Equal(t, 5*time.Minute, got, "a longer-lived JWT keeps the configured TTL")

	_, ok = capTTLToJWTExpiry(5*time.Minute, signedJWT(t, &past))
	require.False(t, ok, "an already-expired JWT must not be cached")

	got, ok = capTTLToJWTExpiry(5*time.Minute, signedJWT(t, nil))
	require.True(t, ok)
	require.Equal(t, 5*time.Minute, got, "no exp: nothing to cap against")

	got, ok = capTTLToJWTExpiry(5*time.Minute, "not-a-jwt")
	require.True(t, ok)
	require.Equal(t, 5*time.Minute, got)
}

// A resolution cached for the default 5 minutes used to keep authenticating after the JWT it
// resolved to had expired, because a cache hit skips JWT validation.
func TestUnary_JWTKindCacheDoesNotOutliveTheJWT(t *testing.T) {
	cache := servicescache.NewMemoryCache()
	t.Cleanup(func() { _ = cache.Close() })
	next := &recordingNext{claims: []fluffycore_contracts_common.Claim{{Type: "sub", Value: "jwt-user"}}}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithCache(cache), WithKnownPrefixes(jwtRefPrefix))
	// JWT NumericDate has one-second resolution; use a whole-second expiry 1-2s out so the first calls are
	// unambiguously before it.
	exp := time.Now().Add(2 * time.Second).Truncate(time.Second)
	token := "Bearer " + jwtRefPrefix + signedJWT(t, &exp)
	call := func() {
		t.Helper()
		_, err := interceptor(newTestContext(t, token), "req", &grpc.UnaryServerInfo{}, (&recordingHandler{}).handler())
		require.NoError(t, err)
	}

	before := atomic.LoadInt32(&theStubResolver.calls)
	call()
	call()
	require.EqualValues(t, before+1, atomic.LoadInt32(&theStubResolver.calls), "second call is a cache hit")

	time.Sleep(time.Until(exp) + 100*time.Millisecond) // the JWT has expired; the default TTL (5m) has not
	call()
	require.EqualValues(t, before+2, atomic.LoadInt32(&theStubResolver.calls), "after the JWT's exp the resolution must be redone, not served from cache")
}
