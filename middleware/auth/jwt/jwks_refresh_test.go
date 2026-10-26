package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	status "github.com/gogo/status"
	jwa "github.com/lestrrat-go/jwx/v4/jwa"
	jwxk "github.com/lestrrat-go/jwx/v4/jwk"
	jwxt "github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
)

// fakeIdP serves a JWKS whose keys can change, and counts fetches.
type fakeIdP struct {
	srv     *httptest.Server
	fetches atomic.Int64
	mu      sync.Mutex
	keys    jwxk.Set
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	idp := &fakeIdP{keys: jwxk.NewSet()}
	idp.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idp.fetches.Add(1)
		idp.mu.Lock()
		b, _ := json.Marshal(idp.keys)
		idp.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	t.Cleanup(idp.srv.Close)
	return idp
}

func (idp *fakeIdP) issuer() string  { return idp.srv.URL }
func (idp *fakeIdP) jwksURL() string { return idp.srv.URL + "/jwks" }

func (idp *fakeIdP) publish(t *testing.T, priv jwxk.Key) {
	t.Helper()
	pub, err := jwxk.PublicKeyOf(priv)
	require.NoError(t, err)
	idp.mu.Lock()
	defer idp.mu.Unlock()
	require.NoError(t, idp.keys.AddKey(pub))
}

func newSigningKey(t *testing.T, kid string) jwxk.Key {
	t.Helper()
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	k, err := jwxk.Import[jwxk.Key](raw)
	require.NoError(t, err)
	require.NoError(t, k.Set(jwxk.KeyIDKey, kid))
	require.NoError(t, k.Set(jwxk.AlgorithmKey, jwa.RS256()))
	return k
}

func signedToken(t *testing.T, issuer string, key jwxk.Key) string {
	t.Helper()
	tok, err := jwxt.NewBuilder().Issuer(issuer).Subject("user-1").
		IssuedAt(time.Now()).Expiration(time.Now().Add(time.Hour)).Build()
	require.NoError(t, err)
	b, err := jwxt.Sign(tok, jwxt.WithKey(jwa.RS256(), key))
	require.NoError(t, err)
	return string(b)
}

// unsignedRandomKidToken is what an attacker can send knowing only the public issuer URL.
func unsignedRandomKidToken(issuer string, i int) string {
	enc := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	return enc(fmt.Sprintf(`{"alg":"RS256","kid":"random-%d"}`, i)) + "." +
		enc(fmt.Sprintf(`{"iss":%q}`, issuer)) + ".AAAA"
}

// withIdP wires the package-level validators to idp for one test. The jwt package pins its
// validators with a package-level sync.Once, so this resets that state (and the refresh
// interval) around the test.
func withIdP(t *testing.T, idp *fakeIdP, minRefresh time.Duration) grpc.UnaryServerInterceptor {
	t.Helper()
	root := withIdPRoot(t, idp, minRefresh)
	interceptor := UnaryServerInterceptor(root)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		scope := di.Get[di.ScopeFactory](root).CreateScope()
		defer scope.Dispose()
		return interceptor(dicontext.SetRequestContainer(ctx, scope.Container()), req, info, handler)
	}
}

// withIdPRoot does withIdP's setup and returns the root container.
func withIdPRoot(t *testing.T, idp *fakeIdP, minRefresh time.Duration) di.Container {
	t.Helper()
	prevInterval := time.Duration(_minJWKSRefreshInterval.Load())
	SetMinJWKSRefreshInterval(minRefresh)
	t.Cleanup(func() {
		SetMinJWKSRefreshInterval(prevInterval)
		_validatorsOnce = sync.Once{}
		_validators = nil
	})
	_validatorsOnce = sync.Once{}
	_validators = nil

	b := di.Builder()
	fluffycore_services_common.AddCommonServices(b)
	AddValidators(b, &fluffycore_contracts_middleware_auth_jwt.IssuerConfigs{
		IssuerConfigs: []*fluffycore_contracts_middleware_auth_jwt.IssuerConfig{
			{OAuth2Config: &fluffycore_contracts_middleware_auth_jwt.OAuth2Config{Issuer: idp.issuer(), JWKSUrl: idp.jwksURL()}},
		},
	})
	return b.Build()
}

