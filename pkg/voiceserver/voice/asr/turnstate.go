// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

// TurnState is an idle/listening/finalizing FSM around an ASR session.
// It exists alongside IncrementalState (which dedups cumulative text)
// to give the dialog plane a clean view of *turn lifecycle* rather
// than just text deltas:
//
//	Idle        — no audio in, no partials seen
//	Listening   — audio is being fed but no partial has arrived yet
//	Speaking    — at least one non-empty partial is in flight
//	Finalizing  — final received OR silence detected; waiting for
//	              the dialog plane / LLM to consume the result
//	Idle (back to start)
//
// Transitions are pushed by ProcessPCM (audio in), partial / final
// callbacks (text in), and explicit Reset / EndOfTurn calls. The FSM
// is goroutine-safe; the text + audio paths typically run in
// different goroutines on busy calls.
//
// What the FSM does NOT decide:
//   - When to send "user finished speaking" to the LLM: that's a
//     policy question (silence VAD + grace timer) the gateway client
//     owns. The FSM exposes the current phase so a policy can be
//     written cleanly; it doesn't prescribe one.
//   - Whether a partial is "good enough" to act on: caller's choice.

import (
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"
)

// TurnPhase enumerates the FSM states. String() is provided so logs
// don't have to carry a switch statement everywhere.
type TurnPhase int32

const (
	PhaseIdle TurnPhase = iota
	PhaseListening
	PhaseSpeaking
	PhaseFinalizing
)

func (p TurnPhase) String() string {
	switch p {
	case PhaseIdle:
		return "idle"
	case PhaseListening:
		return "listening"
	case PhaseSpeaking:
		return "speaking"
	case PhaseFinalizing:
		return "finalizing"
	default:
		return "unknown"
	}
}

// TurnState tracks one ASR turn at a time. The zero value is unusable;
// always construct via NewTurnState. Concurrent-safe.
type TurnState struct {
	phase atomic.Int32 // TurnPhase; atomic so cheap reads don't need the mu

	mu             sync.Mutex
	turnIndex      int       // 1-based; bumped on each EndOfTurn
	lastAudioAt    time.Time // last ProcessPCM observed (used by IsSilentFor)
	lastPartialAt  time.Time
	lastFinalText  string
	lastFinalAt    time.Time
	lastTransition time.Time // timestamp of the last phase change

	// onTransition fires (without the mutex held) whenever the phase
	// changes. nil = silent. Useful for emitting structured events
	// to dashboards / metrics.
	onTransition func(prev, next TurnPhase, turnIndex int)
}

// NewTurnState returns an FSM in PhaseIdle.
func NewTurnState() *TurnState {
	t := &TurnState{}
	t.phase.Store(int32(PhaseIdle))
	t.lastTransition = time.Now()
	return t
}

// Phase returns the current state. Cheap (single atomic load).
func (t *TurnState) Phase() TurnPhase {
	if t == nil {
		return PhaseIdle
	}
	return TurnPhase(t.phase.Load())
}

// SetTransitionHook wires (or clears, with nil) the callback fired
// after every phase change. The hook runs on the goroutine that
// triggered the transition; keep it cheap or fan out to a worker.
func (t *TurnState) SetTransitionHook(fn func(prev, next TurnPhase, turnIndex int)) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.onTransition = fn
	t.mu.Unlock()
}

// OnAudio records that audio was fed at the given wall clock. If we
// were Idle, transitions to Listening. Returns the phase after the
// call (for callers that want to chain a metrics emit).
func (t *TurnState) OnAudio(at time.Time) TurnPhase {
	if t == nil {
		return PhaseIdle
	}
	t.mu.Lock()
	t.lastAudioAt = at
	prev := TurnPhase(t.phase.Load())
	if prev == PhaseIdle {
		t.transitionLocked(PhaseListening)
	}
	cur := TurnPhase(t.phase.Load())
	t.mu.Unlock()
	return cur
}

// OnPartial records a non-empty partial transcript and transitions
// to Speaking (if not already there or beyond).
func (t *TurnState) OnPartial(at time.Time, text string) TurnPhase {
	if t == nil {
		return PhaseIdle
	}
	t.mu.Lock()
	t.lastPartialAt = at
	prev := TurnPhase(t.phase.Load())
	// Idle/Listening/Speaking → Speaking; Finalizing means we already
	// got a final, so a stray late partial shouldn't downgrade us.
	if prev != PhaseFinalizing && text != "" {
		t.transitionLocked(PhaseSpeaking)
	}
	cur := TurnPhase(t.phase.Load())
	t.mu.Unlock()
	return cur
}

