package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// fakeService is a deterministic Service that emits a fixed PCM payload per
// distinct text and counts how many times it was actually called. This lets
// tests assert that cache HITs do not delegate, and MISSes do.
type fakeService struct {
	calls int32
	pcm   map[string][]byte
}

func newFakeService(pcm map[string][]byte) *fakeService {
	return &fakeService{pcm: pcm}
}

func (f *fakeService) SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	atomic.AddInt32(&f.calls, 1)
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	pcm, ok := f.pcm[text]
	if !ok {
		return errors.New("fake: unknown text")
	}
	// Emit in 3 chunks to mimic streaming.
	step := len(pcm) / 3
	if step < 1 {
		step = 1
	}
	for i := 0; i < len(pcm); i += step {
		end := i + step
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := onPCMChunk(pcm[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func collect(svc Service, text string) ([]byte, error) {
	var buf bytes.Buffer
	err := svc.SynthesizeStream(context.Background(), text, func(chunk []byte) error {
		buf.Write(chunk)
		return nil
	})
	return buf.Bytes(), err
}

func TestCache_PutGetEvict(t *testing.T) {
	c := NewCache(2, 0)
	c.Put("a", []byte("1234"))
	c.Put("b", []byte("5678"))
	if c.Len() != 2 {
		t.Fatalf("len=%d, want 2", c.Len())
	}
	// Adding a 3rd evicts oldest (a).
	c.Put("c", []byte("9012"))
	if _, ok := c.Get("a"); ok {
		t.Errorf("a should have been evicted (LRU)")
	}
	if _, ok := c.Get("b"); !ok {
		t.Errorf("b should still be present")
	}
	if _, ok := c.Get("c"); !ok {
		t.Errorf("c should be present")
	}
}

func TestCache_BytesCap(t *testing.T) {
	c := NewCache(0, 6) // 6 byte budget
	c.Put("a", []byte("12"))
	c.Put("b", []byte("34"))
	c.Put("c", []byte("56"))
	// All three fit (2+2+2 = 6).
	if c.Bytes() != 6 || c.Len() != 3 {
		t.Fatalf("len=%d bytes=%d, want 3/6", c.Len(), c.Bytes())
	}
	// Adding 2 more evicts oldest (a) to fit.
	c.Put("d", []byte("78"))
	if _, ok := c.Get("a"); ok {
		t.Errorf("a should have been evicted by byte cap")
	}
}

func TestCachingService_HitSkipsInner(t *testing.T) {
	inner := newFakeService(map[string][]byte{
		"hello": []byte("AUDIO-HELLO-PCM"),
	})
	svc, err := NewCachingService(inner, CacheConfig{
		Cache:    NewCache(0, 0),
		VoiceKey: "test-v1",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	// First call: miss.
	got, err := collect(svc, "hello")
	if err != nil {
		t.Fatalf("first collect: %v", err)
	}
	if !bytes.Equal(got, []byte("AUDIO-HELLO-PCM")) {
		t.Fatalf("first pcm = %q, want %q", got, "AUDIO-HELLO-PCM")
	}
	if calls := atomic.LoadInt32(&inner.calls); calls != 1 {
		t.Fatalf("first call inner calls = %d, want 1", calls)
	}
	// Second call: hit, inner should NOT be invoked.
	got2, err := collect(svc, "hello")
	if err != nil {
		t.Fatalf("second collect: %v", err)
	}
	if !bytes.Equal(got2, []byte("AUDIO-HELLO-PCM")) {
		t.Fatalf("second pcm = %q, want %q", got2, "AUDIO-HELLO-PCM")
	}
	if calls := atomic.LoadInt32(&inner.calls); calls != 1 {
		t.Fatalf("second call inner calls = %d, want 1 (cache hit must not delegate)", calls)
	}
}

func TestCachingService_VoiceKeyIsolation(t *testing.T) {
	pcm := map[string][]byte{"hi": []byte("XYZ")}
	cache := NewCache(0, 0)
	svcA, _ := NewCachingService(newFakeService(pcm), CacheConfig{Cache: cache, VoiceKey: "voice-A"})
	svcB, _ := NewCachingService(newFakeService(pcm), CacheConfig{Cache: cache, VoiceKey: "voice-B"})
	// Render via A first.
	if _, err := collect(svcA, "hi"); err != nil {
		t.Fatalf("A: %v", err)
	}
	// B should be a miss because the key is different.
	innerB := svcB.inner.(*fakeService)
	if _, err := collect(svcB, "hi"); err != nil {
		t.Fatalf("B: %v", err)
	}
	if calls := atomic.LoadInt32(&innerB.calls); calls != 1 {
		t.Fatalf("inner B calls = %d, want 1 (different voice key must not share cache entry)", calls)
	}
}

func TestCachingService_Prewarm(t *testing.T) {
	inner := newFakeService(map[string][]byte{
		"welcome":  []byte("AAAA"),
		"fallback": []byte("BBBB"),
	})
	svc, _ := NewCachingService(inner, CacheConfig{Cache: NewCache(0, 0), VoiceKey: "v"})
	svc.Prewarm(context.Background(), []string{"welcome", "fallback"}, nil)
	// Both should now be cache hits — calls should NOT increase on Speak.
	prewarmCalls := atomic.LoadInt32(&inner.calls)
	if prewarmCalls != 2 {
		t.Fatalf("prewarm inner calls = %d, want 2", prewarmCalls)
	}
	if _, err := collect(svc, "welcome"); err != nil {
		t.Fatalf("welcome: %v", err)
	}
	if _, err := collect(svc, "fallback"); err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if got := atomic.LoadInt32(&inner.calls); got != prewarmCalls {
		t.Errorf("after warm reads inner calls = %d, want unchanged %d", got, prewarmCalls)
	}
}

func TestCachingService_MaxRunesSkipsLargeWrites(t *testing.T) {
	long := "这是一个非常非常长的句子，绝对不应该被缓存因为它是 LLM 一次性生成的独特回答而不是常用应答语句"
	inner := newFakeService(map[string][]byte{long: []byte("LONG-PCM-DATA-XXXXXXXXX")})
	cache := NewCache(0, 0)
	svc, _ := NewCachingService(inner, CacheConfig{
		Cache:    cache,
		VoiceKey: "v",
		MaxRunes: 16,
	})
	if _, err := collect(svc, long); err != nil {
		t.Fatalf("collect: %v", err)
	}
	if cache.Len() != 0 {
		t.Errorf("expected cache to skip long text (MaxRunes=16), got %d entries", cache.Len())
	}
}

func TestReplayPCM_ChunkShape(t *testing.T) {
	pcm := []byte("0123456789")
	var got [][]byte
	if err := replayPCM(context.Background(), pcm, 3, func(b []byte) error {
		// Copy because callback owns the slice only for the call duration.
		got = append(got, append([]byte(nil), b...))
		return nil
	}); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("chunks = %d, want 4 (3+3+3+1)", len(got))
	}
	if !bytes.Equal(got[3], []byte("9")) {
		t.Errorf("last chunk = %q, want %q", got[3], "9")
	}
}
