package claimsprincipal

import (
	"context"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	claimsprincipalContracts "github.com/fluffy-bunny/fluffycore/contracts/common"
	middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	services_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	status "github.com/gogo/status"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
)

const (
	openMethod      = "/test.Svc/Open"
	protectedMethod = "/test.Svc/Protected"
	unlistedMethod  = "/test.Svc/Unlisted"
)

var gateRoot = func() di.Container {
	b := di.Builder()
	services_claimsprincipal.AddClaimsPrincipal(b)
	return b.Build()
}()

func gateMap() map[string]claimsprincipalContracts.IEntryPointConfig {
	builder := services_claimsprincipal.NewEntryPointClaimsBuilder()
	builder.WithGrpcEntrypointPermissionsClaimsMapOpen(openMethod)
	builder.WithGrpcEntrypointClams(protectedMethod,
		services_claimsprincipal.NewClaimFact(claimsprincipalContracts.Claim{Type: "permissions", Value: "write"}))
	return builder.GetEntryPointClaimsMap()
}

// requestCtx returns a context carrying a fresh request scope, with the given claims
// already on its IClaimsPrincipal (as the JWT/mtls interceptors would have added).
func requestCtx(t *testing.T, claims ...claimsprincipalContracts.Claim) context.Context {
	t.Helper()
	scope := di.Get[di.ScopeFactory](gateRoot).CreateScope()
	t.Cleanup(scope.Dispose)
	di.Get[claimsprincipalContracts.IClaimsPrincipal](scope.Container()).AddClaim(claims...)
	return middleware_dicontext.SetRequestContainer(context.Background(), scope.Container())
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeStream) Context() context.Context { return f.ctx }

// gate is one entry point implemented both ways, so every case below runs against the
// unary and the stream interceptor and asserts they reach the same decision.
type gate struct {
	name string
	call func(ctx context.Context, method string, m map[string]claimsprincipalContracts.IEntryPointConfig, zeroTrust bool) (reachedHandler bool, err error)
}

var gates = []gate{
	{"unary", func(ctx context.Context, method string, m map[string]claimsprincipalContracts.IEntryPointConfig, zeroTrust bool) (bool, error) {
		reached := false
		_, err := FinalAuthVerificationMiddlewareUsingClaimsMapWithTrustOptionV2(m, zeroTrust)(ctx, nil,
			&grpc.UnaryServerInfo{FullMethod: method},
			func(ctx context.Context, req interface{}) (interface{}, error) { reached = true; return nil, nil })
		return reached, err
	}},
	{"stream", func(ctx context.Context, method string, m map[string]claimsprincipalContracts.IEntryPointConfig, zeroTrust bool) (bool, error) {
		reached := false
		err := FinalAuthVerificationStreamMiddlewareUsingClaimsMapWithTrustOptionV2(m, zeroTrust)(nil,
			&fakeStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: method},
			func(srv interface{}, ss grpc.ServerStream) error { reached = true; return nil })
		return reached, err
	}},
}

func TestFinalAuthGate(t *testing.T) {
	write := claimsprincipalContracts.Claim{Type: "permissions", Value: "write"}
	cases := []struct {
		desc      string
		ctx       func(t *testing.T) context.Context
		method    string
		zeroTrust bool
		wantCode  codes.Code // codes.OK means the handler must be reached
	}{
		{"zero trust denies an unlisted method", func(t *testing.T) context.Context { return requestCtx(t) }, unlistedMethod, true, codes.PermissionDenied},
		{"zero trust off allows an unlisted method", func(t *testing.T) context.Context { return requestCtx(t) }, unlistedMethod, false, codes.OK},
		{"open method is allowed with no claims", func(t *testing.T) context.Context { return requestCtx(t) }, openMethod, true, codes.OK},
		{"protected method denied without the claim", func(t *testing.T) context.Context { return requestCtx(t) }, protectedMethod, true, codes.PermissionDenied},
		{"protected method allowed with the claim", func(t *testing.T) context.Context { return requestCtx(t, write) }, protectedMethod, true, codes.OK},
		// The fail-open this change fixes: previously a missing request container skipped
		// every check, even under zero trust.
		{"protected method fails closed with no request container", func(t *testing.T) context.Context { return context.Background() }, protectedMethod, true, codes.PermissionDenied},
		{"unlisted method still denied with no request container", func(t *testing.T) context.Context { return context.Background() }, unlistedMethod, true, codes.PermissionDenied},
		{"open method needs no request container", func(t *testing.T) context.Context { return context.Background() }, openMethod, true, codes.OK},
	}
	for _, g := range gates {
		for _, tc := range cases {
			t.Run(g.name+"/"+tc.desc, func(t *testing.T) {
				reached, err := g.call(tc.ctx(t), tc.method, gateMap(), tc.zeroTrust)
				if tc.wantCode == codes.OK {
					require.NoError(t, err)
					require.True(t, reached, "handler must run")
					return
				}
				require.Error(t, err)
				require.Equal(t, tc.wantCode, status.Code(err))
				require.False(t, reached, "handler must not run when the gate rejects")
			})
		}
	}
}
