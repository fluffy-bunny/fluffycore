package jwt

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	copier "github.com/jinzhu/copier"
	jwkfetch "github.com/jwx-go/jwkfetch/v4"
	httprc "github.com/lestrrat-go/httprc/v3"
	jwxk "github.com/lestrrat-go/jwx/v4/jwk"
	jws "github.com/lestrrat-go/jwx/v4/jws"
	jwxt "github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
)

type (
	service struct {
		config       *fluffycore_contracts_middleware_auth_jwt.IssuerConfig
		nilValidator bool
	}
)

var _cache *jwkfetch.Cache
var _issuerConfigs map[string]*fluffycore_contracts_middleware_auth_jwt.IssuerConfig
var _validatorsOnce sync.Once

// DefaultMinJWKSRefreshInterval is the default minimum time between forced JWKS refreshes
// for any one JWKS URL. See SetMinJWKSRefreshInterval.
const DefaultMinJWKSRefreshInterval = 30 * time.Second

var _minJWKSRefreshInterval atomic.Int64

// _jwksRefreshGates holds a *jwksRefreshGate per JWKS URL.
var _jwksRefreshGates sync.Map

type jwksRefreshGate struct {
	mu   sync.Mutex
	last time.Time
}

// SetMinJWKSRefreshInterval sets the minimum time between forced JWKS refreshes per JWKS
// URL. A token whose kid isn't in the cached key set triggers a refresh (to pick up a key
// rollover), but at most once per interval: otherwise any caller who knows the public
// issuer URL could send unsigned tokens with random kids and turn every request into an
// outbound fetch to the IdP, before any signature is checked. d <= 0 restores the default.
func SetMinJWKSRefreshInterval(d time.Duration) {
	if d <= 0 {
		d = DefaultMinJWKSRefreshInterval
	}
	_minJWKSRefreshInterval.Store(int64(d))
}

// keySetForKid returns a key set containing kid, refreshing the JWKS URL if needed but no
// more than once per min interval. ok is false when kid is still unknown.
func keySetForKid(ctx context.Context, jwksURL, kid string) (set jwxk.Set, ok bool, err error) {
	v, _ := _jwksRefreshGates.LoadOrStore(jwksURL, &jwksRefreshGate{})
	gate := v.(*jwksRefreshGate)

	set, err = _cache.Lookup(ctx, jwksURL)
	if err != nil {
		// Not fetched yet (the URL is registered without waiting).
		if set, err = loadKeySet(ctx, jwksURL, gate); err != nil {
			return nil, false, err
		}
	}
	if _, found := set.LookupKeyID(kid); found {
		return set, true, nil
	}

	gate.mu.Lock()
	defer gate.mu.Unlock()

	// Another caller may have refreshed while we waited for the gate.
	if set, err = _cache.Lookup(ctx, jwksURL); err == nil {
		if _, found := set.LookupKeyID(kid); found {
			return set, true, nil
		}
	}
	if !gate.last.IsZero() && time.Since(gate.last) < time.Duration(_minJWKSRefreshInterval.Load()) {
		// Refreshed recently and the kid still isn't there: don't fetch again.
		return set, false, nil
	}
	// Record the attempt before fetching, so a failing IdP isn't hammered either.
	gate.last = time.Now()
	set, err = refreshJWKS(ctx, jwksURL, "unknown kid")
	if err != nil {
		return nil, false, err
	}
	_, found := set.LookupKeyID(kid)
	return set, found, nil
}

// refreshJWKS fetches a JWKS URL now and logs how long that took, and why it was needed. Fetches are rare (a
// first load, or a key rollover) and a slow or failing IdP is what makes a request slow, so they are worth a
// log line each.
func refreshJWKS(ctx context.Context, jwksURL string, reason string) (jwxk.Set, error) {
	started := time.Now()
	set, err := _cache.Refresh(ctx, jwksURL)
	took := time.Since(started)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("jwks_url", jwksURL).Str("reason", reason).Dur("took", took).
			Msg("JWKS fetch failed")
		return nil, err
	}
	zerolog.Ctx(ctx).Info().Str("jwks_url", jwksURL).Str("reason", reason).Int("keys", set.Len()).Dur("took", took).
		Msg("JWKS fetched")
	return set, nil
}

