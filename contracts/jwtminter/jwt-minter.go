package jwtminter

import (
	"context"
	"time"

	fluffycore_contracts_claims "github.com/fluffy-bunny/fluffycore/contracts/claims"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
)

type (
	PublicJwk struct {
		Alg string `json:"alg"`
		Crv string `json:"crv"`
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	PrivateJwk struct {
		Alg string `json:"alg"`
		Crv string `json:"crv"`
		D   string `json:"d"`
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	SigningKey struct {
		PrivateKey          string     `json:"private_key"`
		DecryptedPrivateKey string     `json:"decrypted_private_key"`
		PublicKey           string     `json:"public_key"`
		NotBefore           time.Time  `json:"not_before"`
		NotAfter            time.Time  `json:"not_after"`
		Password            string     `json:"password"`
		Kid                 string     `json:"kid"`
		PublicJwk           PublicJwk  `json:"public_jwk"`
		PrivateJwk          PrivateJwk `json:"private_jwk"`
	}
	KeyMaterial struct {
		SigningKeys []*SigningKey `json:"signing_keys"`
	}
	IKeyMaterial interface {
		GetSigningKey() (*SigningKey, error)
		GetSigningKeys() ([]*SigningKey, error)
		GetPublicWebKeys() ([]*PublicJwk, error)
		CreateKeySet() (jwk.Set, error)
	}
	JWKSKeys struct {
		Keys []*PublicJwk `json:"keys"`
	}
	IssuerConfig struct {
		Issuer      string       `json:"issuer"`
		KeyMaterial *KeyMaterial `json:"key_material"`
	}
	// IJWTMinter interface
	IJWTMinter interface {
		MintToken(ctx context.Context, claims fluffycore_contracts_claims.IClaims) (string, error)
		PublicKeys(ctx context.Context) (*JWKSKeys, error)
	}
)
