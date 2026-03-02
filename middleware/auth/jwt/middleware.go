package jwt

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	copier "github.com/jinzhu/copier"
	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	jws "github.com/lestrrat-go/jwx/v2/jws"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
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

var _cache *jwxk.Cache
var _issuerConfigs map[string]*fluffycore_contracts_middleware_auth_jwt.IssuerConfig
var _validatorsOnce sync.Once

func init() {
	var _ fluffycore_contracts_middleware_auth_jwt.IValidator = &service{}
	_cache = jwxk.NewCache(context.Background())
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
		_cache.Register(issuerConfig.OAuth2Config.JWKSUrl)
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

	kid := rawToken.JWSMessage.Signatures()[0].ProtectedHeaders().KeyID()
	issuer := strings.ToLower(rawToken.Token.Issuer())
	if issuer != s.config.OAuth2Config.Issuer {
		// not our issuer, so we aren't handling it and are not returning an error
		return false, nil
	}
	// check if the issuer is in the list of issuers
	issuerConfig := _issuerConfigs[issuer]
	set, err := _cache.Get(ctx, issuerConfig.OAuth2Config.JWKSUrl)
	if err != nil {
		return true, status.Errorf(codes.Unauthenticated, "cache.Get error: %v", err)
	}
	_, ok := set.LookupKeyID(kid)
	if !ok {
		// try to refresh the cache, maybe a rollover
		set, err = _cache.Refresh(ctx, issuerConfig.OAuth2Config.JWKSUrl)
		if err != nil {
			return true, status.Errorf(codes.Unauthenticated, "cache.Refresh error: %v", err)
		}
		_, ok = set.LookupKeyID(kid)
		if !ok {
			return true, status.Errorf(codes.Unauthenticated, "no keys found")
		}
	}
	parseOptions := []jwxt.ParseOption{
		jwxt.WithKeySet(set),
		jwxt.WithAcceptableSkew(time.Minute * 5),
	}
	trustedToken, err := jwxt.ParseString(rawToken.AccessToken,
		parseOptions...)
	if err != nil {
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
	kid := rt.JWSMessage.Signatures()[0].ProtectedHeaders().KeyID()
	if len(kid) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no kid found in token")
	}
	issuer := rt.Token.Issuer()
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

// UnaryServerInterceptor returns a gRPC unary server interceptor that validates JWTs
// and populates the scoped IClaimsPrincipal with token claims.
func UnaryServerInterceptor(rootContainer di.Container, opts ...ValidationOption) grpc.UnaryServerInterceptor {
	_loadValidators(rootContainer)
	validation := &Validation{}
	for _, opt := range opts {
		opt(validation)
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		scopedContainer := dicontext.GetRequestContainer(ctx)
		claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)
		propertyBag := di.Get[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](scopedContainer)

		requestContextClaimsToPropagate, err := di.TryGet[*fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig](scopedContainer)
		if err != nil && requestContextClaimsToPropagate == nil {
			requestContextClaimsToPropagate = &fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig{
				ClaimToContextMap: map[string]string{
					fluffycore_wellknown.ClaimTypeSub:      fluffycore_wellknown.ClaimTypeSub,
					fluffycore_wellknown.ClaimTypeClientID: fluffycore_wellknown.ClaimTypeClientID,
					fluffycore_wellknown.ClaimTypeEmail:    fluffycore_wellknown.ClaimTypeEmail,
					fluffycore_wellknown.ClaimTypeAud:      fluffycore_wellknown.ClaimTypeAud,
				},
				ContextOrigin: fluffycore_wellknown.ContextOriginKey,
			}
		} else {
			// Create a new map instead of modifying the potentially shared one
			// to avoid concurrent map writes when multiple goroutines access this
			newMap := make(map[string]string, len(requestContextClaimsToPropagate.ClaimToContextMap)+4)
			for k, v := range requestContextClaimsToPropagate.ClaimToContextMap {
				newMap[k] = v
			}
			newMap[fluffycore_wellknown.ClaimTypeSub] = fluffycore_wellknown.ClaimTypeSub
			newMap[fluffycore_wellknown.ClaimTypeClientID] = fluffycore_wellknown.ClaimTypeClientID
			newMap[fluffycore_wellknown.ClaimTypeEmail] = fluffycore_wellknown.ClaimTypeEmail
			newMap[fluffycore_wellknown.ClaimTypeAud] = fluffycore_wellknown.ClaimTypeAud
			requestContextClaimsToPropagate = &fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig{
				ClaimToContextMap: newMap,
				ContextOrigin:     requestContextClaimsToPropagate.ContextOrigin,
			}
		}

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
		claimsPrincipal.AddClaim(claimsPrincipalScratch.GetClaims()...)
		if !fluffycore_utils.IsEmptyOrNil(jwtToken.GetID()) {
			claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
				Type:  fluffycore_wellknown.ClaimTypeSub,
				Value: jwtToken.GetID(),
			})
		}
		for claimType, ctxType := range requestContextClaimsToPropagate.ClaimToContextMap {
			if claimsPrincipal.HasClaimType(claimType) {
				claimVal := claimsPrincipal.GetClaimsByType(claimType)
				if fluffycore_utils.IsNotEmptyOrNil(claimVal) {
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
func StreamServerInterceptor(rootContainer di.Container) grpc.StreamServerInterceptor {
	_loadValidators(rootContainer)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		rt, err := _validate(ctx)
		if err != nil {
			return err
		}
		scopedContainer := dicontext.GetRequestContainer(ctx)
		claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)
		jwtToken := newJWTToken(rt.Token, rt.AccessToken)
		claimsPrincipalScratch := fluffycore_services_common_claimsprincipal.ClaimsPrincipalFromClaimsMap(jwtToken.GetClaims())
		// transfer the claims over to the scoped IClaimsPrincipal
		claimsPrincipal.AddClaim(claimsPrincipalScratch.GetClaims()...)

		if !fluffycore_utils.IsEmptyOrNil(jwtToken.GetID()) {
			claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
				Type:  fluffycore_wellknown.ClaimTypeSub,
				Value: jwtToken.GetID(),
			})
		}
		return handler(srv, ss)
	}
}

func getRawToken(ctx context.Context, accessToken string) (*fluffycore_contracts_middleware_auth_jwt.ParsedToken, error) {
	// Just parse JWT w/o signature check
	notTrustedToken, err := jwxt.ParseString(accessToken,
		jwxt.WithValidate(false),
		jwxt.WithVerify(false))
	if err != nil {
		msg := "Failed to parse JWT. Invalid format"
		zerolog.Ctx(ctx).Warn().Err(err).Msg(msg)
		return nil, status.Error(codes.Unauthenticated, msg)
	}

	// Parse JWT headers
	notValidatedTokenMsg, err := jws.ParseString(accessToken)
	if err != nil {
		msg := "Failed to parse JWT. Invalid headers"
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
	for _, v := range token.Audience() {
		if v == expectedAudience {
			return true
		}
	}

	return false
}
