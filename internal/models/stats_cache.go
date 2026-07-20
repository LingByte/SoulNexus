package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmcache "github.com/LingByte/lingllm/cache"
)

// StatsCacheTTL is the default TTL for dashboard / analytics snapshots.
const StatsCacheTTL = 5 * time.Minute

// StatsCacheKey builds a colon-separated cache key.
func StatsCacheKey(parts ...string) string {
	return strings.Join(parts, ":")
}

// OverviewCacheKey keys overview dashboard payloads.
func OverviewCacheKey(tenantID uint, startDay, endDay string, platformAdmin bool) string {
	if platformAdmin {
		return StatsCacheKey("overview", "platform", startDay, endDay)
	}
	return StatsCacheKey("overview", "tenant", fmt.Sprintf("%d", tenantID), startDay, endDay)
}

// CallAnalyticsCacheKey keys tenant call analytics dashboards.
func CallAnalyticsCacheKey(tenantID uint, startDay, endDay string) string {
	return StatsCacheKey("call_analytics", fmt.Sprintf("%d", tenantID), startDay, endDay)
}

// CalloutAnalysisCacheKey keys outbound callout analysis payloads.
func CalloutAnalysisCacheKey(tenantID uint, startDay, endDay string) string {
	return StatsCacheKey("callout_analysis", fmt.Sprintf("%d", tenantID), startDay, endDay)
}

// CallerAttributesCacheKey keys paginated caller attribute lists.
func CallerAttributesCacheKey(tenantID uint, startDay, endDay, province string, page, size int) string {
	return StatsCacheKey(
		"caller_attributes",
		fmt.Sprintf("%d", tenantID),
		startDay,
		endDay,
		strings.TrimSpace(province),
		fmt.Sprintf("%d", page),
		fmt.Sprintf("%d", size),
	)
}

// StatsCacheGetJSON loads a JSON-encoded value from the global cache.
func StatsCacheGetJSON(ctx context.Context, key string, dest any) bool {
	if dest == nil || strings.TrimSpace(key) == "" {
		return false
	}
	raw, ok := llmcache.Get(ctx, key)
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case string:
		return json.Unmarshal([]byte(v), dest) == nil
	case []byte:
		return json.Unmarshal(v, dest) == nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return false
		}
		return json.Unmarshal(b, dest) == nil
	}
}

// StatsCacheSetJSON stores a JSON-encoded value in the global cache.
func StatsCacheSetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if value == nil || strings.TrimSpace(key) == "" {
		return
	}
	if ttl <= 0 {
		ttl = StatsCacheTTL
	}
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = llmcache.Set(ctx, key, string(b), ttl)
}

// StatsCacheBypass reports whether the client asked for a fresh load (?refresh=1).
func StatsCacheBypass(refreshQuery string) bool {
	return strings.TrimSpace(refreshQuery) == "1"
}

type statsCacheBypassKey struct{}

// WithStatsCacheBypass skips read-through cache for one request context.
func WithStatsCacheBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, statsCacheBypassKey{}, true)
}

// StatsCacheBypassed reports whether caching should be skipped for this context.
func StatsCacheBypassed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(statsCacheBypassKey{}).(bool)
	return v
}
