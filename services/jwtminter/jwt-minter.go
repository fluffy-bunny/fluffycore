package jwtminter

import (
	"context"
	"strings"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_claims "github.com/fluffy-bunny/fluffycore/contracts/claims"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	status "github.com/gogo/status"
	jwt "github.com/golang-jwt/jwt/v5"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
		keyMaterial fluffycore_contracts_jwtminter.IKeyMaterial
	}
)

var stemService = (*service)(nil)

func init() {
	var _ fluffycore_contracts_jwtminter.IJWTMinter = stemService
}

func (s *service) Ctor(keyMaterial fluffycore_contracts_jwtminter.IKeyMaterial) (fluffycore_contracts_jwtminter.IJWTMinter, error) {
	return &service{
		keyMaterial: keyMaterial,
	}, nil
}

// AddSingletonIJWTMinter ...
func AddSingletonIJWTMinter(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_jwtminter.IJWTMinter](builder, stemService.Ctor)
}

func (s *service) MintToken(ctx context.Context, claims fluffycore_contracts_claims.IClaims) (string, error) {
	signingKey, err := s.keyMaterial.GetSigningKey()
	if err != nil {
		return "", err
	}
	var method jwt.SigningMethod
	switch signingKey.PrivateJwk.Alg {
	case "RS256":
		method = jwt.SigningMethodRS256
	case "RS384":
		method = jwt.SigningMethodRS384
	case "RS512":
		method = jwt.SigningMethodRS512
	case "ES256":
		method = jwt.SigningMethodES256
	case "ES384":
		method = jwt.SigningMethodES384
	case "ES512":
		method = jwt.SigningMethodES512
	case "EdDSA":
		method = jwt.SigningMethodEdDSA
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported signing method: %s", signingKey.PrivateJwk.Alg)
	}
	kid := signingKey.Kid
	signedKey := []byte(signingKey.DecryptedPrivateKey)

	var getKey = func() (interface{}, error) {
		var key interface{}

		if strings.HasPrefix(signingKey.PrivateJwk.Alg, "Ed") {
			v, err := jwt.ParseEdPrivateKeyFromPEM(signedKey)
			if err != nil {
				return "", err
			}
			key = v
			return key, nil
		}

		if strings.HasPrefix(signingKey.PrivateJwk.Alg, "ES") {
			v, err := jwt.ParseECPrivateKeyFromPEM(signedKey)
			if err != nil {
				return "", err
			}
			key = v
			return key, nil
		}

		v, err := jwt.ParseRSAPrivateKeyFromPEM(signedKey)
		if err != nil {
			return "", err
		}
		key = v
		return key, nil
	}

	token := jwt.NewWithClaims(method, claims.JwtClaims())
	token.Header["kid"] = kid
	key, err := getKey()
	if err != nil {
		return "", err
	}

	// special case, aud is allowed

	jwtToken, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return jwtToken, nil

}
func (s *service) PublicKeys(ctx context.Context) (*fluffycore_contracts_jwtminter.JWKSKeys, error) {
	keys, err := s.keyMaterial.GetPublicWebKeys()
	if err != nil {
		return nil, err
	}
	return &fluffycore_contracts_jwtminter.JWKSKeys{
		Keys: keys,
	}, nil
}
