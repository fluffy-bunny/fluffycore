package KeyMaterialValidator

import (
	"context"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
)

type (
	service struct {
		keyMaterial         fluffycore_contracts_jwtminter.IKeyMaterial
		JWTValidatorOptions *fluffycore_contracts_middleware_auth_jwt.JWTValidatorOptions
		keySet              jwxk.Set
	}
)

var stemService = (*service)(nil)

func init() {
	var _ fluffycore_contracts_middleware_auth_jwt.IValidator = stemService
}

func (s *service) Ctor(
	keyMaterial fluffycore_contracts_jwtminter.IKeyMaterial,
	options *fluffycore_contracts_middleware_auth_jwt.JWTValidatorOptions) (fluffycore_contracts_middleware_auth_jwt.IValidator, error) {
	keySet, err := keyMaterial.CreateKeySet()
	if err != nil {
		return nil, err
	}
	ss := &service{
		keyMaterial:         keyMaterial,
		JWTValidatorOptions: options,
		keySet:              keySet,
	}

	return ss, nil
}

// AddSingletonIKeyMaterialValidator ...
func AddSingletonIKeyMaterialValidator(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_middleware_auth_jwt.IValidator](builder, stemService.Ctor)
}

func (s *service) ValidateAccessToken(cxt context.Context,
	rawToken *fluffycore_contracts_middleware_auth_jwt.ParsedToken) (bool, error) {
	_, err := s.ParseAccessTokenRaw(cxt, rawToken.AccessToken)
	if err != nil {
		return false, err
	}
	return true, nil
}
func (s *service) shouldValidateSignature() bool {
	if s.JWTValidatorOptions.ValidateSignature == nil {
		return true
	}
	return *s.JWTValidatorOptions.ValidateSignature
}
func (s *service) shouldValidateIssuer() bool {
	if s.JWTValidatorOptions.ValidateIssuer == nil {
		return true
	}
	return *s.JWTValidatorOptions.ValidateIssuer
}
func (s *service) ParseAccessTokenRaw(ctx context.Context, accessToken string) (jwxt.Token, error) {
	// Parse the JWT
	parseOptions := []jwxt.ParseOption{}

	if s.shouldValidateSignature() {
		jwkSet := s.keySet
		parseOptions = append(parseOptions, jwxt.WithKeySet(jwkSet))
	}

	token, err := jwxt.ParseString(accessToken, parseOptions...)
	if err != nil {
		return nil, err
	}

	// This set had a key that worked
	var validationOpts []jwxt.ValidateOption
	if s.shouldValidateIssuer() {
		validationOpts = append(validationOpts, jwxt.WithIssuer(s.JWTValidatorOptions.Issuer))
	}
	// Allow clock skew
	validationOpts = append(validationOpts,
		jwxt.WithAcceptableSkew(time.Minute*time.Duration(s.JWTValidatorOptions.ClockSkewMinutes)))

	opts := validationOpts
	err = jwxt.Validate(token, opts...)
	if err != nil {
		return nil, err
	}
	return token, nil
}
