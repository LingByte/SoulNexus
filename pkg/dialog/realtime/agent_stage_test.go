// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

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

// fakeAgent records PushAudio / Cancel / Close calls and exposes a
// sink so tests can drive Event delivery as if it came from a real
// provider's WS read loop.
type fakeAgent struct {
	startErr  error
	pushErr   error
	mu        sync.Mutex
	pushed    [][]byte
	cancelled atomic.Int32
	closed    atomic.Int32
	sinkMu    sync.Mutex
	sink      EventSink
	ready     chan struct{}
}

func newFakeAgent() *fakeAgent { return &fakeAgent{ready: make(chan struct{})} }

func (a *fakeAgent) setSink(s EventSink) {
	a.sinkMu.Lock()
	a.sink = s
	a.sinkMu.Unlock()
	close(a.ready)
}

func (a *fakeAgent) emit(t *testing.T, ev Event) {
	t.Helper()
	select {
	case <-a.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("agent sink not wired within 2s")
	}
	a.sinkMu.Lock()
	s := a.sink
	a.sinkMu.Unlock()
	s.Emit(ev)
}

func (a *fakeAgent) Start(_ context.Context) error { return a.startErr }

func (a *fakeAgent) PushAudio(pcm []byte) error {
	if a.pushErr != nil {
		return a.pushErr
	}
	a.mu.Lock()
	a.pushed = append(a.pushed, append([]byte(nil), pcm...))
	a.mu.Unlock()
	return nil
}

func (a *fakeAgent) Cancel() error {
	a.cancelled.Add(1)
	return nil
}

func (a *fakeAgent) Close() error {
	a.closed.Add(1)
	return nil
}

func (a *fakeAgent) pushedCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.pushed)
}

// fakeBuilder hands out the supplied fakeAgent and captures the sink
// so the test can drive events.
type fakeBuilder struct {
	agent    *fakeAgent
	buildErr error
}

func (b *fakeBuilder) Build(sink EventSink) (Agent, error) {
	if b.buildErr != nil {
		return nil, b.buildErr
	}
	b.agent.setSink(sink)
	return b.agent, nil
}

// runAgentStage starts the stage in a goroutine and returns the
// output channel + a function that stops it cleanly. The wait helper
// first gives the stage up to 1s to exit on its own (in close /
// fatal event) before cancelling — this avoids a race where the
// cancel beats the in-close read in tests that drive teardown via
// close(in).
func runAgentStage(t *testing.T, s *agentStage, in <-chan pipeline.Frame) (<-chan pipeline.Frame, func() error) {
	t.Helper()
	out := make(chan pipeline.Frame, 32)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { errCh <- s.Run(ctx, in, out, engine.NopLogger{}) }()
	wait := func() error {
		select {
		case err := <-errCh:
			return err
		case <-time.After(1 * time.Second):
			cancel()
		}
		select {
		case err := <-errCh:
			return err
		case <-time.After(1 * time.Second):
			return errors.New("stage Run did not exit within 2s")
		}
	}
	return out, wait
}

func drainFrames(ch <-chan pipeline.Frame, want int, maxWait time.Duration) []pipeline.Frame {
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()
	out := make([]pipeline.Frame, 0, want)
	for len(out) < want {
		select {
		case f, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, f)
		case <-deadline.C:
			return out
		}
	}
	return out
}

func TestAgentStage_Name(t *testing.T) {
	if got := newAgentStage(nil).Name(); got != "realtime_agent" {
		t.Errorf("Name() = %q, want realtime_agent", got)
	}
}

func TestAgentStage_NilBuilderError(t *testing.T) {
	s := newAgentStage(nil)
	in := make(chan pipeline.Frame)
	close(in)
	out := make(chan pipeline.Frame, 1)
	err := s.Run(context.Background(), in, out, engine.NopLogger{})
	if err == nil {
		t.Fatal("Run with nil builder must error")
	}
}

func TestAgentStage_PushAudioForwarded(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}}}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{3, 4}}}
	close(in)

	out, wait := runAgentStage(t, s, in)
	// Wait for stage to exit on in-close.
	if err := wait(); err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Run = %v, want nil or Canceled", err)
	}
	_ = drainFrames(out, 0, 100*time.Millisecond)
	if got := agent.pushedCount(); got != 2 {
		t.Errorf("pushed = %d, want 2", got)
	}
	if agent.closed.Load() == 0 {
		t.Error("Close not called on stage exit")
	}
}

func TestAgentStage_ResamplesUplink8kToAgent16k(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	pcm8k := make([]byte, 8000*2*20/1000) // 20ms @ 8 kHz
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: pcm8k, SampleRate: 8000}}
	close(in)

	out, wait := runAgentStage(t, s, in)
	if err := wait(); err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Run = %v", err)
	}
	_ = drainFrames(out, 0, 50*time.Millisecond)

	agent.mu.Lock()
	defer agent.mu.Unlock()
	if len(agent.pushed) != 1 {
		t.Fatalf("pushed frames = %d, want 1", len(agent.pushed))
	}
	got := len(agent.pushed[0])
	want := AgentInputRate * 2 * 20 / 1000 // 640
	if got < want*8/10 || got > want*12/10 {
		t.Fatalf("pushed bytes = %d, want ~%d (8k→16k resample)", got, want)
	}
}

