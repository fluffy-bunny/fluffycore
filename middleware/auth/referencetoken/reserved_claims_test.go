package referencetoken

import (
	"context"
	"errors"
	"testing"
	"time"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	servicescache "github.com/fluffy-bunny/fluffycore/services/common/cache"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
)

var (
	mtlsVerifiedClaim = fluffycore_contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeMTLSVerified, Value: "true"}
	jwtSubClaim       = fluffycore_contracts_common.Claim{Type: "sub", Value: "jwt-user"}
	handlerClaim      = fluffycore_contracts_common.Claim{Type: "handler_added", Value: "x"}
	// connClaim stands in for a non-mtls claim an earlier interceptor put on the principal.
	connClaim = fluffycore_contracts_common.Claim{Type: "conn_claim", Value: "y"}
)

func cachedClaims(t *testing.T, cache fluffycore_contracts_common.ICache, rawToken string) []fluffycore_contracts_common.Claim {
	t.Helper()
	v, err := cache.Get(newEngine(testRootContainer).cacheKey(rawToken))
	if err != nil {
		return nil
	}
	claims, ok := v.([]fluffycore_contracts_common.Claim)
	require.True(t, ok)
	return claims
}

// claimAddingHandler simulates a handler (or a downstream interceptor) that adds claims to
// the request's principal -- those are not part of the token's resolution.
func claimAddingHandler(claims ...fluffycore_contracts_common.Claim) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		claimsPrincipalOf(ctx).AddClaim(claims...)
		return "ok", nil
	}
}

// The cache used to snapshot the WHOLE principal after the call: the first caller's
// connection-derived mtls_* claims (and anything the handler added) were cached under the
// token, then replayed to a later caller presenting the same token with no client cert.
func TestUnary_DelegateToJWT_CachesOnlyTheJWTsClaims(t *testing.T) {
	cache := servicescache.NewMemoryCache()
	// The "JWT validator" also tries to add a forged mtls claim -- it must not be cached either.
	forged := fluffycore_contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeMTLSCommonName, Value: "forged"}
	next := &recordingNext{claims: []fluffycore_contracts_common.Claim{jwtSubClaim, forged}}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithCache(cache))
	token := jwtRefPrefix + "reserved-claims-unary"

	// Caller 1: mTLS connection (the mtls interceptor already ran), handler adds a claim.
	ctx1 := newTestContext(t, "Bearer "+token)
	claimsPrincipalOf(ctx1).AddClaim(mtlsVerifiedClaim, connClaim)
	_, err := interceptor(ctx1, "req", &grpc.UnaryServerInfo{}, claimAddingHandler(handlerClaim))
	require.NoError(t, err)
	require.ElementsMatch(t, []fluffycore_contracts_common.Claim{jwtSubClaim}, cachedClaims(t, cache, token))

	// Caller 2: same token, no client cert. Cache hit must not hand it caller 1's mtls claim.
	ctx2 := newTestContext(t, "Bearer "+token)
	handler := &recordingHandler{}
	_, err = interceptor(ctx2, "req", &grpc.UnaryServerInfo{}, handler.handler())
	require.NoError(t, err)
	require.EqualValues(t, 1, next.calls, "second call is a cache hit")
	p := claimsPrincipalOf(ctx2)
	require.True(t, p.HasClaim(jwtSubClaim))
	require.False(t, p.HasClaimType(fluffycore_wellknown.ClaimTypeMTLSVerified))
	require.False(t, p.HasClaimType(fluffycore_wellknown.ClaimTypeMTLSCommonName))
	require.False(t, p.HasClaimType("handler_added"))
	require.False(t, p.HasClaimType("conn_claim"))
}

func TestUnary_DelegateToJWT_JWTRejected_NothingCached(t *testing.T) {
	cache := servicescache.NewMemoryCache()
	next := &recordingNext{failWith: errors.New("bad signature")}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithCache(cache))
	token := jwtRefPrefix + "reserved-claims-rejected"

	_, err := interceptor(newTestContext(t, "Bearer "+token), "req", &grpc.UnaryServerInfo{}, (&recordingHandler{}).handler())
	require.Error(t, err)
	require.Nil(t, cachedClaims(t, cache, token))
}

func TestUnary_ResolvedClaimsPrincipal_ReservedClaimsDropped(t *testing.T) {
	cache := servicescache.NewMemoryCache()
	interceptor := UnaryServerInterceptor(testRootContainer, (&recordingNext{}).interceptor(), WithCache(cache))
	token := mtlsClaimsRefPrefix + "x"

	ctx := newTestContext(t, "Bearer "+token)
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, (&recordingHandler{}).handler())
	require.NoError(t, err)
	p := claimsPrincipalOf(ctx)
	require.Equal(t, "resolved-user", p.GetClaimsByType("sub")[0].Value)
	require.False(t, p.HasClaimType(fluffycore_wellknown.ClaimTypeMTLSVerified), "a resolver must not be able to assert mtls_* claims")
	for _, c := range cachedClaims(t, cache, token) {
		require.NotEqual(t, fluffycore_wellknown.ClaimTypeMTLSVerified, c.Type)
	}
}

// A shared/persistent cache may already hold entries written by an older version.
func TestUnary_CacheHit_ReservedClaimsInEntryAreDropped(t *testing.T) {
	cache := servicescache.NewMemoryCache()
	token := claimsRefPrefix + "poisoned-entry"
	require.NoError(t, cache.SetWithTTL(newEngine(testRootContainer).cacheKey(token),
		[]fluffycore_contracts_common.Claim{jwtSubClaim, mtlsVerifiedClaim}, time.Minute))
	interceptor := UnaryServerInterceptor(testRootContainer, (&recordingNext{}).interceptor(), WithCache(cache))

	ctx := newTestContext(t, "Bearer "+token)
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, (&recordingHandler{}).handler())
	require.NoError(t, err)
	p := claimsPrincipalOf(ctx)
	require.True(t, p.HasClaim(jwtSubClaim))
	require.False(t, p.HasClaimType(fluffycore_wellknown.ClaimTypeMTLSVerified))
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *fakeServerStream) Context() context.Context { return s.ctx }

func TestStream_DelegateToJWT_CachesOnlyTheJWTsClaims(t *testing.T) {
	cache := servicescache.NewMemoryCache()
	next := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		claimsPrincipalOf(ss.Context()).AddClaim(jwtSubClaim)
		return handler(srv, ss)
	}
	interceptor := StreamServerInterceptor(testRootContainer, next, WithCache(cache))
	token := jwtRefPrefix + "reserved-claims-stream"

	ctx := newTestContext(t, "Bearer "+token)
	claimsPrincipalOf(ctx).AddClaim(mtlsVerifiedClaim, connClaim)
	err := interceptor(nil, &fakeServerStream{ctx: ctx}, &grpc.StreamServerInfo{}, func(srv interface{}, ss grpc.ServerStream) error {
		claimsPrincipalOf(ss.Context()).AddClaim(handlerClaim)
		return nil
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []fluffycore_contracts_common.Claim{jwtSubClaim}, cachedClaims(t, cache, token))
}
