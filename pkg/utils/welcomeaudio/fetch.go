// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package welcomeaudio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var errDecodeWAVCallbackRequired = errors.New("decodeWAV callback is required")

// fetchTimeout is the runtime-side download budget. Welcome audio fires
// once per inbound INVITE (after first cache miss), and any RTP delay
// past ~3 s is audible to the caller as dead air. We keep this tighter
// than validateTimeout because by the time we're here the URL was
// already validated at write time — failures here are mostly transient.
const fetchTimeout = 3 * time.Second

// cacheTTL controls how long a fetched WAV's decoded PCM stays cached
// in-process. Tunable via SetCacheTTL for tests. Default 10 minutes
// balances "operator can hot-swap a CDN file" vs "don't re-download
// every call". A 0 TTL disables expiry — the entry lives until process
// restart (use only in tests).
var (
	cacheMu  sync.RWMutex
	cacheTTL = 10 * time.Minute
)

// cacheEntry holds the decoded PCM at a specific bridge sample rate
// alongside the wall-clock the entry was inserted. The decode rate is
// part of the cache key (URL + sampleRate) because the same WAV may
// be requested at 8k for one DID and 16k for another.
type cacheEntry struct {
	pcm    []byte
	loaded time.Time
}

// cache is a process-local map keyed by "<url>|<sampleRate>". sync.Map
// is appropriate here because (a) reads vastly outnumber writes after
// warmup, (b) we never delete entries during a hit (only TTL evict on
// next miss), (c) we do not need ordered iteration.
var cache sync.Map // map[string]*cacheEntry

// SetCacheTTL overrides the default cache TTL. Intended for tests; in
// production callers should leave the default alone. A negative or
// zero value disables expiry.
func SetCacheTTL(d time.Duration) {
	cacheMu.Lock()
	cacheTTL = d
	cacheMu.Unlock()
}

// getCacheTTL is the read-side mirror of SetCacheTTL.
func getCacheTTL() time.Duration {
	cacheMu.RLock()
	d := cacheTTL
	cacheMu.RUnlock()
	return d
}

// httpClientForFetch is a longer-timeout client used at runtime. We
// keep keep-alives enabled (default Transport) so repeated calls to
// the same CDN reuse the connection; fetchTimeout caps each request.
func httpClientForFetch() *http.Client {
	return &http.Client{Timeout: fetchTimeout}
}

// FetchPCM returns mono s16le PCM at sampleRate for the given welcome
// audio URL, using a process-local cache. Prefer decodeWAV =
// common.LoadWAVAsPCM16FromBytes; the callback exists so tests
// can inject stubs without HTTP/file I/O.
//
// Error semantics:
//   - rawURL fails ValidateURL-style scheme/host parsing → ErrUnsupportedScheme
//   - HTTP error / non-2xx / oversize → ErrUnreachable
//   - body fails RIFF/WAVE magic → ErrNotAudio
//   - decoder error → wrapped, NOT ErrNotAudio (file passed magic but
//     fmt/data chunks were unparseable; operator should investigate).
//
// Returned PCM is safe for the caller to read but MUST NOT be mutated:
// the same backing slice is shared across all cache hits.
var welcomeFetchPolicy = wavFetchPolicy{
	maxBytes: MaxBytes,
	validate: ValidateBytes,
}

type wavFetchPolicy struct {
	maxBytes    int
	validate    func([]byte) error
	cachePrefix string
}

func FetchPCM(ctx context.Context, rawURL string, sampleRate int, decodeWAV func(raw []byte, sampleRate int) ([]byte, error)) ([]byte, error) {
	return fetchWAVPCMAt(ctx, rawURL, sampleRate, welcomeFetchPolicy, decodeWAV)
}

func fetchWAVPCMAt(ctx context.Context, rawURL string, sampleRate int, policy wavFetchPolicy, decodeWAV func(raw []byte, sampleRate int) ([]byte, error)) ([]byte, error) {
	if decodeWAV == nil {
		return nil, errDecodeWAVCallbackRequired
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if policy.maxBytes <= 0 {
		policy.maxBytes = MaxBytes
	}
	if policy.validate == nil {
		policy.validate = ValidateBytes
	}
	u, err := parseHTTPURL(rawURL)
	if err != nil {
		return nil, err
	}
	key := policy.cachePrefix + fmt.Sprintf("%s|%d", u.String(), sampleRate)

	if v, ok := cache.Load(key); ok {
		entry := v.(*cacheEntry)
		ttl := getCacheTTL()
		if ttl <= 0 || time.Since(entry.loaded) < ttl {
			return entry.pcm, nil
		}
		cache.Delete(key)
	}

	if body, ok := loadDiskWAV(u.String()); ok {
		if err := policy.validate(body); err != nil {
			return nil, err
		}
		pcm, err := decodeWAV(body, sampleRate)
		if err != nil {
			return nil, fmt.Errorf("decode wav: %w", err)
		}
		cache.Store(key, &cacheEntry{pcm: pcm, loaded: time.Now()})
		return pcm, nil
	}

	dlCtx, cancel := context.WithTimeout(context.Background(), diskFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	resp, err := (&http.Client{Timeout: diskFetchTimeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("%w: GET %d", ErrUnreachable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(policy.maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUnreachable, err)
	}
	if len(body) > policy.maxBytes {
		return nil, fmt.Errorf("%w: body exceeds %d bytes", ErrUnreachable, policy.maxBytes)
	}
	if err := policy.validate(body); err != nil {
		return nil, err
	}
	saveDiskWAV(u.String(), body)
	pcm, err := decodeWAV(body, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("decode wav: %w", err)
	}
	cache.Store(key, &cacheEntry{pcm: pcm, loaded: time.Now()})
	return pcm, nil
}

// PurgeCache drops all cached PCM. Exposed for ops endpoints (future
// work) and tests. No-op when the cache is empty.
func PurgeCache() {
	cache.Range(func(k, _ any) bool {
		cache.Delete(k)
		return true
	})
}
