package mtls

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/url"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	"github.com/fluffy-bunny/fluffycore/utils/certgen"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/stretchr/testify/require"
	credentials "google.golang.org/grpc/credentials"
	grpc "google.golang.org/grpc"
	peer "google.golang.org/grpc/peer"
)

var testRootContainer = func() di.Container {
	builder := di.Builder()
	fluffycore_services_common_claimsprincipal.AddClaimsPrincipal(builder)
	return builder.Build()
}()

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	scopeFactory := di.Get[di.ScopeFactory](testRootContainer)
	scope := scopeFactory.CreateScope()
	t.Cleanup(scope.Dispose)
	return dicontext.SetRequestContainer(context.Background(), scope.Container())
}

func claimsPrincipalOf(ctx context.Context) fluffycore_contracts_common.IClaimsPrincipal {
	container := dicontext.GetRequestContainer(ctx)
	return di.Get[fluffycore_contracts_common.IClaimsPrincipal](container)
}

// clientCertFor generates a leaf certificate signed by a fresh CA and returns
// it parsed, for building a fake tls.ConnectionState in tests.
func clientCertFor(t *testing.T, cn string, uris ...string) *x509.Certificate {
	t.Helper()
	ca, err := certgen.NewCA(certgen.CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)

	var parsedURIs []*url.URL
	for _, u := range uris {
		parsed, err := url.Parse(u)
		require.NoError(t, err)
		parsedURIs = append(parsedURIs, parsed)
	}
	leaf, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: cn, URIs: parsedURIs})
	require.NoError(t, err)
	return leaf.Cert
}

func ctxWithVerifiedPeerCert(ctx context.Context, cert *x509.Certificate) context.Context {
	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{cert}},
		},
	}
	return peer.NewContext(ctx, &peer.Peer{AuthInfo: tlsInfo})
}

// ctxWithUnverifiedPeerCert simulates tls.RequestClientCert mode: a certificate
// is present on the connection, but the TLS stack never verified it (empty
// VerifiedChains).
func ctxWithUnverifiedPeerCert(ctx context.Context, cert *x509.Certificate) context.Context {
	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{cert},
		},
	}
	return peer.NewContext(ctx, &peer.Peer{AuthInfo: tlsInfo})
}

func callUnary(t *testing.T, ctx context.Context) (called bool, finalCtx context.Context) {
	t.Helper()
	interceptor := UnaryServerInterceptor()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		finalCtx = ctx
		return "ok", nil
	}
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, handler)
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
	return called, finalCtx
}

func TestUnary_NoPeerInfo_NoClaimsAdded(t *testing.T) {
	ctx := newTestContext(t)
	called, _ := callUnary(t, ctx)
	require.True(t, called)
	require.Empty(t, claimsPrincipalOf(ctx).GetClaims())
}

func TestUnary_PeerWithoutTLSInfo_NoClaimsAdded(t *testing.T) {
	ctx := newTestContext(t)
	ctx = peer.NewContext(ctx, &peer.Peer{AuthInfo: nil})
	called, _ := callUnary(t, ctx)
	require.True(t, called)
	require.Empty(t, claimsPrincipalOf(ctx).GetClaims())
}

func TestUnary_UnverifiedPeerCert_NoClaimsAdded(t *testing.T) {
	// A certificate is present (tls.RequestClientCert mode) but never verified --
	// must NOT be trusted.
	cert := clientCertFor(t, "unverified-client")
	ctx := newTestContext(t)
	ctx = ctxWithUnverifiedPeerCert(ctx, cert)

	called, _ := callUnary(t, ctx)
	require.True(t, called)
	require.Empty(t, claimsPrincipalOf(ctx).GetClaims(), "an unverified peer certificate must never produce mTLS claims")
}

func TestUnary_VerifiedPeerCert_AddsClaims(t *testing.T) {
	cert := clientCertFor(t, "test-client", "spiffe://cluster.local/ns/foo/sa/bar")
	ctx := newTestContext(t)
	ctx = ctxWithVerifiedPeerCert(ctx, cert)

	called, _ := callUnary(t, ctx)
	require.True(t, called)

	principal := claimsPrincipalOf(ctx)
	require.Equal(t, "true", principal.GetClaimsByType(fluffycore_wellknown.ClaimTypeMTLSVerified)[0].Value)
	require.Equal(t, "test-client", principal.GetClaimsByType(fluffycore_wellknown.ClaimTypeMTLSCommonName)[0].Value)

	wantFingerprint := sha256.Sum256(cert.Raw)
	require.Equal(t, hex.EncodeToString(wantFingerprint[:]), principal.GetClaimsByType(fluffycore_wellknown.ClaimTypeMTLSFingerprint)[0].Value)

	require.Equal(t, "spiffe://cluster.local/ns/foo/sa/bar", principal.GetClaimsByType(fluffycore_wellknown.ClaimTypeMTLSSANURI)[0].Value)
}

func TestUnary_VerifiedPeerCert_NoURISAN_NoSANClaim(t *testing.T) {
	cert := clientCertFor(t, "test-client-no-san")
	ctx := newTestContext(t)
	ctx = ctxWithVerifiedPeerCert(ctx, cert)

	callUnary(t, ctx)

	principal := claimsPrincipalOf(ctx)
	require.False(t, principal.HasClaimType(fluffycore_wellknown.ClaimTypeMTLSSANURI))
}

func TestStream_VerifiedPeerCert_AddsClaims(t *testing.T) {
	cert := clientCertFor(t, "stream-client")
	ctx := newTestContext(t)
	ctx = ctxWithVerifiedPeerCert(ctx, cert)

	ss := &fakeServerStream{ctx: ctx}
	interceptor := StreamServerInterceptor()
	called := false
	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)

	principal := claimsPrincipalOf(ctx)
	require.Equal(t, "true", principal.GetClaimsByType(fluffycore_wellknown.ClaimTypeMTLSVerified)[0].Value)
}

// fakeServerStream is a minimal grpc.ServerStream stand-in that just returns a
// fixed context.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context { return f.ctx }
