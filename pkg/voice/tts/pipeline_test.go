package tts

import (
	"context"
	"testing"
	"time"
)

type pipelineFakeService struct {
	pcm []byte
}

func (f *pipelineFakeService) SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error {
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
		Service:       &pipelineFakeService{pcm: pcm},
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

	// New semantics (SoulNexus): Speak() no longer zero-pads the sub-frame
	// tail. The tail is stashed as residual to be either consumed by the
	// next Speak() (continuity) or drained by Finalize() at end-of-turn.
	// So we expect exactly 2 full frames here, not 3.
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames (2 full, tail stashed as residual), got=%d", len(frames))
	}
	if len(frames[0]) != frameBytes || len(frames[1]) != frameBytes {
		t.Fatalf("expected both frames size=%d, got=%d,%d", frameBytes, len(frames[0]), len(frames[1]))
	}

	// Finalize drains the residual as one zero-padded frame.
	if err := p.Finalize(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}
	if len(frames) != 3 {
		t.Fatalf("expected 3 frames after Finalize, got=%d", len(frames))
	}
	if len(frames[2]) != frameBytes {
		t.Fatalf("expected padded tail size=%d, got=%d", frameBytes, len(frames[2]))
	}
	tailOffset := len(pcm) - 100
	if string(frames[2][:100]) != string(pcm[tailOffset:]) {
		t.Fatalf("Finalize frame should preserve first 100 bytes of tail")
	}
	for i := 100; i < frameBytes; i++ {
		if frames[2][i] != 0 {
			t.Fatalf("expected zero padding after tail at byte %d, got %d", i, frames[2][i])
		}
	}
}

// TestPipeline_ResidualCarriesAcrossSpeak verifies that two consecutive
// Speak() calls on the same Pipeline produce a CONTINUOUS audio stream
// — the sub-frame tail of Speak #1 is consumed at the start of Speak #2
// rather than being zero-padded (which would have produced the "滋滋"
// hiss the user reported between LLM-streamed sentences).
func TestPipeline_ResidualCarriesAcrossSpeak(t *testing.T) {
	const frameBytes = 1920 // 16kHz × 60ms × 2 bytes
	// First call: 1 full frame + 200-byte tail. With residual carry,
	// only 1 frame should be emitted (the tail is stashed).
	pcm1 := bytes200(frameBytes+200, 0xAA)
	// Second call: 100 bytes. Combined with the 200-byte carry that is
	// 300 bytes — still less than a full frame, so STILL no emit. The
	// carry grows to 300 bytes.
	pcm2 := bytes200(100, 0xBB)
	// Third call: enough bytes to push past one full frame.
	pcm3 := bytes200(frameBytes, 0xCC)

	var frames [][]byte
	sink := func(frame []byte) error {
		cp := make([]byte, len(frame))
		copy(cp, frame)
		frames = append(frames, cp)
		return nil
	}

	mk := func(body []byte) *pipelineFakeService { return &pipelineFakeService{pcm: body} }
	cfg := Config{SampleRate: 16000, Channels: 1, FrameDuration: 60 * time.Millisecond, SendPCMFrame: sink}

	cfg.Service = mk(pcm1)
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.Start(context.Background())
	defer p.Stop()
	if err := p.Speak("s1"); err != nil {
		t.Fatalf("Speak s1: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("after Speak s1: expected 1 frame, got %d", len(frames))
	}

	// Reuse the same Pipeline; swap the service so Speak #2 streams pcm2.
	p.cfg.Service = mk(pcm2)
	if err := p.Speak("s2"); err != nil {
		t.Fatalf("Speak s2: %v", err)
	}
	// 200 carry + 100 new = 300 bytes; still < 1920. No new frame.
	if len(frames) != 1 {
		t.Fatalf("after Speak s2: expected 1 frame (residual grows), got %d", len(frames))
	}

	p.cfg.Service = mk(pcm3)
	if err := p.Speak("s3"); err != nil {
		t.Fatalf("Speak s3: %v", err)
	}
	// 300 carry + 1920 new = 2220 bytes; one full frame emitted, 300 carry remains.
	if len(frames) != 2 {
		t.Fatalf("after Speak s3: expected 2 frames, got %d", len(frames))
	}
	// The 2nd emitted frame must begin with the tail bytes of pcm1
	// (the original carry) — proves continuity is preserved across
	// Speak boundaries.
	want := byte(0xAA)
	if frames[1][0] != want {
		t.Fatalf("residual carry broken: frame[1][0]=%#x want %#x", frames[1][0], want)
	}
}

func bytes200(n int, v byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}

// TestPipeline_StopClearsResidual verifies that Stop() discards any
// residual, so a barge-in mid-utterance does NOT splice the cancelled
// turn's audio tail into the next reply.
func TestPipeline_StopClearsResidual(t *testing.T) {
	const frameBytes = 1920
	pcm := bytes200(frameBytes+200, 0xAA)
	var frames [][]byte
	p, err := New(Config{
		Service:       &pipelineFakeService{pcm: pcm},
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
		t.Fatalf("New: %v", err)
	}
	p.Start(context.Background())
	if err := p.Speak("x"); err != nil {
		t.Fatalf("Speak: %v", err)
	}
	p.Stop()
	// Finalize after Stop must NOT emit anything because residual was discarded.
	before := len(frames)
	if err := p.Finalize(); err != nil {
		t.Fatalf("Finalize after Stop: %v", err)
	}
	if len(frames) != before {
		t.Fatalf("Stop must discard residual; Finalize emitted %d extra frames", len(frames)-before)
	}
}
