package wellknown

const (
	// Standard JWT claim types (RFC 7519)

	// ClaimTypeSub is the "subject" claim.
	ClaimTypeSub = "sub"
	// ClaimTypeAud is the "audience" claim.
	ClaimTypeAud = "aud"
	// ClaimTypeIss is the "issuer" claim.
	ClaimTypeIss = "iss"
	// ClaimTypeExp is the "expiration time" claim.
	ClaimTypeExp = "exp"
	// ClaimTypeIat is the "issued at" claim.
	ClaimTypeIat = "iat"
	// ClaimTypeNbf is the "not before" claim.
	ClaimTypeNbf = "nbf"
	// ClaimTypeJti is the "JWT ID" claim.
	ClaimTypeJti = "jti"

	// OAuth2 / custom claim types

	// ClaimTypeClientID is the "client_id" claim used in OAuth2.
	ClaimTypeClientID = "client_id"
	// ClaimTypeEmail is the "email" claim.
	ClaimTypeEmail = "email"
	// ClaimTypeScope is the "scope" claim.
	ClaimTypeScope = "scope"
	// ClaimTypePermissions is the "permissions" claim (e.g., Auth0).
	ClaimTypePermissions = "permissions"

	// Identity values

	// AnonymousSubject is the subject value used for unauthenticated requests.
	AnonymousSubject = "anonymous"

	// mTLS claim types -- populated by middleware/auth/mtls from the gRPC
	// connection's verified client certificate (see that package's doc comment).

	// ClaimTypeMTLSVerified is set to "true" when the request arrived over a
	// connection where the client presented an X.509 certificate that the TLS
	// stack cryptographically verified against the server's configured client
	// CA bundle. Gate a method on mutual TLS by requiring this claim.
	ClaimTypeMTLSVerified = "mtls_verified"
	// ClaimTypeMTLSCommonName is the verified client certificate's subject
	// CommonName.
	ClaimTypeMTLSCommonName = "mtls_cn"
	// ClaimTypeMTLSFingerprint is the SHA-256 fingerprint (hex) of the verified
	// client certificate's raw DER bytes -- useful for pinning to one specific
	// certificate rather than trusting CommonName alone.
	ClaimTypeMTLSFingerprint = "mtls_fingerprint"
	// ClaimTypeMTLSSANURI is a URI SAN from the verified client certificate,
	// e.g. a SPIFFE ID such as "spiffe://cluster.local/ns/foo/sa/bar" -- the
	// identity form HashiCorp Vault's PKI secrets engine commonly issues. A
	// certificate may carry more than one; each is added as a separate claim
	// value under this same type.
	ClaimTypeMTLSSANURI = "mtls_san_uri"
)
