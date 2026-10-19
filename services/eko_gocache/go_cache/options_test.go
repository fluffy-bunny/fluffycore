package go_cache

import (
	"context"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_eko_gocache "github.com/fluffy-bunny/fluffycore/contracts/eko_gocache"
	"github.com/stretchr/testify/require"
)

// The options passed here used to be ignored: items never expired.
func TestInMemoryCache_HonorsDefaultExpiration(t *testing.T) {
	expiry, cleanup := 50*time.Millisecond, 10*time.Millisecond
	b := di.Builder()
	AddISingletonInMemoryCacheWithOptions(b, &InMemoryCacheOptions{DefaultExpiration: &expiry, CleanupInterval: &cleanup})
	c := di.Get[fluffycore_contracts_eko_gocache.IGoCache](b.Build())

	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", []byte("v")))
	_, err := c.Get(ctx, "k")
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond)
	_, err = c.Get(ctx, "k")
	require.Error(t, err, "item should have expired after DefaultExpiration")
}
