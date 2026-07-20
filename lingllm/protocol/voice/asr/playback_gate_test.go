package asr

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPlaybackGateTail(t *testing.T) {
	var playing atomic.Bool
	playing.Store(true)
	gate := NewPlaybackGate(func() bool { return playing.Load() }, func() int { return 0 }, 100*time.Millisecond)

	if !gate.IsEchoSuppressActive() {
		t.Fatal("expected active while playing")
	}

	playing.Store(false)
	if !gate.IsEchoSuppressActive() {
		t.Fatal("expected tail after stop")
	}

	time.Sleep(120 * time.Millisecond)
	if gate.IsEchoSuppressActive() {
		t.Fatal("tail should expire")
	}
}

func TestPlaybackGateQueued(t *testing.T) {
	gate := NewPlaybackGate(func() bool { return false }, func() int { return 2 }, 0)
	if !gate.IsBargeInWindow() {
		t.Fatal("queued utterances should keep barge-in window open")
	}
}
