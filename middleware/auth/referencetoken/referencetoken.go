// Package referencetoken provides a gRPC middleware that recognizes opaque
// external "reference tokens" -- most commonly Personal Access Tokens (PATs) --
// presented as a normal `Authorization: Bearer <token>` header, resolves them via
// one or more registered IResolver implementations, and caches the resulting
// claims for a TTL so repeat requests skip resolution entirely.
//
// A reference token resolves to one of two things (contracts/middleware/auth/referencetoken.ResolvedKind):
//
//   - ResolvedKindJWT: the reference token stands in for a real JWT. This
//     middleware swaps the opaque token for the resolved JWT in the request's
//     Authorization metadata and hands the request to the normal JWT validation
//     pipeline (next), which validates it exactly as if the caller had presented
//     that JWT directly. Once next returns successfully, the claims it populated
//     onto the scoped IClaimsPrincipal are snapshotted into the cache.
//   - ResolvedKindClaimsPrincipal: resolution already produced the final claims
//     (e.g. from an introspection call). This middleware loads them directly onto
//     the scoped IClaimsPrincipal, caches them, and calls straight through to the
//     rest of the pipeline -- next (the JWT validator) is never invoked.
//
// In both cases the opaque reference token is stripped out of (or replaced in)
// the request's metadata before the request proceeds, so no other middleware or
// handler ever observes it.
//
// This middleware must run in place of -- immediately wrapping -- the JWT
// validation interceptor: after dicontext's request-scope interceptor and before
// the claims-principal authorization gate. It relies on the scoped
// IClaimsPrincipal being empty when it runs.
//
//	jwtInterceptor := jwt.UnaryServerInterceptor(rootContainer)
//	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(
//	    referencetoken.UnaryServerInterceptor(rootContainer, jwtInterceptor,
//	        referencetoken.WithCache(memoryCache),
//	        referencetoken.WithDefaultTTL(fluffycore_contracts_common.FiveMinutes),
//	    ),
//	))
//
// Requests carrying an ordinary JWT (three dot-separated segments) or no
// Authorization header at all are structurally cheap to recognize and are passed
// straight through to next unchanged -- resolvers are only ever consulted for
// opaque bearer tokens.
package referencetoken

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_middleware_auth_referencetoken "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/referencetoken"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	metautils "github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
)

// AddResolver registers a reference token resolver constructor into the DI
// container. ctor may take any parameters resolvable from the container (plain DI
// constructor injection, e.g. a database client or an HTTP client for calling an
// introspection endpoint) and must return a type implementing IResolver.
//
// Register as many resolvers as needed -- e.g. one per PAT format/issuer -- they
// are tried in registration order per request until one returns handled=true.
func AddResolver(builder di.ContainerBuilder, ctor any) {
	di.AddSingleton[fluffycore_contracts_middleware_auth_referencetoken.IResolver](builder, ctor)
}

type (
	// Config holds the tunables for the reference token middleware.
	Config struct {
		// Cache stores resolved claims keyed by a hash of the reference token, so a
		// resolver -- and, for ResolvedKindJWT, the downstream JWT validator -- only
		// runs once per token per TTL window. If nil, resolution is never cached: every
		// request bearing a reference token re-resolves it.
		Cache fluffycore_contracts_common.ISingletonMemoryCache
		// DefaultTTL is used when a resolver's Resolved.TTL is <= 0.
		DefaultTTL time.Duration
		// CacheKeyPrefix namespaces this middleware's cache keys, useful when Cache is
		// shared with other middleware/services.
		CacheKeyPrefix string
		// KnownPrefixes are the static prefixes your reference tokens are tagged with
		// (e.g. "pat_", "mysvc_pat_") -- the same convention used by GitHub ("ghp_"),
		// GitLab ("glpat-"), Stripe ("sk_live_"), Slack ("xoxb-"), etc. When set, a
		// token is treated as a reference token if and only if it starts with one of
		// these prefixes; everything else passes straight through to the JWT pipeline
		// unchanged, no shape-guessing involved. When empty, detection falls back to
		// the structural looksLikeJWT heuristic (see that function's doc comment).
		// Configuring this is the recommended, standard way to make reference tokens
		// unambiguous -- prefer it over relying on the fallback.
		KnownPrefixes []string
	}
	// Option configures a Config.
	Option func(*Config)
)

