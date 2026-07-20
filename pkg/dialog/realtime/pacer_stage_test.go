// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// manualTicker drives pacerStage's tick channel from the test
// goroutine so timing assertions are deterministic.
type manualTicker struct {
	c chan time.Time
}

func newManualTicker() *manualTicker { return &manualTicker{c: make(chan time.Time, 32)} }

func (m *manualTicker) tick() { m.c <- time.Now() }

// runPacerStage drives the stage with a manual ticker; returns the
// output channel and a stop helper. Caller drives ticks via mt.tick.
func runPacerStage(t *testing.T, cfg PacerConfig, in <-chan pipeline.Frame) (
	<-chan pipeline.Frame, *manualTicker, func() error,
) {
	t.Helper()
	mt := newManualTicker()
	s := newPacerStage(cfg)
	s.tickerF = func(time.Duration) (<-chan time.Time, func()) {
		return mt.c, func() {}
	}
	out := make(chan pipeline.Frame, 64)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { errCh <- s.Run(ctx, in, out, engine.NopLogger{}) }()
	stop := func() error {
		select {
		case err := <-errCh:
			return err
		case <-time.After(time.Second):
			cancel()
			return <-errCh
		}
	}
	return out, mt, stop
}

func TestPacerStage_Name(t *testing.T) {
	if got := newPacerStage(PacerConfig{}).Name(); got != "pcm_pacer" {
		t.Errorf("Name() = %q, want pcm_pacer", got)
	}
}

func TestPacerStage_NonPCMPassthroughImmediate(t *testing.T) {
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "hi"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	out, _, stop := runPacerStage(t, PacerConfig{}, in)
	frames := drainFrames(out, 2, 200*time.Millisecond)
	_ = stop()

	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2 passthrough", len(frames))
	}
	if frames[0].Kind != pipeline.KindTextFinal || frames[0].Text != "hi" {
		t.Errorf("frame0 = %+v", frames[0])
	}
	if frames[1].Kind != pipeline.KindAITextDone {
		t.Errorf("frame1 = %+v", frames[1])
	}
}

func TestPacerStage_PrebufferGatesFirstFrame(t *testing.T) {
	// 16 kHz @ 20 ms = 640 bytes/frame; default prebuffer = 3
	// frames = 1920 bytes. Anything less than that must NOT
	// produce a frame on tick.
	cfg := PacerConfig{} // defaults
	in := make(chan pipeline.Frame, 4)
	out, mt, stop := runPacerStage(t, cfg, in)

	// Send 1 frame's worth (640 bytes). Prebuffer requires 3 →
	// tick should yield nothing.
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{
		Data: bytes.Repeat([]byte{1}, 640),
	}}
	mt.tick()
	frames := drainFrames(out, 1, 100*time.Millisecond)
	if len(frames) != 0 {
		t.Errorf("got %d frames before prebuffer satisfied, want 0", len(frames))
	}
	// Top up to 1920 bytes (3 frames worth). One tick = one frame
	// emitted (the first one of the buffer).
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{
		Data: bytes.Repeat([]byte{2}, 1280),
	}}
	mt.tick()
	frames = drainFrames(out, 1, 200*time.Millisecond)
	close(in)
	_ = stop()

	if len(frames) != 1 || frames[0].Kind != pipeline.KindPCM {
		t.Fatalf("got %+v, want one KindPCM after prebuffer", frames)
	}
	if got := len(frames[0].PCM.Data); got != 640 {
		t.Errorf("frame size = %d, want 640", got)
	}
}

func TestPacerStage_UnderrunSkipsTick(t *testing.T) {
	cfg := PacerConfig{PrebufferFrames: 1} // start fast
	in := make(chan pipeline.Frame, 4)
	out, mt, stop := runPacerStage(t, cfg, in)

	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{
		Data: bytes.Repeat([]byte{1}, 640),
	}}
	mt.tick() // emits one frame (drainInbound pulls the PCM first)
	mt.tick() // buffer empty, must skip (no zero-fill)
	frames := drainFrames(out, 2, 200*time.Millisecond)
	close(in)
	_ = stop()
	if len(frames) != 1 {
		t.Errorf("got %d frames, want 1 (second tick must skip on underrun)", len(frames))
	}
}

func TestPacerStage_DrainOnClose(t *testing.T) {
	cfg := PacerConfig{PrebufferFrames: 1}
	in := make(chan pipeline.Frame, 4)
	out, _, stop := runPacerStage(t, cfg, in)

	// Send 2 frames worth, then close immediately. Drain path must
	// emit both on close even without ticks.
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{
		Data: bytes.Repeat([]byte{3}, 1280),
	}}
	close(in)
	frames := drainFrames(out, 2, 500*time.Millisecond)
	_ = stop()
	if len(frames) != 2 {
		t.Errorf("got %d frames on drain, want 2", len(frames))
	}
}

func TestPacerStage_FrameSizeMatchesConfig(t *testing.T) {
	// 8 kHz @ 20 ms = 320 bytes/frame.
	cfg := PacerConfig{SampleRate: 8000, FrameMillis: 20, PrebufferFrames: 1}
	in := make(chan pipeline.Frame, 4)
	out, mt, stop := runPacerStage(t, cfg, in)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{
		Data: bytes.Repeat([]byte{7}, 640),
	}}
	mt.tick()
	frames := drainFrames(out, 1, 200*time.Millisecond)
	close(in)
	_ = stop()
	if len(frames) != 1 || len(frames[0].PCM.Data) != 320 {
		t.Errorf("frame size = %d, want 320 for 8k/20ms", len(frames[0].PCM.Data))
	}
	if frames[0].PCM.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", frames[0].PCM.SampleRate)
	}
}

func TestPacerStage_PartialTrailDropped(t *testing.T) {
	// 640 bytes/frame; supply 700. Drain must emit one full
	// frame and drop the trailing 60 bytes (no zero-pad).
	cfg := PacerConfig{PrebufferFrames: 1}
	in := make(chan pipeline.Frame, 4)
	out, _, stop := runPacerStage(t, cfg, in)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{
		Data: bytes.Repeat([]byte{9}, 700),
	}}
	close(in)
	frames := drainFrames(out, 2, 500*time.Millisecond)
	_ = stop()
	if len(frames) != 1 {
		t.Errorf("got %d frames, want exactly 1 (trailing 60 bytes dropped)", len(frames))
	}
}