func init() {
	SetMinJWKSRefreshInterval(DefaultMinJWKSRefreshInterval)
	var _ fluffycore_contracts_middleware_auth_jwt.IValidator = &service{}
	cache, err := jwkfetch.NewCache(context.Background(), httprc.NewClient())
	if err != nil {
		panic(err)
	}
	_cache = cache
	_issuerConfigs = make(map[string]*fluffycore_contracts_middleware_auth_jwt.IssuerConfig)
}

// AddValidators registers JWT validators for each issuer configuration into the DI container.
func AddValidators(builder di.ContainerBuilder, config *fluffycore_contracts_middleware_auth_jwt.IssuerConfigs) {
	for _, issuerConfig := range config.IssuerConfigs {
		dst := &fluffycore_contracts_middleware_auth_jwt.IssuerConfig{}
		err := copier.Copy(dst, issuerConfig)
		if err != nil {
			panic(err)
		}

		_issuerConfigs[issuerConfig.OAuth2Config.Issuer] = dst
		// Several issuers can share one JWKS URL (for example the same server reached under two names, each
		// name being its own issuer), and AddValidators can be called again for the same URL. The cache is keyed
		// by URL and refuses a second registration, so register each URL once. Each issuer still gets its own
		// validator below.
		jwksURL := issuerConfig.OAuth2Config.JWKSUrl
		if !_cache.IsRegistered(context.Background(), jwksURL) {
			// Don't block startup on the first fetch (v2 fetched lazily); keySetForKid fetches on demand.
			if err := _cache.Register(context.Background(), jwksURL, jwkfetch.WithWaitReady(false)); err != nil {
				panic(err)
			}
		}
		// STOP: we want multiple validators even though it looks like we are adding the same one over and over.
		// each validator targets a specific issuer.
		di.AddSingleton[fluffycore_contracts_middleware_auth_jwt.IValidator](builder, func() *service {
			return &service{
				config: dst,
			}
		})
	}
}

// AddNilValidator registers a no-op JWT validator, replacing any existing validators.
func AddNilValidator(builder di.ContainerBuilder) {
	// we don't want any other validators in here.
	builder.Remove(reflect.TypeOf((*fluffycore_contracts_middleware_auth_jwt.IValidator)(nil)).Elem())
	di.AddSingleton[fluffycore_contracts_middleware_auth_jwt.IValidator](builder, func() *service {
		return &service{
			nilValidator: true,
		}
	})
}

func (s *service) ValidateAccessToken(ctx context.Context, rawToken *fluffycore_contracts_middleware_auth_jwt.ParsedToken) (bool, error) {
	if s.nilValidator {
		return true, nil
	}

	kid, _ := rawToken.JWSMessage.Signatures()[0].ProtectedHeaders().KeyID()
	tokenIssuer, _ := rawToken.Token.Issuer()
	issuer := strings.ToLower(tokenIssuer)
	if issuer != s.config.OAuth2Config.Issuer {
		// not our issuer, so we aren't handling it and are not returning an error
		return false, nil
	}
	// check if the issuer is in the list of issuers
	issuerConfig := _issuerConfigs[issuer]
	// An unknown kid may be a key rollover, so the key set is refreshed -- but at most once
	// per SetMinJWKSRefreshInterval per JWKS URL (see keySetForKid).
	set, ok, err := keySetForKid(ctx, issuerConfig.OAuth2Config.JWKSUrl, kid)
	if err != nil {
		return true, status.Errorf(codes.Unauthenticated, "jwks fetch error: %v", err)
	}
	if !ok {
		return true, status.Errorf(codes.Unauthenticated, "no keys found")
	}
	// Verify the signature over the exact bytes the claims were parsed from, then validate
	// those claims -- instead of parsing the whole token a third time to do both.
	if _, err := jws.Verify([]byte(rawToken.AccessToken), jws.WithKeySet(set)); err != nil {
		return true, status.Errorf(codes.Unauthenticated, "token signature not valid")
	}
	trustedToken := rawToken.Token
	// exp is required: jwxt.Validate accepts a token with no exp, and a signed token that
	// never expires is a credential that can't be revoked by waiting.
	if err := jwxt.Validate(trustedToken, jwxt.WithAcceptableSkew(time.Minute*5), jwxt.WithRequiredClaim(jwxt.ExpirationKey)); err != nil {
		return true, status.Errorf(codes.Unauthenticated, "token signature not valid")
	}

	if len(s.config.Audiences) > 0 {
		// check audience
		matchedAudience := ""
		for _, aud := range s.config.Audiences {
			if checkAudienceMatch(trustedToken, aud) {
				matchedAudience = aud
				break
			}
		}
		if matchedAudience == "" {
			msg := "JWT audience do not match"
			return true, status.Error(codes.Unauthenticated, msg)
		}
	}
	return true, nil
}

