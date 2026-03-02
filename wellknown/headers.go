package wellknown

const (
	// HTTP/gRPC header and metadata keys

	// HeaderAuthorization is the standard HTTP Authorization header (title-case for HTTP).
	HeaderAuthorization = "Authorization"
	// MetadataKeyAuthorization is the gRPC metadata key for authorization (lowercase per gRPC convention).
	MetadataKeyAuthorization = "authorization"
	// HeaderContentType is the standard HTTP Content-Type header.
	HeaderContentType = "Content-Type"
	// ContentTypeJSON is the MIME type for JSON.
	ContentTypeJSON = "application/json"
	// HeaderXApiKey is the X-Api-Key header used for API key authentication.
	HeaderXApiKey = "X-Api-Key"

	// Auth schemes

	// AuthSchemeBearer is the lowercase bearer auth scheme for comparison.
	AuthSchemeBearer = "bearer"
	// AuthSchemeBearerPrefix is the "Bearer " prefix for constructing Authorization header values.
	AuthSchemeBearerPrefix = "Bearer "

	// Context propagation

	// ContextOriginKey is the metadata/config key for context origin propagation.
	ContextOriginKey = "ctxOrigin"
)
