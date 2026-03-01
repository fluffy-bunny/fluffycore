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
)
