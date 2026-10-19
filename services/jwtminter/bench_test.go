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
)

func BenchmarkMintToken_ES256(b *testing.B) {
	km := &fluffycore_contracts_jwtminter.KeyMaterial{}
	if err := json.Unmarshal([]byte(GetSigningKeysES256_JSON()), km); err != nil {
		b.Fatal(err)
	}
	builder := di.Builder()
	di.AddInstance[*fluffycore_contracts_jwtminter.KeyMaterial](builder, km)
	AddSingletonIJWTMinter(builder)
	fluffycore_services_keymaterial.AddSingletonIKeyMaterial(builder)
	minter := di.Get[fluffycore_contracts_jwtminter.IJWTMinter](builder.Build())
	claims := fluffycore_services_claims.NewClaims()
	claims.Set("sub", "u")
	claims.Set("exp", time.Now().Add(time.Hour).Unix())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := minter.MintToken(context.Background(), claims); err != nil {
			b.Fatal(err)
		}
	}
}