func callWithToken(interceptor grpc.UnaryServerInterceptor, token string) (reached bool, err error) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Svc/M"},
		func(ctx context.Context, req interface{}) (interface{}, error) { reached = true; return nil, nil })
	return reached, err
}

// The attack: unsigned tokens with random kids used to force a JWKS fetch each (20
// requests -> 21 fetches). Now at most one forced refresh per interval.
func TestJWKS_RandomKidFloodIsThrottled(t *testing.T) {
	idp := newFakeIdP(t)
	interceptor := withIdP(t, idp, time.Minute)

	for i := 0; i < 20; i++ {
		reached, err := callWithToken(interceptor, unsignedRandomKidToken(idp.issuer(), i))
		require.False(t, reached)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	}
	require.LessOrEqual(t, idp.fetches.Load(), int64(2), "initial load + at most one forced refresh")
}

func TestJWKS_ConcurrentRandomKidFloodIsThrottled(t *testing.T) {
	idp := newFakeIdP(t)
	interceptor := withIdP(t, idp, time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			reached, err := callWithToken(interceptor, unsignedRandomKidToken(idp.issuer(), i))
			if reached || status.Code(err) != codes.Unauthenticated {
				t.Errorf("request %d: reached=%v err=%v", i, reached, err)
			}
		}(i)
	}
	wg.Wait()
	require.LessOrEqual(t, idp.fetches.Load(), int64(2), "concurrent misses must not each fetch")
}

// Throttling must not break genuine key rollover: the IdP publishes a new key, and the first
// token signed with it triggers a refresh (none has happened yet) and is accepted.
func TestJWKS_KeyRolloverPickedUpImmediately(t *testing.T) {
	idp := newFakeIdP(t)
	oldKey := newSigningKey(t, "old")
	idp.publish(t, oldKey)
	interceptor := withIdP(t, idp, time.Minute)

	reached, err := callWithToken(interceptor, signedToken(t, idp.issuer(), oldKey))
	require.NoError(t, err)
	require.True(t, reached)

	newKey := newSigningKey(t, "new")
	idp.publish(t, newKey)
	reached, err = callWithToken(interceptor, signedToken(t, idp.issuer(), newKey))
	require.NoError(t, err, "a genuine rollover is picked up by the first refresh")
	require.True(t, reached)

	// The old key still works too (it's still published).
	reached, err = callWithToken(interceptor, signedToken(t, idp.issuer(), oldKey))
	require.NoError(t, err)
	require.True(t, reached)
}

// A kid that shows up after a recent refresh is picked up once the interval passes.
func TestJWKS_UnknownKidRetriedAfterInterval(t *testing.T) {
	idp := newFakeIdP(t)
	interval := 300 * time.Millisecond
	interceptor := withIdP(t, idp, interval)
	newKey := newSigningKey(t, "late")
	token := signedToken(t, idp.issuer(), newKey)

	// Not published yet: initial load + one forced refresh, both without the kid.
	_, err := callWithToken(interceptor, token)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	fetchesAfterFirst := idp.fetches.Load()

	idp.publish(t, newKey)
	// Inside the interval: rejected without fetching again.
	_, err = callWithToken(interceptor, token)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.Equal(t, fetchesAfterFirst, idp.fetches.Load(), "no fetch inside the interval")

	time.Sleep(interval + 100*time.Millisecond)
	reached, err := callWithToken(interceptor, token)
	require.NoError(t, err, "after the interval the new key is fetched and the token accepted")
	require.True(t, reached)
}

// A request whose kid is already cached must never trigger a refresh.
func TestJWKS_KnownKidNeverRefreshes(t *testing.T) {
	idp := newFakeIdP(t)
	key := newSigningKey(t, "k1")
	idp.publish(t, key)
	interceptor := withIdP(t, idp, time.Minute)

	for i := 0; i < 10; i++ {
		reached, err := callWithToken(interceptor, signedToken(t, idp.issuer(), key))
		require.NoError(t, err)
		require.True(t, reached)
	}
	require.EqualValues(t, 1, idp.fetches.Load(), "only the initial load")
}
