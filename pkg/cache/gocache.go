package cache

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// goCacheWrapper go-cache包装器
type goCacheWrapper struct {
	cache *gocache.Cache
}

// NewGoCache 创建基于go-cache的本地缓存
func NewGoCache(config LocalConfig) Cache {
	// 将配置转换为go-cache的配置
	defaultExpiration := config.DefaultExpiration
	cleanupInterval := config.CleanupInterval

	// 创建go-cache实例
	c := gocache.New(defaultExpiration, cleanupInterval)

	// 设置最大项数（go-cache本身没有这个限制，但我们可以通过监控来实现）

	return &goCacheWrapper{
		cache: c,
	}
}

// Get 获取缓存值
func (gc *goCacheWrapper) Get(ctx context.Context, key string) (interface{}, bool) {
	if value, found := gc.cache.Get(key); found {
		return value, true
	}
	return nil, false
}

// Set 设置缓存值
func (gc *goCacheWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	gc.cache.Set(key, value, expiration)
	return nil
}

// Delete 删除缓存
func (gc *goCacheWrapper) Delete(ctx context.Context, key string) error {
	gc.cache.Delete(key)
	return nil
}

// Exists 检查键是否存在
func (gc *goCacheWrapper) Exists(ctx context.Context, key string) bool {
	_, found := gc.cache.Get(key)
	return found
}

// Clear 清空所有缓存
func (gc *goCacheWrapper) Clear(ctx context.Context) error {
	gc.cache.Flush()
	return nil
}

// GetMulti 批量获取
func (gc *goCacheWrapper) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if value, found := gc.cache.Get(key); found {
			result[key] = value
		}
	}
	return result
}

// SetMulti 批量设置
func (gc *goCacheWrapper) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	for key, value := range data {
		gc.cache.Set(key, value, expiration)
	}
	return nil
}

// DeleteMulti 批量删除
func (gc *goCacheWrapper) DeleteMulti(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		gc.cache.Delete(key)
	}
	return nil
}

// Increment 自增
func (gc *goCacheWrapper) Increment(ctx context.Context, key string, value int64) (int64, error) {
	// go-cache支持IncrementInt64，返回新值
	if newValue, err := gc.cache.IncrementInt64(key, value); err == nil {
		return newValue, nil
	}

	// 如果键不存在，先设置为初始值
	gc.cache.Set(key, value, gocache.DefaultExpiration)
	return value, nil
}

// Decrement 自减
func (gc *goCacheWrapper) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	// go-cache支持DecrementInt64，返回新值
	if newValue, err := gc.cache.DecrementInt64(key, value); err == nil {
		return newValue, nil
	}

	// 如果键不存在，先设置为初始值
	gc.cache.Set(key, -value, gocache.DefaultExpiration)
	return -value, nil
}

// GetWithTTL 获取值并返回剩余TTL
func (gc *goCacheWrapper) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	// go-cache没有直接获取TTL的方法，但我们可以通过GetWithExpiration获取
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

// Close 关闭缓存连接
func (gc *goCacheWrapper) Close() error {
	// go-cache不需要关闭连接
	return nil
}

// 额外的go-cache特有方法

// GetWithExpiration 获取值和过期时间
func (gc *goCacheWrapper) GetWithExpiration(key string) (interface{}, time.Time, bool) {
	return gc.cache.GetWithExpiration(key)
}

// Items 获取所有缓存项（用于调试和监控）
func (gc *goCacheWrapper) Items() map[string]gocache.Item {
	return gc.cache.Items()
}

// ItemCount 获取缓存项数量
func (gc *goCacheWrapper) ItemCount() int {
	return gc.cache.ItemCount()
}

// Flush 清空缓存
func (gc *goCacheWrapper) Flush() {
	gc.cache.Flush()
}
