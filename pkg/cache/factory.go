package cache

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	KindLocal   = "local"   // local
	KindGoCache = "gocache" // gocache
	KindRedis   = "redis"   // redis
)

// NewCache creates a cache instance based on configuration
func NewCache(config Config) (Cache, error) {
	switch strings.ToLower(config.Type) {
	case KindLocal:
		return NewLocalCache(config.Local), nil
	case KindGoCache:
		return NewGoCache(config.Local), nil
	case KindRedis:
		return NewRedisCache(config.Redis)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", config.Type)
	}
}

// NewCacheWithOptions creates a cache instance with additional options
func NewCacheWithOptions(config Config, options *Options) (Cache, error) {
	if options == nil {
		options = DefaultOptions()
	}

	// If local cache is enabled as L1 cache, create layered cache
	if options.UseLocalCache && config.Type != "local" && config.Type != "gocache" {
		return NewLayeredCache(config, options)
	}

	return NewCache(config)
}

// NewLayeredCache creates a layered cache (local cache + distributed cache)
func NewLayeredCache(config Config, options *Options) (Cache, error) {
	// Create local cache as L1 cache
	localConfig := config.Local
	if options.LocalExpiration > 0 {
		localConfig.DefaultExpiration = options.LocalExpiration
	}

	localCache := NewLocalCache(localConfig)

	// Create distributed cache as L2 cache
	var distributedCache Cache
	var err error

	switch strings.ToLower(config.Type) {
	case "redis":
		distributedCache, err = NewRedisCache(config.Redis)
		if err != nil {
			return nil, fmt.Errorf("failed to create redis cache: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported distributed cache type: %s", config.Type)
	}

	return &layeredCache{
		local:       localCache,
		distributed: distributedCache,
		options:     options,
	}, nil
}

// layeredCache implements layered caching strategy
type layeredCache struct {
	local       Cache
	distributed Cache
	options     *Options
}

// Get retrieves from local cache first, then from distributed cache and backfills local cache
func (lc *layeredCache) Get(ctx context.Context, key string) (interface{}, bool) {
	// Try local cache first
	if value, exists := lc.local.Get(ctx, key); exists {
		return value, true
	}

	// Get from distributed cache
	if value, exists := lc.distributed.Get(ctx, key); exists {
		// Backfill to local cache
		lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
		return value, true
	}

	return nil, false
}

// Set stores to both local and distributed cache
func (lc *layeredCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Set to distributed cache
	if err := lc.distributed.Set(ctx, key, value, expiration); err != nil {
		return err
	}

	// Set to local cache
	return lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
}

// Delete removes from both cache layers
func (lc *layeredCache) Delete(ctx context.Context, key string) error {
	// Delete from local cache
	if err := lc.local.Delete(ctx, key); err != nil {
		return err
	}

	// Delete from distributed cache
	return lc.distributed.Delete(ctx, key)
}

// Exists checks if key exists in either cache layer
func (lc *layeredCache) Exists(ctx context.Context, key string) bool {
	return lc.local.Exists(ctx, key) || lc.distributed.Exists(ctx, key)
}

// Clear removes all entries from both cache layers
func (lc *layeredCache) Clear(ctx context.Context) error {
	// Clear local cache
	if err := lc.local.Clear(ctx); err != nil {
		return err
	}

	// Clear distributed cache
	return lc.distributed.Clear(ctx)
}

// GetMulti retrieves multiple values by keys
func (lc *layeredCache) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})

	// Get from local cache first
	localResult := lc.local.GetMulti(ctx, keys...)
	for key, value := range localResult {
		result[key] = value
	}

	// Find keys missing from local cache
	missingKeys := make([]string, 0)
	for _, key := range keys {
		if _, exists := result[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	// Get missing keys from distributed cache
	if len(missingKeys) > 0 {
		distributedResult := lc.distributed.GetMulti(ctx, missingKeys...)
		for key, value := range distributedResult {
			result[key] = value
			// Backfill to local cache
			lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
		}
	}

	return result
}

// SetMulti stores multiple key-value pairs to both cache layers
func (lc *layeredCache) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	// Set to distributed cache
	if err := lc.distributed.SetMulti(ctx, data, expiration); err != nil {
		return err
	}

	// Set to local cache
	return lc.local.SetMulti(ctx, data, lc.options.LocalExpiration)
}

// DeleteMulti removes multiple keys from both cache layers
func (lc *layeredCache) DeleteMulti(ctx context.Context, keys ...string) error {
	// Delete from local cache
	if err := lc.local.DeleteMulti(ctx, keys...); err != nil {
		return err
	}

	// Delete from distributed cache
	return lc.distributed.DeleteMulti(ctx, keys...)
}

// Increment increments a numeric value in distributed cache and updates local cache
func (lc *layeredCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	// Increment in distributed cache
	result, err := lc.distributed.Increment(ctx, key, value)
	if err != nil {
		return 0, err
	}

	// Update local cache
	lc.local.Set(ctx, key, result, lc.options.LocalExpiration)
	return result, nil
}

// Decrement decrements a numeric value in distributed cache and updates local cache
func (lc *layeredCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	// Decrement in distributed cache
	result, err := lc.distributed.Decrement(ctx, key, value)
	if err != nil {
		return 0, err
	}

	// Update local cache
	lc.local.Set(ctx, key, result, lc.options.LocalExpiration)
	return result, nil
}

// GetWithTTL retrieves value and TTL from cache layers
func (lc *layeredCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	// Try local cache first
	if value, ttl, exists := lc.local.GetWithTTL(ctx, key); exists {
		return value, ttl, true
	}

	// Get from distributed cache
	if value, ttl, exists := lc.distributed.GetWithTTL(ctx, key); exists {
		// Backfill to local cache
		lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
		return value, ttl, true
	}

	return nil, 0, false
}

// Close closes connections for both cache layers
func (lc *layeredCache) Close() error {
	// Close local cache
	if err := lc.local.Close(); err != nil {
		return err
	}

	// Close distributed cache
	return lc.distributed.Close()
}
