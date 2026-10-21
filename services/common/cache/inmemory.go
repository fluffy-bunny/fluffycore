package cache

import (
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	ttlcache "github.com/jellydator/ttlcache/v2"
	singleflight "golang.org/x/sync/singleflight"
)

type (
	service struct {
		ttlCache ttlcache.SimpleCache
		inflight singleflight.Group
	}
)

// AddMemoryCache adds service to the DI container
func AddMemoryCache(b di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.ISingletonMemoryCache](b, func() fluffycore_contracts_common.ISingletonMemoryCache {
		return NewMemoryCache()
	})
}

// NewMemoryCache returns an in-memory cache whose TTLs are ABSOLUTE: an entry set with a
// ttl expires ttl after it was set, however often it is read. ttlcache's default is to
// slide the expiry forward on every hit, which silently turned any regularly-read entry
// into one that never expires -- e.g. a reference token's resolved claims stayed cached
// (and a revoked token kept authenticating) as long as it was used once per TTL window.
func NewMemoryCache() *service {
	c := ttlcache.NewCache()
	c.SkipTTLExtensionOnHit(true)
	s := &service{
		ttlCache: c,
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
	s.dropExpired(key)
	return s.ttlCache.Set(key, data)
}

// dropExpired clears key if it holds an expired entry that ttlcache has not cleaned up yet.
// ttlcache treats such an entry as absent, so a Set builds a NEW item while the stale one is
// still in its expiry queue; when the cleaner later evicts the stale one it deletes by key and
// takes the fresh item with it, long before its TTL. Under load the cleaner runs late enough
// for this to happen regularly: a re-resolved reference token was dropped again and resolved
// a second time inside one TTL window.
func (s *service) dropExpired(key string) {
	if _, err := s.ttlCache.Get(key); err != nil {
		// Absent or expired; Remove on an absent key is a harmless ErrNotFound.
		_ = s.ttlCache.Remove(key)
	}
}

func (s *service) SetTTL(ttl time.Duration) error {
	return s.ttlCache.SetTTL(ttl)
}

func (s *service) SetWithTTL(key string, data interface{}, ttl time.Duration) error {
	s.dropExpired(key)
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
	result, _ := s.GetOrInsertE(k, adder)
	return result
}

// GetOrInsertE returns the adder's error instead of swallowing it, and runs the adder once
// for concurrent misses on the same key (it used to run once per caller -- a thundering herd
// on whatever the adder calls when a hot key expires).
func (s *service) GetOrInsertE(k string, adder func() (interface{}, time.Duration, error)) (interface{}, error) {
	if result, err := s.Get(k); err == nil && result != nil {
		return result, nil
	}
	result, err, _ := s.inflight.Do(k, func() (interface{}, error) {
		// Another caller may have filled k while we waited to get in.
		if result, err := s.Get(k); err == nil && result != nil {
			return result, nil
		}
		obj, ttl, err := adder()
		if err != nil {
			return nil, err
		}
		if err := s.SetWithTTL(k, obj, ttl); err != nil {
			return nil, err
		}
		return obj, nil
	})
	return result, err
}
