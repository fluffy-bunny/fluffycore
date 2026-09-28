package keymaterial

import (
	"encoding/pem"
	"sync"
	"testing"
	"time"

	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	"github.com/stretchr/testify/require"
)

func testKey(kid string, notBefore, notAfter time.Time) *fluffycore_contracts_jwtminter.SigningKey {
	return &fluffycore_contracts_jwtminter.SigningKey{
		Kid:        kid,
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "TEST KEY", Bytes: []byte("key-" + kid)})),
		NotBefore:  notBefore,
		NotAfter:   notAfter,
		PublicJwk:  fluffycore_contracts_jwtminter.PublicJwk{Kid: kid},
	}
}

func newTestService(keys ...*fluffycore_contracts_jwtminter.SigningKey) *service {
	km, _ := stemService.Ctor(&fluffycore_contracts_jwtminter.KeyMaterial{SigningKeys: keys})
	return km.(*service)
}

// The configured KeyMaterial must never be written to. Previously DecryptedPrivateKey was
// set on the shared config struct on every call, racing with minters reading it.
func TestGetSigningKey_DoesNotMutateConfig(t *testing.T) {
	now := time.Now()
	cfgKey := testKey("a", now.Add(-time.Hour), now.Add(time.Hour))
	s := newTestService(cfgKey)

	got, err := s.GetSigningKey()
	require.NoError(t, err)
	require.NotEmpty(t, got.DecryptedPrivateKey)
	require.Empty(t, cfgKey.DecryptedPrivateKey, "config struct must not be mutated")
	require.NotSame(t, cfgKey, got, "caller gets a service-owned copy, not the config pointer")
}

// Between key boundaries the active key is served from cache: same instance, no rebuild.
// Previously every call rebuilt (and re-decrypted) under the exclusive lock.
func TestGetSigningKey_CachedBetweenBoundaries(t *testing.T) {
	now := time.Now()
	s := newTestService(testKey("a", now.Add(-time.Hour), now.Add(time.Hour)))

	first, err := s.GetSigningKey()
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		again, err := s.GetSigningKey()
		require.NoError(t, err)
		require.Same(t, first, again)
	}
	require.WithinDuration(t, now.Add(time.Hour), s.nextFetchTime, time.Second,
		"next rebuild is scheduled at the active key's NotAfter")
}

// A key that becomes valid later must be picked up once its NotBefore passes -- the cache
// must not pin the old key past the boundary.
func TestGetSigningKey_RebuildsAtNextBoundary(t *testing.T) {
	now := time.Now()
	later := now.Add(150 * time.Millisecond)
	s := newTestService(
		testKey("old", now.Add(-time.Hour), now.Add(time.Hour)),
		testKey("new", later, now.Add(time.Hour)), // last active key wins once valid
	)

	got, err := s.GetSigningKey()
	require.NoError(t, err)
	require.Equal(t, "old", got.Kid)
	jwks, err := s.GetPublicWebKeys()
	require.NoError(t, err)
	require.Len(t, jwks, 1)

	time.Sleep(time.Until(later) + 50*time.Millisecond)

	got, err = s.GetSigningKey()
	require.NoError(t, err)
	require.Equal(t, "new", got.Kid)
	jwks, err = s.GetPublicWebKeys()
	require.NoError(t, err)
	require.Len(t, jwks, 2)
}

func TestGetSigningKey_NoActiveKey(t *testing.T) {
	now := time.Now()
	s := newTestService(testKey("expired", now.Add(-2*time.Hour), now.Add(-time.Hour)))
	_, err := s.GetSigningKey()
	require.Error(t, err)
}

func TestGetSigningKey_ConcurrentCallersSeeOneKey(t *testing.T) {
	now := time.Now()
	s := newTestService(testKey("a", now.Add(-time.Hour), now.Add(time.Hour)))

	const n = 64
	results := make([]*fluffycore_contracts_jwtminter.SigningKey, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			k, err := s.GetSigningKey()
			if err == nil {
				results[i] = k
			}
		}(i)
	}
	wg.Wait()
	for i := range results {
		require.NotNil(t, results[i])
		require.Same(t, results[0], results[i], "all concurrent callers get the one cached key")
	}
}

func TestNextKeyBoundary(t *testing.T) {
	now := time.Now()
	keys := []*fluffycore_contracts_jwtminter.SigningKey{
		testKey("expired", now.Add(-3*time.Hour), now.Add(-2*time.Hour)),
		testKey("active", now.Add(-time.Hour), now.Add(3*time.Hour)),
		testKey("future", now.Add(2*time.Hour), now.Add(5*time.Hour)),
	}
	require.Equal(t, now.Add(2*time.Hour), nextKeyBoundary(keys, now), "future key's NotBefore is the earliest upcoming boundary")

	past := []*fluffycore_contracts_jwtminter.SigningKey{testKey("expired", now.Add(-3*time.Hour), now.Add(-2*time.Hour))}
	require.True(t, nextKeyBoundary(past, now).After(now.Add(100*365*24*time.Hour)), "no future boundary means never rebuild")
}
