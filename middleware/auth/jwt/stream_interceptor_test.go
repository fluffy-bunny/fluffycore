package jwt

import (
	"context"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
)

var streamTestRoot = func() di.Container {
	b := di.Builder()
	fluffycore_services_common_claimsprincipal.AddClaimsPrincipal(b)
	return b.Build()
}()

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeStream) Context() context.Context { return f.ctx }

// streamCtx builds a request-scoped context with incoming metadata, as the real server
// (which always supplies incoming metadata) plus dicontext would.
func streamCtx(t *testing.T, authorization string) (context.Context, di.Container) {
	t.Helper()
	scope := di.Get[di.ScopeFactory](streamTestRoot).CreateScope()
	t.Cleanup(scope.Dispose)
	md := metadata.MD{}
	if authorization != "" {
		md = metadata.Pairs(fluffycore_wellknown.MetadataKeyAuthorization, authorization)
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)
	return dicontext.SetRequestContainer(ctx, scope.Container()), scope.Container()
}

func runStream(ctx context.Context, opts ...ValidationOption) (reached bool, err error) {
	err = StreamServerInterceptor(streamTestRoot, opts...)(nil, &fakeStream{ctx: ctx},
		&grpc.StreamServerInfo{FullMethod: "/test.Svc/Stream"},
		func(srv interface{}, ss grpc.ServerStream) error { reached = true; return nil })
	return reached, err
}

func isAnonymous(c di.Container) bool {
	return di.Get[fluffycore_contracts_common.IClaimsPrincipal](c).HasClaim(fluffycore_contracts_common.Claim{
		Type: fluffycore_wellknown.ClaimTypeSub, Value: fluffycore_wellknown.AnonymousSubject,
	})
}

// A stream with no Authorization header must proceed as anonymous -- the same as the unary
// interceptor -- so the claims-principal gate decides. It used to be rejected with NotFound.
func TestStreamInterceptor_NoToken_ProceedsAsAnonymous(t *testing.T) {
	ctx, c := streamCtx(t, "")
	reached, err := runStream(ctx)
	require.NoError(t, err)
	require.True(t, reached)
	require.True(t, isAnonymous(c))
}

func TestStreamInterceptor_InvalidToken_Rejected(t *testing.T) {
	ctx, _ := streamCtx(t, "Bearer not-a-jwt")
	reached, err := runStream(ctx)
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.False(t, reached)
}

func TestStreamInterceptor_InvalidToken_AnonymousOnFailure(t *testing.T) {
	ctx, c := streamCtx(t, "Bearer not-a-jwt")
	reached, err := runStream(ctx, WithAnonymousOnFailure())
	require.NoError(t, err)
	require.True(t, reached)
	require.True(t, isAnonymous(c))
}

// Previously this nil-dereferenced inside di.Get and took down the process (there was no
// stream recovery interceptor either).
func TestStreamInterceptor_NoRequestContainer_Internal(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	reached, err := runStream(ctx)
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
	require.False(t, reached)
}
