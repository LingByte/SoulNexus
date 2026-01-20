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

// NewCache 创建缓存实例
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

// NewCacheWithOptions 创建带选项的缓存实例
func NewCacheWithOptions(config Config, options *Options) (Cache, error) {
	if options == nil {
		options = DefaultOptions()
	}

	// 如果启用本地缓存作为一级缓存，创建分层缓存
	if options.UseLocalCache && config.Type != "local" && config.Type != "gocache" {
		return NewLayeredCache(config, options)
	}

	return NewCache(config)
}

// NewLayeredCache 创建分层缓存（本地缓存 + 分布式缓存）
func NewLayeredCache(config Config, options *Options) (Cache, error) {
	// 创建本地缓存作为一级缓存
	localConfig := config.Local
	if options.LocalExpiration > 0 {
		localConfig.DefaultExpiration = options.LocalExpiration
	}

	localCache := NewLocalCache(localConfig)

	// 创建分布式缓存作为二级缓存
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

// layeredCache 分层缓存实现
type layeredCache struct {
	local       Cache
	distributed Cache
	options     *Options
}

// Get 从本地缓存获取，如果没有则从分布式缓存获取并回填本地缓存
func (lc *layeredCache) Get(ctx context.Context, key string) (interface{}, bool) {
	// 先从本地缓存获取
	if value, exists := lc.local.Get(ctx, key); exists {
		return value, true
	}

	// 从分布式缓存获取
	if value, exists := lc.distributed.Get(ctx, key); exists {
		// 回填到本地缓存
		lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
		return value, true
	}

	return nil, false
}

// Set 同时设置到本地和分布式缓存
func (lc *layeredCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 设置到分布式缓存
	if err := lc.distributed.Set(ctx, key, value, expiration); err != nil {
		return err
	}

	// 设置到本地缓存
	return lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
}

// Delete 从两个缓存层删除
func (lc *layeredCache) Delete(ctx context.Context, key string) error {
	// 删除本地缓存
	if err := lc.local.Delete(ctx, key); err != nil {
		return err
	}

	// 删除分布式缓存
	return lc.distributed.Delete(ctx, key)
}

// Exists 检查键是否存在
func (lc *layeredCache) Exists(ctx context.Context, key string) bool {
	return lc.local.Exists(ctx, key) || lc.distributed.Exists(ctx, key)
}

// Clear 清空两个缓存层
func (lc *layeredCache) Clear(ctx context.Context) error {
	// 清空本地缓存
	if err := lc.local.Clear(ctx); err != nil {
		return err
	}

	// 清空分布式缓存
	return lc.distributed.Clear(ctx)
}

// GetMulti 批量获取
func (lc *layeredCache) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})

	// 先从本地缓存获取
	localResult := lc.local.GetMulti(ctx, keys...)
	for key, value := range localResult {
		result[key] = value
	}

	// 查找本地缓存中没有的键
	missingKeys := make([]string, 0)
	for _, key := range keys {
		if _, exists := result[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	// 从分布式缓存获取缺失的键
	if len(missingKeys) > 0 {
		distributedResult := lc.distributed.GetMulti(ctx, missingKeys...)
		for key, value := range distributedResult {
			result[key] = value
			// 回填到本地缓存
			lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
		}
	}

	return result
}

// SetMulti 批量设置
func (lc *layeredCache) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	// 设置到分布式缓存
	if err := lc.distributed.SetMulti(ctx, data, expiration); err != nil {
		return err
	}

	// 设置到本地缓存
	return lc.local.SetMulti(ctx, data, lc.options.LocalExpiration)
}

// DeleteMulti 批量删除
func (lc *layeredCache) DeleteMulti(ctx context.Context, keys ...string) error {
	// 删除本地缓存
	if err := lc.local.DeleteMulti(ctx, keys...); err != nil {
		return err
	}

	// 删除分布式缓存
	return lc.distributed.DeleteMulti(ctx, keys...)
}

// Increment 自增
func (lc *layeredCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	// 分布式缓存的Increment操作
	result, err := lc.distributed.Increment(ctx, key, value)
	if err != nil {
		return 0, err
	}

	// 更新本地缓存
	lc.local.Set(ctx, key, result, lc.options.LocalExpiration)
	return result, nil
}

// Decrement 自减
func (lc *layeredCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	// 分布式缓存的Decrement操作
	result, err := lc.distributed.Decrement(ctx, key, value)
	if err != nil {
		return 0, err
	}

	// 更新本地缓存
	lc.local.Set(ctx, key, result, lc.options.LocalExpiration)
	return result, nil
}

// GetWithTTL 获取值和TTL
func (lc *layeredCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	// 先从本地缓存获取
	if value, ttl, exists := lc.local.GetWithTTL(ctx, key); exists {
		return value, ttl, true
	}

	// 从分布式缓存获取
	if value, ttl, exists := lc.distributed.GetWithTTL(ctx, key); exists {
		// 回填到本地缓存
		lc.local.Set(ctx, key, value, lc.options.LocalExpiration)
		return value, ttl, true
	}

	return nil, 0, false
}

// Close 关闭缓存连接
func (lc *layeredCache) Close() error {
	// 关闭本地缓存
	if err := lc.local.Close(); err != nil {
		return err
	}

	// 关闭分布式缓存
	return lc.distributed.Close()
}
