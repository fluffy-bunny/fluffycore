package jwt

import (
	"context"
	"sync"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
)

// Two issuers that are the same server under two names share one JWKS URL, e.g.
//
//	jwtValidators__issuers  = http://localhost:9802,http://business-micro:9802
//	jwtValidators__jwksUrls = http://localhost:9802/.well-known/jwks,http://localhost:9802/.well-known/jwks
//
// AddValidators used to register the URL with the JWKS cache once per issuer, and the cache refuses a second
// registration, so the service panicked at startup with "resource already exists".
func TestAddValidators_IssuersSharingOneJWKSURL(t *testing.T) {
	idp := newFakeIdP(t)
	key := newSigningKey(t, "kid-1")
	idp.publish(t, key)

	const secondIssuer = "http://second-name-for-the-same-server"

	prevInterval := time.Duration(_minJWKSRefreshInterval.Load())
	t.Cleanup(func() {
		SetMinJWKSRefreshInterval(prevInterval)
		_validatorsOnce = sync.Once{}
		_validators = nil
	})
	_validatorsOnce = sync.Once{}
	_validators = nil

	configs := &fluffycore_contracts_middleware_auth_jwt.IssuerConfigs{
		IssuerConfigs: []*fluffycore_contracts_middleware_auth_jwt.IssuerConfig{
			{OAuth2Config: &fluffycore_contracts_middleware_auth_jwt.OAuth2Config{Issuer: idp.issuer(), JWKSUrl: idp.jwksURL()}},
			{OAuth2Config: &fluffycore_contracts_middleware_auth_jwt.OAuth2Config{Issuer: secondIssuer, JWKSUrl: idp.jwksURL()}},
		},
	}
	b := di.Builder()
	fluffycore_services_common.AddCommonServices(b)
	require.NotPanics(t, func() { AddValidators(b, configs) }, "two issuers sharing a JWKS URL")
	// and the same URLs again, as when AddValidators is called twice in one process
	require.NotPanics(t, func() { AddValidators(di.Builder(), configs) }, "the same URL registered twice")
	root := b.Build()

	interceptor := UnaryServerInterceptor(root)
	call := func(token string) (bool, error) {
		scoped := func(ic grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
			return func(ctx2 context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				scope := di.Get[di.ScopeFactory](root).CreateScope()
				defer scope.Dispose()
				return ic(dicontext.SetRequestContainer(ctx2, scope.Container()), req, info, handler)
			}
		}(interceptor)
		return callWithToken(scoped, token)
	}

	// each issuer has its own validator, and both are checked against the shared keys
	reached, err := call(signedToken(t, idp.issuer(), key))
	require.NoError(t, err)
	require.True(t, reached, "a token from the first issuer must be accepted")

	reached, err = call(signedToken(t, secondIssuer, key))
	require.NoError(t, err)
	require.True(t, reached, "a token from the second issuer must be accepted")

	reached, err = call(signedToken(t, "http://not-a-configured-issuer", key))
	require.Error(t, err)
	require.False(t, reached, "an issuer nobody configured must still be rejected")
}
