package jwt

import (
	"context"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	jwa "github.com/lestrrat-go/jwx/v2/jwa"
	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// tokenAssertingMTLS is a validly signed JWT that also claims mTLS -- e.g. minted by an IdP
// that lets clients request arbitrary claims, or copied from a request made over mTLS.
func tokenAssertingMTLS(t *testing.T, issuer string, key jwxk.Key) string {
	t.Helper()
	tok, err := jwxt.NewBuilder().Issuer(issuer).Subject("user-1").
		IssuedAt(time.Now()).Expiration(time.Now().Add(time.Hour)).
		Claim(fluffycore_wellknown.ClaimTypeMTLSVerified, "true").
		Claim(fluffycore_wellknown.ClaimTypeMTLSCommonName, "trusted-service").
		Build()
	require.NoError(t, err)
	b, err := jwxt.Sign(tok, jwxt.WithKey(jwa.RS256, key))
	require.NoError(t, err)
	return string(b)
}

func principalFrom(ctx context.Context) fluffycore_contracts_common.IClaimsPrincipal {
	return di.Get[fluffycore_contracts_common.IClaimsPrincipal](dicontext.GetRequestContainer(ctx))
}

func requireNoMTLSClaims(t *testing.T, p fluffycore_contracts_common.IClaimsPrincipal) {
	t.Helper()
	require.True(t, p.HasClaim(fluffycore_contracts_common.Claim{Type: "sub", Value: "user-1"}), "the token itself was accepted")
	for _, c := range p.GetClaims() {
		require.NotContains(t, []string{fluffycore_wellknown.ClaimTypeMTLSVerified, fluffycore_wellknown.ClaimTypeMTLSCommonName}, c.Type,
			"a token must not be able to assert mtls_* claims")
	}
}

func TestUnary_TokenCannotAssertMTLSClaims(t *testing.T) {
	idp := newFakeIdP(t)
	key := newSigningKey(t, "k1")
	idp.publish(t, key)
	interceptor := withIdP(t, idp, time.Minute)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenAssertingMTLS(t, idp.issuer(), key)))
	var p fluffycore_contracts_common.IClaimsPrincipal
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Svc/M"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			p = principalFrom(ctx)
			return nil, nil
		})
	require.NoError(t, err)
	requireNoMTLSClaims(t, p)
}

func TestStream_TokenCannotAssertMTLSClaims(t *testing.T) {
	idp := newFakeIdP(t)
	key := newSigningKey(t, "k1")
	idp.publish(t, key)
	root := withIdPRoot(t, idp, time.Minute)

	scope := di.Get[di.ScopeFactory](root).CreateScope()
	t.Cleanup(scope.Dispose)
	ctx := dicontext.SetRequestContainer(context.Background(), scope.Container())
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer "+tokenAssertingMTLS(t, idp.issuer(), key)))

	var p fluffycore_contracts_common.IClaimsPrincipal
	err := StreamServerInterceptor(root)(nil, &fakeStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/test.Svc/S"},
		func(srv interface{}, ss grpc.ServerStream) error { p = principalFrom(ss.Context()); return nil })
	require.NoError(t, err)
	requireNoMTLSClaims(t, p)
}
