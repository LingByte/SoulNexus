package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

// Why this file exists
//
// QCloud (and most cloud-TTS) round-trip latency is dominated by *cold-start*:
// DNS + TLS handshake + auth + initial WS frame ≈ 200-500 ms before the first
// PCM byte arrives. For phrases the dialog repeats often (welcome line, ASR
// fallback, "请稍等" hold tones) we already know the audio bytes — there is
// no reason to re-pay that round-trip every time.
//
// CachingService wraps any TTS Service with a content-addressable PCM cache.
// On a hit it replays the cached PCM via the same `onPCMChunk` callback the
// underlying service would have used, so the downstream Pipeline cannot tell
// the difference. On a miss it delegates and tees the PCM into the cache so
// the *second* utterance of any phrase is instant.
//
// Process-singleton vs per-call
//
// The Cache is process-level (DefaultCache) so warm entries survive across
// SIP calls. The CachingService instance, by contrast, is per-call because
// the upstream synthesizer profile (sample rate, voice type, speed) varies
// per leg. The cache key encodes both the voice profile and the text, so
// two calls with different sample rates do not collide.
//
// Memory bounds
//
// PCM at 16 kHz mono 16-bit ≈ 32 KB/sec. A 5-second sentence ≈ 160 KB. The
// default cache holds 64 entries / 16 MB total — about ~100 short sentences,
// well within reach for a server. Eviction is naive LRU (insertion-order
// slice + map). When the cap is hit, the oldest entry is dropped.
//
// Thread-safety
//
// All Cache and CachingService methods are safe for concurrent use. Two
// in-flight Synthesize calls for the *same* missing key will both call the
// underlying service (we do not collapse them via singleflight) — the
// duplicate work is acceptable because Prewarm is the recommended path for
// known-hot phrases and one-off duplicates are rare.

// Cache holds rendered PCM keyed by an opaque string. Safe for concurrent
// access. Use the package-level DefaultCache to share entries across calls
// in the same process.
type Cache struct {
	mu      sync.RWMutex
	entries map[string][]byte
	order   []string // LRU: front = oldest

	maxEntries int
	maxBytes   int
	curBytes   int
}

// NewCache returns an empty cache with the given caps. maxEntries and
// maxBytes both apply; whichever is hit first triggers eviction. Pass 0 to
// disable that particular cap.
func NewCache(maxEntries, maxBytes int) *Cache {
	if maxEntries < 0 {
		maxEntries = 0
	}
	if maxBytes < 0 {
		maxBytes = 0
	}
	return &Cache{
		entries:    make(map[string][]byte, maxEntries),
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
	}
}

// Get returns (pcm, true) on a hit. The returned slice is owned by the cache;
// callers must not mutate it (use it as a read-only source).
func (c *Cache) Get(key string) ([]byte, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	pcm, ok := c.entries[key]
	return pcm, ok
}

// Put stores a copy of pcm under key. Evicts oldest entries if either cap is
// exceeded.
func (c *Cache) Put(key string, pcm []byte) {
	if c == nil || key == "" || len(pcm) == 0 {
		return
	}
	cp := make([]byte, len(pcm))
	copy(cp, pcm)
	c.mu.Lock()
	defer c.mu.Unlock()
	if old, ok := c.entries[key]; ok {
		c.curBytes -= len(old)
		c.entries[key] = cp
		c.curBytes += len(cp)
		c.touch(key)
		c.evictLocked()
		return
	}
	c.entries[key] = cp
	c.order = append(c.order, key)
	c.curBytes += len(cp)
	c.evictLocked()
}

// Len returns the number of cached entries.
func (c *Cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Bytes returns the current total PCM bytes held.
func (c *Cache) Bytes() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.curBytes
}

// touch moves key to the end of order (newest). Caller holds mu.
func (c *Cache) touch(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}

// evictLocked drops oldest entries until both caps are satisfied.
func (c *Cache) evictLocked() {
	for {
		overEntries := c.maxEntries > 0 && len(c.order) > c.maxEntries
		overBytes := c.maxBytes > 0 && c.curBytes > c.maxBytes
		if !overEntries && !overBytes {
			return
		}
		if len(c.order) == 0 {
			return
		}
		oldest := c.order[0]
		c.order = c.order[1:]
		if pcm, ok := c.entries[oldest]; ok {
			c.curBytes -= len(pcm)
			delete(c.entries, oldest)
		}
	}
}

// DefaultCache is the process-wide TTS PCM cache. Wire CachingService to it
// (the default) so phrases warmed during one call serve the next call too.
//
//   - 128 entries / 32 MiB defaults out of the box
//   - Replace before any service uses it if your deployment has different
//     memory budgets (e.g. NewCache(512, 128<<20)).
var DefaultCache = NewCache(128, 32<<20)

// CacheConfig configures a CachingService.
type CacheConfig struct {
	// Cache to use. nil → DefaultCache.
	Cache *Cache

	// VoiceKey is a stable string identifying the voice profile (vendor +
	// voice type + sample rate + speed). It is mixed into every cache key
	// so two profiles cannot collide on the same text. Required.
	VoiceKey string

	// MaxRunes drops cache writes for texts longer than this. 0 = no limit.
	// Useful to avoid caching arbitrarily-long LLM replies that will never
	// repeat. Reads always check the cache regardless.
	MaxRunes int

	// ChunkBytes is the chunk size used when replaying a cached entry to
	// the downstream callback. 0 → emit the whole PCM in one chunk (the
	// Pipeline's framer does the rest). 1024-4096 is a reasonable choice
	// when you want callback-shape parity with live streaming.
	ChunkBytes int
}

