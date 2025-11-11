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
		Crv string `json:"crv,omitempty"` // For EC keys
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		X   string `json:"x,omitempty"` // For EC keys
		Y   string `json:"y,omitempty"` // For EC keys
		E   string `json:"e,omitempty"` // For RSA keys
		N   string `json:"n,omitempty"` // For RSA keys
	}
	PrivateJwk struct {
		Alg string `json:"alg"`
		Crv string `json:"crv,omitempty"` // For EC keys
		D   string `json:"d"`
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		X   string `json:"x,omitempty"`  // For EC keys
		Y   string `json:"y,omitempty"`  // For EC keys
		E   string `json:"e,omitempty"`  // For RSA keys
		N   string `json:"n,omitempty"`  // For RSA keys
		P   string `json:"p,omitempty"`  // For RSA keys
		Q   string `json:"q,omitempty"`  // For RSA keys
		Dp  string `json:"dp,omitempty"` // For RSA keys
		Dq  string `json:"dq,omitempty"` // For RSA keys
		Qi  string `json:"qi,omitempty"` // For RSA keys
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