func getTokenFromAuthorizationHeader(ctx context.Context) (*string, error) {
	// pull bearer token from context using metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "no metadata found")
	}
	// its an Authorization : Bearer {{token}}
	bear := md.Get(fluffycore_wellknown.MetadataKeyAuthorization)
	if len(bear) == 0 {
		// not having anything is ok.
		return nil, nil
	}
	authorization := strings.Split(bear[0], " ")
	if len(authorization) != 2 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization")
	}
	if strings.ToLower(authorization[0]) != fluffycore_wellknown.AuthSchemeBearer {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization")
	}
	token := authorization[1]
	return &token, nil
}

var _validators []fluffycore_contracts_middleware_auth_jwt.IValidator

func _validate(ctx context.Context) (*fluffycore_contracts_middleware_auth_jwt.ParsedToken, error) {
	// parse the token
	tokenPtr, err := getTokenFromAuthorizationHeader(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "token parse error")
	}
	if tokenPtr == nil {
		return nil, status.Error(codes.NotFound, "no token found")
	}
	token := *tokenPtr

	rt, err := getRawToken(ctx, token)
	if err != nil {
		return nil, err
	}
	kid, _ := rt.JWSMessage.Signatures()[0].ProtectedHeaders().KeyID()
	if len(kid) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no kid found in token")
	}
	issuer, _ := rt.Token.Issuer()
	if len(issuer) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no issuer found in token")
	}

	// validate the token
	for _, validator := range _validators {
		handled, err := validator.ValidateAccessToken(ctx, rt)
		if handled {
			if err != nil {
				return nil, err
			}
			return rt, nil
		}
	}
	return nil, status.Error(codes.Unauthenticated, "token validation error")

}
func _loadValidators(rootContainer di.Container) {
	_validatorsOnce.Do(func() {
		_validators = di.Get[[]fluffycore_contracts_middleware_auth_jwt.IValidator](rootContainer)
	})
}

// Validation holds configuration options for JWT validation interceptors.
type Validation struct {
	AnonymousOnFailure bool
}

// ValidationOption is a functional option for configuring JWT validation.
type ValidationOption func(*Validation)

// WithAnonymousOnFailure returns a ValidationOption that allows requests to proceed
// as anonymous when JWT validation fails, instead of returning an error.
func WithAnonymousOnFailure() ValidationOption {
	return func(v *Validation) {
		v.AnonymousOnFailure = true
	}
}

// claimsToPropagate returns which claims to copy into the logging property bag: the default
// set plus any registered RequestClaimsContextPropagateConfig. It is resolved once, from the
// root container, when the interceptor is built -- register the config as an instance or
// singleton. It used to be looked up (usually failing, as few apps register it) and the map
// rebuilt on every request.
func claimsToPropagate(rootContainer di.Container) *fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig {
	merged := &fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig{
		ClaimToContextMap: map[string]string{},
		ContextOrigin:     fluffycore_wellknown.ContextOriginKey,
	}
	if rootContainer != nil {
		if cfg, err := di.TryGet[*fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig](rootContainer); err == nil && cfg != nil {
			for k, v := range cfg.ClaimToContextMap {
				merged.ClaimToContextMap[k] = v
			}
			merged.ContextOrigin = cfg.ContextOrigin
		}
	}
	for _, claimType := range []string{
		fluffycore_wellknown.ClaimTypeSub,
		fluffycore_wellknown.ClaimTypeClientID,
		fluffycore_wellknown.ClaimTypeEmail,
		fluffycore_wellknown.ClaimTypeAud,
	} {
		merged.ClaimToContextMap[claimType] = claimType
	}
	return merged
}

