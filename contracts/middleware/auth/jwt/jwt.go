package jwt

import (
	"context"

	jws "github.com/lestrrat-go/jwx/v2/jws"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
)

type (
	OIDCConfig struct {
		Issuer string `json:"issuer" mapstructure:"ISSUER"`
	}
	OAuth2Config struct {
		Issuer  string `json:"issuer" mapstructure:"ISSUER"`
		JWKSUrl string `json:"jwks_url" mapstructure:"JWKS_URL"`
	}
	IssuerConfig struct {
		OIDCConfig   *OIDCConfig   `json:"oidc" mapstructure:"OIDC"`
		OAuth2Config *OAuth2Config `json:"oauth2" mapstructure:"OAUTH2"`
		Audiences    []string      `json:"audiences" mapstructure:"AUDIENCES"`
	}
	IssuerConfigs struct {
		IssuerConfigs []*IssuerConfig `json:"issuer_configs"  mapstructure:"ISSUER_CONFIGS"`
	}
	ParsedToken struct {
		Token       jwxt.Token
		JWSMessage  *jws.Message
		AccessToken string
	}
	JWTValidatorOptions struct {
		ClockSkewMinutes    int    `json:"clockSkewMinutes"`
		ValidateSignature   *bool  `json:"validateSignature"`
		ValidateIssuer      *bool  `json:"validateIssuer"`
		Issuer              string `json:"issuer"`
		AltJWKSUrl          string `json:"altJWKSUrl"`
		AltTokenEndpointUrl string `json:"altTokenEndpointUrl"`
	}
	IValidator interface {
		ValidateAccessToken(cxt context.Context, rawToken *ParsedToken) (bool, error)
	}
)