// CachingService wraps an inner Service with a process-level PCM cache.
// Hits skip the network; misses delegate and tee the PCM into the cache.
type CachingService struct {
	inner Service
	cfg   CacheConfig
}

// NewCachingService validates cfg and returns a Service whose
// SynthesizeStream is cache-aware.
func NewCachingService(inner Service, cfg CacheConfig) (*CachingService, error) {
	if inner == nil {
		return nil, errors.New("voice/tts: nil inner service")
	}
	if strings.TrimSpace(cfg.VoiceKey) == "" {
		return nil, errors.New("voice/tts: empty VoiceKey")
	}
	if cfg.Cache == nil {
		cfg.Cache = DefaultCache
	}
	return &CachingService{inner: inner, cfg: cfg}, nil
}

// CacheKey returns the canonical key used for `text`. Exposed for callers
// that want to seed a cache from somewhere else (file, KV).
func (c *CachingService) CacheKey(text string) string {
	if c == nil {
		return ""
	}
	return cacheKey(c.cfg.VoiceKey, text)
}

// Cache returns the underlying cache for stats / introspection.
func (c *CachingService) Cache() *Cache {
	if c == nil {
		return nil
	}
	return c.cfg.Cache
}

// SynthesizeStream implements Service. On cache hit it replays cached PCM via
// onPCMChunk synchronously and returns nil. On miss it delegates to the inner
// service, copies the PCM into a buffer, then writes the buffer to the cache
// after the inner call returns successfully.
func (c *CachingService) SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if c == nil || c.inner == nil {
		return errors.New("voice/tts: nil caching service")
	}
	t := strings.TrimSpace(text)
	if t == "" {
		return nil
	}
	key := cacheKey(c.cfg.VoiceKey, t)
	if pcm, ok := c.cfg.Cache.Get(key); ok {
		return replayPCM(ctx, pcm, c.cfg.ChunkBytes, onPCMChunk)
	}

	// Miss: tee inner output into a buffer for caching.
	var collected []byte
	if c.cacheable(t) {
		collected = make([]byte, 0, 32*1024)
	}
	err := c.inner.SynthesizeStream(ctx, t, func(chunk []byte) error {
		if collected != nil && len(chunk) > 0 {
			collected = append(collected, chunk...)
		}
		return onPCMChunk(chunk)
	})
	if err != nil {
		return err
	}
	if collected != nil && len(collected) > 0 && ctx.Err() == nil {
		c.cfg.Cache.Put(key, collected)
	}
	return nil
}

// Prewarm renders each text once and stores the PCM in the cache. Errors
// per-text are returned via the optional onErr callback (nil → silently
// dropped). Texts that are already cached are skipped.
//
// Typical use: at process startup or per-call attach, prime the cache with
// the welcome line and any fallback strings so the first user-visible speak
// has near-zero first-byte latency.
func (c *CachingService) Prewarm(ctx context.Context, texts []string, onErr func(text string, err error)) {
	if c == nil || c.inner == nil || len(texts) == 0 {
		return
	}
	for _, raw := range texts {
		if ctx.Err() != nil {
			return
		}
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		key := cacheKey(c.cfg.VoiceKey, t)
		if _, ok := c.cfg.Cache.Get(key); ok {
			continue
		}
		var buf []byte
		err := c.inner.SynthesizeStream(ctx, t, func(chunk []byte) error {
			buf = append(buf, chunk...)
			return nil
		})
		if err != nil {
			if onErr != nil {
				onErr(t, err)
			}
			continue
		}
		if len(buf) > 0 {
			c.cfg.Cache.Put(key, buf)
		}
	}
}

// cacheable returns whether text is short enough to be worth caching.
func (c *CachingService) cacheable(text string) bool {
	if c == nil {
		return false
	}
	if c.cfg.MaxRunes <= 0 {
		return true
	}
	n := 0
	for range text {
		n++
		if n > c.cfg.MaxRunes {
			return false
		}
	}
	return true
}

// cacheKey returns "<voice>:<sha1(text)>" — short, collision-resistant, and
// easy to compare. SHA-1 is fine here: this is a content-address cache, not
// a security boundary.
func cacheKey(voiceKey, text string) string {
	h := sha1.Sum([]byte(text))
	return voiceKey + ":" + hex.EncodeToString(h[:])
}

// replayPCM emits a cached buffer through onPCMChunk. With chunkBytes <= 0,
// the whole buffer flushes in one call (lowest latency). With chunkBytes > 0,
// the buffer is sliced so the callback shape mirrors live streaming.
func replayPCM(ctx context.Context, pcm []byte, chunkBytes int, onPCMChunk func([]byte) error) error {
	if onPCMChunk == nil || len(pcm) == 0 {
		return nil
	}
	if chunkBytes <= 0 {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		return onPCMChunk(pcm)
	}
	for i := 0; i < len(pcm); i += chunkBytes {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		end := i + chunkBytes
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := onPCMChunk(pcm[i:end]); err != nil {
			return err
		}
	}
	return nil
}