// UnaryServerInterceptor returns a gRPC unary server interceptor that validates JWTs
// and populates the scoped IClaimsPrincipal with token claims.
func UnaryServerInterceptor(rootContainer di.Container, opts ...ValidationOption) grpc.UnaryServerInterceptor {
	_loadValidators(rootContainer)
	validation := &Validation{}
	for _, opt := range opts {
		opt(validation)
	}
	requestContextClaimsToPropagate := claimsToPropagate(rootContainer)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		scopedContainer := dicontext.GetRequestContainer(ctx)
		claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)
		propertyBag := di.Get[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](scopedContainer)

		rt, err := _validate(ctx)
		if err != nil {
			if validation.AnonymousOnFailure {
				claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
					Type:  fluffycore_wellknown.ClaimTypeSub,
					Value: fluffycore_wellknown.AnonymousSubject,
				})
				propertyBag.Set(fluffycore_wellknown.ClaimTypeSub, fluffycore_wellknown.AnonymousSubject)
				return handler(ctx, req)
			}
			e, ok := status.FromError(err)
			if ok {
				if e.Code() == codes.NotFound {
					claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
						Type:  fluffycore_wellknown.ClaimTypeSub,
						Value: fluffycore_wellknown.AnonymousSubject,
					})
					propertyBag.Set(fluffycore_wellknown.ClaimTypeSub, fluffycore_wellknown.AnonymousSubject)
					return handler(ctx, req)
				}
			}
			return nil, err
		}
		jwtToken := newJWTToken(rt.Token, rt.AccessToken)
		claimsPrincipalScratch := fluffycore_services_common_claimsprincipal.ClaimsPrincipalFromClaimsMap(jwtToken.GetClaims())
		// transfer the claims over to the scoped IClaimsPrincipal
		// mtls_* claims are reserved for the mtls interceptor; a token must not assert them.
		claimsPrincipal.AddClaim(fluffycore_services_common_claimsprincipal.WithoutReservedMTLSClaims(claimsPrincipalScratch.GetClaims())...)
		if jwtToken.GetID() != "" {
			claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
				Type:  fluffycore_wellknown.ClaimTypeSub,
				Value: jwtToken.GetID(),
			})
		}
		for claimType, ctxType := range requestContextClaimsToPropagate.ClaimToContextMap {
			if claimsPrincipal.HasClaimType(claimType) {
				claimVal := claimsPrincipal.GetClaimsByType(claimType)
				if len(claimVal) > 0 {
					if len(claimVal) == 1 {
						propertyBag.Set(ctxType, claimVal[0].Value)
					} else {
						values := make([]string, 0, len(claimVal))
						for _, v := range claimVal {
							values = append(values, v.Value)
						}
						propertyBag.Set(ctxType, values)
					}
				}
			}
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that validates JWTs
// and populates the scoped IClaimsPrincipal with token claims.
//
// It follows the same rules as UnaryServerInterceptor: a request with no Authorization
// header at all proceeds as anonymous (sub=anonymous) so the claims-principal gate --
// not this interceptor -- decides whether the method allows that; an invalid token is
// rejected unless WithAnonymousOnFailure is set.
func StreamServerInterceptor(rootContainer di.Container, opts ...ValidationOption) grpc.StreamServerInterceptor {
	_loadValidators(rootContainer)
	validation := &Validation{}
	for _, opt := range opts {
		opt(validation)
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		scopedContainer := dicontext.GetRequestContainer(ctx)
		if scopedContainer == nil {
			return status.Error(codes.Internal, "authentication unavailable")
		}
		claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)
		rt, err := _validate(ctx)
		if err != nil {
			if e, ok := status.FromError(err); validation.AnonymousOnFailure || (ok && e.Code() == codes.NotFound) {
				claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
					Type:  fluffycore_wellknown.ClaimTypeSub,
					Value: fluffycore_wellknown.AnonymousSubject,
				})
				return handler(srv, ss)
			}
			return err
		}
		jwtToken := newJWTToken(rt.Token, rt.AccessToken)
		claimsPrincipalScratch := fluffycore_services_common_claimsprincipal.ClaimsPrincipalFromClaimsMap(jwtToken.GetClaims())
		// transfer the claims over to the scoped IClaimsPrincipal
		// mtls_* claims are reserved for the mtls interceptor; a token must not assert them.
		claimsPrincipal.AddClaim(fluffycore_services_common_claimsprincipal.WithoutReservedMTLSClaims(claimsPrincipalScratch.GetClaims())...)

		if jwtToken.GetID() != "" {
			claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
				Type:  fluffycore_wellknown.ClaimTypeSub,
				Value: jwtToken.GetID(),
			})
		}
		return handler(srv, ss)
	}
}

