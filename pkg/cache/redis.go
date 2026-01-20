package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCache Redis缓存实现
type redisCache struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisCache 创建Redis缓存
func NewRedisCache(config RedisConfig) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.IdleTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &redisCache{
		client: client,
		config: config,
	}, nil
}

// Get 获取缓存值
func (rc *redisCache) Get(ctx context.Context, key string) (interface{}, bool) {
	result := rc.client.Get(ctx, key)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, false
		}
		return nil, false
	}

	var value interface{}
	if err := json.Unmarshal([]byte(result.Val()), &value); err != nil {
		// 如果JSON解析失败，尝试直接返回字符串
		return result.Val(), true
	}
	return value, true
}

// Set 设置缓存值
func (rc *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return rc.client.Set(ctx, key, data, expiration).Err()
}

// Delete 删除缓存
func (rc *redisCache) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

// Exists 检查键是否存在
func (rc *redisCache) Exists(ctx context.Context, key string) bool {
	result := rc.client.Exists(ctx, key)
	return result.Val() > 0
}

// Clear 清空所有缓存
func (rc *redisCache) Clear(ctx context.Context) error {
	return rc.client.FlushDB(ctx).Err()
}

// GetMulti 批量获取
func (rc *redisCache) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	if len(keys) == 0 {
		return make(map[string]interface{})
	}

	// 使用 Pipeline 批量获取
	pipe := rc.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})
	for i, cmd := range cmds {
		if cmd.Err() == nil {
			var value interface{}
			// 先尝试 JSON 解码
			if err := json.Unmarshal([]byte(cmd.Val()), &value); err != nil {
				// 如果解码失败，返回原始字符串值
				result[keys[i]] = cmd.Val()
			} else {
				result[keys[i]] = value
			}
		}
	}

	return result
}

// SetMulti 批量设置
func (rc *redisCache) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	if len(data) == 0 {
		return nil
	}

	pipe := rc.client.Pipeline()
	for key, value := range data {
		dataBytes, err := json.Marshal(value) // 确保值被序列化
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, dataBytes, expiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// DeleteMulti 批量删除
func (rc *redisCache) DeleteMulti(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	return rc.client.Del(ctx, keys...).Err()
}

// Increment 自增
func (rc *redisCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	result := rc.client.IncrBy(ctx, key, value)
	return result.Val(), result.Err()
}

// Decrement 自减
func (rc *redisCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	result := rc.client.DecrBy(ctx, key, value)
	return result.Val(), result.Err()
}

// GetWithTTL 获取值并返回剩余TTL
func (rc *redisCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	// 获取值
	value, exists := rc.Get(ctx, key)
	if !exists {
		return nil, 0, false
	}

	// 获取TTL
	ttl := rc.client.TTL(ctx, key)
	if ttl.Err() != nil {
		return value, 0, true
	}

	return value, ttl.Val(), true
}

// Close 关闭缓存连接
func (rc *redisCache) Close() error {
	return rc.client.Close()
}
