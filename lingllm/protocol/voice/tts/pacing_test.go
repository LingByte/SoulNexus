package tts

import (
	"context"
	"testing"
	"time"
)

type slowMockTTS struct{}

func (slowMockTTS) Synthesize(ctx context.Context, text string, cb func([]byte) error) error {
	// Emit 3 frames worth instantly
	for i := 0; i < 3; i++ {
		if err := cb(make([]byte, 1920)); err != nil {
			return err
		}
	}
	return nil
}

func TestPaceRealtime(t *testing.T) {
	var sent int
	p, err := NewTTSPipeline(TTSPipelineConfig{
		TTSService:       slowMockTTS{},
		PaceRealtime:     true,
		FrameDuration:    20 * time.Millisecond,
		TargetSampleRate: 16000,
		SendCallback: func([]byte) error {
			sent++
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Start(context.Background())

	start := time.Now()
	_ = p.Speak("hello")
	elapsed := time.Since(start)

	if sent < 2 {
		t.Fatalf("sent %d frames", sent)
	}
	// 3 frames @ 20ms ≈ 40ms+ ; without pacing would be <5ms
	if elapsed < 35*time.Millisecond {
		t.Errorf("paced too fast: %v", elapsed)
	}
}
