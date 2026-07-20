package session

import (
	"context"
	"testing"
	"time"
)

func TestEmitWelcomePCM_PacesWhenRequested(t *testing.T) {
	pcm := make([]byte, 640*3) // 3 frames @ 16kHz
	var n int
	start := time.Now()
	err := emitWelcomePCM(context.Background(), pcm, 16000, func(b []byte) error {
		n++
		return nil
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("frames=%d want 3", n)
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("paced emit too fast: %v", elapsed)
	}
}

func TestEmitWelcomePCM_NoPaceBurst(t *testing.T) {
	pcm := make([]byte, 640*5)
	start := time.Now()
	_ = emitWelcomePCM(context.Background(), pcm, 16000, func([]byte) error { return nil }, false)
	if elapsed := time.Since(start); elapsed > 30*time.Millisecond {
		t.Fatalf("unpaced emit too slow: %v", elapsed)
	}
}
