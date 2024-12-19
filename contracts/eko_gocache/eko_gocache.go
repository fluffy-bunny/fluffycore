package eko_cache

import (
	"context"

	"github.com/eko/gocache/lib/v4/store"
)

type (
	IGoCache interface {
		store.StoreInterface
		GetOrInsert(ctx context.Context, key string, f func(ctx context.Context) (any, error), options ...store.Option) (any, error)
	}

	RedisCacheOptions struct {
		// Default is tcp.
		Network string `json:"network"`
		// host:port address.
		Addr string `json:"addr"`
		// Use the specified Username to authenticate the current connection
		// with one of the connections defined in the ACL list when connecting
		// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
		Username string `json:"username"`
		// Optional password. Must match the password specified in the
		// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
		// or the User Password when connecting to a Redis 6.0 instance, or greater,
		// that is using the Redis ACL system.
		Password string `json:"password"`
	}
	IRedisCache interface {
		IGoCache
	}
)
