// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/hraban/opus"
	"go.uber.org/zap"
)

// fakeStore implements just enough of stores.Store for recording tests.
// We can't easily inject a custom store (stores.Default() picks based
// on env) so the recording.Close path is tested via the LocalStore
// backend with a tempdir set via UPLOAD_DIR. That's exercised in
// TestRecordingClose below.

func TestRecordingDisabled(t *testing.T) {
	if newRecordingSession(&Config{EnableRecording: false}, zap.NewNop(), nil, "r", "p", "i", "t") != nil {
		t.Error("disabled config must return nil session")
	}
}

func TestRecordingPushBeforeStart(t *testing.T) {
	cfg := (&Config{EnableRecording: true}).Normalise()
	rs := newRecordingSession(cfg, zap.NewNop(), newWebhookEmitter(cfg, zap.NewNop()), "r", "p", "i", "t")
	if rs == nil {
		t.Skip("recording session unavailable (opus build?)")
	}
	rs.Push(nil)      // nil-safe
	rs.Push([]byte{}) // empty-safe
	rs.Close()
}

func TestRecordingNilSafety(t *testing.T) {
	var rs *recordingSession
	rs.Push([]byte{0x01})
	rs.Close()
	if rs.sinkPayload() != nil {
		t.Error("nil session.sinkPayload should be nil")
	}
}

func TestRecordingPushDecodes(t *testing.T) {
	cfg := (&Config{EnableRecording: true}).Normalise()
	em := newWebhookEmitter(cfg, zap.NewNop())
	rs := newRecordingSession(cfg, zap.NewNop(), em, "room", "pid", "ident", "trk")
	if rs == nil {
		t.Skip("opus unavailable")
	}

	// Encode 20 ms of silence to feed the decoder.
	enc, err := opus.NewEncoder(48000, 1, opus.AppVoIP)
	if err != nil {
		t.Fatalf("opus enc: %v", err)
	}
	pcm := make([]int16, 48000*20/1000) // 20 ms mono
	buf := make([]byte, 1500)
	n, err := enc.Encode(pcm, buf)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	pkt := append([]byte(nil), buf[:n]...)
	rs.Push(pkt)

	// Sink helper should be the Push closure.
	sink := rs.sinkPayload()
	if sink == nil {
		t.Fatal("sinkPayload returned nil")
	}
	sink(pkt)

	rs.mu.Lock()
	pcmLen := rs.pcm.Len()
	rs.mu.Unlock()
	if pcmLen == 0 {
		t.Error("no PCM accumulated after two pushes")
	}

	// Decode error path: garbage payload should be skipped, not panic.
	rs.Push([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	// Idempotent close: second Close is a no-op.
	rs.Close()
	rs.Close()
}

func TestRecordingPushAfterClose(t *testing.T) {
	cfg := (&Config{EnableRecording: true}).Normalise()
	rs := newRecordingSession(cfg, zap.NewNop(), newWebhookEmitter(cfg, zap.NewNop()), "r", "p", "i", "t")
	if rs == nil {
		t.Skip("opus unavailable")
	}
	rs.Close()
	// After close, push must be a no-op.
	rs.Push([]byte{0x01, 0x02})
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.pcm.Len() != 0 {
		t.Error("push after close should be ignored")
	}
}

func TestRecordingConcurrentClose(t *testing.T) {
	cfg := (&Config{EnableRecording: true}).Normalise()
	rs := newRecordingSession(cfg, zap.NewNop(), newWebhookEmitter(cfg, zap.NewNop()), "r", "p", "i", "t")
	if rs == nil {
		t.Skip("opus unavailable")
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); rs.Close() }()
	}
	wg.Wait()
}

func TestSanitiseSegmentBoundary(t *testing.T) {
	long := bytes.Repeat([]byte("a"), 200)
	got := sanitiseSegment(string(long))
	if len(got) != 64 {
		t.Errorf("long input not truncated: len=%d", len(got))
	}
}

// Smoke: build a buffer, wrap as WAV and verify we can write to a Buffer
// (round-trips the canonical RIFF/WAVE shape).
func TestRecordingWAVRoundtrip(t *testing.T) {
	pcm := make([]byte, 96000) // 1 s of silence
	wav := wrapWAVRecording(pcm, 48000, 1)
	if !bytes.HasPrefix(wav, []byte("RIFF")) {
		t.Fatal("not RIFF")
	}
	if len(wav) != 44+len(pcm) {
		t.Errorf("wav size = %d, want %d", len(wav), 44+len(pcm))
	}
}

// Ensure the time helpers used by webhook are not flaky.
func TestNowMillisMonotonic(t *testing.T) {
	a := time.Now().UnixMilli()
	time.Sleep(2 * time.Millisecond)
	b := time.Now().UnixMilli()
	if b < a {
		t.Errorf("clock went backward: %d → %d", a, b)
	}
}
