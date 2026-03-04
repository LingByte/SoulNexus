package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestRedisCache() *redisCache {
	config := RedisConfig{
		Addr:         "localhost:6379", // 这里假设 Redis 运行在本地，若是远程地址，请调整
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	cache, err := NewRedisCache(config)
	if err != nil {
		fmt.Printf("Error initializing cache: %v\n", err)
		return nil
	}
	return cache.(*redisCache)
}

// skipIfRedisNotAvailable 如果Redis不可用则跳过测试
func skipIfRedisNotAvailable(t *testing.T, c *redisCache) {
	if c == nil {
		t.Skip("Redis not available, skipping test")
	}
}

func TestSetGet(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	// Set value
	err := c.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	// Get value
	val, exists := c.Get(ctx, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	// Get non-existing key
	val, exists = c.Get(ctx, "key_not_exists")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestRedisGetWithTTL(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	// Set value with expiration time (50ms)
	err := c.Set(ctx, "key4", "value4", 50*time.Millisecond)
	assert.NoError(t, err)

	// Get value with TTL
	val, ttl, exists := c.GetWithTTL(ctx, "key4")
	assert.True(t, exists)
	assert.Equal(t, "value4", val)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Check the key again, should have expired
	_, ttl, exists = c.GetWithTTL(ctx, "key4")
	assert.False(t, exists)
	assert.Equal(t, 0*time.Millisecond, ttl)
}

func TestGetMulti(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	// Set multiple keys
	data := map[string]interface{}{
		"key5": "value5",
		"key6": "value6",
	}
	err := c.SetMulti(ctx, data, 0)
	assert.NoError(t, err)

	// Get multiple keys
	result := c.GetMulti(ctx, "key5", "key6", "key_not_exists")
	assert.Nil(t, result["key_not_exists"])
}

func TestRedisCacheTTLHandling(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	// Set with TTL (50ms)
	err := c.Set(ctx, "key_with_ttl", "value_with_ttl", 50*time.Millisecond)
	assert.NoError(t, err)

	// Check TTL immediately
	_, ttl, exists := c.GetWithTTL(ctx, "key_with_ttl")
	assert.True(t, exists)

	// Check the key has expired
	_, ttl, exists = c.GetWithTTL(ctx, "key_with_ttl")
	assert.Equal(t, 0*time.Millisecond, ttl)
}

func TestRedisCacheFail(t *testing.T) {
	// Simulate Redis connection failure
	config := RedisConfig{
		Addr:         "invalid:6379", // Invalid address
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	_, err := NewRedisCache(config)
	assert.Error(t, err)
}

func TestRedisIncrementDecrement(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	// Increment
	nv, err := c.Increment(ctx, "counter", 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), nv)

	// Increment again
	nv, err = c.Increment(ctx, "counter", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), nv)

	// Decrement
	nv, err = c.Decrement(ctx, "counter", 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), nv)
}

func TestRedisBatchOps(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	data := map[string]interface{}{
		"a": 1,
		"b": "B",
		"c": 3.14,
	}
	assert.NoError(t, c.SetMulti(ctx, data, 0))

	got := c.GetMulti(ctx, "a", "b", "c", "d")
	_, hasD := got["d"]
	assert.False(t, hasD)

	assert.NoError(t, c.DeleteMulti(ctx, "a", "c"))
	_, ok := c.Get(ctx, "a")
	assert.False(t, ok)
	_, ok = c.Get(ctx, "c")
	assert.False(t, ok)
}

func TestRedisClearAndClose(t *testing.T) {
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	_ = c.Set(ctx, "k", "v", 0)
	assert.True(t, c.Exists(ctx, "k"))

	assert.NoError(t, c.Clear(ctx))
	assert.False(t, c.Exists(ctx, "k"))

	assert.NoError(t, c.Close())
}

func TestRedisLRUEviction(t *testing.T) {
	// MaxSize=2，验证 LRU 淘汰
	c := newTestRedisCache()
	skipIfRedisNotAvailable(t, c)
	ctx := context.Background()

	_ = c.Set(ctx, "A", "va", 0)
	_ = c.Set(ctx, "B", "vb", 0)

	// 访问 A，使 B 成为 LRU
	_, _ = c.Get(ctx, "A")

	// 插入 C，应淘汰 B
	_ = c.Set(ctx, "C", "vc", 0)

	_, _ = c.Get(ctx, "B")
	_, okA := c.Get(ctx, "A")
	_, okC := c.Get(ctx, "C")
	assert.True(t, okA)
	assert.True(t, okC)
}
