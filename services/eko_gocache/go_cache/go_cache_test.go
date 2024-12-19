package go_cache

import (
	"context"
	"testing"
	"time"

	store "github.com/eko/gocache/lib/v4/store"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	reflectx "github.com/fluffy-bunny/fluffy-dozm-di/reflectx"
	fluffycore_contracts_eko_gocache "github.com/fluffy-bunny/fluffycore/contracts/eko_gocache"
	require "github.com/stretchr/testify/require"
)

type (
	IMySingletonInMemoryCache interface {
		fluffycore_contracts_eko_gocache.IGoCache
	}
)

func TestMemoryCache(t *testing.T) {
	//var err error
	b := di.Builder()
	b.ConfigureOptions(func(o *di.Options) {
		o.ValidateScopes = true
		o.ValidateOnBuild = true
	})
	// order maters for Singleton and Transient, they are both app scoped and the last one wins
	AddSingletonInMemoryCacheNoExpiration(b, reflectx.TypeOf[IMySingletonInMemoryCache]())
	container := b.Build()

	var meCache IMySingletonInMemoryCache
	require.NotPanics(t, func() {
		meCache = di.Get[IMySingletonInMemoryCache](container)
	})
	require.NotNil(t, meCache)

	ctx := context.Background()
	val, err := meCache.Get(ctx, "test")
	require.Error(t, err)
	require.Nil(t, val)

	meCache.Set(ctx, "test", "bob", store.WithExpiration(time.Second))
	val, err = meCache.Get(ctx, "test")
	require.NoError(t, err)
	require.Equal(t, "bob", val)
	time.Sleep(time.Second)

	val, err = meCache.Get(ctx, "test")
	require.Error(t, err)
	require.Nil(t, val)

	val, err = meCache.GetOrInsert(ctx, "dog", func(ctx context.Context) (interface{}, error) {
		return "Bowie", nil
	}, store.WithExpiration(time.Second))
	require.NoError(t, err)
	require.Equal(t, "Bowie", val)
	time.Sleep(time.Second)

	val, err = meCache.Get(ctx, "dog")
	require.Error(t, err)
	require.Nil(t, val)
}