// WithCache sets the TTL cache used to remember resolved claims between requests.
func WithCache(cache fluffycore_contracts_common.ISingletonMemoryCache) Option {
	return func(c *Config) {
		c.Cache = cache
	}
}

// WithDefaultTTL sets the fallback TTL applied when a resolver doesn't specify one.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
	}
}

// WithCacheKeyPrefix namespaces this middleware's cache keys.
func WithCacheKeyPrefix(prefix string) Option {
	return func(c *Config) {
		c.CacheKeyPrefix = prefix
	}
}

// WithKnownPrefixes declares the static prefix(es) your reference tokens are
// tagged with, e.g. WithKnownPrefixes("pat_"). This is the recommended way to
// distinguish reference tokens from JWTs: exact and unambiguous, unlike the
// structural fallback (see looksLikeJWT). Tag every reference token your
// resolvers issue with one of these prefixes.
func WithKnownPrefixes(prefixes ...string) Option {
	return func(c *Config) {
		c.KnownPrefixes = prefixes
	}
}

var (
	_resolvers     []fluffycore_contracts_middleware_auth_referencetoken.IResolver
	_resolversOnce sync.Once
)

func loadResolvers(rootContainer di.Container) []fluffycore_contracts_middleware_auth_referencetoken.IResolver {
	_resolversOnce.Do(func() {
		_resolvers = di.Get[[]fluffycore_contracts_middleware_auth_referencetoken.IResolver](rootContainer)
	})
	return _resolvers
}

// engine holds the resolved configuration/resolvers shared by the unary and
// stream interceptors built from a single UnaryServerInterceptor/StreamServerInterceptor call.
type engine struct {
	config    *Config
	resolvers []fluffycore_contracts_middleware_auth_referencetoken.IResolver
}

func newEngine(rootContainer di.Container, opts ...Option) *engine {
	config := &Config{
		DefaultTTL:     fluffycore_contracts_common.FiveMinutes,
		CacheKeyPrefix: "reftok:",
	}
	for _, opt := range opts {
		opt(config)
	}
	return &engine{
		config:    config,
		resolvers: loadResolvers(rootContainer),
	}
}

type outcomeKind int

const (
	// outcomePassThrough means there is no reference token to resolve (no
	// Authorization header, or it already looks like a JWT) -- call next unchanged.
	outcomePassThrough outcomeKind = iota
	// outcomeDelegateToJWT means the reference token resolved to a real JWT -- call
	// next with the rewritten context, then cache whatever claims it populates.
	outcomeDelegateToJWT
	// outcomeDirect means claims are already final (cache hit, or a
	// ResolvedKindClaimsPrincipal resolution) -- skip next entirely and call the
	// real downstream handler directly with the rewritten context.
	outcomeDirect
	// outcomeError means authentication must fail outright.
	outcomeError
)

// resolveOutcome carries the decision + whatever context/err/cache bookkeeping the
// caller (Unary/StreamServerInterceptor) needs to act on it.
type resolveOutcome struct {
	kind     outcomeKind
	ctx      context.Context
	err      error
	cacheKey string
	ttl      time.Duration
}

