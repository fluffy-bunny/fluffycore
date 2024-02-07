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

func AddISingletonInMemoryCache(cb di.ContainerBuilder, implementedInterfaceTypes ...reflect.Type) {
	reflectType := []reflect.Type{
		reflect.TypeOf((*fluffycore_contracts_eko_gocache.ISingletonInMemoryCache)(nil)),
		reflect.TypeOf((*fluffycore_contracts_eko_gocache.IGoCache)(nil)),
	}
	reflectType = append(reflectType, implementedInterfaceTypes...)
	di.AddSingleton[*service](cb, stemService.Ctor, reflectType...)
}
func (s *service) GetType() string {
	return "go-cache"
}
