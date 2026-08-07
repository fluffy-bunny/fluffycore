package referencetoken

import (
	"context"
	"time"
)

// ResolvedKind identifies what an IResolver produced for a reference token it recognized.
type ResolvedKind int

const (
	// ResolvedKindUnhandled is the zero value. A Resolved should never be returned with
	// this Kind -- return handled=false instead so other resolvers get a chance.
	ResolvedKindUnhandled ResolvedKind = iota
	// ResolvedKindJWT means RawJWT holds a real JWT. The reference token middleware
	// substitutes RawJWT for the opaque reference token in the request's Authorization
	// metadata and hands the request to the normal JWT validation pipeline, exactly as
	// if the caller had presented RawJWT directly. Use this when the external token
	// (e.g. a Personal Access Token) is itself backed by / exchanged for a JWT that your
	// existing JWT validators already know how to check.
	ResolvedKindJWT
	// ResolvedKindClaimsPrincipal means Claims already holds the final, trusted claims
	// for this token. No downstream JWT validation is performed -- the reference token
	// middleware loads Claims directly onto the request's scoped IClaimsPrincipal. Use
	// this when resolution (e.g. an introspection call, a database lookup) already
	// establishes the caller's identity and there is no JWT to hand off.
	ResolvedKindClaimsPrincipal
)

// Resolved is what an IResolver returns for a reference token it recognizes.
type Resolved struct {
	// Kind selects which of RawJWT / Claims is populated.
	Kind ResolvedKind
	// RawJWT holds the underlying JWT when Kind == ResolvedKindJWT.
	RawJWT string
	// Claims holds the final claims when Kind == ResolvedKindClaimsPrincipal. Uses the
	// same shape a parsed JWT's claim set has (string, bool, float64, and []string /
	// []interface{} of those), so it can be fed straight into
	// claimsprincipal.ClaimsPrincipalFromClaimsMap.
	Claims map[string]interface{}
	// TTL controls how long this resolution's *result* (the final claims) may be cached
	// before the next request bearing the same reference token must resolve again.
	// <= 0 uses the middleware's configured default TTL.
	TTL time.Duration
}

// ICache is the reference-token middleware's cache contract: look up a previously-resolved
// token by key, and store a fresh resolution with a TTL. It is intentionally the minimal
// subset of fluffycore_contracts_common.ISingletonMemoryCache the middleware actually calls
// (see referencetoken.go's resolve/cacheResolvedClaims) -- any ISingletonMemoryCache already
// satisfies it, so existing in-process callers (referencetoken.WithCache(memoryCache)) keep
// working unchanged, but implementations are no longer required to be in-process or
// singleton-scoped. A shared, durable backing store (Mongo, Redis, a KV store, ...) satisfies
// this just as well, and is what you want once resolution should be cached once cluster-wide
// rather than once per replica.
type ICache interface {
	// Get returns the cached value for key, or a non-nil error if there is no (unexpired)
	// entry. The returned value must be a []fluffycore_contracts_common.Claim -- that is the
	// only type the middleware ever stores.
	Get(key string) (interface{}, error)
	// SetWithTTL stores data under key for ttl. Implementations that cannot honor ttl
	// precisely (e.g. a store with only coarse-grained expiry) should round up, never down --
	// serving a resolution slightly past its TTL is far cheaper than re-resolving a valid one.
	SetWithTTL(key string, data interface{}, ttl time.Duration) error
}

// IResolver resolves an opaque external reference token -- e.g. a Personal Access
// Token (PAT) -- into either a JWT for the normal JWT validation pipeline to
// validate, or a fully-formed set of claims to trust as-is.
//
// Implementations that don't recognize a token's shape (wrong prefix, wrong length,
// not present in their backing store, ...) MUST return handled=false, resolved=nil,
// err=nil so other registered resolvers get a chance. Only return a non-nil error
// for tokens the resolver does recognize but cannot honor (revoked, expired, backing
// store unavailable, ...) -- that is treated as a hard authentication failure and
// short-circuits the request, it does not fall through to other resolvers.
type IResolver interface {
	Resolve(ctx context.Context, rawToken string) (handled bool, resolved *Resolved, err error)
}