// resolve is the transport-agnostic core: given the inbound context, decide
// whether/how to resolve a reference token, mutating the scoped IClaimsPrincipal
// and metadata as needed, and report what the caller should do next.
func (e *engine) resolve(ctx context.Context) resolveOutcome {
	rawToken, present, err := extractBearerToken(ctx)
	if err != nil {
		return resolveOutcome{kind: outcomeError, err: err}
	}
	if !present || rawToken == "" {
		return resolveOutcome{kind: outcomePassThrough}
	}
	if !e.isReferenceToken(rawToken) {
		return resolveOutcome{kind: outcomePassThrough}
	}

	scopedContainer := dicontext.GetRequestContainer(ctx)
	if scopedContainer == nil {
		return resolveOutcome{kind: outcomeError, err: status.Error(codes.Internal,
			"referencetoken middleware: no request-scoped container found -- is dicontext.ScopedContextUnaryServerInterceptor/StreamServerInterceptor registered before this middleware?")}
	}
	claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)

	cacheKey := e.cacheKey(rawToken)
	if e.config.Cache != nil {
		if cached, cacheErr := e.config.Cache.Get(cacheKey); cacheErr == nil {
			if claims, ok := cached.([]fluffycore_contracts_common.Claim); ok {
				zerolog.Ctx(ctx).Debug().Msg("referencetoken: cache hit, skipping resolution and JWT validation")
				claimsPrincipal.AddClaim(claims...)
				return resolveOutcome{kind: outcomeDirect, ctx: stripAuthorization(ctx)}
			}
		}
	}

	resolved, resolveErr := e.resolveViaResolvers(ctx, rawToken)
	if resolveErr != nil {
		return resolveOutcome{kind: outcomeError, err: resolveErr}
	}
	if resolved == nil {
		return resolveOutcome{kind: outcomeError, err: status.Error(codes.Unauthenticated, "unrecognized reference token")}
	}

	ttl := resolved.TTL
	if ttl <= 0 {
		ttl = e.config.DefaultTTL
	}

	switch resolved.Kind {
	case fluffycore_contracts_middleware_auth_referencetoken.ResolvedKindJWT:
		return resolveOutcome{
			kind:     outcomeDelegateToJWT,
			ctx:      rewriteAuthorizationToJWT(ctx, resolved.RawJWT),
			cacheKey: cacheKey,
			ttl:      ttl,
		}
	case fluffycore_contracts_middleware_auth_referencetoken.ResolvedKindClaimsPrincipal:
		scratch := fluffycore_services_common_claimsprincipal.ClaimsPrincipalFromClaimsMap(resolved.Claims)
		claims := scratch.GetClaims()
		claimsPrincipal.AddClaim(claims...)
		if e.config.Cache != nil && len(claims) > 0 {
			_ = e.config.Cache.SetWithTTL(cacheKey, claims, ttl)
		}
		return resolveOutcome{kind: outcomeDirect, ctx: stripAuthorization(ctx)}
	default:
		return resolveOutcome{kind: outcomeError, err: status.Errorf(codes.Internal,
			"referencetoken middleware: resolver returned unrecognized ResolvedKind %v", resolved.Kind)}
	}
}

// isReferenceToken decides whether rawToken should be routed to resolvers at
// all. When KnownPrefixes is configured, that's the whole answer: prefix match
// wins, no shape-guessing. Otherwise it falls back to looksLikeJWT.
func (e *engine) isReferenceToken(rawToken string) bool {
	if len(e.config.KnownPrefixes) > 0 {
		for _, prefix := range e.config.KnownPrefixes {
			if strings.HasPrefix(rawToken, prefix) {
				return true
			}
		}
		return false
	}
	return !looksLikeJWT(rawToken)
}

func (e *engine) resolveViaResolvers(ctx context.Context, rawToken string) (*fluffycore_contracts_middleware_auth_referencetoken.Resolved, error) {
	for _, resolver := range e.resolvers {
		handled, resolved, err := resolver.Resolve(ctx, rawToken)
		if err != nil {
			return nil, err
		}
		if handled {
			return resolved, nil
		}
	}
	return nil, nil
}

// cacheResolvedClaims snapshots the scoped IClaimsPrincipal (populated by `next`,
// the JWT validator, in the outcomeDelegateToJWT path) into the cache. Called only
// after next has returned successfully.
func (e *engine) cacheResolvedClaims(ctx context.Context, cacheKey string, ttl time.Duration) {
	if e.config.Cache == nil || cacheKey == "" {
		return
	}
	scopedContainer := dicontext.GetRequestContainer(ctx)
	if scopedContainer == nil {
		return
	}
	claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)
	claims := claimsPrincipal.GetClaims()
	if len(claims) == 0 {
		// Nothing to cache -- avoid caching an empty/failed resolution.
		return
	}
	_ = e.config.Cache.SetWithTTL(cacheKey, claims, ttl)
}

func (e *engine) cacheKey(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return e.config.CacheKeyPrefix + hex.EncodeToString(sum[:])
}

