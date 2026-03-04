package cache

import (
	"context"
	"testing"
	"time"
)

func TestLocalCache(t *testing.T) {
	config := LocalConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}

	cache := NewLocalCache(config)
	defer cache.Close()

	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		key := "test_key"
		value := "test_value"
		expiration := 1 * time.Minute

		err := cache.Set(ctx, key, value, expiration)
		if err != nil {
			t.Errorf("Failed to set cache: %v", err)
		}

		if retrieved, exists := cache.Get(ctx, key); !exists {
			t.Error("Cache value not found")
		} else if retrieved != value {
			t.Errorf("Expected %v, got %v", value, retrieved)
		}
	})
}
