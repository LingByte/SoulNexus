package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestGoCache() *goCacheWrapper {
	config := LocalConfig{
		DefaultExpiration: 1 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewGoCache(config)
	return cache.(*goCacheWrapper)
}

func TestGoCacheSetGet(t *testing.T) {
	c := newTestGoCache()
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

func TestGoCacheDelete(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set and Delete value
	err := c.Set(ctx, "key2", "value2", 0)
	assert.NoError(t, err)
	err = c.Delete(ctx, "key2")
	assert.NoError(t, err)

	// Check if the key was deleted
	_, exists := c.Get(ctx, "key2")
	assert.False(t, exists)
}

func TestGoCacheExists(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set value and check existence
	c.Set(ctx, "key3", "value3", 0)
	exists := c.Exists(ctx, "key3")
	assert.True(t, exists)

	// Check non-existing key
	exists = c.Exists(ctx, "key_not_exists")
	assert.False(t, exists)
}

func TestGoCacheGetWithTTL(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set value with expiration time
	c.Set(ctx, "key4", "value4", 50*time.Millisecond)

	// Get value with TTL
	val, ttl, exists := c.GetWithTTL(ctx, "key4")
	assert.True(t, exists)
	assert.Equal(t, "value4", val)
	assert.Greater(t, ttl, 0*time.Millisecond)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)
	_, ttl, exists = c.GetWithTTL(ctx, "key4")
	assert.False(t, exists)
	assert.Equal(t, 0*time.Millisecond, ttl)
}

func TestGoCacheIncrementDecrement(t *testing.T) {
	c := newTestGoCache()
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

func TestGoCacheGetMulti(t *testing.T) {
	c := newTestGoCache()
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
	assert.Equal(t, "value5", result["key5"])
	assert.Equal(t, "value6", result["key6"])
	assert.Nil(t, result["key_not_exists"])
}

func TestGoCacheDeleteMulti(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set multiple keys
	data := map[string]interface{}{
		"key7": "value7",
		"key8": "value8",
	}
	err := c.SetMulti(ctx, data, 0)
	assert.NoError(t, err)

	// Delete multiple keys
	err = c.DeleteMulti(ctx, "key7", "key8")
	assert.NoError(t, err)

	// Check if they are deleted
	_, exists := c.Get(ctx, "key7")
	assert.False(t, exists)
	_, exists = c.Get(ctx, "key8")
	assert.False(t, exists)
}

func TestGoCacheClear(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set multiple keys
	data := map[string]interface{}{
		"key9":  "value9",
		"key10": "value10",
	}
	err := c.SetMulti(ctx, data, 0)
	assert.NoError(t, err)

	// Clear the cache
	err = c.Clear(ctx)
	assert.NoError(t, err)

	// Check if all keys are cleared
	_, exists := c.Get(ctx, "key9")
	assert.False(t, exists)
	_, exists = c.Get(ctx, "key10")
	assert.False(t, exists)
}

func TestGoCacheClose(t *testing.T) {
	c := newTestGoCache()
	err := c.Close()
	assert.NoError(t, err)
}

func TestGoCacheWrapperIntegration(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set a value
	err := c.Set(ctx, "integration_key", "integration_value", 0)
	assert.NoError(t, err)

	// Get the value
	val, exists := c.Get(ctx, "integration_key")
	assert.True(t, exists)
	assert.Equal(t, "integration_value", val)

	// Clean up
	err = c.Delete(ctx, "integration_key")
	assert.NoError(t, err)
}

func TestItems(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set multiple keys
	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := c.SetMulti(ctx, data, 0)
	assert.NoError(t, err)

	// Get all items
	items := c.Items()
	assert.Equal(t, 2, len(items))
}

func TestItemCount(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set some keys
	c.Set(ctx, "key1", "value1", 0)
	c.Set(ctx, "key2", "value2", 0)

	// Check item count
	count := c.ItemCount()
	assert.Equal(t, 2, count)
}

func TestFlush(t *testing.T) {
	c := newTestGoCache()
	ctx := context.Background()

	// Set keys
	c.Set(ctx, "key1", "value1", 0)
	c.Set(ctx, "key2", "value2", 0)

	// Flush the cache
	c.Flush()

	// Verify that cache is cleared
	_, exists := c.Get(ctx, "key1")
	assert.False(t, exists)
	_, exists = c.Get(ctx, "key2")
	assert.False(t, exists)
}