// UnaryServerInterceptor returns a gRPC unary server interceptor implementing the
// reference-token resolution described in the package doc. next is normally
// middleware/auth/jwt.UnaryServerInterceptor -- it is invoked for ordinary JWTs and
// for reference tokens that resolve to a JWT (ResolvedKindJWT); it is skipped
// entirely on a cache hit or a ResolvedKindClaimsPrincipal resolution.
func UnaryServerInterceptor(rootContainer di.Container, next grpc.UnaryServerInterceptor, opts ...Option) grpc.UnaryServerInterceptor {
	e := newEngine(rootContainer, opts...)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		out := e.resolve(ctx)
		switch out.kind {
		case outcomeError:
			return nil, out.err
		case outcomePassThrough:
			return next(ctx, req, info, handler)
		case outcomeDirect:
			return handler(out.ctx, req)
		case outcomeDelegateToJWT:
			resp, err := next(out.ctx, req, info, handler)
			if err != nil {
				return nil, err
			}
			e.cacheResolvedClaims(out.ctx, out.cacheKey, out.ttl)
			return resp, nil
		default:
			return nil, status.Error(codes.Internal, "referencetoken middleware: unhandled outcome")
		}
	}
}

// StreamServerInterceptor is the streaming counterpart to UnaryServerInterceptor.
// next is normally middleware/auth/jwt.StreamServerInterceptor.
func StreamServerInterceptor(rootContainer di.Container, next grpc.StreamServerInterceptor, opts ...Option) grpc.StreamServerInterceptor {
	e := newEngine(rootContainer, opts...)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		out := e.resolve(ctx)
		switch out.kind {
		case outcomeError:
			return out.err
		case outcomePassThrough:
			return next(srv, ss, info, handler)
		case outcomeDirect:
			sw := fluffycore_middleware.NewStreamContextWrapper(ss)
			sw.SetContext(out.ctx)
			return handler(srv, sw)
		case outcomeDelegateToJWT:
			sw := fluffycore_middleware.NewStreamContextWrapper(ss)
			sw.SetContext(out.ctx)
			if err := next(srv, sw, info, handler); err != nil {
				return err
			}
			e.cacheResolvedClaims(out.ctx, out.cacheKey, out.ttl)
			return nil
		default:
			return status.Error(codes.Internal, "referencetoken middleware: unhandled outcome")
		}
	}
}

// looksLikeJWT is the fallback structural check used only when the caller hasn't
// configured WithKnownPrefixes: a JWT is always header.payload.signature
// (exactly two dots); reference tokens/PATs are expected to be opaque strings
// without that shape. Prefer WithKnownPrefixes -- the standard, unambiguous way
// to tag a reference token (mirroring GitHub's "ghp_", GitLab's "glpat-",
// Stripe's "sk_live_", etc.) -- over relying on this guess.
func looksLikeJWT(token string) bool {
	return strings.Count(token, ".") == 2
}

// extractBearerToken pulls the raw token out of the incoming "authorization"
// metadata. present=false (with no error) means there was nothing to extract --
// callers should treat that as "nothing to do here", exactly like the JWT
// middleware treats a missing token as fine (anonymous) rather than a failure.
func extractBearerToken(ctx context.Context) (token string, present bool, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false, nil
	}
	values := md.Get(fluffycore_wellknown.MetadataKeyAuthorization)
	if len(values) == 0 {
		return "", false, nil
	}
	parts := strings.SplitN(values[0], " ", 2)
	if len(parts) != 2 {
		return "", false, status.Error(codes.Unauthenticated, "invalid authorization header")
	}
	if !strings.EqualFold(parts[0], fluffycore_wellknown.AuthSchemeBearer) {
		return "", false, status.Error(codes.Unauthenticated, "invalid authorization header")
	}
	return parts[1], true, nil
}

// stripAuthorization removes the authorization header entirely, so nothing
// downstream of this middleware can observe the raw reference token.
func stripAuthorization(ctx context.Context) context.Context {
	md := metautils.ExtractIncoming(ctx)
	md = md.Del(fluffycore_wellknown.MetadataKeyAuthorization)
	return md.ToIncoming(ctx)
}

// rewriteAuthorizationToJWT replaces the authorization header's value with the
// resolved JWT, so the downstream JWT pipeline (and nothing else) sees it.
func rewriteAuthorizationToJWT(ctx context.Context, jwt string) context.Context {
	md := metautils.ExtractIncoming(ctx)
	md = md.Set(fluffycore_wellknown.MetadataKeyAuthorization, fluffycore_wellknown.AuthSchemeBearerPrefix+jwt)
	return md.ToIncoming(ctx)
}
