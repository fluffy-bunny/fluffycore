package jwtminter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	fluffycore_services_claims "github.com/fluffy-bunny/fluffycore/services/claims"
	fluffycore_services_keymaterial "github.com/fluffy-bunny/fluffycore/services/keymaterial"
	require "github.com/stretchr/testify/require"
)

func TestJwtMinter(t *testing.T) {
	b := di.Builder()
	b.ConfigureOptions(func(o *di.Options) {
		o.ValidateScopes = true
		o.ValidateOnBuild = true
	})
	keyMaterialJSON := GetSigningKeysJSON()
	keymaterial := &fluffycore_contracts_jwtminter.KeyMaterial{}
	err := json.Unmarshal([]byte(keyMaterialJSON), keymaterial)
	require.NoError(t, err)
	di.AddInstance[*fluffycore_contracts_jwtminter.KeyMaterial](b, keymaterial)
	// order maters for Singleton and Transient, they are both app scoped and the last one wins
	AddSingletonIJWTMinter(b)
	fluffycore_services_keymaterial.AddSingletonIKeyMaterial(b)
	container := b.Build()

	jwtMinter := di.Get[fluffycore_contracts_jwtminter.IJWTMinter](container)
	require.NotNil(t, jwtMinter)

	now := time.Now()
	expiration := now.Add(24 * time.Hour).Unix()
	claims := fluffycore_services_claims.NewClaims()
	claims.Set("sub", "1234567890")
	claims.Set("name", "John Doe")
	claims.Set("iss", "http://example.com")
	claims.Set("exp", expiration)

	token, err := jwtMinter.MintToken(context.TODO(), claims)
	require.NoError(t, err)
	require.NotEmpty(t, token)

}
