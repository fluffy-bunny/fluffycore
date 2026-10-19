package jwtminter

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	fluffycore_services_claims "github.com/fluffy-bunny/fluffycore/services/claims"
	fluffycore_services_keymaterial "github.com/fluffy-bunny/fluffycore/services/keymaterial"
	pkcs8enc "github.com/fluffy-bunny/fluffycore/utils/pkcs8enc"
	"github.com/stretchr/testify/require"
)

// A signing key stored as encrypted PKCS#8 (what utils/ecdsa.Encode writes now, or
// `openssl pkcs8 -topk8`) must be usable by the minter.
func TestMintToken_EncryptedPKCS8SigningKey(t *testing.T) {
	km := &fluffycore_contracts_jwtminter.KeyMaterial{}
	require.NoError(t, json.Unmarshal([]byte(GetSigningKeysES256_JSON()), km))
	sk := km.SigningKeys[0]
	block, _ := pem.Decode([]byte(sk.PrivateKey))
	ecKey, err := x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	enc, err := pkcs8enc.Encrypt(der, "pw")
	require.NoError(t, err)
	sk.PrivateKey = string(pem.EncodeToMemory(enc))
	sk.Password = "pw"

	b := di.Builder()
	di.AddInstance[*fluffycore_contracts_jwtminter.KeyMaterial](b, km)
	AddSingletonIJWTMinter(b)
	fluffycore_services_keymaterial.AddSingletonIKeyMaterial(b)
	minter := di.Get[fluffycore_contracts_jwtminter.IJWTMinter](b.Build())

	claims := fluffycore_services_claims.NewClaims()
	claims.Set("sub", "u")
	claims.Set("exp", time.Now().Add(time.Hour).Unix())
	token, err := minter.MintToken(context.Background(), claims)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}