func TestAgentStage_TranslatesUserTranscriptFinal(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out, wait := runAgentStage(t, s, in)

	// Wait for sink wiring.
	agent.emit(t, Event{Kind: EventUserTranscript, Text: "hello", Final: true})

	frames := drainFrames(out, 1, 200*time.Millisecond)
	close(in)
	_ = wait()

	if len(frames) != 1 || frames[0].Kind != pipeline.KindTextFinal || frames[0].Text != "hello" {
		t.Errorf("got %+v, want one KindTextFinal/hello", frames)
	}
}

func TestAgentStage_TranslatesAssistantTurn(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out, wait := runAgentStage(t, s, in)
	// Open the turn with a non-final AI text fragment, then final.
	agent.emit(t, Event{Kind: EventAssistantText, Text: "It is "})
	agent.emit(t, Event{Kind: EventAssistantText, Text: "noon.", Final: true})

	frames := drainFrames(out, 3, 300*time.Millisecond)
	close(in)
	_ = wait()

	// Expect: KindAIText("It is "), KindAIText("noon."), KindAITextDone.
	if len(frames) < 3 {
		t.Fatalf("got %d frames, want 3: %+v", len(frames), frames)
	}
	if frames[0].Kind != pipeline.KindAIText || frames[0].Text != "It is " {
		t.Errorf("frame0 = %+v", frames[0])
	}
	if frames[1].Kind != pipeline.KindAIText || frames[1].Text != "noon." {
		t.Errorf("frame1 = %+v", frames[1])
	}
	if frames[2].Kind != pipeline.KindAITextDone {
		t.Errorf("frame2 = %+v, want KindAITextDone", frames[2])
	}
}

func TestAgentStage_BargeInCancelsAgent(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out, wait := runAgentStage(t, s, in)
	// Server-VAD barge-in from agent.
	agent.emit(t, Event{Kind: EventUserSpeechStarted})
	frames := drainFrames(out, 1, 200*time.Millisecond)
	close(in)
	_ = wait()
	if agent.cancelled.Load() != 1 {
		t.Errorf("Cancel call count = %d, want 1 (server-VAD speech started)", agent.cancelled.Load())
	}
	if len(frames) != 1 || frames[0].Kind != pipeline.KindBargeIn {
		t.Errorf("frames = %+v, want one KindBargeIn", frames)
	}
}

func TestAgentStage_InjectedBargeInCancelsAgent(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out, wait := runAgentStage(t, s, in)
	// External KindBargeIn → stage must Cancel() the agent AND
	// forward the frame.
	in <- pipeline.Frame{Kind: pipeline.KindBargeIn}
	frames := drainFrames(out, 1, 200*time.Millisecond)
	close(in)
	_ = wait()
	if agent.cancelled.Load() != 1 {
		t.Errorf("Cancel call count = %d, want 1", agent.cancelled.Load())
	}
	if len(frames) != 1 || frames[0].Kind != pipeline.KindBargeIn {
		t.Errorf("frames = %+v, want one KindBargeIn forwarded", frames)
	}
}

func TestAgentStage_AssistantAudioBecomesPCM(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out, wait := runAgentStage(t, s, in)
	agent.emit(t, Event{Kind: EventAssistantAudio, Audio: []byte{9, 9, 9, 9}, SampleRate: 24000})
	frames := drainFrames(out, 1, 200*time.Millisecond)
	close(in)
	_ = wait()
	if len(frames) != 1 || frames[0].Kind != pipeline.KindPCM {
		t.Fatalf("frames = %+v, want one KindPCM", frames)
	}
	if frames[0].PCM.SampleRate != 24000 {
		t.Errorf("SampleRate = %d, want 24000", frames[0].PCM.SampleRate)
	}
	if string(frames[0].PCM.Data) != "\x09\x09\x09\x09" {
		t.Errorf("Data = %v", frames[0].PCM.Data)
	}
}

func TestAgentStage_FatalErrorTeardown(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out := make(chan pipeline.Frame, 8)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { errCh <- s.Run(ctx, in, out, engine.NopLogger{}) }()
	agent.emit(t, Event{Kind: EventError, Fatal: true, Err: errors.New("boom")})
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Run = nil, want fatal error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stage did not exit on fatal error")
	}
}

func TestAgentStage_SessionCloseFlushesActiveTurn(t *testing.T) {
	agent := newFakeAgent()
	s := newAgentStage(&fakeBuilder{agent: agent})

	in := make(chan pipeline.Frame, 1)
	out, wait := runAgentStage(t, s, in)
	// Start an assistant turn (no Final), then session closes
	// mid-stream — stage must synthesise KindAITextDone so
	// downstream observers don't leak the turn.
	agent.emit(t, Event{Kind: EventAssistantText, Text: "halfway"})
	agent.emit(t, Event{Kind: EventSessionClose})

	frames := drainFrames(out, 2, 300*time.Millisecond)
	close(in)
	_ = wait()

	if len(frames) < 2 {
		t.Fatalf("got %d frames, want at least 2: %+v", len(frames), frames)
	}
	if frames[len(frames)-1].Kind != pipeline.KindAITextDone {
		t.Errorf("last frame = %+v, want KindAITextDone", frames[len(frames)-1])
	}
}
