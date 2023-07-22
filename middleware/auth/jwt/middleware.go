package jwt

import (
	"context"
	"reflect"
	"strings"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	jws "github.com/lestrrat-go/jwx/v2/jws"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
	log "github.com/rs/zerolog/log"
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

func init() {
	_cache = jwxk.NewCache(context.Background())
	var _ fluffycore_contracts_middleware_auth_jwt.IValidator = &service{}
	_issuerConfigs = make(map[string]*fluffycore_contracts_middleware_auth_jwt.IssuerConfig)
}

func AddValidators(builder di.ContainerBuilder, config *fluffycore_contracts_middleware_auth_jwt.IssuerConfigs) {
	for _, issuerConfig := range config.IssuerConfigs {
		_issuerConfigs[strings.ToLower(issuerConfig.OAuth2Config.Issuer)] = issuerConfig
		_cache.Register(issuerConfig.OAuth2Config.JWKSUrl)
		di.AddSingleton[fluffycore_contracts_middleware_auth_jwt.IValidator](builder, func(config *fluffycore_contracts_middleware_auth_jwt.IssuerConfig) *service {
			issuerConfig.OAuth2Config.Issuer = strings.ToLower(issuerConfig.OAuth2Config.Issuer)
			return &service{
				config: issuerConfig,
			}
		})
	}
}

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
		return true, status.Errorf(codes.Unauthenticated, msg)
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
	bear := md.Get("authorization")
	if len(bear) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "no authorization found")
	}
	authorization := strings.Split(bear[0], " ")
	if len(authorization) != 2 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization")
	}
	if strings.ToLower(authorization[0]) != "bearer" {
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
	token := ""
	if tokenPtr != nil {
		token = *tokenPtr
	}
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
		if err != nil {
			log.Error().Msg("ValidateAccessToken")
		}
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
	if _validators != nil {
		return
	}
	_validators = di.Get[[]fluffycore_contracts_middleware_auth_jwt.IValidator](rootContainer)
}
func UnaryServerInterceptor(rootContainer di.Container) grpc.UnaryServerInterceptor {
	_loadValidators(rootContainer)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		rt, err := _validate(ctx)
		if err != nil {
			return nil, err
		}
		jwtToken := newJWTToken(rt.Token, rt.AccessToken)
		scopedContainer := dicontext.GetRequestContainer(ctx)
		claimsPrincipal := di.Get[fluffycore_contracts_common.IClaimsPrincipal](scopedContainer)
		claimsPrincipalScratch := fluffycore_services_common_claimsprincipal.ClaimsPrincipalFromClaimsMap(jwtToken.GetClaims())
		// transfer the claims over to the scoped IClaimsPrincipal
		claimsPrincipal.AddClaim(claimsPrincipalScratch.GetClaims()...)
		if !utils.IsEmptyOrNil(jwtToken.GetID()) {
			claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
				Type:  string("sub"),
				Value: jwtToken.GetID(),
			})
		}
		return handler(ctx, req)
	}
}

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

		if !utils.IsEmptyOrNil(jwtToken.GetID()) {
			claimsPrincipal.AddClaim(fluffycore_contracts_common.Claim{
				Type:  string("sub"),
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
		log.Warn().Err(err).Msg(msg)
		return nil, status.Errorf(codes.Unauthenticated, msg)
	}

	// Parse JWT headers
	notValidatedTokenMsg, err := jws.ParseString(accessToken)
	if err != nil {
		msg := "Failed to parse JWT. Invalid headers"
		log.Warn().Err(err).Msg(msg)
		return nil, status.Errorf(codes.Unauthenticated, msg)
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
