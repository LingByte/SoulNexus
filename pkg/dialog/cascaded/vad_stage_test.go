// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// fakeDetector implements BargeInDetector for deterministic tests.
// It fires positive whenever pcm contains any non-zero byte AND
// synthPlaying is true. The "non-zero byte" heuristic is enough to
// distinguish "silence" from "speech" frames without needing to
// craft real RMS-positive samples.
type fakeDetector struct {
	calls atomic.Int32
}

func (d *fakeDetector) CheckBargeIn(pcm []byte, synthPlaying bool) bool {
	d.calls.Add(1)
	if !synthPlaying {
		return false
	}
	for _, b := range pcm {
		if b != 0 {
			return true
		}
	}
	return false
}

// drainOutput pulls every frame out of the stage's output channel
// until it closes (or the test deadline expires). Returns the frames
// observed in order.
func drainOutput(t *testing.T, out <-chan pipeline.Frame, deadline time.Duration) []pipeline.Frame {
	t.Helper()
	var got []pipeline.Frame
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case f, ok := <-out:
			if !ok {
				return got
			}
			got = append(got, f)
		case <-timer.C:
			t.Fatalf("drainOutput: timed out after %v with %d frames", deadline, len(got))
		}
	}
}

// runStage spins up vadStage.Run in a goroutine and returns the
// out-channel + a "wait for Run to return" helper. Tests close `in`
// (or cancel ctx) to end the stage.
func runStage(t *testing.T, s *vadStage, ctx context.Context, in <-chan pipeline.Frame) (<-chan pipeline.Frame, func() error) {
	t.Helper()
	out := make(chan pipeline.Frame, 16)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx, in, out, engine.NopLogger{})
	}()
	wait := func() error { return <-errCh }
	return out, wait
}

// --- Passthrough semantics -----------------------------------------

func TestVADStage_PassthroughWhenNoDetector(t *testing.T) {
	s := newVADStage(nil, func() bool { return true }, func() {}, 0, nil)
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2, 3, 4}}}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "hi"}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d frames, want 2 passthrough", len(got))
	}
	if got[0].Kind != pipeline.KindPCM || got[1].Kind != pipeline.KindTextFinal {
		t.Errorf("passthrough order/kind wrong: %+v", got)
	}
}

func TestVADStage_PassthroughWhenNoPredicate(t *testing.T) {
	det := &fakeDetector{}
	s := newVADStage(det, nil, func() {}, 0, nil)
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 1, 1, 1}}}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Kind != pipeline.KindPCM {
		t.Errorf("expected single PCM passthrough, got %+v", got)
	}
	if det.calls.Load() != 0 {
		t.Errorf("detector should not be called without a predicate; calls=%d", det.calls.Load())
	}
}

// --- TTS-gated barge-in --------------------------------------------

func TestVADStage_NoFireWhenTTSNotPlaying(t *testing.T) {
	det := &fakeDetector{}
	var hits atomic.Int32
	s := newVADStage(det, func() bool { return false }, func() { hits.Add(1) }, 0, nil)
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 1, 1, 1}}}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	if hits.Load() != 0 {
		t.Errorf("onBargeIn fired with tts not playing; hits=%d", hits.Load())
	}
	if len(got) != 1 || got[0].Kind != pipeline.KindPCM {
		t.Errorf("PCM should still pass through, got %+v", got)
	}
}

func TestVADStage_FiresOnSpeechDuringTTS(t *testing.T) {
	det := &fakeDetector{}
	var hits atomic.Int32
	s := newVADStage(det, func() bool { return true }, func() { hits.Add(1) }, 0, nil)

	in := make(chan pipeline.Frame, 4)
	// Non-zero PCM during TTS → barge-in.
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{0, 1, 0, 1}}}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}
	if hits.Load() != 1 {
		t.Errorf("onBargeIn hits = %d, want 1", hits.Load())
	}
	// Expect: KindBargeIn control + KindPCM passthrough, in that order.
	if len(got) != 2 {
		t.Fatalf("got %d frames, want KindBargeIn + KindPCM", len(got))
	}
	if got[0].Kind != pipeline.KindBargeIn {
		t.Errorf("frame[0].Kind = %v, want KindBargeIn", got[0].Kind)
	}
	if got[1].Kind != pipeline.KindPCM {
		t.Errorf("frame[1].Kind = %v, want KindPCM passthrough", got[1].Kind)
	}
}

func TestVADStage_DebouncesBargeInDuringSameTTSEpisode(t *testing.T) {
	det := &fakeDetector{}
	var hits atomic.Int32
	s := newVADStage(det, func() bool { return true }, func() { hits.Add(1) }, 0, nil)
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{0, 1}}}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{0, 2}}}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	if hits.Load() != 1 {
		t.Errorf("onBargeIn hits = %d, want 1 (debounced)", hits.Load())
	}
	bargeIn := 0
	for _, f := range got {
		if f.Kind == pipeline.KindBargeIn {
			bargeIn++
		}
	}
	if bargeIn != 1 {
		t.Errorf("KindBargeIn count = %d, want 1", bargeIn)
	}
}

