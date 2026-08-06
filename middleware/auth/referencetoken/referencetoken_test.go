package referencetoken

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_middleware_auth_referencetoken "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/referencetoken"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	servicescache "github.com/fluffy-bunny/fluffycore/services/common/cache"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	status "github.com/gogo/status"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
)

// stubResolver recognizes two opaque token shapes so tests can exercise both
// ResolvedKind branches, plus a magic "error-token" for the hard-failure path.
// It is registered exactly once, in the package-level test root container below --
// UnaryServerInterceptor/StreamServerInterceptor memoize the resolver list for the
// lifetime of the process (mirroring middleware/auth/jwt's _loadValidators), so
// every test in this file shares one resolver set and tracks its own call count
// via before/after deltas.
type stubResolver struct {
	calls int32
}

const (
	jwtRefPrefix    = "jwt-ref:"
	claimsRefPrefix = "claims-ref:"
	errorToken      = "error-token"
)

func (r *stubResolver) Resolve(ctx context.Context, rawToken string) (bool, *fluffycore_contracts_middleware_auth_referencetoken.Resolved, error) {
	atomic.AddInt32(&r.calls, 1)
	switch {
	case rawToken == errorToken:
		return true, nil, status.Error(codes.Unauthenticated, "token revoked")
	case strings.HasPrefix(rawToken, jwtRefPrefix):
		return true, &fluffycore_contracts_middleware_auth_referencetoken.Resolved{
			Kind:   fluffycore_contracts_middleware_auth_referencetoken.ResolvedKindJWT,
			RawJWT: strings.TrimPrefix(rawToken, jwtRefPrefix),
		}, nil
	case strings.HasPrefix(rawToken, claimsRefPrefix):
		return true, &fluffycore_contracts_middleware_auth_referencetoken.Resolved{
			Kind: fluffycore_contracts_middleware_auth_referencetoken.ResolvedKindClaimsPrincipal,
			Claims: map[string]interface{}{
				"sub": "resolved-user",
			},
		}, nil
	default:
		return false, nil, nil
	}
}

var theStubResolver = &stubResolver{}

var testRootContainer = func() di.Container {
	builder := di.Builder()
	fluffycore_services_common_claimsprincipal.AddClaimsPrincipal(builder)
	AddResolver(builder, func() fluffycore_contracts_middleware_auth_referencetoken.IResolver {
		return theStubResolver
	})
	return builder.Build()
}()

// newTestContext builds a fresh request-scoped di.Container (as
// dicontext.ScopedContextUnaryServerInterceptor would) plus an incoming context
// carrying authHeader (verbatim, e.g. "Bearer sometoken"), or no authorization
// metadata at all when authHeader == "".
func newTestContext(t *testing.T, authHeader string) context.Context {
	t.Helper()
	scopeFactory := di.Get[di.ScopeFactory](testRootContainer)
	scope := scopeFactory.CreateScope()
	t.Cleanup(scope.Dispose)

	ctx := dicontext.SetRequestContainer(context.Background(), scope.Container())
	md := metadata.MD{}
	if authHeader != "" {
		md = metadata.New(map[string]string{"authorization": authHeader})
	}
	return metadata.NewIncomingContext(ctx, md)
}

func claimsPrincipalOf(ctx context.Context) fluffycore_contracts_common.IClaimsPrincipal {
	container := dicontext.GetRequestContainer(ctx)
	return di.Get[fluffycore_contracts_common.IClaimsPrincipal](container)
}

// recordingNext builds a stand-in for the downstream JWT validator interceptor.
// When told to succeed it adds claims (simulating a JWT validator populating the
// scoped IClaimsPrincipal) and calls through to handler; it always records the
// context it was invoked with and how many times it was called.
type recordingNext struct {
	calls    int32
	lastCtx  context.Context
	claims   []fluffycore_contracts_common.Claim
	failWith error
}

func (n *recordingNext) interceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		atomic.AddInt32(&n.calls, 1)
		n.lastCtx = ctx
		if n.failWith != nil {
			return nil, n.failWith
		}
		if len(n.claims) > 0 {
			claimsPrincipalOf(ctx).AddClaim(n.claims...)
		}
		return handler(ctx, req)
	}
}

