package knowledge

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// QueryCache provides caching for query results to improve performance
type QueryCache struct {
	mu        sync.RWMutex
	cache     map[string]*CacheEntry
	maxSize   int
	ttl       time.Duration
	hitCount  int64
	missCount int64
}

// CacheEntry represents a cached query result
type CacheEntry struct {
	Results   []QueryResult
	Timestamp time.Time
}

// NewQueryCache creates a new query cache with specified size and TTL
func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &QueryCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a cached query result
func (qc *QueryCache) Get(ctx context.Context, query string) ([]QueryResult, bool) {
	if qc == nil {
		return nil, false
	}

	key := qc.hashKey(query)
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	entry, ok := qc.cache[key]
	if !ok {
		qc.missCount++
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.Timestamp) > qc.ttl {
		qc.missCount++
		return nil, false
	}

	qc.hitCount++
	return entry.Results, true
}

// Set stores a query result in the cache
func (qc *QueryCache) Set(ctx context.Context, query string, results []QueryResult) {
	if qc == nil {
		return
	}

	key := qc.hashKey(query)
	qc.mu.Lock()
	defer qc.mu.Unlock()

	// Simple eviction: if cache is full, clear it
	if len(qc.cache) >= qc.maxSize {
		qc.cache = make(map[string]*CacheEntry)
	}

	qc.cache[key] = &CacheEntry{
		Results:   results,
		Timestamp: time.Now(),
	}
}

// Clear clears all cached entries
func (qc *QueryCache) Clear() {
	if qc == nil {
		return
	}

	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.cache = make(map[string]*CacheEntry)
}

// Stats returns cache statistics
func (qc *QueryCache) Stats() map[string]any {
	if qc == nil {
		return nil
	}

	qc.mu.RLock()
	defer qc.mu.RUnlock()

	total := qc.hitCount + qc.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(qc.hitCount) / float64(total)
	}

	return map[string]any{
		"hit_count":   qc.hitCount,
		"miss_count":  qc.missCount,
		"total":       total,
		"hit_rate":    hitRate,
		"size":        len(qc.cache),
		"max_size":    qc.maxSize,
		"ttl_seconds": int64(qc.ttl.Seconds()),
	}
}

// hashKey generates a hash key for the query
func (qc *QueryCache) hashKey(query string) string {
	hash := md5.Sum([]byte(query))
	return fmt.Sprintf("%x", hash)
}

// VectorCache provides caching for embedding vectors
type VectorCache struct {
	mu      sync.RWMutex
	cache   map[string][]float32
	maxSize int
}

// NewVectorCache creates a new vector cache
func NewVectorCache(maxSize int) *VectorCache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &VectorCache{
		cache:   make(map[string][]float32),
		maxSize: maxSize,
	}
}

// Get retrieves a cached vector
func (vc *VectorCache) Get(text string) ([]float32, bool) {
	if vc == nil {
		return nil, false
	}

	key := vc.hashKey(text)
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	vec, ok := vc.cache[key]
	return vec, ok
}

// Set stores a vector in the cache
func (vc *VectorCache) Set(text string, vector []float32) {
	if vc == nil {
		return
	}

	key := vc.hashKey(text)
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Simple eviction: if cache is full, clear it
	if len(vc.cache) >= vc.maxSize {
		vc.cache = make(map[string][]float32)
	}

	// Store a copy to prevent external modifications
	vec := make([]float32, len(vector))
	copy(vec, vector)
	vc.cache[key] = vec
}

// Clear clears all cached vectors
func (vc *VectorCache) Clear() {
	if vc == nil {
		return
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.cache = make(map[string][]float32)
}

// Size returns the current cache size
func (vc *VectorCache) Size() int {
	if vc == nil {
		return 0
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return len(vc.cache)
}

// hashKey generates a hash key for the text
func (vc *VectorCache) hashKey(text string) string {
	hash := md5.Sum([]byte(text))
	return fmt.Sprintf("%x", hash)
}
