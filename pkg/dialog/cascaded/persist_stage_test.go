// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

type recordingPersister struct {
	mu  sync.Mutex
	got []TurnRecord
}

func (r *recordingPersister) PersistTurn(_ context.Context, rec TurnRecord) {
	r.mu.Lock()
	r.got = append(r.got, rec)
	r.mu.Unlock()
}

func (r *recordingPersister) snapshot() []TurnRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]TurnRecord, len(r.got))
	copy(out, r.got)
	return out
}

func runPersistStage(t *testing.T, s *persistStage, in <-chan pipeline.Frame) []pipeline.Frame {
	t.Helper()
	out := make(chan pipeline.Frame, 32)
	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(context.Background(), in, out, engine.NopLogger{}) }()
	got := drainOutput(t, out, time.Second)
	if err := <-errCh; err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	return got
}

// stepClock returns base + N*step on the Nth call. Deterministic
// across goroutine scheduling — used so timing assertions don't
// race against the stage's Run goroutine consuming frames.
type stepClock struct {
	mu   sync.Mutex
	base time.Time
	step time.Duration
	n    int
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.base.Add(time.Duration(c.n) * c.step)
	c.n++
	return t
}

func TestPersistStage_Name(t *testing.T) {
	if got := newPersistStage(nil, nil).Name(); got != "persist" {
		t.Errorf("Name() = %q, want %q", got, "persist")
	}
}

func TestPersistStage_NilPersisterPassthrough(t *testing.T) {
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "hi"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "hello back"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1}}}
	close(in)

	got := runPersistStage(t, newPersistStage(nil, nil), in)
	if len(got) != 4 {
		t.Errorf("got %d frames, want 4 passthrough", len(got))
	}
}

func TestPersistStage_HappyPathOneTurn(t *testing.T) {
	rp := &recordingPersister{}
	// stepClock returns base + N*100ms on the Nth Now() call. Now()
	// is invoked exactly 3 times in this test (the 2nd KindAIText
	// doesn't call Now() because firstAIAt is already set):
	//   call 0: KindTextFinal       startedAt   = base+0ms
	//   call 1: KindAIText (first)  firstAIAt   = base+100ms
	//   call 2: KindAITextDone      completedAt = base+200ms
	// → LLMFirstMs = 100, LLMWallMs = 200.
	clk := &stepClock{base: time.Unix(1700000000, 0), step: 100 * time.Millisecond}
	s := newPersistStage(rp, nil)
	s.nowFn = clk.Now

	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "  what time is it  "}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "It is "}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "noon."}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	r := got[0]
	if r.UserText != "what time is it" {
		t.Errorf("UserText = %q, want trimmed", r.UserText)
	}
	if r.AIText != "It is noon." {
		t.Errorf("AIText = %q, want concatenated", r.AIText)
	}
	if r.LLMFirstMs != 100 {
		t.Errorf("LLMFirstMs = %d, want 100", r.LLMFirstMs)
	}
	if r.LLMWallMs != 200 {
		t.Errorf("LLMWallMs = %d, want 200", r.LLMWallMs)
	}
	if r.PipelineMs != r.LLMWallMs {
		t.Errorf("PipelineMs (%d) != LLMWallMs (%d)", r.PipelineMs, r.LLMWallMs)
	}
}

func TestPersistStage_TTSFirstByteRecorded(t *testing.T) {
	rp := &recordingPersister{}
	// Now() only advances when timestamps are stamped:
	//   call 0: KindTextFinal       startedAt=speechEndedAt
	//   call 1: KindAIText (first)  firstAIAt
	//   uplink PCM is ignored for TTS (and silent → no speech stamp)
	//   call 2: KindPCM synth       firstPCMAt
	//   call 3: KindAITextDone      completedAt
	clk := &stepClock{base: time.Unix(1700000000, 0), step: 100 * time.Millisecond}
	s := newPersistStage(rp, nil)
	s.nowFn = clk.Now

	in := make(chan pipeline.Frame, 6)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "a"}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{9}}} // uplink — ignored
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1}, Synthesized: true}}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{2}, Synthesized: true}}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].TTSFirstByteMs != 200 {
		t.Errorf("TTSFirstByteMs = %d, want 200 (skip uplink PCM)", got[0].TTSFirstByteMs)
	}
	if got[0].E2EFirstByteMs != got[0].TTSFirstByteMs {
		t.Errorf("E2EFirstByteMs = %d, want fallback TTSFirstByteMs %d", got[0].E2EFirstByteMs, got[0].TTSFirstByteMs)
	}
}

func loudPCM16(nSamples int, amp int16) []byte {
	if nSamples < 1 {
		nSamples = 80
	}
	out := make([]byte, nSamples*2)
	for i := 0; i < nSamples; i++ {
		v := amp
		if i%2 == 1 {
			v = -amp
		}
		out[i*2] = byte(uint16(v))
		out[i*2+1] = byte(uint16(v) >> 8)
	}
	return out
}

func TestPersistStage_E2EFromUserSpeechEnd(t *testing.T) {
	rp := &recordingPersister{}
	// call 0: uplink speech → lastUserSpeechAt
	// call 1: TextFinal startedAt (speechEnd reuses lastUserSpeechAt)
	// call 2: AIText firstAI
	// call 3: synth PCM firstPCM
	// call 4: Done
	// TTS = call3-call1 = 200ms; E2E = call3-call0 = 300ms
	clk := &stepClock{base: time.Unix(1700000000, 0), step: 100 * time.Millisecond}
	s := newPersistStage(rp, nil)
	s.nowFn = clk.Now

	in := make(chan pipeline.Frame, 8)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: loudPCM16(160, 4000)}}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "喂"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "您好"}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}, Synthesized: true}}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].TTSFirstByteMs != 200 {
		t.Errorf("TTSFirstByteMs = %d, want 200", got[0].TTSFirstByteMs)
	}
	if got[0].E2EFirstByteMs != 300 {
		t.Errorf("E2EFirstByteMs = %d, want 300 (speech end → first client PCM)", got[0].E2EFirstByteMs)
	}
}

