package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExpiredLRUCache_Basic(t *testing.T) {
	cache := NewExpiredLRUCache[string, string](2, 100*time.Millisecond)

	// 测试添加与获取（未过期）
	cache.Add("a", "valueA")
	val, ok := cache.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "valueA", val)

	// 测试更新 key，值是否替换
	cache.Add("a", "newA")
	val, ok = cache.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "newA", val)

	// 测试 Contains（不过期情况）
	assert.True(t, cache.Contains("a"))

	// 测试 Remove
	ok = cache.Remove("a")
	assert.True(t, ok)
	_, ok = cache.Get("a")
	assert.False(t, ok)
}

func TestExpiredLRUCache_Expiration(t *testing.T) {
	cache := NewExpiredLRUCache[string, string](2, 50*time.Millisecond)

	cache.Add("k1", "v1")
	time.Sleep(60 * time.Millisecond)
	_, ok := cache.Get("k1") // 已过期
	assert.False(t, ok)

	// 测试过期后 key 自动删除
	assert.False(t, cache.Contains("k1"))
}

func TestExpiredLRUCache_Eviction(t *testing.T) {
	cache := NewExpiredLRUCache[string, string](1, 1*time.Second)

	// 插入第一个 key
	evicted := cache.Add("key1", "val1")
	assert.False(t, evicted) // 第一次不应该驱逐

	// 插入第二个 key，应该驱逐 key1（容量 1）
	evicted = cache.Add("key2", "val2")
	assert.True(t, evicted)

	_, ok := cache.Get("key1")
	assert.False(t, ok)

	val, ok := cache.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, "val2", val)
}