func getRawToken(ctx context.Context, accessToken string) (*fluffycore_contracts_middleware_auth_jwt.ParsedToken, error) {
	// Decode the JWS once, and read the (not yet trusted) claims from its payload. This used
	// to parse the whole token twice here (jwt.Parse, then jws.Parse) and a third time to
	// verify; ValidateAccessToken now verifies the signature and validates these same claims.
	notValidatedTokenMsg, err := jws.ParseString(accessToken)
	if err != nil || len(notValidatedTokenMsg.Signatures()) == 0 {
		msg := "Failed to parse JWT. Invalid headers"
		zerolog.Ctx(ctx).Warn().Err(err).Msg(msg)
		return nil, status.Error(codes.Unauthenticated, msg)
	}
	notTrustedToken, err := jwxt.Parse(notValidatedTokenMsg.Payload(),
		jwxt.WithValidate(false),
		jwxt.WithVerify(false))
	if err != nil {
		msg := "Failed to parse JWT. Invalid format"
		zerolog.Ctx(ctx).Warn().Err(err).Msg(msg)
		return nil, status.Error(codes.Unauthenticated, msg)
	}
	rt := &fluffycore_contracts_middleware_auth_jwt.ParsedToken{
		Token:       notTrustedToken,
		JWSMessage:  notValidatedTokenMsg,
		AccessToken: accessToken,
	}
	return rt, nil
}
func checkAudienceMatch(token jwxt.Token, expectedAudience string) bool {
	// If a wildcard, do not check audience
	if expectedAudience == "*" {
		return true
	}

	// JWT may contain several audiences
	// At least one of them should match
	audiences, _ := token.Audience()
	for _, v := range audiences {
		if v == expectedAudience {
			return true
		}
	}

	return false
}

// firstLoadWait is the longest a request waits for a first fetch that is still running before it fetches the
// keys itself.
const firstLoadWait = 2 * time.Second

// loadKeySet returns the key set of a JWKS URL that has not been fetched yet.
//
// If the first fetch that Register started in the background is still running, it waits for it (up to
// firstLoadWait) instead of forcing a refresh on top of it: a second fetch would be wasted, and httprc
// (3.0.6) deadlocks when more concurrent refreshes arrive than it has workers, which a burst of first
// requests would cause.
//
// If that fetch is not running, it already failed, and the cache's own retry is minutes away. The usual
// cause is the JWKS URL being served by this very process (the issuer is the server's own token service,
// for example), which is not listening yet when Register runs at startup. Waiting would never help, so the
// keys are fetched here straight away, one request at a time per URL (the gate serialises them, which also
// keeps the refreshes below httprc's worker count) and at most once per minimum refresh interval.
// Requests that arrive in between fail at once instead of each waiting.
func loadKeySet(ctx context.Context, jwksURL string, gate *jwksRefreshGate) (jwxk.Set, error) {
	r, err := _cache.LookupResource(ctx, jwksURL)
	if err != nil {
		return nil, err
	}

	// A fetch already failed a moment ago: don't make every request wait through it again.
	gate.mu.Lock()
	recentlyFailed := !gate.last.IsZero() && time.Since(gate.last) < time.Duration(_minJWKSRefreshInterval.Load())
	gate.mu.Unlock()
	if recentlyFailed {
		if set, err := _cache.Lookup(ctx, jwksURL); err == nil {
			return set, nil
		}
		return nil, fmt.Errorf("the keys at %s are not available (the last attempt to fetch them failed)", jwksURL)
	}

	if r.IsBusy() {
		waitStarted := time.Now()
		waitCtx, cancel := context.WithTimeout(ctx, firstLoadWait)
		err := r.Ready(waitCtx)
		cancel()
		zerolog.Ctx(ctx).Debug().Str("jwks_url", jwksURL).Bool("ready", err == nil).Dur("waited", time.Since(waitStarted)).
			Msg("waited for the first JWKS fetch")
		if err == nil {
			return _cache.Lookup(ctx, jwksURL)
		}
	}

	gate.mu.Lock()
	defer gate.mu.Unlock()
	// another request may have loaded it while this one waited for the gate
	if set, err := _cache.Lookup(ctx, jwksURL); err == nil {
		return set, nil
	}
	if !gate.last.IsZero() && time.Since(gate.last) < time.Duration(_minJWKSRefreshInterval.Load()) {
		return nil, fmt.Errorf("the keys at %s are not available (the last attempt to fetch them failed)", jwksURL)
	}
	gate.last = time.Now()
	return refreshJWKS(ctx, jwksURL, "first load")
}
