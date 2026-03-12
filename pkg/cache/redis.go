package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCache implements Redis cache
type redisCache struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisCache creates a new Redis cache instance
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

	// Test connection
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
		// If JSON parsing fails, try to return the string directly
		return result.Val(), true
	}
	return value, true
}

// Set stores a value in cache
func (rc *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return rc.client.Set(ctx, key, data, expiration).Err()
}

// Delete removes a cached value
func (rc *redisCache) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
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

	// Use Pipeline for batch retrieval
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
			// Try JSON decoding first
			if err := json.Unmarshal([]byte(cmd.Val()), &value); err != nil {
				// If decoding fails, return the raw string value
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
		dataBytes, err := json.Marshal(value) // Ensure value is serialized
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

// GetWithTTL retrieves value with remaining TTL
func (rc *redisCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	// Get value
	value, exists := rc.Get(ctx, key)
	if !exists {
		return nil, 0, false
	}

	// Get TTL
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
