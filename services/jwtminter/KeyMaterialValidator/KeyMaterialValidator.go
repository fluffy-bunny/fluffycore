package KeyMaterialValidator

import (
	"context"
	"fmt"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	jws "github.com/lestrrat-go/jwx/v2/jws"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
)

type (
	service struct {
		keyMaterial         fluffycore_contracts_jwtminter.IKeyMaterial
		JWTValidatorOptions *fluffycore_contracts_middleware_auth_jwt.JWTValidatorOptions
		keySet              jwxk.Set
		keysByKid           map[string]*fluffycore_contracts_jwtminter.SigningKey
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
	keys, err := keyMaterial.GetSigningKeys()
	if err != nil {
		return nil, err
	}
	keysByKid := make(map[string]*fluffycore_contracts_jwtminter.SigningKey, len(keys))
	for _, k := range keys {
		keysByKid[k.Kid] = k
	}
	ss := &service{
		keyMaterial:         keyMaterial,
		JWTValidatorOptions: options,
		keySet:              keySet,
		keysByKid:           keysByKid,
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
func (s *service) clockSkew() time.Duration {
	return time.Minute * time.Duration(s.JWTValidatorOptions.ClockSkewMinutes)
}

func (s *service) ParseAccessTokenRaw(ctx context.Context, accessToken string) (jwxt.Token, error) {
	// jwxt.Parse validates by default -- with zero skew, before the Validate below ever
	// ran, so ClockSkewMinutes had no effect. Parse without validating; Validate does it.
	parseOptions := []jwxt.ParseOption{jwxt.WithValidate(false)}
	if s.shouldValidateSignature() {
		parseOptions = append(parseOptions, jwxt.WithKeySet(s.keySet))
	} else {
		// Without this jwx refuses to parse at all ("no key provided"), so
		// ValidateSignature=false used to reject every token.
		parseOptions = append(parseOptions, jwxt.WithVerify(false))
	}

	token, err := jwxt.ParseString(accessToken, parseOptions...)
	if err != nil {
		return nil, err
	}

	var validationOpts []jwxt.ValidateOption
	if s.shouldValidateIssuer() {
		validationOpts = append(validationOpts, jwxt.WithIssuer(s.JWTValidatorOptions.Issuer))
	}
	validationOpts = append(validationOpts, jwxt.WithAcceptableSkew(s.clockSkew()))
	if err := jwxt.Validate(token, validationOpts...); err != nil {
		return nil, err
	}

	if s.shouldValidateSignature() {
		if err := s.checkSigningKeyWindow(accessToken, token); err != nil {
			return nil, err
		}
	}
	return token, nil
}

// checkSigningKeyWindow rejects a token whose signing key was not valid when the token was
// issued. The key set holds every configured key -- retired (NotAfter passed) and not-yet-
// active ones too -- and used to trust all of them forever. A key now only vouches for
// tokens issued (iat) inside its NotBefore..NotAfter window, so tokens signed just before a
// rotation keep working until they expire, but a retired key can't mint new ones. Without an
// iat, the key must be valid now.
func (s *service) checkSigningKeyWindow(accessToken string, token jwxt.Token) error {
	msg, err := jws.ParseString(accessToken)
	if err != nil {
		return err
	}
	if len(msg.Signatures()) == 0 {
		return fmt.Errorf("token has no signature")
	}
	kid := msg.Signatures()[0].ProtectedHeaders().KeyID()
	key, ok := s.keysByKid[kid]
	if !ok {
		return fmt.Errorf("token signed with unknown key %q", kid)
	}
	issued := token.IssuedAt()
	if issued.IsZero() {
		issued = time.Now()
	}
	skew := s.clockSkew()
	if issued.Before(key.NotBefore.Add(-skew)) || issued.After(key.NotAfter.Add(skew)) {
		return fmt.Errorf("token issued at %s outside signing key %q validity [%s, %s]",
			issued.UTC().Format(time.RFC3339), kid, key.NotBefore.UTC().Format(time.RFC3339), key.NotAfter.UTC().Format(time.RFC3339))
	}
	return nil
}
