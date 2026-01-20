package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestCache(maxSize int, defaultExp, cleanup time.Duration) Cache {
	return NewLocalCache(LocalConfig{
		MaxSize:           maxSize,
		DefaultExpiration: defaultExp,
		CleanupInterval:   cleanup,
	})
}

func TestSetGetExistsDelete(t *testing.T) {
	c := newTestCache(10, time.Minute, time.Hour)
	ctx := context.Background()

	// Set & Get
	err := c.Set(ctx, "k1", "v1", 0)
	assert.NoError(t, err)

	v, ok := c.Get(ctx, "k1")
	assert.True(t, ok)
	assert.Equal(t, "v1", v)

	// Exists
	assert.True(t, c.Exists(ctx, "k1"))
	assert.False(t, c.Exists(ctx, "nope"))

	// Delete
	assert.NoError(t, c.Delete(ctx, "k1"))
	_, ok = c.Get(ctx, "k1")
	assert.False(t, ok)
}

func TestExpirationOnGetAndExists(t *testing.T) {
	c := newTestCache(10, time.Minute, time.Hour)
	ctx := context.Background()

	err := c.Set(ctx, "exp", 42, 50*time.Millisecond)
	assert.NoError(t, err)

	// 立即可读
	v, ok := c.Get(ctx, "exp")
	assert.True(t, ok)
	assert.Equal(t, 42, v)

	// 等待过期
	time.Sleep(70 * time.Millisecond)

	_, ok = c.Get(ctx, "exp")
	assert.False(t, ok)
	assert.False(t, c.Exists(ctx, "exp"))
}

func TestGetWithTTL(t *testing.T) {
	c := newTestCache(10, time.Minute, time.Hour)
	ctx := context.Background()

	err := c.Set(ctx, "ttl", "vv", 80*time.Millisecond)
	assert.NoError(t, err)

	_, ttl, ok := c.GetWithTTL(ctx, "ttl")
	assert.True(t, ok)
	assert.Greater(t, ttl, 0*time.Millisecond)
	assert.LessOrEqual(t, ttl, 80*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	_, ttl, ok = c.GetWithTTL(ctx, "ttl")
	assert.False(t, ok)
	assert.Equal(t, 0*time.Millisecond, ttl)
}

func TestIncrementDecrement(t *testing.T) {
	c := newTestCache(10, 200*time.Millisecond, time.Hour)
	ctx := context.Background()

	// 不存在键：创建并返回 value
	nv, err := c.Increment(ctx, "cnt", 5)
	assert.NoError(t, err)
	assert.EqualValues(t, 5, nv)

	// 已存在为 int64：继续自增
	nv, err = c.Increment(ctx, "cnt", 3)
	assert.NoError(t, err)
	assert.EqualValues(t, 8, nv)

	// 自减
	nv, err = c.Decrement(ctx, "cnt", 2)
	assert.NoError(t, err)
	assert.EqualValues(t, 6, nv)

	// 用 int 类型
	assert.NoError(t, c.Set(ctx, "i", int(10), 0))
	nv, err = c.Increment(ctx, "i", 4)
	assert.NoError(t, err)
	assert.EqualValues(t, 14, nv)

	// 用 float64 类型
	assert.NoError(t, c.Set(ctx, "f", float64(1.2), 0))
	nv, err = c.Increment(ctx, "f", 10)
	assert.NoError(t, err)
	assert.EqualValues(t, 11, nv)

	// 不支持类型：重置为增量值
	assert.NoError(t, c.Set(ctx, "x", "bad-type", 0))
	nv, err = c.Increment(ctx, "x", 99)
	assert.NoError(t, err)
	assert.EqualValues(t, 99, nv)
}

func TestBatchOps(t *testing.T) {
	c := newTestCache(10, time.Minute, time.Hour)
	ctx := context.Background()

	data := map[string]interface{}{
		"a": 1,
		"b": "B",
		"c": 3.14,
	}
	assert.NoError(t, c.SetMulti(ctx, data, 0))

	got := c.GetMulti(ctx, "a", "b", "c", "d")
	assert.Equal(t, 1, got["a"])
	assert.Equal(t, "B", got["b"])
	assert.Equal(t, 3.14, got["c"])
	_, hasD := got["d"]
	assert.False(t, hasD)

	assert.NoError(t, c.DeleteMulti(ctx, "a", "c"))
	_, ok := c.Get(ctx, "a")
	assert.False(t, ok)
	_, ok = c.Get(ctx, "c")
	assert.False(t, ok)
}

func TestClearAndClose(t *testing.T) {
	c := newTestCache(10, time.Minute, time.Hour)
	ctx := context.Background()

	_ = c.Set(ctx, "k", "v", 0)
	assert.True(t, c.Exists(ctx, "k"))

	assert.NoError(t, c.Clear(ctx))
	assert.False(t, c.Exists(ctx, "k"))

	assert.NoError(t, c.Close())
}

func TestLRUEviction(t *testing.T) {
	// MaxSize=2，验证 LRU 淘汰
	c := newTestCache(2, time.Minute, time.Hour)
	ctx := context.Background()

	_ = c.Set(ctx, "A", "va", 0)
	_ = c.Set(ctx, "B", "vb", 0)

	// 访问 A，使 B 成为 LRU
	_, _ = c.Get(ctx, "A")

	// 插入 C，应淘汰 B
	_ = c.Set(ctx, "C", "vc", 0)

	_, okB := c.Get(ctx, "B")
	_, okA := c.Get(ctx, "A")
	_, okC := c.Get(ctx, "C")

	assert.False(t, okB, "B 应当被淘汰")
	assert.True(t, okA)
	assert.True(t, okC)
}

func TestBackgroundCleanup(t *testing.T) {
	// 将 CleanupInterval 设短一些，确保后台清理线程会运行
	c := newTestCache(10, time.Minute, 20*time.Millisecond)
	ctx := context.Background()

	_ = c.Set(ctx, "short", "x", 30*time.Millisecond)
	_ = c.Set(ctx, "long", "y", 500*time.Millisecond)

	// 先确保都存在
	_, ok := c.Get(ctx, "short")
	assert.True(t, ok)
	_, ok = c.Get(ctx, "long")
	assert.True(t, ok)

	// 等待 short 过期 + 等待一次清理周期
	time.Sleep(90 * time.Millisecond)

	// 即使不主动访问，也应被清掉（依赖 cleanup 协程）
	_, ok = c.Get(ctx, "short")
	assert.False(t, ok, "短期项目应被后台清理")
	_, ok = c.Get(ctx, "long")
	assert.True(t, ok, "长期项目不应被清理")
}

func TestSetGetWithExpiration(t *testing.T) {
	c := newTestCache(10, 200*time.Millisecond, time.Minute)
	ctx := context.Background()

	// 设置带过期时间的缓存
	_ = c.Set(ctx, "exp_key", "exp_value", 100*time.Millisecond)

	// 立即读取
	val, ok := c.Get(ctx, "exp_key")
	assert.True(t, ok)
	assert.Equal(t, "exp_value", val)

	// 等待过期
	time.Sleep(150 * time.Millisecond)
	_, ok = c.Get(ctx, "exp_key")
	assert.False(t, ok, "Key should be expired after timeout")
}

func TestDeleteAfterExpiration(t *testing.T) {
	c := newTestCache(10, 100*time.Millisecond, time.Minute)
	ctx := context.Background()

	// 设置带过期时间的缓存
	_ = c.Set(ctx, "exp_key", "exp_value", 100*time.Millisecond)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 检查缓存是否已删除
	_, ok := c.Get(ctx, "exp_key")
	assert.False(t, ok)
}
