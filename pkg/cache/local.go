package cache

import (
	"context"
	"sync"
	"time"
)

// localCache implements local in-memory cache
type localCache struct {
	config LocalConfig
	cache  *lruCache
	mu     sync.RWMutex
}

// lruCache implements LRU (Least Recently Used) cache
type lruCache struct {
	maxSize int
	items   map[string]*cacheItem
	keys    []string
	mu      sync.RWMutex
}

// cacheItem represents a single cache entry
type cacheItem struct {
	value      interface{}
	expiration time.Time
	lastAccess time.Time
}

// NewLocalCache creates a new local cache instance
func NewLocalCache(config LocalConfig) Cache {
	lc := &localCache{
		config: config,
		cache: &lruCache{
			maxSize: config.MaxSize,
			items:   make(map[string]*cacheItem),
			keys:    make([]string, 0),
		},
	}

	// Start cleanup goroutine
	go lc.startCleanup()

	return lc
}

// Get retrieves a value from cache by key
func (lc *localCache) Get(ctx context.Context, key string) (interface{}, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.cache.get(key)
	if !exists {
		return nil, false
	}

	// Check if expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.delete(key)
		return nil, false
	}

	// Update last access time
	item.lastAccess = time.Now()
	return item.value, true
}

// Set stores a value in cache with expiration
func (lc *localCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	item := &cacheItem{
		value:      value,
		expiration: exp,
		lastAccess: time.Now(),
	}

	lc.cache.set(key, item)
	return nil
}

// Delete removes a key from cache
func (lc *localCache) Delete(ctx context.Context, key string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.cache.delete(key)
	return nil
}

// Exists checks if a key exists in cache
func (lc *localCache) Exists(ctx context.Context, key string) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.cache.get(key)
	if !exists {
		return false
	}

	// Check if expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.delete(key)
		return false
	}

	return true
}

// Clear removes all entries from cache
func (lc *localCache) Clear(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.cache.clear()
	return nil
}

// GetMulti retrieves multiple values by keys
func (lc *localCache) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if value, exists := lc.Get(ctx, key); exists {
			result[key] = value
		}
	}
	return result
}

// SetMulti stores multiple key-value pairs with expiration
func (lc *localCache) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	for key, value := range data {
		if err := lc.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMulti removes multiple keys from cache
func (lc *localCache) DeleteMulti(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := lc.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Increment increments a numeric value by the given amount
func (lc *localCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	item, exists := lc.cache.get(key)
	if !exists {
		// If key doesn't exist, create new value
		newValue := value
		lc.cache.set(key, &cacheItem{
			value:      newValue,
			expiration: time.Now().Add(lc.config.DefaultExpiration),
			lastAccess: time.Now(),
		})
		return newValue, nil
	}

	// Check if expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.delete(key)
		newValue := value
		lc.cache.set(key, &cacheItem{
			value:      newValue,
			expiration: time.Now().Add(lc.config.DefaultExpiration),
			lastAccess: time.Now(),
		})
		return newValue, nil
	}

	// Try to convert to number and increment
	switch v := item.value.(type) {
	case int:
		newValue := int64(v) + value
		item.value = newValue
		item.lastAccess = time.Now()
		return newValue, nil
	case int64:
		newValue := v + value
		item.value = newValue
		item.lastAccess = time.Now()
		return newValue, nil
	case float64:
		newValue := int64(v) + value
		item.value = newValue
		item.lastAccess = time.Now()
		return newValue, nil
	default:
		// If type is not supported, reset to specified value
		item.value = value
		item.lastAccess = time.Now()
		return value, nil
	}
}

// Decrement decrements a numeric value by the given amount
func (lc *localCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	return lc.Increment(ctx, key, -value)
}

// GetWithTTL retrieves a value and its remaining TTL
func (lc *localCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.cache.get(key)
	if !exists {
		return nil, 0, false
	}

	// Check if expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.delete(key)
		return nil, 0, false
	}

	var ttl time.Duration
	if !item.expiration.IsZero() {
		ttl = item.expiration.Sub(time.Now())
		if ttl < 0 {
			ttl = 0
		}
	}

	// Update last access time
	item.lastAccess = time.Now()
	return item.value, ttl, true
}

// Close closes the cache connection (no-op for local cache)
func (lc *localCache) Close() error {
	// Local cache doesn't need to close connections
	return nil
}

// startCleanup starts the cleanup goroutine
func (lc *localCache) startCleanup() {
	ticker := time.NewTicker(lc.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		lc.cleanup()
	}
}

// cleanup removes expired items from cache
func (lc *localCache) cleanup() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	now := time.Now()
	for key, item := range lc.cache.items {
		if !item.expiration.IsZero() && now.After(item.expiration) {
			lc.cache.delete(key)
		}
	}
}

// LRU cache method implementations
func (lc *lruCache) get(key string) (*cacheItem, bool) {
	item, exists := lc.items[key]
	if !exists {
		return nil, false
	}

	// Update access order
	lc.updateAccessOrder(key)
	return item, true
}

func (lc *lruCache) set(key string, item *cacheItem) {
	// If key already exists, delete it first
	if _, exists := lc.items[key]; exists {
		lc.delete(key)
	}

	// If max size reached, evict least recently used item
	if len(lc.items) >= lc.maxSize {
		lc.evictLRU()
	}

	lc.items[key] = item
	lc.keys = append(lc.keys, key)
}

func (lc *lruCache) delete(key string) {
	delete(lc.items, key)
	// Remove from keys slice
	for i, k := range lc.keys {
		if k == key {
			lc.keys = append(lc.keys[:i], lc.keys[i+1:]...)
			break
		}
	}
}

func (lc *lruCache) clear() {
	lc.items = make(map[string]*cacheItem)
	lc.keys = make([]string, 0)
}

func (lc *lruCache) updateAccessOrder(key string) {
	// Move accessed key to end
	for i, k := range lc.keys {
		if k == key {
			lc.keys = append(lc.keys[:i], lc.keys[i+1:]...)
			lc.keys = append(lc.keys, key)
			break
		}
	}
}

func (lc *lruCache) evictLRU() {
	if len(lc.keys) == 0 {
		return
	}

	// Remove least recently used item (first one)
	oldestKey := lc.keys[0]
	delete(lc.items, oldestKey)
	lc.keys = lc.keys[1:]
}