type recordingHandler struct {
	calls   int32
	lastCtx context.Context
}

func (h *recordingHandler) handler() grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		atomic.AddInt32(&h.calls, 1)
		h.lastCtx = ctx
		return "ok", nil
	}
}

func TestUnary_NoAuthorizationHeader_PassesThroughToNext(t *testing.T) {
	next := &recordingNext{}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor())

	ctx := newTestContext(t, "")
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler.handler())

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 1, next.calls)
	require.EqualValues(t, 1, handler.calls)
}

func TestUnary_OrdinaryJWT_PassesThroughUnchanged(t *testing.T) {
	before := atomic.LoadInt32(&theStubResolver.calls)

	next := &recordingNext{}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor())

	// Three dot-separated segments -- looks like a JWT, so resolvers must not run.
	ctx := newTestContext(t, "Bearer header.payload.signature")
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler.handler())

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 1, next.calls)
	require.EqualValues(t, before, atomic.LoadInt32(&theStubResolver.calls), "resolvers must not be consulted for JWT-shaped tokens")

	md, ok := metadata.FromIncomingContext(next.lastCtx)
	require.True(t, ok)
	require.Equal(t, "Bearer header.payload.signature", md.Get("authorization")[0], "authorization header must be untouched for ordinary JWTs")
}

func TestUnary_UnrecognizedOpaqueToken_FailsAuthentication(t *testing.T) {
	next := &recordingNext{}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor())

	ctx := newTestContext(t, "Bearer totally-opaque-nobody-recognizes-this")
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler.handler())

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.EqualValues(t, 0, next.calls)
	require.EqualValues(t, 0, handler.calls)
}

func TestUnary_ResolverError_FailsAuthenticationWithoutFallthrough(t *testing.T) {
	next := &recordingNext{}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor())

	ctx := newTestContext(t, "Bearer "+errorToken)
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler.handler())

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Contains(t, st.Message(), "revoked")
	require.EqualValues(t, 0, next.calls)
	require.EqualValues(t, 0, handler.calls)
}

func TestUnary_ResolvedKindJWT_DelegatesToNextAndCaches(t *testing.T) {
	before := atomic.LoadInt32(&theStubResolver.calls)
	cache := servicescache.NewMemoryCache()

	next := &recordingNext{claims: []fluffycore_contracts_common.Claim{{Type: "sub", Value: "jwt-user"}}}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithCache(cache))

	// First request: cache miss -- resolver runs, next (the JWT validator stand-in)
	// is invoked with the resolved JWT swapped into the authorization header.
	ctx1 := newTestContext(t, "Bearer "+jwtRefPrefix+"the-real-jwt")
	resp, err := interceptor(ctx1, "req", &grpc.UnaryServerInfo{}, handler.handler())
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 1, next.calls)
	require.EqualValues(t, before+1, atomic.LoadInt32(&theStubResolver.calls))

	md, ok := metadata.FromIncomingContext(next.lastCtx)
	require.True(t, ok)
	require.Equal(t, "Bearer the-real-jwt", md.Get("authorization")[0], "next must see the resolved JWT, not the opaque reference token")

	require.Equal(t, "jwt-user", claimsPrincipalOf(ctx1).GetClaimsByType("sub")[0].Value)

	// Second request, same reference token: cache hit -- next (and the resolver)
	// must NOT run again, yet the claims still land on the (fresh) scoped principal.
	ctx2 := newTestContext(t, "Bearer "+jwtRefPrefix+"the-real-jwt")
	resp, err = interceptor(ctx2, "req", &grpc.UnaryServerInfo{}, handler.handler())
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 1, next.calls, "next must not be called again on a cache hit")
	require.EqualValues(t, before+1, atomic.LoadInt32(&theStubResolver.calls), "resolver must not be called again on a cache hit")
	require.Equal(t, "jwt-user", claimsPrincipalOf(ctx2).GetClaimsByType("sub")[0].Value)

	md2, ok := metadata.FromIncomingContext(handler.lastCtx)
	require.True(t, ok)
	require.Empty(t, md2.Get("authorization"), "the reference token must be stripped from metadata on a cache hit")
}