func TestPersistStage_E2EIgnoresFinalTimeEmittedAt(t *testing.T) {
	// ASR often stamps EmittedAt ≈ TextFinal when a late correction lands
	// with the final. That must not erase energy-based speech end.
	rp := &recordingPersister{}
	clk := &stepClock{base: time.Unix(1700000000, 0), step: 100 * time.Millisecond}
	s := newPersistStage(rp, nil)
	s.nowFn = clk.Now

	in := make(chan pipeline.Frame, 8)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: loudPCM16(160, 4000)}} // t0 speech
	_ = clk.Now()
	_ = clk.Now()
	finalAt := clk.Now() // t3 — TextFinal wall time
	in <- pipeline.Frame{
		Kind:      pipeline.KindTextFinal,
		Text:      "需要预约课程",
		EmittedAt: finalAt, // same as final — content_change 0 pattern
	}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "您好"}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}, Synthesized: true}}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	// speech at t0, first PCM several steps later — E2E must use t0 not finalAt.
	if got[0].E2EFirstByteMs < 300 {
		t.Errorf("E2EFirstByteMs = %d, want >= 300 (energy speech end, not final-time EmittedAt)", got[0].E2EFirstByteMs)
	}
}


func TestPersistStage_PCMBeforeTurnIgnored(t *testing.T) {
	// PCM frames arriving before a turn starts must not seed
	// firstPCMAt (which would produce negative / nonsensical
	// TTSFirstByteMs once the turn does begin).
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1}}}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "a"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].TTSFirstByteMs != 0 {
		t.Errorf("TTSFirstByteMs = %d, want 0 (no PCM in turn)", got[0].TTSFirstByteMs)
	}
}

func TestPersistStage_MultipleTurns(t *testing.T) {
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 6)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q1"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "a1"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q2"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "a2"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	if got[0].UserText != "q1" || got[0].AIText != "a1" {
		t.Errorf("turn 0 = %+v", got[0])
	}
	if got[1].UserText != "q2" || got[1].AIText != "a2" {
		t.Errorf("turn 1 = %+v", got[1])
	}
}

func TestPersistStage_NewTurnFlushesStale(t *testing.T) {
	// If a new KindTextFinal arrives BEFORE KindAITextDone (e.g.
	// pre-emption), the stale accumulator must still be flushed so
	// no turn silently disappears.
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q1"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "a1-partial"}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q2"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2 (stale + new)", len(got))
	}
	if got[0].UserText != "q1" || got[0].AIText != "a1-partial" {
		t.Errorf("stale turn = %+v", got[0])
	}
	if got[1].UserText != "q2" || got[1].AIText != "" {
		t.Errorf("new turn = %+v (want q2 + empty AI)", got[1])
	}
}

func TestPersistStage_UserFinalRevisionBeforeAssistant(t *testing.T) {
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "您好，"}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "你好，听到我说话吗？"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "听得见。"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].UserText != "你好，听到我说话吗？" {
		t.Errorf("UserText = %q, want revised final", got[0].UserText)
	}
	if got[0].AIText != "听得见。" {
		t.Errorf("AIText = %q", got[0].AIText)
	}
}

func TestPersistStage_CumulativeAssistantText(t *testing.T) {
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 5)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "听得见。"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "听得见。有什么可以帮您的？"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	want := "听得见。有什么可以帮您的？"
	if got[0].AIText != want {
		t.Errorf("AIText = %q, want %q", got[0].AIText, want)
	}
}

func TestPersistStage_EmptyTurnNotPersisted(t *testing.T) {
	// KindAITextDone without any KindTextFinal / KindAIText
	// must not create a phantom record.
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 1)
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	close(in)

	_ = runPersistStage(t, s, in)
	if got := rp.snapshot(); len(got) != 0 {
		t.Errorf("got %d records, want 0", len(got))
	}
}

func TestPersistStage_DraftDrainOnClose(t *testing.T) {
	// If the input channel closes mid-turn (e.g. call ends after AI
	// started replying), an incomplete-but-non-empty draft should
	// still be persisted so the partial AI utterance is recorded.
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "q"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "partial..."}
	close(in)

	_ = runPersistStage(t, s, in)
	got := rp.snapshot()
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1 partial flush", len(got))
	}
	if got[0].AIText != "partial..." {
		t.Errorf("AIText = %q, want partial", got[0].AIText)
	}
}

func TestPersistStage_PassthroughIntact(t *testing.T) {
	// All frames must reach downstream, with text NOT mutated.
	rp := &recordingPersister{}
	s := newPersistStage(rp, nil)
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "user q"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "ai reply"}
	in <- pipeline.Frame{Kind: pipeline.KindAITextDone}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1, 2}}}
	close(in)

	got := runPersistStage(t, s, in)
	if len(got) != 4 {
		t.Errorf("got %d frames, want 4 passthrough", len(got))
	}
	// First frame text intact
	if got[0].Text != "user q" {
		t.Errorf("got[0].Text = %q, want intact", got[0].Text)
	}
}
