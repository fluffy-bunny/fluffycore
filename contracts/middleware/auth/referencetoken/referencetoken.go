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
