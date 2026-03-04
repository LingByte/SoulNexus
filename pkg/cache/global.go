package cache

import (
	"context"
	"sync"
	"time"
)

var (
	globalCache Cache
	globalOnce  sync.Once
	globalMu    sync.RWMutex
)

// InitGlobalCache 初始化全局缓存实例
func InitGlobalCache(config Config) error {
	var err error
	globalOnce.Do(func() {
		globalMu.Lock()
		defer globalMu.Unlock()

		globalCache, err = NewCache(config)
		if err != nil {
			return
		}
	})
	return err
}

// InitGlobalCacheWithOptions 使用选项初始化全局缓存实例
func InitGlobalCacheWithOptions(config Config, options *Options) error {
	var err error
	globalOnce.Do(func() {
		globalMu.Lock()
		defer globalMu.Unlock()

		globalCache, err = NewCacheWithOptions(config, options)
		if err != nil {
			return
		}
	})
	return err
}

// GetGlobalCache 获取全局缓存实例
// 如果未初始化，返回一个默认的本地缓存实例
func GetGlobalCache() Cache {
	globalMu.RLock()
	if globalCache != nil {
		globalMu.RUnlock()
		return globalCache
	}
	globalMu.RUnlock()

	// 双重检查，如果未初始化，创建一个默认的本地缓存
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalCache == nil {
		globalCache = NewLocalCache(LocalConfig{
			MaxSize:           1000,
			DefaultExpiration: 5 * time.Minute,
			CleanupInterval:   10 * time.Minute,
		})
	}
	return globalCache
}

// SetGlobalCache 设置全局缓存实例（主要用于测试）
func SetGlobalCache(c Cache) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalCache = c
}

// CloseGlobalCache 关闭全局缓存连接
func CloseGlobalCache() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalCache != nil {
		err := globalCache.Close()
		globalCache = nil
		return err
	}
	return nil
}

// ==================== 便捷方法：直接使用全局缓存实例 ====================

// Get 从全局缓存获取值
func Get(ctx context.Context, key string) (interface{}, bool) {
	return GetGlobalCache().Get(ctx, key)
}

// Set 设置全局缓存值
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return GetGlobalCache().Set(ctx, key, value, expiration)
}

// Delete 从全局缓存删除
func Delete(ctx context.Context, key string) error {
	return GetGlobalCache().Delete(ctx, key)
}

// Exists 检查全局缓存中键是否存在
func Exists(ctx context.Context, key string) bool {
	return GetGlobalCache().Exists(ctx, key)
}

// Clear 清空全局缓存
func Clear(ctx context.Context) error {
	return GetGlobalCache().Clear(ctx)
}

// GetMulti 批量获取
func GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	return GetGlobalCache().GetMulti(ctx, keys...)
}

// SetMulti 批量设置
func SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	return GetGlobalCache().SetMulti(ctx, data, expiration)
}

// DeleteMulti 批量删除
func DeleteMulti(ctx context.Context, keys ...string) error {
	return GetGlobalCache().DeleteMulti(ctx, keys...)
}
