package jwt

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	jwa "github.com/lestrrat-go/jwx/v4/jwa"
	jwxt "github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
)

// The token is now parsed once and its signature verified separately; each of these must
// still be rejected.
func TestValidate_RejectsBadTokensAfterSingleParse(t *testing.T) {
	idp := newFakeIdP(t)
	key := newSigningKey(t, "k1")
	idp.publish(t, key)
	interceptor := withIdP(t, idp, time.Minute)

	valid := signedToken(t, idp.issuer(), key)
	reached, err := callWithToken(interceptor, valid)
	require.NoError(t, err)
	require.True(t, reached)

	// Claims swapped under an unchanged signature.
	parts := strings.Split(valid, ".")
	forged := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"` + idp.issuer() + `","sub":"admin","exp":9999999999}`))
	tampered := parts[0] + "." + forged + "." + parts[2]

	expiredTok, err := jwxt.NewBuilder().Issuer(idp.issuer()).Subject("u").
		IssuedAt(time.Now().Add(-2 * time.Hour)).Expiration(time.Now().Add(-time.Hour)).Build()
	require.NoError(t, err)
	expired, err := jwxt.Sign(expiredTok, jwxt.WithKey(jwa.RS256(), key))
	require.NoError(t, err)

	otherKey := newSigningKey(t, "k1") // same kid, different key: signature can't verify
	wrongKey := signedToken(t, idp.issuer(), otherKey)

	for name, tok := range map[string]string{"tampered claims": tampered, "expired": string(expired), "wrong key": wrongKey, "garbage": "a.b.c", "no signature": parts[0] + "." + parts[1]} {
		t.Run(name, func(t *testing.T) {
			reached, err := callWithToken(interceptor, tok)
			require.False(t, reached)
			require.Equal(t, codes.Unauthenticated, status.Code(err), "err=%v", err)
		})
	}
}

func TestClaimsToPropagate_MergesRootConfigWithDefaults(t *testing.T) {
	require.Equal(t, fluffycore_wellknown.ContextOriginKey, claimsToPropagate(nil).ContextOrigin)

	b := di.Builder()
	di.AddInstance[*fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig](b, &fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig{
		ClaimToContextMap: map[string]string{"tenant": "tenant_id"},
		ContextOrigin:     "x-origin",
	})
	got := claimsToPropagate(b.Build())
	require.Equal(t, "x-origin", got.ContextOrigin)
	require.Equal(t, "tenant_id", got.ClaimToContextMap["tenant"])
	require.Equal(t, fluffycore_wellknown.ClaimTypeSub, got.ClaimToContextMap[fluffycore_wellknown.ClaimTypeSub])
}
