package cache

import (
	"context"
	"sync"
	"time"
)

// localCache 本地缓存实现
type localCache struct {
	config LocalConfig
	cache  *lruCache
	mu     sync.RWMutex
}

// lruCache LRU缓存
type lruCache struct {
	maxSize int
	items   map[string]*cacheItem
	keys    []string
	mu      sync.RWMutex
}

// cacheItem 缓存项
type cacheItem struct {
	value      interface{}
	expiration time.Time
	lastAccess time.Time
}

// NewLocalCache 创建本地缓存
func NewLocalCache(config LocalConfig) Cache {
	lc := &localCache{
		config: config,
		cache: &lruCache{
			maxSize: config.MaxSize,
			items:   make(map[string]*cacheItem),
			keys:    make([]string, 0),
		},
	}

	// 启动清理协程
	go lc.startCleanup()

	return lc
}

// Get 获取缓存值
func (lc *localCache) Get(ctx context.Context, key string) (interface{}, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.cache.get(key)
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.delete(key)
		return nil, false
	}

	// 更新最后访问时间
	item.lastAccess = time.Now()
	return item.value, true
}

// Set 设置缓存值
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

// Delete 删除缓存
func (lc *localCache) Delete(ctx context.Context, key string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.cache.delete(key)
	return nil
}

// Exists 检查键是否存在
func (lc *localCache) Exists(ctx context.Context, key string) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.cache.get(key)
	if !exists {
		return false
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.delete(key)
		return false
	}

	return true
}

// Clear 清空所有缓存
func (lc *localCache) Clear(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.cache.clear()
	return nil
}

// GetMulti 批量获取
func (lc *localCache) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if value, exists := lc.Get(ctx, key); exists {
			result[key] = value
		}
	}
	return result
}

// SetMulti 批量设置
func (lc *localCache) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	for key, value := range data {
		if err := lc.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMulti 批量删除
func (lc *localCache) DeleteMulti(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := lc.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Increment 自增
func (lc *localCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	item, exists := lc.cache.get(key)
	if !exists {
		// 如果不存在，创建新值
		newValue := value
		lc.cache.set(key, &cacheItem{
			value:      newValue,
			expiration: time.Now().Add(lc.config.DefaultExpiration),
			lastAccess: time.Now(),
		})
		return newValue, nil
	}

	// 检查是否过期
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

	// 尝试转换为数字并自增
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
		// 如果类型不支持，重置为指定值
		item.value = value
		item.lastAccess = time.Now()
		return value, nil
	}
}

// Decrement 自减
func (lc *localCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	return lc.Increment(ctx, key, -value)
}

// GetWithTTL 获取值并返回剩余TTL
func (lc *localCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.cache.get(key)
	if !exists {
		return nil, 0, false
	}

	// 检查是否过期
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

	// 更新最后访问时间
	item.lastAccess = time.Now()
	return item.value, ttl, true
}

// Close 关闭缓存连接
func (lc *localCache) Close() error {
	// 本地缓存不需要关闭连接
	return nil
}

// startCleanup 启动清理协程
func (lc *localCache) startCleanup() {
	ticker := time.NewTicker(lc.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		lc.cleanup()
	}
}

// cleanup 清理过期项
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

// LRU缓存方法实现
func (lc *lruCache) get(key string) (*cacheItem, bool) {
	item, exists := lc.items[key]
	if !exists {
		return nil, false
	}

	// 更新访问顺序
	lc.updateAccessOrder(key)
	return item, true
}

func (lc *lruCache) set(key string, item *cacheItem) {
	// 如果键已存在，先删除
	if _, exists := lc.items[key]; exists {
		lc.delete(key)
	}

	// 如果达到最大大小，删除最久未使用的项
	if len(lc.items) >= lc.maxSize {
		lc.evictLRU()
	}

	lc.items[key] = item
	lc.keys = append(lc.keys, key)
}

func (lc *lruCache) delete(key string) {
	delete(lc.items, key)
	// 从keys中删除
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
	// 将访问的键移到末尾
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

	// 删除最久未使用的项（第一个）
	oldestKey := lc.keys[0]
	delete(lc.items, oldestKey)
	lc.keys = lc.keys[1:]
}
