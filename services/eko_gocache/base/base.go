package base

import (
	"context"
	"encoding/json"
	"time"

	cache "github.com/eko/gocache/lib/v4/cache"
	store "github.com/eko/gocache/lib/v4/store"
)

type (
	BaseEkoGoCache struct {
		CacheManager *cache.Cache[[]byte]
	}
)

func (s *BaseEkoGoCache) Set(ctx context.Context, key any, value any, options ...store.Option) error {
	generic, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.CacheManager.Set(ctx, key, generic, options...)
}

func (s *BaseEkoGoCache) GetOrInsert(ctx context.Context, key string, f func(ctx context.Context) (any, error), options ...store.Option) (any, error) {
	result, err := s.Get(ctx, key)
	if err != nil {
		result, err = f(ctx)
		if err != nil {
			return nil, err
		}
		err = s.Set(ctx, key, result, options...)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return result, nil
}

func (s *BaseEkoGoCache) Get(ctx context.Context, key any) (any, error) {
	resultB, err := s.CacheManager.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var result any
	err = json.Unmarshal(resultB, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *BaseEkoGoCache) GetWithTTL(ctx context.Context, key any) (any, time.Duration, error) {
	resultB, duration, err := s.CacheManager.GetWithTTL(ctx, key)
	if err != nil {
		return nil, 0, err
	}
	var result any
	err = json.Unmarshal(resultB, &result)
	if err != nil {
		return nil, 0, err
	}
	return result, duration, nil
}

func (s *BaseEkoGoCache) Delete(ctx context.Context, key any) error {
	return s.CacheManager.Delete(ctx, key)
}

func (s *BaseEkoGoCache) Invalidate(ctx context.Context, options ...store.InvalidateOption) error {
	return s.CacheManager.Invalidate(ctx, options...)
}

func (s *BaseEkoGoCache) Clear(ctx context.Context) error {
	return s.CacheManager.Clear(ctx)
}

func (s *BaseEkoGoCache) GetType() string {
	return "eko_gocache"
}
