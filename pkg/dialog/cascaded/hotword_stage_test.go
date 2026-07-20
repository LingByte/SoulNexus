// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// fakeRewriter is a TextRewriter that swaps a fixed substring.
type fakeRewriter struct {
	from, to string
	calls    atomic.Int32
}

func (f *fakeRewriter) Correct(text string) string {
	f.calls.Add(1)
	return strings.ReplaceAll(text, f.from, f.to)
}

func runHotwordStage(t *testing.T, s *hotwordStage, in <-chan pipeline.Frame) []pipeline.Frame {
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

func TestHotwordStage_Name(t *testing.T) {
	if got := newHotwordStage(nil).Name(); got != "hotword" {
		t.Errorf("Name() = %q, want %q", got, "hotword")
	}
}

func TestHotwordStage_NilRewriterPassesThrough(t *testing.T) {
	in := make(chan pipeline.Frame, 3)
	in <- pipeline.Frame{Kind: pipeline.KindTextInterim, Text: "hello"}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "world"}
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1}}}
	close(in)

	got := runHotwordStage(t, newHotwordStage(nil), in)
	if len(got) != 3 {
		t.Errorf("got %d frames, want 3 passthrough", len(got))
	}
	for i, f := range got {
		if f.Kind == pipeline.KindTextInterim && f.Text != "hello" {
			t.Errorf("frame[%d] text mutated: %q", i, f.Text)
		}
	}
}

func TestHotwordStage_RewritesTextFrames(t *testing.T) {
	rw := &fakeRewriter{from: "wrong", to: "right"}
	in := make(chan pipeline.Frame, 4)
	in <- pipeline.Frame{Kind: pipeline.KindTextInterim, Text: "the wrong word"}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "still wrong"}
	in <- pipeline.Frame{Kind: pipeline.KindAIText, Text: "AI says wrong"} // not rewritten
	in <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: engine.PCMFrame{Data: []byte{1}}}
	close(in)

	got := runHotwordStage(t, newHotwordStage(rw), in)
	if len(got) != 4 {
		t.Fatalf("got %d frames, want 4", len(got))
	}
	if got[0].Text != "the right word" {
		t.Errorf("interim text = %q, want corrected", got[0].Text)
	}
	if got[1].Text != "still right" {
		t.Errorf("final text = %q, want corrected", got[1].Text)
	}
	// AI text is NOT rewritten — only ASR-side kinds
	if got[2].Text != "AI says wrong" {
		t.Errorf("AI text mutated: %q", got[2].Text)
	}
	// Correct() called only for the two ASR text frames
	if calls := rw.calls.Load(); calls != 2 {
		t.Errorf("Correct() calls = %d, want 2", calls)
	}
}

func TestHotwordStage_NoOpRewriteUsesOriginal(t *testing.T) {
	// rewriter that returns input unchanged → frame must pass
	// through without triggering the observer.
	rw := &fakeRewriter{from: "xxxx", to: "yyyy"} // never matches
	var observerHits atomic.Int32
	s := newHotwordStage(rw, withHotwordObserver(func(raw, corrected string, isFinal bool) {
		observerHits.Add(1)
	}))
	in := make(chan pipeline.Frame, 1)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "untouched"}
	close(in)

	got := runHotwordStage(t, s, in)
	if len(got) != 1 || got[0].Text != "untouched" {
		t.Errorf("expected single passthrough, got %+v", got)
	}
	if observerHits.Load() != 0 {
		t.Errorf("observer fired on no-op rewrite (= %d)", observerHits.Load())
	}
}

func TestHotwordStage_ObserverFiresOnRewrite(t *testing.T) {
	rw := &fakeRewriter{from: "foo", to: "bar"}
	var observed atomic.Int32
	s := newHotwordStage(rw, withHotwordObserver(func(raw, corrected string, isFinal bool) {
		observed.Add(1)
		if !isFinal {
			t.Errorf("observer isFinal = %v, want true (sent KindTextFinal)", isFinal)
		}
		if raw != "foo bar baz" || corrected != "bar bar baz" {
			t.Errorf("observer payload mismatch: raw=%q corrected=%q", raw, corrected)
		}
	}))
	in := make(chan pipeline.Frame, 1)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "foo bar baz"}
	close(in)
	_ = runHotwordStage(t, s, in)
	if observed.Load() != 1 {
		t.Errorf("observer fired %d times, want 1", observed.Load())
	}
}

func TestHotwordStage_EmptyTextFramesPassThrough(t *testing.T) {
	rw := &fakeRewriter{from: "x", to: "y"}
	in := make(chan pipeline.Frame, 2)
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: "   "}
	in <- pipeline.Frame{Kind: pipeline.KindTextFinal, Text: ""}
	close(in)
	got := runHotwordStage(t, newHotwordStage(rw), in)
	if len(got) != 2 {
		t.Errorf("got %d frames, want 2 passthrough", len(got))
	}
	if rw.calls.Load() != 0 {
		t.Errorf("Correct() called for empty input (= %d)", rw.calls.Load())
	}
}
