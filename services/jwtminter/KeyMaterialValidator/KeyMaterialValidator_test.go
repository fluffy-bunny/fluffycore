package KeyMaterialValidator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	jwtminter "github.com/fluffy-bunny/fluffycore/services/jwtminter"
	fluffycore_services_keymaterial "github.com/fluffy-bunny/fluffycore/services/keymaterial"
	jwa "github.com/lestrrat-go/jwx/v2/jwa"
	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
)

const issuer = "https://issuer.example"

// keyMaterial returns an ES256 key that was valid from 48h ago until 24h ago (retired) and
// an EdDSA key that is valid now.
func keyMaterial(t *testing.T) (*fluffycore_contracts_jwtminter.KeyMaterial, *fluffycore_contracts_jwtminter.SigningKey, *fluffycore_contracts_jwtminter.SigningKey) {
	t.Helper()
	load := func(js string) *fluffycore_contracts_jwtminter.SigningKey {
		km := &fluffycore_contracts_jwtminter.KeyMaterial{}
		require.NoError(t, json.Unmarshal([]byte(js), km))
		return km.SigningKeys[0]
	}
	retired := load(jwtminter.GetSigningKeysES256_JSON())
	retired.NotBefore = time.Now().Add(-48 * time.Hour)
	retired.NotAfter = time.Now().Add(-24 * time.Hour)
	active := load(jwtminter.GetSigningKeysEdDSA_JSON())
	return &fluffycore_contracts_jwtminter.KeyMaterial{SigningKeys: []*fluffycore_contracts_jwtminter.SigningKey{retired, active}}, retired, active
}

func newValidator(t *testing.T, km *fluffycore_contracts_jwtminter.KeyMaterial, opts *fluffycore_contracts_middleware_auth_jwt.JWTValidatorOptions) *service {
	t.Helper()
	b := di.Builder()
	di.AddInstance[*fluffycore_contracts_jwtminter.KeyMaterial](b, km)
	fluffycore_services_keymaterial.AddSingletonIKeyMaterial(b)
	ikm := di.Get[fluffycore_contracts_jwtminter.IKeyMaterial](b.Build())
	v, err := stemService.Ctor(ikm, opts)
	require.NoError(t, err)
	return v.(*service)
}

func sign(t *testing.T, key *fluffycore_contracts_jwtminter.SigningKey, iat, exp time.Time) string {
	t.Helper()
	raw, err := json.Marshal(key.PrivateJwk)
	require.NoError(t, err)
	priv, err := jwxk.ParseKey(raw)
	require.NoError(t, err)
	b := jwxt.NewBuilder().Issuer(issuer).Subject("u").Expiration(exp)
	if !iat.IsZero() {
		b = b.IssuedAt(iat)
	}
	tok, err := b.Build()
	require.NoError(t, err)
	signed, err := jwxt.Sign(tok, jwxt.WithKey(jwa.SignatureAlgorithm(key.PrivateJwk.Alg), priv))
	require.NoError(t, err)
	return string(signed)
}

func opts(skewMinutes int) *fluffycore_contracts_middleware_auth_jwt.JWTValidatorOptions {
	return &fluffycore_contracts_middleware_auth_jwt.JWTValidatorOptions{Issuer: issuer, ClockSkewMinutes: skewMinutes}
}

func TestValidator_ActiveKeyAccepted(t *testing.T) {
	km, _, active := keyMaterial(t)
	v := newValidator(t, km, opts(0))
	_, err := v.ParseAccessTokenRaw(context.Background(), sign(t, active, time.Now(), time.Now().Add(time.Hour)))
	require.NoError(t, err)
}

// jwxt.Parse used to validate with zero skew before the skew-aware Validate ran.
func TestValidator_ClockSkewApplies(t *testing.T) {
	km, _, active := keyMaterial(t)
	justExpired := sign(t, active, time.Now().Add(-10*time.Minute), time.Now().Add(-30*time.Second))

	_, err := newValidator(t, km, opts(1)).ParseAccessTokenRaw(context.Background(), justExpired)
	require.NoError(t, err, "30s past exp is within a 1 minute skew")
	_, err = newValidator(t, km, opts(0)).ParseAccessTokenRaw(context.Background(), justExpired)
	require.Error(t, err)
}

// ValidateSignature=false used to fail every token: jwx refuses to parse without a key
// unless verification is explicitly turned off.
func TestValidator_ValidateSignatureFalse(t *testing.T) {
	km, _, active := keyMaterial(t)
	o := opts(0)
	no := false
	o.ValidateSignature = &no
	_, err := newValidator(t, km, o).ParseAccessTokenRaw(context.Background(), sign(t, active, time.Now(), time.Now().Add(time.Hour)))
	require.NoError(t, err)
}

// Every configured key used to verify tokens forever, including retired ones.
func TestValidator_RetiredKeyOnlyVouchesForTokensIssuedInItsWindow(t *testing.T) {
	km, retired, _ := keyMaterial(t)
	v := newValidator(t, km, opts(0))
	ctx := context.Background()

	_, err := v.ParseAccessTokenRaw(ctx, sign(t, retired, time.Now(), time.Now().Add(time.Hour)))
	require.Error(t, err, "a retired key must not vouch for a token issued now")

	_, err = v.ParseAccessTokenRaw(ctx, sign(t, retired, time.Time{}, time.Now().Add(time.Hour)))
	require.Error(t, err, "without iat the key must be valid now")

	_, err = v.ParseAccessTokenRaw(ctx, sign(t, retired, time.Now().Add(-30*time.Hour), time.Now().Add(time.Hour)))
	require.NoError(t, err, "a token issued while the key was valid still verifies until it expires")
}
