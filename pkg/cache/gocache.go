package cache

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// goCacheWrapper wraps go-cache package for unified interface
type goCacheWrapper struct {
	cache *gocache.Cache
}

// NewGoCache creates a local cache based on go-cache package
func NewGoCache(config LocalConfig) Cache {
	// Convert config to go-cache configuration
	defaultExpiration := config.DefaultExpiration
	cleanupInterval := config.CleanupInterval

	// Create go-cache instance
	c := gocache.New(defaultExpiration, cleanupInterval)

	// Set max items (go-cache doesn't have this limit natively, but we can implement it through monitoring)

	return &goCacheWrapper{
		cache: c,
	}
}

// Get retrieves a value from cache by key
func (gc *goCacheWrapper) Get(ctx context.Context, key string) (interface{}, bool) {
	if value, found := gc.cache.Get(key); found {
		return value, true
	}
	return nil, false
}

// Set stores a value in cache with expiration
func (gc *goCacheWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	gc.cache.Set(key, value, expiration)
	return nil
}

// Delete removes a key from cache
func (gc *goCacheWrapper) Delete(ctx context.Context, key string) error {
	gc.cache.Delete(key)
	return nil
}

// Exists checks if a key exists in cache
func (gc *goCacheWrapper) Exists(ctx context.Context, key string) bool {
	_, found := gc.cache.Get(key)
	return found
}

// Clear removes all entries from cache
func (gc *goCacheWrapper) Clear(ctx context.Context) error {
	gc.cache.Flush()
	return nil
}

// GetMulti retrieves multiple values by keys
func (gc *goCacheWrapper) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if value, found := gc.cache.Get(key); found {
			result[key] = value
		}
	}
	return result
}

// SetMulti stores multiple key-value pairs with expiration
func (gc *goCacheWrapper) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	for key, value := range data {
		gc.cache.Set(key, value, expiration)
	}
	return nil
}

// DeleteMulti removes multiple keys from cache
func (gc *goCacheWrapper) DeleteMulti(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		gc.cache.Delete(key)
	}
	return nil
}

// Increment increments a numeric value by the given amount
func (gc *goCacheWrapper) Increment(ctx context.Context, key string, value int64) (int64, error) {
	// go-cache supports IncrementInt64, returns new value
	if newValue, err := gc.cache.IncrementInt64(key, value); err == nil {
		return newValue, nil
	}

	// If key doesn't exist, set it to initial value
	gc.cache.Set(key, value, gocache.DefaultExpiration)
	return value, nil
}

// Decrement decrements a numeric value by the given amount
func (gc *goCacheWrapper) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	// go-cache supports DecrementInt64, returns new value
	if newValue, err := gc.cache.DecrementInt64(key, value); err == nil {
		return newValue, nil
	}

	// If key doesn't exist, set it to initial value
	gc.cache.Set(key, -value, gocache.DefaultExpiration)
	return -value, nil
}

// GetWithTTL retrieves a value and its remaining TTL
func (gc *goCacheWrapper) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	// go-cache doesn't have direct TTL method, but we can use GetWithExpiration
	if value, expiration, found := gc.cache.GetWithExpiration(key); found {
		var ttl time.Duration
		if !expiration.IsZero() {
			ttl = expiration.Sub(time.Now())
			if ttl < 0 {
				ttl = 0
			}
		}
		return value, ttl, true
	}
	return nil, 0, false
}

// Close closes the cache connection (no-op for go-cache)
func (gc *goCacheWrapper) Close() error {
	// go-cache doesn't need to close connections
	return nil
}

// Additional go-cache specific methods

// GetWithExpiration retrieves value with expiration time
func (gc *goCacheWrapper) GetWithExpiration(key string) (interface{}, time.Time, bool) {
	return gc.cache.GetWithExpiration(key)
}

// Items returns all cache items (for debugging and monitoring)
func (gc *goCacheWrapper) Items() map[string]gocache.Item {
	return gc.cache.Items()
}

// ItemCount returns the number of cache items
func (gc *goCacheWrapper) ItemCount() int {
	return gc.cache.ItemCount()
}

// Flush clears all cache entries
func (gc *goCacheWrapper) Flush() {
	gc.cache.Flush()
}