// OnFinal records a final transcript and moves to Finalizing. Most
// dialog flows then push the final text into the LLM and call
// EndOfTurn once the LLM accepts ownership.
func (t *TurnState) OnFinal(at time.Time, text string) TurnPhase {
	if t == nil {
		return PhaseIdle
	}
	t.mu.Lock()
	t.lastFinalAt = at
	t.lastFinalText = text
	t.transitionLocked(PhaseFinalizing)
	cur := TurnPhase(t.phase.Load())
	t.mu.Unlock()
	return cur
}

// EndOfTurn closes the current turn and returns to Idle. Bumps
// turnIndex so callers can correlate logs / events to "the third
// user turn of this call".
func (t *TurnState) EndOfTurn() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	t.turnIndex++
	idx := t.turnIndex
	t.transitionLocked(PhaseIdle)
	t.mu.Unlock()
	return idx
}

// IsSilentFor reports whether no audio has arrived for the supplied
// duration. Useful for "user paused" detection without a full VAD —
// pair with a 600-1200 ms threshold to drive end-of-turn logic.
func (t *TurnState) IsSilentFor(d time.Duration, now time.Time) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lastAudioAt.IsZero() {
		return false
	}
	return now.Sub(t.lastAudioAt) >= d
}

// LastFinal returns the most recently observed final text and its
// timestamp. (string, time.Time) — both zero values if no final yet.
func (t *TurnState) LastFinal() (string, time.Time) {
	if t == nil {
		return "", time.Time{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastFinalText, t.lastFinalAt
}

// TurnIndex returns the number of completed turns (i.e. EndOfTurn
// invocations) on this state. 0 = no turn has finished.
func (t *TurnState) TurnIndex() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.turnIndex
}

// transitionLocked changes phase, stamps the transition time, and
// schedules the hook if registered. The mutex MUST be held by caller.
// We capture the hook value before unlocking because the hook may
// take a while and we don't want to hold the FSM lock across it.
func (t *TurnState) transitionLocked(next TurnPhase) {
	prev := TurnPhase(t.phase.Load())
	if prev == next {
		return
	}
	t.phase.Store(int32(next))
	t.lastTransition = time.Now()
	hook := t.onTransition
	idx := t.turnIndex
	if hook != nil {
		// Run after the lock is released by spawning to a goroutine
		// would be safer for slow hooks, but turn transitions are
		// rare (a few per minute per call) and the cost of an extra
		// goroutine per transition probably outweighs the safety.
		// Instead document: hook MUST be quick.
		go hook(prev, next, idx)
	}
	_ = idx
}

// SentenceEndings is the set of punctuation the helpers below treat
// as sentence boundaries. Mirrors the Chinese-first set LingEchoX
// uses; ASCII variants included for English and code-mixed input.
var SentenceEndings = []rune{'。', '！', '？', '.', '!', '?', '\n'}

// FindLastSentenceEnding returns the byte offset of the last sentence
// terminator in text, or -1 if none. Operates on runes so multi-byte
// punctuation is matched correctly.
func FindLastSentenceEnding(text string) int {
	if text == "" {
		return -1
	}
	last := -1
	for i, r := range text {
		for _, e := range SentenceEndings {
			if r == e {
				last = i + utf8.RuneLen(r)
			}
		}
	}
	if last < 0 {
		return -1
	}
	return last - 1 // return offset of the terminator byte itself
}

// IsSentenceTerminator reports whether r is one of SentenceEndings.
func IsSentenceTerminator(r rune) bool {
	for _, e := range SentenceEndings {
		if r == e {
			return true
		}
	}
	return false
}

// NormalizeForCompare strips whitespace and punctuation, leaving only
// letters / digits / Han characters; collapses repeated identical
// runes; returns the result. Used for fuzzy-equality between two ASR
// hypotheses where the recognizer is jittering on inserted spaces or
// commas. Keeps the same shape as LingEchoX's normalizeTextFast.
func NormalizeForCompare(text string) string {
	if text == "" {
		return ""
	}
	out := make([]rune, 0, len(text))
	var last rune
	hasLast := false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !hasLast || r != last {
				out = append(out, r)
				last = r
				hasLast = true
			}
		}
	}
	return string(out)
}
