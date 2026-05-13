// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package recorder

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"testing"
	"time"
)

// memStore is a minimal stores.Store impl backed by an in-process map so
// the test can inspect the WAV bytes without touching disk. We only
// implement what the recorder actually calls (Write); the rest of the
// interface is satisfied with no-op stubs.
type memStore struct{ m map[string][]byte }

func newMemStore() *memStore { return &memStore{m: map[string][]byte{}} }

func (s *memStore) Write(bucket, key string, body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.m[bucket+"/"+key] = b
	return nil
}
func (s *memStore) Read(_, _ string) (io.ReadCloser, int64, error) { return nil, 0, nil }
func (s *memStore) Exists(_, _ string) (bool, error)               { return false, nil }
func (s *memStore) Delete(bucket, key string) error {
	delete(s.m, bucket+"/"+key)
	return nil
}
func (s *memStore) PublicURL(_, _ string) string                   { return "" }

func TestRecorder_FlushProducesValidStereoWAV(t *testing.T) {
	store := newMemStore()
	r := New(Config{
		CallID:     "call-test-1",
		Bucket:     "test-recordings",
		SampleRate: 16000,
		Transport:  "test",
		Store:      store,
	})
	if r == nil {
		t.Fatal("New returned nil")
	}

	// Write 100 ms of caller PCM (3200 bytes = 1600 samples @ 16 kHz).
	caller := make([]byte, 3200)
	r.WriteCaller(caller)
	// Stagger AI write by 30 ms so wall-clock alignment puts a small
	// silence at the start of the right channel.
	time.Sleep(30 * time.Millisecond)
	ai := make([]byte, 3200)
	for i := range ai {
		ai[i] = byte(i % 256) // non-zero so we can detect it in the WAV
	}
	r.WriteAI(ai)

	info, ok := r.Flush(context.Background())
	if !ok {
		t.Fatal("Flush returned ok=false")
	}
	if info.Format != "wav" || info.Channels != 2 || info.SampleRate != 16000 {
		t.Fatalf("info: %+v", info)
	}
	if info.Bytes < 44 {
		t.Fatalf("wav too small: %d", info.Bytes)
	}
	stored := store.m[info.URL]
	if len(stored) != info.Bytes {
		t.Fatalf("stored %d bytes, info says %d", len(stored), info.Bytes)
	}
	// Validate RIFF/WAVE header.
	if string(stored[0:4]) != "RIFF" || string(stored[8:12]) != "WAVE" {
		t.Fatalf("not a WAVE file: %q ... %q", stored[0:4], stored[8:12])
	}
	// fmt chunk: PCM=1, channels=2, sample rate=16000, bits=16.
	channels := binary.LittleEndian.Uint16(stored[22:24])
	sr := binary.LittleEndian.Uint32(stored[24:28])
	bits := binary.LittleEndian.Uint16(stored[34:36])
	if channels != 2 || sr != 16000 || bits != 16 {
		t.Fatalf("fmt: ch=%d sr=%d bits=%d", channels, sr, bits)
	}
	// Find the data chunk and confirm it contains the AI bytes interleaved
	// in the right channel. Header is 44 bytes for canonical PCM WAV.
	data := stored[44:]
	if !bytes.Contains(data, []byte{0x01, 0x02, 0x03, 0x04}) {
		// Right-channel bytes are interleaved every 4 bytes; samples 1..N
		// of the AI track produce the pattern 1,0,2,0,3,0,4,0... in the
		// right pair after the alignment offset, but a coarse contains
		// check on a recognisable AI byte (0x10) in the right slot is
		// enough: at least one byte from the AI buffer must appear.
		// Use a loose substring check.
		var found bool
		for i := 2; i+1 < len(data); i += 4 {
			if data[i] != 0 || data[i+1] != 0 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("right channel appears to be all zero (AI not interleaved)")
		}
	}
}

func TestRecorder_FlushIdempotent(t *testing.T) {
	store := newMemStore()
	r := New(Config{CallID: "c1", SampleRate: 16000, Store: store})
	r.WriteCaller(make([]byte, 320))
	if _, ok := r.Flush(context.Background()); !ok {
		t.Fatal("first flush failed")
	}
	if _, ok := r.Flush(context.Background()); ok {
		t.Fatal("second flush should be a no-op")
	}
}

func TestRecorder_NilSafeMethods(t *testing.T) {
	var r *Recorder
	r.WriteCaller([]byte{1, 2})
	r.WriteAI([]byte{3, 4})
	if _, ok := r.Flush(context.Background()); ok {
		t.Fatal("nil receiver Flush should return ok=false")
	}
}

func TestRecorder_RollingChunkUpload(t *testing.T) {
	store := newMemStore()
	r := New(Config{
		CallID:        "chunked-call",
		Bucket:        "test",
		SampleRate:    16000,
		Store:         store,
		ChunkInterval: 50 * time.Millisecond,
	})
	if r == nil {
		t.Fatal("New returned nil")
	}
	// Push a frame, wait for a tick, push another, wait again, then flush.
	r.WriteCaller(make([]byte, 320))
	time.Sleep(80 * time.Millisecond)
	r.WriteAI(make([]byte, 320))
	time.Sleep(80 * time.Millisecond)
	if _, ok := r.Flush(context.Background()); !ok {
		t.Fatal("flush failed")
	}
	// Expect at least one chunk + one final WAV.
	var parts, finals int
	for k := range store.m {
		if bytes.Contains([]byte(k), []byte("-part-")) {
			parts++
		} else if bytes.HasSuffix([]byte(k), []byte(".wav")) {
			finals++
		}
	}
	if parts != 0 {
		t.Fatalf("expected 0 part-* after Flush (chunks should be reclaimed), got %d. keys: %v", parts, keysOf(store.m))
	}
	if finals < 1 {
		t.Fatalf("expected ≥1 final wav, got %d. keys: %v", finals, keysOf(store.m))
	}
}

func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestNew_RejectsBadConfig(t *testing.T) {
	if r := New(Config{SampleRate: 16000}); r != nil {
		t.Fatal("expected nil for empty CallID")
	}
	if r := New(Config{CallID: "x", SampleRate: 0}); r != nil {
		t.Fatal("expected nil for zero SampleRate")
	}
}
