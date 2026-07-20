package knowledge

import (
	"context"
	"testing"
	"time"
)

func TestQueryCache_GetSet(t *testing.T) {
	cache := NewQueryCache(10, time.Minute)
	ctx := context.Background()

	query := "test query"
	results := []QueryResult{
		{
			Record: Record{ID: "1", Content: "test content"},
			Score:  0.95,
		},
	}

	// Initially should not be in cache
	_, ok := cache.Get(ctx, query)
	if ok {
		t.Error("expected cache miss, got hit")
	}

	// Set and retrieve
	cache.Set(ctx, query, results)
	retrieved, ok := cache.Get(ctx, query)
	if !ok {
		t.Error("expected cache hit, got miss")
	}

	if len(retrieved) != len(results) {
		t.Errorf("expected %d results, got %d", len(results), len(retrieved))
	}
}

func TestQueryCache_Stats(t *testing.T) {
	cache := NewQueryCache(10, time.Minute)
	ctx := context.Background()

	// Initial stats
	stats := cache.Stats()
	if stats["hit_count"] != int64(0) {
		t.Error("expected 0 hits initially")
	}

	// Add some cache hits and misses
	cache.Get(ctx, "query1") // miss
	cache.Get(ctx, "query2") // miss
	cache.Set(ctx, "query3", []QueryResult{})
	cache.Get(ctx, "query3") // hit

	stats = cache.Stats()
	if stats["hit_count"] != int64(1) {
		t.Errorf("expected 1 hit, got %v", stats["hit_count"])
	}
	if stats["miss_count"] != int64(2) {
		t.Errorf("expected 2 misses, got %v", stats["miss_count"])
	}
}

func TestQueryCache_Clear(t *testing.T) {
	cache := NewQueryCache(10, time.Minute)
	ctx := context.Background()

	cache.Set(ctx, "query1", []QueryResult{})
	cache.Set(ctx, "query2", []QueryResult{})

	cache.Clear()

	_, ok := cache.Get(ctx, "query1")
	if ok {
		t.Error("expected cache miss after clear")
	}
}

func TestVectorCache_GetSet(t *testing.T) {
	cache := NewVectorCache(10)

	text := "test text"
	vector := []float32{0.1, 0.2, 0.3}

	// Initially should not be in cache
	_, ok := cache.Get(text)
	if ok {
		t.Error("expected cache miss, got hit")
	}

	// Set and retrieve
	cache.Set(text, vector)
	retrieved, ok := cache.Get(text)
	if !ok {
		t.Error("expected cache hit, got miss")
	}

	if len(retrieved) != len(vector) {
		t.Errorf("expected %d elements, got %d", len(vector), len(retrieved))
	}

	// Verify values
	for i := range vector {
		if retrieved[i] != vector[i] {
			t.Errorf("expected %f, got %f", vector[i], retrieved[i])
		}
	}
}

func TestVectorCache_Size(t *testing.T) {
	cache := NewVectorCache(10)

	if cache.Size() != 0 {
		t.Error("expected size 0 initially")
	}

	cache.Set("text1", []float32{0.1})
	cache.Set("text2", []float32{0.2})

	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}
}

func TestVectorCache_Clear(t *testing.T) {
	cache := NewVectorCache(10)

	cache.Set("text1", []float32{0.1})
	cache.Set("text2", []float32{0.2})

	cache.Clear()

	if cache.Size() != 0 {
		t.Error("expected size 0 after clear")
	}

	_, ok := cache.Get("text1")
	if ok {
		t.Error("expected cache miss after clear")
	}
}

func TestVectorCache_Eviction(t *testing.T) {
	cache := NewVectorCache(2)

	cache.Set("text1", []float32{0.1})
	cache.Set("text2", []float32{0.2})

	// Adding a third item should trigger eviction
	cache.Set("text3", []float32{0.3})

	// Cache should be cleared and only contain the new item
	if cache.Size() != 1 {
		t.Errorf("expected size 1 after eviction, got %d", cache.Size())
	}
}

func TestQueryCache_Expiration(t *testing.T) {
	cache := NewQueryCache(10, 100*time.Millisecond)
	ctx := context.Background()

	cache.Set(ctx, "query", []QueryResult{})

	// Should be in cache immediately
	_, ok := cache.Get(ctx, "query")
	if !ok {
		t.Error("expected cache hit before expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, ok = cache.Get(ctx, "query")
	if ok {
		t.Error("expected cache miss after expiration")
	}
}
