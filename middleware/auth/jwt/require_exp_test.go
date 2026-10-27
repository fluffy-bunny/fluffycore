package jwt

import (
	"testing"
	"time"

	jwa "github.com/lestrrat-go/jwx/v4/jwa"
	jwxk "github.com/lestrrat-go/jwx/v4/jwk"
	jwxt "github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// A correctly signed token from a trusted issuer but without exp never expires.
// jwxt.Validate accepts it, so the validator must require exp itself.
func TestJWT_SignedTokenWithoutExpIsRejected(t *testing.T) {
	idp := newFakeIdP(t)
	key := newSigningKey(t, "k1")
	idp.publish(t, key)
	interceptor := withIdP(t, idp, time.Minute)

	reached, err := callWithToken(interceptor, signedToken(t, idp.issuer(), key))
	require.NoError(t, err)
	require.True(t, reached, "control: the same token with an exp is accepted")

	reached, err = callWithToken(interceptor, signedTokenNoExp(t, idp.issuer(), key))
	require.False(t, reached)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func signedTokenNoExp(t *testing.T, issuer string, key jwxk.Key) string {
	t.Helper()
	tok, err := jwxt.NewBuilder().Issuer(issuer).Subject("user-1").IssuedAt(time.Now()).Build()
	require.NoError(t, err)
	b, err := jwxt.Sign(tok, jwxt.WithKey(jwa.RS256(), key))
	require.NoError(t, err)
	return string(b)
}
