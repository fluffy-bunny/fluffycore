package cache

import (
	"time"

	"github.com/dozm/di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	ttlcache "github.com/jellydator/ttlcache/v2"
)

type (
	service struct {
		ttlCache ttlcache.SimpleCache
	}
)

// AddMemoryCache adds service to the DI container
func AddMemoryCache(b di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.IMemoryCache](b, func() fluffycore_contracts_common.IMemoryCache {
		return NewMemoryCache()
	})
	di.AddScoped[fluffycore_contracts_common.IScopedMemoryCache](b, func() fluffycore_contracts_common.IMemoryCache {
		return NewMemoryCache()
	})
}

func NewMemoryCache() fluffycore_contracts_common.IMemoryCache {
	s := &service{
		ttlCache: ttlcache.NewCache(),
	}
	s.SetTTL(fluffycore_contracts_common.Forever)
	return s
}

func (s *service) Get(key string) (interface{}, error) {
	return s.ttlCache.Get(key)
}

func (s *service) GetWithTTL(key string) (interface{}, time.Duration, error) {
	return s.ttlCache.GetWithTTL(key)
}

func (s *service) Set(key string, data interface{}) error {
	return s.ttlCache.Set(key, data)
}

func (s *service) SetTTL(ttl time.Duration) error {
	return s.ttlCache.SetTTL(ttl)
}

func (s *service) SetWithTTL(key string, data interface{}, ttl time.Duration) error {
	return s.ttlCache.SetWithTTL(key, data, ttl)
}

func (s *service) Remove(key string) error {
	return s.ttlCache.Remove(key)
}

func (s *service) Close() error {
	return s.ttlCache.Close()
}

func (s *service) Purge() error {
	return s.ttlCache.Purge()
}

func (s *service) GetOrInsert(k string, adder func() (interface{}, time.Duration, error)) interface{} {
	result, err := s.Get(k)
	if err != nil || result == nil {
		obj, ttl, err := adder()
		if err != nil {
			return nil
		}
		s.SetWithTTL(k, obj, ttl)
		result = obj
	}
	return result
}
