// Package mtls turns a verified client certificate on the gRPC connection into
// claims on the request's scoped IClaimsPrincipal, so mutual TLS can be gated
// per method using the exact same claims/permissions engine
// (contracts/common.IEntryPointConfig, validated by
// middleware/claimsprincipal.FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2)
// you already use for JWT-derived permissions. See this repo's mTLS README for
// the full walkthrough (config, wiring, diagrams, a worked per-method example).
//
// TLS's handshake -- and therefore whether a client certificate is requested,
// and how strictly it's verified -- happens once per connection, before gRPC
// knows which method will be called; see runtime/servertls for building a
// *tls.Config in "optional" mode (tls.VerifyClientCertIfGiven) so a per-method
// requirement can be layered on top here rather than forced connection-wide.
//
// Claims are only added when the TLS stack itself cryptographically verified
// the presented certificate against the server's configured client CA bundle
// (i.e. tls.ConnectionState.VerifiedChains is non-empty) -- never merely
// because a certificate was present (tls.ConnectionState.PeerCertificates can
// be populated but unverified under tls.RequestClientCert). A plaintext
// connection, a connection with no client certificate, or one whose
// certificate didn't verify all produce no mTLS claims at all -- callers must
// not assume "claim absent" means "not mTLS-capable service", only "not
// verified on this connection".
package mtls

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	grpc "google.golang.org/grpc"
	credentials "google.golang.org/grpc/credentials"
	peer "google.golang.org/grpc/peer"
)

// UnaryServerInterceptor adds mTLS-derived claims (see package doc) onto the
// scoped IClaimsPrincipal, then always calls through to handler -- this
// middleware never fails a request by itself; it only supplies claims for the
// claims-principal authorization gate (or your own resolvers) to act on.
//
// Register this after dicontext.ScopedContextUnaryServerInterceptor (needs the
// request-scoped container) and before the claims-principal authorization
// gate. Safe to register unconditionally, including when TLS is disabled --
// with no TLS peer info on the connection it's a cheap no-op.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		addPeerCertClaims(ctx)
		return handler(ctx, req)
	}
}

// StreamServerInterceptor is the streaming counterpart to UnaryServerInterceptor.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		addPeerCertClaims(ss.Context())
		return handler(srv, ss)
	}
}

func addPeerCertClaims(ctx context.Context) {
	cert := verifiedClientCert(ctx)
	if cert == nil {
		return
	}

	scopedContainer := dicontext.GetRequestContainer(ctx)
	if scopedContainer == nil {
		// dicontext hasn't run yet -- misconfigured registration order. There's
		// nothing to attach claims to; log-free no-op rather than panicking a
		// request over what is, at worst, a missing security signal.
		return
	}
	claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)

	claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
		Type: fluffycore_wellknown.ClaimTypeMTLSVerified, Value: "true",
	})
	if cert.Subject.CommonName != "" {
		claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
			Type: fluffycore_wellknown.ClaimTypeMTLSCommonName, Value: cert.Subject.CommonName,
		})
	}
	sum := sha256.Sum256(cert.Raw)
	claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
		Type: fluffycore_wellknown.ClaimTypeMTLSFingerprint, Value: hex.EncodeToString(sum[:]),
	})
	for _, uri := range cert.URIs {
		claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
			Type: fluffycore_wellknown.ClaimTypeMTLSSANURI, Value: uri.String(),
		})
	}
}

// verifiedClientCert returns the connection's verified client (leaf)
// certificate, or nil if there isn't one. It deliberately checks
// VerifiedChains rather than PeerCertificates: the latter can be populated
// without any verification having taken place (tls.RequestClientCert mode),
// while VerifiedChains is only ever non-empty once the TLS stack has actually
// validated the chain against the server's configured client CA bundle.
func verifiedClientCert(ctx context.Context) *x509.Certificate {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil
	}
	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return nil
	}
	return tlsInfo.State.VerifiedChains[0][0]
}
