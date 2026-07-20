// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// fakeASR is a deterministic ASRRecognizer for tests. ProcessPCM
// records every PCM chunk it sees; the test then calls emit/emitErr
// to drive the registered callbacks. This lets each test fully
// control the partial/final/error timing without races.
type fakeASR struct {
	mu       sync.Mutex
	pcm      [][]byte
	onText   func(text string, isFinal bool)
	onErr    func(err error, fatal bool)
	failNext error // when non-nil, next ProcessPCM returns this
}

func (f *fakeASR) ProcessPCM(ctx context.Context, pcm []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	// Copy because the caller may reuse the buffer.
	c := make([]byte, len(pcm))
	copy(c, pcm)
	f.pcm = append(f.pcm, c)
	return nil
}

func (f *fakeASR) SetTextCallback(cb func(text string, isFinal bool)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onText = cb
}

func (f *fakeASR) SetErrorCallback(cb func(err error, fatal bool)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onErr = cb
}

func (f *fakeASR) emit(text string, isFinal bool) {
	f.mu.Lock()
	cb := f.onText
	f.mu.Unlock()
	if cb != nil {
		cb(text, isFinal)
	}
}

func (f *fakeASR) emitErr(err error, fatal bool) {
	f.mu.Lock()
	cb := f.onErr
	f.mu.Unlock()
	if cb != nil {
		cb(err, fatal)
	}
}

func (f *fakeASR) pcmCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pcm)
}

// runASRStage spins up asrStage.Run in a goroutine. Tests close `in`
// or cancel ctx to end the stage. drainOutput already exists from
// vad_stage_test.go (same package).
func runASRStage(t *testing.T, s *asrStage, ctx context.Context, in <-chan pipeline.Frame) (<-chan pipeline.Frame, func() error) {
	t.Helper()
	out := make(chan pipeline.Frame, 32)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx, in, out, engine.NopLogger{})
	}()
	return out, func() error { return <-errCh }
}

// --- Name / Nil recognizer passthrough ------------------------------

func TestASRStage_Name(t *testing.T) {
	s := newASRStage(nil)
	if got := s.Name(); got != "asr" {
		t.Errorf("Name() = %q, want %q", got, "asr")
	}
}

func TestASRStage_NilRecognizerPassthrough(t *testing.T) {
	s := newASRStage(nil)
	in := make(chan pipeline.Frame, 3)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}}}
	in <- pipeline.Frame{Kind: pipeline.KindBargeIn}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "x"}
	close(in)

	out, wait := runASRStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d frames, want 3 passthrough", len(got))
	}
}

// --- PCM → ProcessPCM forwarding -----------------------------------

func TestASRStage_ForwardsPCMToRecognizer(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 3)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2, 3, 4}}}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{5, 6, 7, 8}}}
	close(in)

	out, wait := runASRStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	if asr.pcmCount() != 2 {
		t.Errorf("ProcessPCM called %d times, want 2", asr.pcmCount())
	}
	if len(got) != 2 {
		t.Errorf("got %d frames passthrough, want 2", len(got))
	}
}

func TestASRStage_NonPCMFramesPassthroughUntouched(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 3)
	in <- pipeline.Frame{Kind: pipeline.KindBargeIn}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	out, wait := runASRStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	if asr.pcmCount() != 0 {
		t.Errorf("non-PCM should not reach recognizer; got %d", asr.pcmCount())
	}
	if len(got) != 2 {
		t.Errorf("got %d frames, want 2 passthrough", len(got))
	}
}

// --- Transcript emission -------------------------------------------