func TestUnary_ResolvedKindClaimsPrincipal_SkipsNextAndCaches(t *testing.T) {
	before := atomic.LoadInt32(&theStubResolver.calls)
	cache := servicescache.NewMemoryCache()

	next := &recordingNext{}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithCache(cache))

	ctx1 := newTestContext(t, "Bearer "+claimsRefPrefix+"whatever")
	resp, err := interceptor(ctx1, "req", &grpc.UnaryServerInfo{}, handler.handler())
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 0, next.calls, "the JWT validator must be skipped entirely for a direct claims resolution")
	require.EqualValues(t, 1, handler.calls)
	require.EqualValues(t, before+1, atomic.LoadInt32(&theStubResolver.calls))
	require.Equal(t, "resolved-user", claimsPrincipalOf(ctx1).GetClaimsByType("sub")[0].Value)

	md, ok := metadata.FromIncomingContext(handler.lastCtx)
	require.True(t, ok)
	require.Empty(t, md.Get("authorization"), "the opaque reference token must never reach the real handler")

	// Second request, same reference token: cache hit -- resolver must not run again.
	ctx2 := newTestContext(t, "Bearer "+claimsRefPrefix+"whatever")
	resp, err = interceptor(ctx2, "req", &grpc.UnaryServerInfo{}, handler.handler())
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 0, next.calls)
	require.EqualValues(t, before+1, atomic.LoadInt32(&theStubResolver.calls), "resolver must not be called again on a cache hit")
	require.Equal(t, "resolved-user", claimsPrincipalOf(ctx2).GetClaimsByType("sub")[0].Value)
}

func TestUnary_WithKnownPrefixes_NonMatchingTokenPassesThroughEvenIfOpaque(t *testing.T) {
	before := atomic.LoadInt32(&theStubResolver.calls)

	next := &recordingNext{}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithKnownPrefixes(jwtRefPrefix, claimsRefPrefix))

	// No dots at all -- under the structural fallback this would be routed to
	// resolvers and fail as "unrecognized". With known prefixes configured it
	// matches neither, so it must be treated as an ordinary bearer credential
	// and passed straight through instead.
	ctx := newTestContext(t, "Bearer some-opaque-uuid-1234")
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler.handler())

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 1, next.calls)
	require.EqualValues(t, before, atomic.LoadInt32(&theStubResolver.calls), "resolvers must not run for a token matching no known prefix")
}

func TestUnary_WithKnownPrefixes_MatchingTokenIsResolvedEvenIfJWTShaped(t *testing.T) {
	// A reference token that, purely by coincidence, has two dots -- under the
	// structural fallback this would be misrouted as an ordinary JWT and passed
	// straight through unresolved. With a known prefix configured, the prefix
	// wins regardless of shape.
	next := &recordingNext{claims: []fluffycore_contracts_common.Claim{{Type: "sub", Value: "jwt-user"}}}
	handler := &recordingHandler{}
	interceptor := UnaryServerInterceptor(testRootContainer, next.interceptor(), WithKnownPrefixes(jwtRefPrefix, claimsRefPrefix))

	ctx := newTestContext(t, "Bearer "+jwtRefPrefix+"a.b.c")
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler.handler())

	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	require.EqualValues(t, 1, next.calls)

	md, ok := metadata.FromIncomingContext(next.lastCtx)
	require.True(t, ok)
	require.Equal(t, "Bearer a.b.c", md.Get("authorization")[0], "the prefix match must route to the resolver even though the token happens to look JWT-shaped")
}

func TestLooksLikeJWT(t *testing.T) {
	require.True(t, looksLikeJWT("header.payload.signature"))
	require.False(t, looksLikeJWT("opaque-token-no-dots"))
	require.False(t, looksLikeJWT("only.onedot"))
}

func TestCacheKey_StableAndDistinct(t *testing.T) {
	e := newEngine(testRootContainer)
	require.Equal(t, e.cacheKey("token-a"), e.cacheKey("token-a"))
	require.NotEqual(t, e.cacheKey("token-a"), e.cacheKey("token-b"))
}
