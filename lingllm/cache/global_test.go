package cache

import (
	"context"
	"testing"
	"time"
)

func TestGlobalCacheInitialization(t *testing.T) {
	// Initialize global cache with Config
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	err := InitGlobalCache(config)
	if err != nil {
		t.Errorf("InitGlobalCache failed: %v", err)
	}
}

func TestGlobalCacheSet(t *testing.T) {
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	InitGlobalCache(config)

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	err := Set(ctx, key, value, 1*time.Hour)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}
}

func TestGlobalCacheGet(t *testing.T) {
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	InitGlobalCache(config)

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value first
	Set(ctx, key, value, 1*time.Hour)

	// Get value
	result, exists := Get(ctx, key)
	if !exists {
		t.Error("Get should return existing value")
	}

	if result != value {
		t.Errorf("Get returned %v, want %v", result, value)
	}
}

func TestGlobalCacheDelete(t *testing.T) {
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	InitGlobalCache(config)

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value first
	Set(ctx, key, value, 1*time.Hour)

	// Delete value
	err := Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify it's deleted
	result, exists := Get(ctx, key)
	if exists && result != nil {
		t.Error("Value should be deleted")
	}
}

func TestGlobalCacheClear(t *testing.T) {
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	InitGlobalCache(config)

	ctx := context.Background()

	// Set multiple values
	Set(ctx, "key1", "value1", 1*time.Hour)
	Set(ctx, "key2", "value2", 1*time.Hour)

	// Clear cache
	err := Clear(ctx)
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}
}

func TestGlobalCacheExists(t *testing.T) {
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	InitGlobalCache(config)

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value
	Set(ctx, key, value, 1*time.Hour)

	// Check if exists
	exists := Exists(ctx, key)
	if !exists {
		t.Error("Key should exist")
	}
}

func TestGlobalCacheGetMulti(t *testing.T) {
	config := Config{
		Type: "local",
		Local: LocalConfig{
			MaxSize:         1000,
			CleanupInterval: 5 * time.Minute,
		},
	}
	InitGlobalCache(config)

	ctx := context.Background()

	// Set multiple values
	Set(ctx, "key1", "value1", 1*time.Hour)
	Set(ctx, "key2", "value2", 1*time.Hour)

	// Get multiple values
	results := GetMulti(ctx, "key1", "key2")

	if len(results) != 2 {
		t.Errorf("GetMulti returned %d results, want 2", len(results))
	}
}