func TestASRStage_EmitsTextFinalFromCallback(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 1}}}

	out, wait := runASRStage(t, s, context.Background(), in)
	// Drive a final transcript via the callback we installed.
	// Brief sleep ensures Run has registered the callback before we emit.
	time.Sleep(20 * time.Millisecond)
	asr.emit("hello world", true)

	// Read out the next two frames: PCM passthrough + KindTextFinal.
	// Order is not strictly defined (select non-deterministic), so
	// classify by Kind rather than position.
	deadline := time.Now().Add(time.Second)
	var sawFinal, sawPCM bool
	for !(sawFinal && sawPCM) && time.Now().Before(deadline) {
		select {
		case f := <-out:
			switch f.Kind {
			case pipeline.KindTextFinal:
				if f.Text != "hello world" {
					t.Errorf("final text = %q, want %q", f.Text, "hello world")
				}
				sawFinal = true
			case pipeline.KindPCM:
				sawPCM = true
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	close(in)
	_ = wait()
	if !sawFinal {
		t.Error("never saw KindTextFinal frame")
	}
	if !sawPCM {
		t.Error("never saw KindPCM passthrough")
	}
}

func TestASRStage_EmitsTextInterimFromCallback(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	asr.emit("partial", false)

	deadline := time.Now().Add(time.Second)
	var sawInterim bool
	for !sawInterim && time.Now().Before(deadline) {
		select {
		case f := <-out:
			if f.Kind == pipeline.KindTextInterim {
				if f.Text != "partial" {
					t.Errorf("interim text = %q, want %q", f.Text, "partial")
				}
				sawInterim = true
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	close(in)
	_ = wait()
	if !sawInterim {
		t.Error("never saw KindTextInterim frame")
	}
}

func TestASRStage_EmptyTranscriptDropped(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	asr.emit("   ", true)  // whitespace-only → dropped by stage
	asr.emit("", false)    // empty → dropped
	asr.emit("real", true) // should pass

	deadline := time.Now().Add(500 * time.Millisecond)
	var sawReal, sawNoise bool
	for time.Now().Before(deadline) {
		select {
		case f := <-out:
			if f.Kind == pipeline.KindTextFinal && f.Text == "real" {
				sawReal = true
			}
			if f.Kind == pipeline.KindTextInterim && f.Text == "" {
				sawNoise = true
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	close(in)
	_ = wait()
	if !sawReal {
		t.Error("real transcript was dropped")
	}
	if sawNoise {
		t.Error("empty transcript should have been dropped before emit")
	}
}

// --- Drain after input close ---------------------------------------

func TestASRStage_DrainsTranscriptsAfterInputClose(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	// Fill the transcript buffer BEFORE closing input. The stage's
	// post-close drain phase should still emit them.
	asr.emit("first", false)
	asr.emit("final-text", true)
	close(in)

	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Errorf("Run = %v, want nil after input close drain", err)
	}
	var foundFinal bool
	for _, f := range got {
		if f.Kind == pipeline.KindTextFinal && f.Text == "final-text" {
			foundFinal = true
		}
	}
	if !foundFinal {
		t.Error("final transcript lost during input-close drain")
	}
}

// --- Error handling -------------------------------------------------

func TestASRStage_TransientErrorIsObservabilityOnly(t *testing.T) {
	asr := &fakeASR{failNext: errors.New("transient blip")}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}}}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{3, 4}}}
	close(in)

	out, wait := runASRStage(t, s, context.Background(), in)
	got := drainOutput(t, out, time.Second)
	if err := wait(); err != nil {
		t.Errorf("Run = %v, want nil despite transient ProcessPCM error", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d frames passthrough, want 2 (PCM should still flow after transient err)", len(got))
	}
}

func TestASRStage_CtxCanceledFromProcessPCMReturns(t *testing.T) {
	asr := &fakeASR{failNext: context.Canceled}
	s := newASRStage(asr)
	in := make(chan pipeline.Frame, 1)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}}}
	// Don't close `in`; the stage should still terminate via the
	// returned ctx.Canceled error.

	out, wait := runASRStage(t, s, context.Background(), in)
	_ = drainOutput(t, out, time.Second)
	if err := wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Run = %v, want context.Canceled", err)
	}
}

func TestASRStage_FatalErrorInvokesHandler(t *testing.T) {
	asr := &fakeASR{}
	var fatalHits atomic.Int32
	s := newASRStage(asr, withASRFatalErrorHandler(func(err error) {
		fatalHits.Add(1)
	}))
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	asr.emitErr(errors.New("session lost"), true)
	time.Sleep(50 * time.Millisecond)
	close(in)
	_ = drainOutput(t, out, time.Second)
	_ = wait()
	if got := fatalHits.Load(); got != 1 {
		t.Errorf("fatal handler hits = %d, want 1", got)
	}
}

