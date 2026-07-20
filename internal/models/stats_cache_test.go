package models

import (
	"context"
	"testing"
	"time"

	llmcache "github.com/LingByte/lingllm/cache"
)

func TestStatsCacheJSONRoundTrip(t *testing.T) {
	llmcache.SetGlobalCache(llmcache.NewLocalCache(llmcache.LocalConfig{
		MaxSize:           16,
		DefaultExpiration: time.Minute,
		CleanupInterval:   time.Minute,
	}))

	key := CallAnalyticsCacheKey(42, "2026-01-01", "2026-01-14")
	want := map[string]any{"callCount": 12, "rangeStart": "2026-01-01"}
	StatsCacheSetJSON(context.Background(), key, want, StatsCacheTTL)

	var got map[string]any
	if !StatsCacheGetJSON(context.Background(), key, &got) {
		t.Fatal("expected cache hit")
	}
	if got["callCount"] != float64(want["callCount"].(int)) {
		t.Fatalf("callCount = %v, want %v", got["callCount"], want["callCount"])
	}
}

func TestStatsCacheBypassed(t *testing.T) {
	ctx := WithStatsCacheBypass(context.Background())
	if !StatsCacheBypassed(ctx) {
		t.Fatal("expected bypass flag")
	}
}