func TestVADStage_SilentPCMDuringTTSDoesNotFire(t *testing.T) {
	det := &fakeDetector{}
	var hits atomic.Int32
	s := newVADStage(det, func() bool { return true }, func() { hits.Add(1) }, 0, nil)
	in := make(chan pipeline.Frame, 2)
	// All-zero PCM → fakeDetector returns false.
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{0, 0, 0, 0}}}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	if hits.Load() != 0 {
		t.Errorf("silent PCM should not fire barge-in; hits=%d", hits.Load())
	}
	if len(got) != 1 || got[0].Kind != pipeline.KindPCM {
		t.Errorf("expected single PCM passthrough, got %+v", got)
	}
}

func TestVADStage_NonPCMFramesBypassDetector(t *testing.T) {
	det := &fakeDetector{}
	s := newVADStage(det, func() bool { return true }, func() {}, 0, nil)
	in := make(chan pipeline.Frame, 3)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "hello"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "world"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	if det.calls.Load() != 0 {
		t.Errorf("detector called on non-PCM frames; calls=%d", det.calls.Load())
	}
	if len(got) != 3 {
		t.Fatalf("got %d frames, want 3 passthrough", len(got))
	}
}

// --- nil callback safety -------------------------------------------

func TestVADStage_NilOnBargeInCallbackIsSafe(t *testing.T) {
	det := &fakeDetector{}
	// onBargeIn nil — should still emit KindBargeIn frame, no panic.
	s := newVADStage(det, func() bool { return true }, nil, 0, nil)
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 0, 1, 0}}}
	close(in)

	out, wait := runStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d frames, want KindBargeIn + KindPCM", len(got))
	}
}

// --- Lifecycle ------------------------------------------------------

func TestVADStage_ContextCancelStopsStage(t *testing.T) {
	det := &fakeDetector{}
	s := newVADStage(det, func() bool { return false }, func() {}, 0, nil)
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan pipeline.Frame) // never write, never close
	out, wait := runStage(t, s, ctx, in)
	cancel()
	// out must close after ctx cancel (defer close(out) in Run).
	select {
	case _, ok := <-out:
		if ok {
			t.Error("expected closed out chan after ctx cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("out did not close after ctx cancel")
	}
	if err := wait(); err == nil {
		t.Error("Run should return ctx.Err() on cancel, got nil")
	}
}

func TestVADStage_Name(t *testing.T) {
	s := newVADStage(nil, nil, nil, 0, nil)
	if got := s.Name(); got != "vad" {
		t.Errorf("Name() = %q, want %q", got, "vad")
	}
}

// --- Engine integration --------------------------------------------

// TestEngine_VADStagePrependedWhenDetectorSet asserts that
// WithVADDetector / WithBargeInHandler are accepted by New and that
// Attach succeeds with the resulting Engine.
//
// We deliberately do NOT assert detector.calls > 0: the engine starts
// the call with ttsPlaying=false (no AI audio out yet), and the VAD
// stage short-circuits its detector call when the predicate reports
// false (Go's && short-circuit). To actually reach the detector via
// this integration test we'd need the engine's output stream to emit
// KindPCM first — and the current TTS stub never does. Stage-level
// tests above already cover the detector code path deterministically.
//
// This test's value: catch a future regression where the Option
// closures fail to install (e.g. typo in the field assignment) or
// where prepending vadStage breaks pipeline.New validation.
func TestEngine_VADStagePrependedWhenDetectorSet(t *testing.T) {
	det := &fakeDetector{}
	var bargeHits atomic.Int32
	e := New(
		engine.Config{Mode: engine.ModeCascaded, CallID: "c-vad", TenantID: "t1"},
		WithVADDetector(det),
		WithBargeInHandler(func() { bargeHits.Add(1) }),
	)
	if e.vadDetector == nil {
		t.Fatal("WithVADDetector did not install detector")
	}
	if e.bargeInHandler == nil {
		t.Fatal("WithBargeInHandler did not install handler")
	}
	port := newFakePort()
	detach, err := e.Attach(context.Background(), port, nil)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	// Push one silent frame and confirm: (a) Attach completes, (b)
	// no spurious barge-in fires while ttsPlaying is still false.
	port.in <- engine.PCMFrame{Data: []byte{0, 0, 0, 0}, SampleRate: 16000}
	time.Sleep(50 * time.Millisecond)
	if bargeHits.Load() != 0 {
		t.Errorf("unexpected barge-in fire pre-tts; hits=%d", bargeHits.Load())
	}
	close(port.in)
	_ = detach(context.Background())
}

// TestEngine_NoVADWhenDetectorAbsent asserts the default-construction
// path (no Options) still runs the 3-stage chain unchanged. Guards
// against accidentally always-prepending vadStage.
func TestEngine_NoVADWhenDetectorAbsent(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c-novad"})
	if e.vadDetector != nil {
		t.Error("default engine should have no detector")
	}
	port := newFakePort()
	detach, err := e.Attach(context.Background(), port, nil)
	if err != nil {
		t.Fatalf("Attach without VAD options: %v", err)
	}
	close(port.in)
	_ = detach(context.Background())
}

// TestEngine_NilOptionsTolerated ensures variadic Options can include
// nil entries (e.g. conditional construction:
// `New(cfg, opts...)` where opts may be a nil-padded slice). Without
// the nil-guard in New, this would panic.
func TestEngine_NilOptionsTolerated(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("New panicked on nil Option: %v", r)
		}
	}()
	_ = New(engine.Config{Mode: engine.ModeCascaded, CallID: "c-nil"}, nil, nil)
}
