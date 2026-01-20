package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Helpers -------------------------------------------------------------

func mustLocalConfig() LocalConfig {
	return LocalConfig{
		MaxSize:           128,
		DefaultExpiration: 200 * time.Millisecond,
		CleanupInterval:   50 * time.Millisecond,
	}
}

func mustRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 1,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
}

func requireRedisOrSkip(t *testing.T) {
	t.Helper()
	_, err := NewRedisCache(mustRedisConfig())
	if err != nil {
		t.Skipf("skip: redis not available at %s: %v", mustRedisConfig().Addr, err)
	}
}

// --- Factory tests -------------------------------------------------------

func TestNewCache_Factory_Local(t *testing.T) {
	cfg := Config{
		Type:  KindLocal,
		Local: mustLocalConfig(),
	}
	c, err := NewCache(cfg)
	assert.NoError(t, err)
	// 应为 *localCache
	_, ok := c.(*localCache)
	assert.True(t, ok, "expect *localCache for KindLocal")
	_ = c.Close()
}

func TestNewCache_Factory_GoCache(t *testing.T) {
	cfg := Config{
		Type:  KindGoCache,
		Local: mustLocalConfig(),
	}
	c, err := NewCache(cfg)
	assert.NoError(t, err)
	// 应为 *goCacheWrapper
	_, ok := c.(*goCacheWrapper)
	assert.True(t, ok, "expect *goCacheWrapper for KindGoCache")
	_ = c.Close()
}

func TestNewCache_Factory_Redis(t *testing.T) {
	requireRedisOrSkip(t)

	cfg := Config{
		Type:  KindRedis,
		Redis: mustRedisConfig(),
	}
	c, err := NewCache(cfg)
	assert.NoError(t, err)
	// 应为 *redisCache
	_, ok := c.(*redisCache)
	assert.True(t, ok, "expect *redisCache for KindRedis")
	_ = c.Close()
}

func TestNewCacheWithOptions_UseLocalLayered(t *testing.T) {
	requireRedisOrSkip(t)

	cfg := Config{
		Type:  KindRedis,
		Local: mustLocalConfig(),
		Redis: mustRedisConfig(),
	}
	opts := &Options{
		UseLocalCache:   true,
		LocalExpiration: 150 * time.Millisecond,
	}
	c, err := NewCacheWithOptions(cfg, opts)
	assert.NoError(t, err)

	// 应该返回 layeredCache
	lc, ok := c.(*layeredCache)
	assert.True(t, ok, "expect *layeredCache when UseLocalCache=true")
	assert.NotNil(t, lc.local)
	assert.NotNil(t, lc.distributed)
	assert.Equal(t, opts, lc.options)

	_ = c.Close()
}

// --- Layered cache behavior ---------------------------------------------

func TestLayeredCache_BasicFlow_SetGetBackfill(t *testing.T) {
	requireRedisOrSkip(t)

	cfg := Config{
		Type:  KindRedis,
		Local: mustLocalConfig(),
		Redis: mustRedisConfig(),
	}
	opts := &Options{
		UseLocalCache:   true,
		LocalExpiration: 300 * time.Millisecond,
	}

	cc, err := NewCacheWithOptions(cfg, opts)
	assert.NoError(t, err)
	lc := cc.(*layeredCache)
	ctx := context.Background()
	key := "layered:k1"

	// 1) Set: 同步到分布式 + 本地
	err = lc.Set(ctx, key, "V1", 1*time.Second)
	assert.NoError(t, err)

	// 2) Get: 应该本地命中
	val, ok := lc.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, "V1", val)

	// 3) 删除本地层，验证分布式回填
	_ = lc.local.Delete(ctx, key)

	// 确认本地不存在
	_, ok = lc.local.Get(ctx, key)
	assert.False(t, ok)

	// 从 layered 读，应该会打到分布式并回填本地
	val, ok = lc.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, "V1", val)

	// 再次直接读本地应命中（已回填）
	val, ok = lc.local.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, "V1", val)

	_ = lc.Delete(ctx, key)
	_ = lc.Close()
}

func TestLayeredCache_MultiOps_BackfillAndTTL(t *testing.T) {
	requireRedisOrSkip(t)

	cfg := Config{
		Type:  KindRedis,
		Local: mustLocalConfig(),
		Redis: mustRedisConfig(),
	}
	opts := &Options{
		UseLocalCache:   true,
		LocalExpiration: 120 * time.Millisecond, // 本地较短 TTL，便于测试
	}

	cc, err := NewCacheWithOptions(cfg, opts)
	assert.NoError(t, err)
	lc := cc.(*layeredCache)
	ctx := context.Background()

	data := map[string]interface{}{
		"layered:k2": "V2",
		"layered:k3": "V3",
	}
	// 1) 批量设置：分布式 + 本地
	err = lc.SetMulti(ctx, data, 1*time.Second)
	assert.NoError(t, err)

	// 2) 批量读取：应命中本地
	m := lc.GetMulti(ctx, "layered:k2", "layered:k3", "layered:missing")
	assert.Equal(t, "V2", m["layered:k2"])
	assert.Equal(t, "V3", m["layered:k3"])
	_, has := m["layered:missing"]
	assert.False(t, has)

	// 3) 本地 TTL 到期后，应可从分布式回填
	time.Sleep(150 * time.Millisecond)
	// 清理本地（有些实现依赖清理器；这里直接 Delete 本地，确保 miss）
	_ = lc.local.DeleteMulti(ctx, "layered:k2", "layered:k3")

	// 再次 GetMulti，会走分布式并回填
	m2 := lc.GetMulti(ctx, "layered:k2", "layered:k3")
	assert.Equal(t, "V2", m2["layered:k2"])
	assert.Equal(t, "V3", m2["layered:k3"])

	// 验证 GetWithTTL：先本地命中
	_, ttl, ok := lc.GetWithTTL(ctx, "layered:k2")
	assert.True(t, ok)
	assert.Greater(t, ttl, 0*time.Millisecond)

	_ = lc.DeleteMulti(ctx, "layered:k2", "layered:k3")
	_ = lc.Close()
}

func TestLayeredCache_IncDec_Clear_Exists_Close(t *testing.T) {
	requireRedisOrSkip(t)

	cfg := Config{
		Type:  KindRedis,
		Local: mustLocalConfig(),
		Redis: mustRedisConfig(),
	}
	opts := &Options{
		UseLocalCache:   true,
		LocalExpiration: 300 * time.Millisecond,
	}

	cc, err := NewCacheWithOptions(cfg, opts)
	assert.NoError(t, err)
	lc := cc.(*layeredCache)
	ctx := context.Background()
	key := "layered:cnt"

	// Increment on distributed, write-through to local
	nv, err := lc.Increment(ctx, key, 5)
	assert.NoError(t, err)
	assert.EqualValues(t, 5, nv)

	// Increment again
	nv, err = lc.Increment(ctx, key, 2)
	assert.NoError(t, err)
	assert.EqualValues(t, 7, nv)

	// Decrement
	nv, err = lc.Decrement(ctx, key, 3)
	assert.NoError(t, err)
	assert.EqualValues(t, 4, nv)

	// Exists: 本地或分布式任一存在即 true
	assert.True(t, lc.Exists(ctx, key))

	// Clear: 清空两层
	err = lc.Clear(ctx)
	assert.NoError(t, err)

	// 应该不存在
	assert.False(t, lc.Exists(ctx, key))

	_ = lc.Close()
}
