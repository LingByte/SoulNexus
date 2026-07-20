package asr

import (
	"context"
	"encoding/binary"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/lingllm/media"
)

func testGate(streaming bool) *PlaybackGate {
	var flag atomic.Bool
	flag.Store(streaming)
	return NewPlaybackGate(func() bool { return flag.Load() }, func() int { return 0 }, 0)
}

func TestVADComponentCreation(t *testing.T) {
	vad := NewVADComponent(DefaultVADConfig(), nil)
	if vad == nil {
		t.Fatal("nil VAD")
	}
	if vad.Name() != "vad" {
		t.Errorf("name = %s", vad.Name())
	}
}

func TestVADComponentBargeInDetection(t *testing.T) {
	cfg := DefaultBargeInVADConfig()
	cfg.Threshold = 1000
	cfg.ConsecutiveFramesNeeded = 1
	gate := testGate(true)
	vad := NewVADComponent(cfg, gate)

	done := make(chan struct{}, 1)
	vad.SetBargeInCallback(func() { done <- struct{}{} })

	speech := make([]byte, 320)
	for i := 0; i < len(speech)-1; i += 2 {
		binary.LittleEndian.PutUint16(speech[i:i+2], 8000)
	}
	if _, _, err := vad.Process(context.Background(), speech); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Error("expected barge-in callback")
	}
}

func TestVADComponentBargeInNotWhenIdle(t *testing.T) {
	cfg := DefaultBargeInVADConfig()
	cfg.Threshold = 1000
	cfg.ConsecutiveFramesNeeded = 1
	gate := testGate(false)
	vad := NewVADComponent(cfg, gate)

	fired := false
	vad.SetBargeInCallback(func() { fired = true })

	speech := make([]byte, 320)
	for i := 0; i < len(speech)-1; i += 2 {
		binary.LittleEndian.PutUint16(speech[i:i+2], 8000)
	}
	_, _, _ = vad.Process(context.Background(), speech)
	if fired {
		t.Error("barge-in should not fire when downlink idle")
	}
}

func TestVADComponentDisabled(t *testing.T) {
	vad := NewVADComponent(DefaultVADConfig(), testGate(true))
	vad.SetEnabled(false)
	speech := make([]byte, 320)
	binary.LittleEndian.PutUint16(speech[0:2], 8000)
	_, _, err := vad.Process(context.Background(), speech)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDSPRMSSpeech(t *testing.T) {
	speech := make([]byte, 320)
	for i := 0; i < len(speech)-1; i += 2 {
		binary.LittleEndian.PutUint16(speech[i:i+2], 5000)
	}
	rms := media.RMSPCM16LE(speech)
	if rms < 4000 || rms > 6000 {
		t.Errorf("rms = %f", rms)
	}
}
