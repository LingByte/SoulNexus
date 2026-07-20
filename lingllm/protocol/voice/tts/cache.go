package tts

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

// Cache holds rendered PCM keyed by an opaque string. Safe for concurrent
// access. Use DefaultCache to share entries across calls in one process.
type Cache struct {
	mu      sync.RWMutex
	entries map[string][]byte
	order   []string // LRU: front = oldest

	maxEntries int
	maxBytes   int
	curBytes   int
}

// NewCache returns an empty cache with the given caps. maxEntries and maxBytes
// both apply; whichever is hit first triggers eviction. Pass 0 to disable that cap.
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

// Get returns (pcm, true) on a hit. The returned slice is owned by the cache.
func (c *Cache) Get(key string) ([]byte, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	pcm, ok := c.entries[key]
	return pcm, ok
}

// Put stores a copy of pcm under key, evicting oldest entries when over cap.
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

func (c *Cache) touch(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}

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

// DefaultCache is the process-wide TTS PCM cache (128 entries / 32 MiB).
var DefaultCache = NewCache(128, 32<<20)

// CacheConfig configures a CachingTTSService.
type CacheConfig struct {
	// Cache to use. nil → DefaultCache.
	Cache *Cache
	// VoiceKey identifies vendor + voice + sample rate + speed. Required.
	VoiceKey string
	// MaxRunes skips cache writes for longer texts. 0 = no limit.
	MaxRunes int
	// ChunkBytes controls replay chunk size. 0 → one shot.
	ChunkBytes int
}

// CachingTTSService wraps TTSService with a PCM cache.
type CachingTTSService struct {
	inner TTSService
	cfg   CacheConfig
}

// NewCachingTTSService validates cfg and returns a cache-aware TTSService.
func NewCachingTTSService(inner TTSService, cfg CacheConfig) (*CachingTTSService, error) {
	if inner == nil {
		return nil, errors.New("tts: nil inner service")
	}
	if strings.TrimSpace(cfg.VoiceKey) == "" {
		return nil, errors.New("tts: empty VoiceKey")
	}
	if cfg.Cache == nil {
		cfg.Cache = DefaultCache
	}
	return &CachingTTSService{inner: inner, cfg: cfg}, nil
}

// CacheKey returns the canonical key for text.
func (c *CachingTTSService) CacheKey(text string) string {
	if c == nil {
		return ""
	}
	return cacheKey(c.cfg.VoiceKey, text)
}

// Cache returns the underlying cache.
func (c *CachingTTSService) Cache() *Cache {
	if c == nil {
		return nil
	}
	return c.cfg.Cache
}

// Synthesize implements TTSService.
func (c *CachingTTSService) Synthesize(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if c == nil || c.inner == nil {
		return errors.New("tts: nil caching service")
	}
	t := strings.TrimSpace(text)
	if t == "" {
		return nil
	}
	key := cacheKey(c.cfg.VoiceKey, t)
	if pcm, ok := c.cfg.Cache.Get(key); ok {
		return replayPCM(ctx, pcm, c.cfg.ChunkBytes, onPCMChunk)
	}

	var collected []byte
	if c.cacheable(t) {
		collected = make([]byte, 0, 32*1024)
	}
	err := c.inner.Synthesize(ctx, t, func(chunk []byte) error {
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

// Prewarm renders texts once and stores PCM. Skips already-cached entries.
func (c *CachingTTSService) Prewarm(ctx context.Context, texts []string, onErr func(text string, err error)) {
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
		err := c.inner.Synthesize(ctx, t, func(chunk []byte) error {
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

func (c *CachingTTSService) cacheable(text string) bool {
	if c == nil || c.cfg.MaxRunes <= 0 {
		return c != nil
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

func cacheKey(voiceKey, text string) string {
	h := sha1.Sum([]byte(text))
	return voiceKey + ":" + hex.EncodeToString(h[:])
}

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
