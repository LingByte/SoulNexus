package tts

import (
	"context"
	"testing"
	"time"
)

type fakeService struct {
	pcm []byte
}

func (f *fakeService) SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error {
	_ = text
	// send pcm in two chunks to test framing across boundaries
	mid := len(f.pcm) / 2
	if err := callback(f.pcm[:mid]); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return callback(f.pcm[mid:])
}

func TestPipeline_Speak_FramesPCM(t *testing.T) {
	// 16000Hz * 60ms * 2 bytes = 1920 bytes per frame
	frameBytes := 1920
	pcm := make([]byte, frameBytes*2+100) // 2 full frames + tail

	var frames [][]byte
	p, err := New(Config{
		Service:       &fakeService{pcm: pcm},
		SampleRate:    16000,
		Channels:      1,
		FrameDuration: 60 * time.Millisecond,
		SendPCMFrame: func(frame []byte) error {
			cp := make([]byte, len(frame))
			copy(cp, frame)
			frames = append(frames, cp)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	p.Start(context.Background())
	defer p.Stop()

	if err := p.Speak("hello"); err != nil {
		t.Fatalf("Speak failed: %v", err)
	}

	if len(frames) != 3 {
		t.Fatalf("expected 3 frames (2 full + tail), got=%d", len(frames))
	}
	if len(frames[0]) != frameBytes || len(frames[1]) != frameBytes {
		t.Fatalf("expected first two frames size=%d, got=%d,%d", frameBytes, len(frames[0]), len(frames[1]))
	}
	// Last partial frame is zero-padded to a full frame for downstream encoders (e.g. Opus).
	if len(frames[2]) != frameBytes {
		t.Fatalf("expected padded tail size=%d, got=%d", frameBytes, len(frames[2]))
	}
	tailOffset := len(pcm) - 100
	if string(frames[2][:100]) != string(pcm[tailOffset:]) {
		t.Fatalf("padded frame should preserve first 100 bytes of tail")
	}
	for i := 100; i < frameBytes; i++ {
		if frames[2][i] != 0 {
			t.Fatalf("expected zero padding after tail at byte %d, got %d", i, frames[2][i])
		}
	}
}