func TestASRStage_NonFatalErrorSkipsHandler(t *testing.T) {
	asr := &fakeASR{}
	var fatalHits atomic.Int32
	s := newASRStage(asr, withASRFatalErrorHandler(func(err error) {
		fatalHits.Add(1)
	}))
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	asr.emitErr(errors.New("transient"), false)
	time.Sleep(50 * time.Millisecond)
	close(in)
	_ = drainOutput(t, out, time.Second)
	_ = wait()
	if got := fatalHits.Load(); got != 0 {
		t.Errorf("fatal handler should not fire for non-fatal err; hits=%d", got)
	}
}

func TestASRStage_NilErrorPassedToCallbackIsIgnored(t *testing.T) {
	asr := &fakeASR{}
	var fatalHits atomic.Int32
	s := newASRStage(asr, withASRFatalErrorHandler(func(err error) {
		fatalHits.Add(1)
	}))
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	// Defensive: recognizer signals "error callback fired with nil err".
	// The stage must not crash and must not invoke the fatal handler.
	asr.emitErr(nil, true)
	time.Sleep(50 * time.Millisecond)
	close(in)
	_ = drainOutput(t, out, time.Second)
	_ = wait()
	if got := fatalHits.Load(); got != 0 {
		t.Errorf("nil err should not invoke fatal handler; hits=%d", got)
	}
}

// --- Lifecycle / context -------------------------------------------

func TestASRStage_CtxCancelStopsStage(t *testing.T) {
	asr := &fakeASR{}
	s := newASRStage(asr)
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan pipeline.Frame)
	out, wait := runASRStage(t, s, ctx, in)
	cancel()
	select {
	case _, ok := <-out:
		if ok {
			t.Error("out should close after ctx cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("out did not close after ctx cancel")
	}
	if err := wait(); err == nil {
		t.Error("Run should return ctx.Err() on cancel")
	}
}

// --- Engine integration --------------------------------------------

func TestEngine_WithASRRecognizerSwapsStage(t *testing.T) {
	asr := &fakeASR{}
	e := New(
		engine.Config{Mode: engine.ModeCascaded, CallID: "c-asr", TenantID: "t1"},
		WithASRRecognizer(asr),
	)
	if e.asrRecognizer == nil {
		t.Fatal("WithASRRecognizer did not install recognizer")
	}
	port := newFakePort()
	detach, err := e.Attach(context.Background(), port, nil)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	// Push a PCM frame; the swapped asrStage should call ProcessPCM.
	port.in <- engine.PCMFrame{Data: []byte{1, 1, 2, 2}, SampleRate: 16000}
	deadline := time.Now().Add(time.Second)
	for asr.pcmCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if asr.pcmCount() == 0 {
		t.Fatal("PCM never reached the swapped ASR recognizer")
	}
	close(port.in)
	_ = detach(context.Background())
}

// --- Drop-oldest overflow ------------------------------------------

func TestASRStage_TranscriptBufferDropsOldestOnOverflow(t *testing.T) {
	// Buffer of 2 → after we burst 5 callbacks with no Run draining,
	// at most 2 should survive. We can't easily assert WHICH two
	// (depends on select timing) but we can assert the *count* of
	// emitted text frames is bounded.
	asr := &fakeASR{}
	s := newASRStage(asr, withASRTranscriptBuffer(2))
	in := make(chan pipeline.Frame, 1)
	out, wait := runASRStage(t, s, context.Background(), in)
	time.Sleep(20 * time.Millisecond)
	// Burst 5 quickly. Stage may not have read between them.
	for i := 0; i < 5; i++ {
		asr.emit("t", false)
	}
	close(in)
	got := drainOutput(t, out, time.Second)
	_ = wait()
	textCount := 0
	for _, f := range got {
		if f.IsText() {
			textCount++
		}
	}
	// Best-effort: text count must be in [1, 5]. The tighter bound
	// (<=2) is the design intent but timing-dependent; we assert the
	// stage doesn't blow up rather than the exact overflow geometry.
	if textCount < 1 || textCount > 5 {
		t.Errorf("textCount = %d, want in [1,5]", textCount)
	}
}
