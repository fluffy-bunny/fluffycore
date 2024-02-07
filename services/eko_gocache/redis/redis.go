package redis

import (
	"reflect"

	cache "github.com/eko/gocache/lib/v4/cache"
	redis_store "github.com/eko/gocache/store/redis/v4"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_eko_gocache "github.com/fluffy-bunny/fluffycore/contracts/eko_gocache"
	services_eko_gocache_base "github.com/fluffy-bunny/fluffycore/services/eko_gocache/base"
	go_redis "github.com/redis/go-redis/v9"
)

type (
	service struct {
		services_eko_gocache_base.BaseEkoGoCache
	}
)

var stemService = (*service)(nil)

func init() {
	var _ fluffycore_contracts_eko_gocache.IGoCache = stemService
	var _ fluffycore_contracts_eko_gocache.IRedisCache = stemService

}
func (s *service) Ctor(options *fluffycore_contracts_eko_gocache.RedisCacheOptions) (*service, error) {

	// TODO: better config when needed
	redisStore := redis_store.NewRedis(go_redis.NewClient(&go_redis.Options{
		Addr:     options.Addr, //"127.0.0.1:6379",
		Network:  options.Network,
		Username: options.Username,
		Password: options.Password,
	}))

	cacheManager := cache.New[[]byte](redisStore)
	ss := &service{
		BaseEkoGoCache: services_eko_gocache_base.BaseEkoGoCache{
			CacheManager: cacheManager,
		},
	}

	return ss, nil
}

func AddIRedisCache(cb di.ContainerBuilder, implementedInterfaceTypes ...reflect.Type) {
	reflectType := []reflect.Type{
		reflect.TypeOf((*fluffycore_contracts_eko_gocache.IRedisCache)(nil)),
		reflect.TypeOf((*fluffycore_contracts_eko_gocache.IGoCache)(nil)),
	}
	reflectType = append(reflectType, implementedInterfaceTypes...)
	di.AddSingleton[*service](cb, stemService.Ctor, reflectType...)
}

func (s *service) GetType() string {
	return "redis-cache"
}
