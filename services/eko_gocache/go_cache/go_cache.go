package go_cache

import (
	"reflect"
	"time"

	cache "github.com/eko/gocache/lib/v4/cache"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_eko_gocache "github.com/fluffy-bunny/fluffycore/contracts/eko_gocache"
	services_eko_gocache_base "github.com/fluffy-bunny/fluffycore/services/eko_gocache/base"
	gocache "github.com/patrickmn/go-cache"
)

type (
	service struct {
		services_eko_gocache_base.BaseEkoGoCache
	}
	InMemoryCacheOptions struct {
		ImplementedInterfaceTypes []reflect.Type
		// DefaultExpiration is the default value of the expiration time of each item in the cache. It can be overridden by passing a custom expiration time when calling the Set method.
		// default value is gocache.NoExpiration
		DefaultExpiration *time.Duration
		// CleanupInterval is the interval between cache cleanup runs. If it is less than or equal to zero, no cleanup goroutine is started.
		// Default is 10 minutes.
		CleanupInterval *time.Duration
	}
)

var stemService = (*service)(nil)

func init() {
	var _ fluffycore_contracts_eko_gocache.ISingletonInMemoryCache = stemService
	var _ fluffycore_contracts_eko_gocache.IGoCache = stemService

}
func (s *service) Ctor() (*service, error) {
	gocacheClient := gocache.New(gocache.NoExpiration, 10*time.Minute)
	gocacheStore := gocache_store.NewGoCache(gocacheClient)

	cacheManager := cache.New[[]byte](gocacheStore)
	ss := &service{
		BaseEkoGoCache: services_eko_gocache_base.BaseEkoGoCache{
			CacheManager: cacheManager,
		},
	}
	return ss, nil
}
func timeDurationPtr(v time.Duration) *time.Duration {
	return &v
}
func AddISingletonInMemoryCacheWithOptions(cb di.ContainerBuilder, options *InMemoryCacheOptions) {
	if options == nil {
		options = &InMemoryCacheOptions{}
	}
	if options.CleanupInterval == nil {
		options.CleanupInterval = timeDurationPtr(10 * time.Minute)
	}
	if options.DefaultExpiration == nil {
		options.DefaultExpiration = timeDurationPtr(gocache.NoExpiration)
	}

	reflectType := []reflect.Type{
		reflect.TypeOf((*fluffycore_contracts_eko_gocache.ISingletonInMemoryCache)(nil)),
		reflect.TypeOf((*fluffycore_contracts_eko_gocache.IGoCache)(nil)),
	}
	reflectType = append(reflectType, options.ImplementedInterfaceTypes...)
	di.AddSingleton[*service](cb, stemService.Ctor, reflectType...)
}
func AddISingletonInMemoryCache(cb di.ContainerBuilder, implementedInterfaceTypes ...reflect.Type) {
	options := &InMemoryCacheOptions{
		ImplementedInterfaceTypes: implementedInterfaceTypes,
		DefaultExpiration:         timeDurationPtr(gocache.NoExpiration),
		CleanupInterval:           timeDurationPtr(10 * time.Minute),
	}
	AddISingletonInMemoryCacheWithOptions(cb, options)
}
func (s *service) GetType() string {
	return "go-cache"
}
