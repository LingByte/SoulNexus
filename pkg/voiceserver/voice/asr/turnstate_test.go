// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import (
	"sync"
	"testing"
	"time"
)

func TestTurnState_HappyPath(t *testing.T) {
	ts := NewTurnState()
	if ts.Phase() != PhaseIdle {
		t.Fatalf("zero state should be Idle, got %v", ts.Phase())
	}

	now := time.Now()
	ts.OnAudio(now)
	if ts.Phase() != PhaseListening {
		t.Fatalf("after OnAudio should be Listening, got %v", ts.Phase())
	}

	ts.OnPartial(now, "hello")
	if ts.Phase() != PhaseSpeaking {
		t.Fatalf("after OnPartial should be Speaking, got %v", ts.Phase())
	}

	ts.OnFinal(now, "hello world")
	if ts.Phase() != PhaseFinalizing {
		t.Fatalf("after OnFinal should be Finalizing, got %v", ts.Phase())
	}

	idx := ts.EndOfTurn()
	if idx != 1 {
		t.Fatalf("first EndOfTurn should yield 1, got %d", idx)
	}
	if ts.Phase() != PhaseIdle {
		t.Fatalf("after EndOfTurn should be Idle, got %v", ts.Phase())
	}

	got, _ := ts.LastFinal()
	if got != "hello world" {
		t.Fatalf("LastFinal=%q", got)
	}
}

func TestTurnState_EmptyPartialDoesNotMoveToSpeaking(t *testing.T) {
	ts := NewTurnState()
	ts.OnAudio(time.Now())
	ts.OnPartial(time.Now(), "") // empty partial — recognizer noise
	if ts.Phase() != PhaseListening {
		t.Fatalf("empty partial should keep us in Listening, got %v", ts.Phase())
	}
}

func TestTurnState_StalePartialAfterFinalIgnored(t *testing.T) {
	ts := NewTurnState()
	ts.OnFinal(time.Now(), "done")
	ts.OnPartial(time.Now(), "late stray")
	if ts.Phase() != PhaseFinalizing {
		t.Fatalf("late partial should not knock us back from Finalizing, got %v", ts.Phase())
	}
}

func TestTurnState_IsSilentFor(t *testing.T) {
	ts := NewTurnState()
	now := time.Now()
	ts.OnAudio(now)
	if ts.IsSilentFor(500*time.Millisecond, now.Add(100*time.Millisecond)) {
		t.Fatal("should not be silent 100ms after audio")
	}
	if !ts.IsSilentFor(500*time.Millisecond, now.Add(700*time.Millisecond)) {
		t.Fatal("should be silent 700ms after audio with 500ms threshold")
	}
}

func TestTurnState_TransitionHookFires(t *testing.T) {
	ts := NewTurnState()
	var (
		mu     sync.Mutex
		calls  []string
	)
	ts.SetTransitionHook(func(prev, next TurnPhase, _ int) {
		mu.Lock()
		calls = append(calls, prev.String()+"->"+next.String())
		mu.Unlock()
	})

	ts.OnAudio(time.Now())
	ts.OnPartial(time.Now(), "x")
	ts.OnFinal(time.Now(), "x done")
	ts.EndOfTurn()
	// hooks run on goroutines; let them settle
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 4 {
		t.Fatalf("expected 4 transitions, got %d: %v", len(calls), calls)
	}
}

func TestFindLastSentenceEnding(t *testing.T) {
	// FindLastSentenceEnding returns the offset of the LAST byte of
	// the terminator (so callers can slice text[:idx+1]). For ASCII
	// punctuation that's the same byte; for multi-byte CJK punct
	// like 。 (3 bytes) it's the 3rd byte.
	cases := []struct {
		in   string
		want int
	}{
		{"", -1},
		{"hello", -1},
		{"hello.", 5},
		{"hello. world", 5},
		{"hello! world?", 12},
		{"你好。", 8}, // 你(0..2) 好(3..5) 。(6..8)
	}
	for _, c := range cases {
		got := FindLastSentenceEnding(c.in)
		if got != c.want {
			t.Errorf("FindLastSentenceEnding(%q)=%d, want %d", c.in, got, c.want)
		}
	}
}

func TestNormalizeForCompare(t *testing.T) {
	// NormalizeForCompare strips punctuation/whitespace and collapses
	// CONSECUTIVE identical runes (so "abccc" becomes "abc"). It does
	// NOT dedup repeated substrings (so "你好你好" stays "你好你好").
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{" hello,  world  ", "heloworld"}, // "ll" → "l", "oo" preserved (no consecutive o)
		{"你好，  你好！", "你好你好"},
		{"abc-123  abc-123", "abc123abc123"},
	}
	for _, c := range cases {
		got := NormalizeForCompare(c.in)
		if got != c.want {
			t.Errorf("NormalizeForCompare(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}
