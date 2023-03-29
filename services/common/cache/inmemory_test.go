package cache

import (
	"testing"
	"time"

	"github.com/dozm/di"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	"github.com/stretchr/testify/require"
)

func TestSameTypeAsScopedTransientSingleton(t *testing.T) {
	//var err error
	b := di.Builder()
	b.ConfigureOptions(func(o *di.Options) {
		o.ValidateScopes = true
		o.ValidateOnBuild = true
	})
	// order maters for Singleton and Transient, they are both app scoped and the last one wins
	AddMemoryCache(b)
	container := b.Build()

	meCache := di.Get[fluffycore_contracts_common.IMemoryCache](container)
	require.Panics(t, func() {
		di.Get[fluffycore_contracts_common.IScopedMemoryCache](container)
	})
	scopeFactory := di.Get[di.ScopeFactory](container)
	scope := scopeFactory.CreateScope()
	scopedContainer := scope.Container()

	meCacheScoped := di.Get[fluffycore_contracts_common.IScopedMemoryCache](scopedContainer)
	require.NotNil(t, meCacheScoped)

	require.NotNil(t, meCache)

	val, err := meCache.Get("test")
	require.Error(t, err)
	require.Nil(t, val)

	meCache.SetWithTTL("test", "bob", time.Second)
	val, err = meCache.Get("test")
	require.NoError(t, err)
	require.Equal(t, "bob", val)
	time.Sleep(time.Second)

	val, err = meCache.Get("test")
	require.Error(t, err)
	require.Nil(t, val)

	val = meCache.GetOrInsert("dog", func() (interface{}, time.Duration, error) {
		return "Bowie", time.Second, nil
	})
	require.Equal(t, "Bowie", val)
	time.Sleep(time.Second)

	val, err = meCache.Get("dog")
	require.Error(t, err)
	require.Nil(t, val)
}
