package embedder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// CachingEmbedder wraps an Embedder with an in-process LRU-ish vector cache.
// Identical query strings (after trim) reuse the prior embedding within TTL.
type CachingEmbedder struct {
	inner   Embedder
	ttl     time.Duration
	maxSize int

	mu    sync.Mutex
	cache map[string]cachingEmbedEntry
	hits  int64
	miss  int64
}

type cachingEmbedEntry struct {
	vec []float32
	at  time.Time
}

// NewCachingEmbedder returns an Embedder that caches single- and multi-text embeds.
// maxSize <= 0 defaults to 2048 entries; ttl <= 0 defaults to 30 minutes.
func NewCachingEmbedder(inner Embedder, maxSize int, ttl time.Duration) *CachingEmbedder {
	if maxSize <= 0 {
		maxSize = 2048
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &CachingEmbedder{
		inner:   inner,
		ttl:     ttl,
		maxSize: maxSize,
		cache:   make(map[string]cachingEmbedEntry),
	}
}

func (c *CachingEmbedder) Name() string {
	if c == nil || c.inner == nil {
		return "caching"
	}
	return c.inner.Name() + "+cache"
}

func (c *CachingEmbedder) Provider() string {
	if c == nil || c.inner == nil {
		return ""
	}
	return c.inner.Provider()
}

func (c *CachingEmbedder) Dimension() int {
	if c == nil || c.inner == nil {
		return 0
	}
	return c.inner.Dimension()
}

func (c *CachingEmbedder) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

func (c *CachingEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, ErrEmptyInput
	}
	return vecs[0], nil
}

func (c *CachingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if c == nil || c.inner == nil {
		return nil, ErrEmptyInput
	}
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	out := make([][]float32, len(texts))
	missingIdx := make([]int, 0, len(texts))
	missingTexts := make([]string, 0, len(texts))

	now := time.Now()
	c.mu.Lock()
	for i, text := range texts {
		key := cacheKey(text)
		if e, ok := c.cache[key]; ok && now.Sub(e.at) <= c.ttl {
			out[i] = copyFloat32(e.vec)
			c.hits++
			continue
		}
		c.miss++
		missingIdx = append(missingIdx, i)
		missingTexts = append(missingTexts, text)
	}
	c.mu.Unlock()

	if len(missingTexts) == 0 {
		return out, nil
	}

	fresh, err := c.inner.Embed(ctx, missingTexts)
	if err != nil {
		return nil, err
	}
	if len(fresh) != len(missingTexts) {
		return nil, ErrEmptyInput
	}

	c.mu.Lock()
	if len(c.cache) >= c.maxSize {
		c.cache = make(map[string]cachingEmbedEntry)
	}
	for j, idx := range missingIdx {
		vec := copyFloat32(fresh[j])
		out[idx] = vec
		c.cache[cacheKey(missingTexts[j])] = cachingEmbedEntry{vec: copyFloat32(vec), at: now}
	}
	c.mu.Unlock()

	return out, nil
}

// Stats returns cache hit/miss counters (for diagnostics).
func (c *CachingEmbedder) Stats() (hits, miss int64) {
	if c == nil {
		return 0, 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.miss
}

func cacheKey(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

func copyFloat32(in []float32) []float32 {
	if in == nil {
		return nil
	}
	out := make([]float32, len(in))
	copy(out, in)
	return out
}
