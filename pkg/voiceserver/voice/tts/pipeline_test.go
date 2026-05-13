package tts

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// staticService emits a pre-supplied payload in N chunks.
type staticService struct {
	chunks [][]byte
	err    error
}

func (s *staticService) SynthesizeStream(ctx context.Context, _ string, on func([]byte) error) error {
	for _, c := range s.chunks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := on(c); err != nil {
			return err
		}
	}
	return s.err
}

func TestPipelineNewValidations(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("want nil-service err")
	}
	if _, err := New(Config{Service: &staticService{}}); err == nil {
		t.Fatal("want nil-sink err")
	}
	if _, err := New(Config{Service: &staticService{}, Sink: func([]byte) error { return nil }}); err == nil {
		t.Fatal("want input-rate err")
	}
}

func TestPipelineSpeakDeliversFrames(t *testing.T) {
	// 20 ms @ 16 kHz PCM16 mono = 640 bytes.
	var sinkBytes atomic.Int64
	var frameCount atomic.Int64
	chunk := make([]byte, 1280) // two full frames
	svc := &staticService{chunks: [][]byte{chunk, chunk}}

	p, err := New(Config{
		Service:         svc,
		InputSampleRate: 16000, OutputSampleRate: 16000,
		FrameDuration: 20 * time.Millisecond,
		Sink: func(frame []byte) error {
			sinkBytes.Add(int64(len(frame)))
			frameCount.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Speak("x"); err == nil {
		t.Fatal("Speak before Start must fail")
	}
	p.Start(context.Background())
	if err := p.Speak("hello"); err != nil {
		t.Fatalf("Speak: %v", err)
	}
	if frameCount.Load() != 4 {
		t.Fatalf("want 4 frames, got %d", frameCount.Load())
	}
	if sinkBytes.Load() != 4*640 {
		t.Fatalf("want %d bytes, got %d", 4*640, sinkBytes.Load())
	}
	p.Stop()
}

func TestPipelineTailPadding(t *testing.T) {
	// 700 bytes → 1 full 640-byte frame + 60-byte tail (padded to 640).
	var frames int
	svc := &staticService{chunks: [][]byte{make([]byte, 700)}}
	p, _ := New(Config{
		Service: svc, InputSampleRate: 16000, FrameDuration: 20 * time.Millisecond,
		Sink: func([]byte) error { frames++; return nil },
	})
	p.Start(context.Background())
	if err := p.Speak("x"); err != nil {
		t.Fatal(err)
	}
	if frames != 2 {
		t.Fatalf("want 2 frames (full + padded tail), got %d", frames)
	}
}

func TestPipelineResampleOutput(t *testing.T) {
	var bytesOut int
	svc := &staticService{chunks: [][]byte{make([]byte, 640)}}
	p, _ := New(Config{
		Service:         svc,
		InputSampleRate: 16000, OutputSampleRate: 8000,
		FrameDuration: 20 * time.Millisecond,
		Sink:          func(f []byte) error { bytesOut += len(f); return nil },
	})
	p.Start(context.Background())
	if err := p.Speak("x"); err != nil {
		t.Fatal(err)
	}
	// Resampled to 8k → 320 bytes/frame.
	if bytesOut == 0 {
		t.Fatalf("resample produced no bytes")
	}
}

func TestPipelineInterrupt(t *testing.T) {
	var wg sync.WaitGroup
	block := make(chan struct{})
	svc := ServiceFunc(func(ctx context.Context, _ string, on func([]byte) error) error {
		// Emit one chunk, then wait until Interrupt cancels ctx.
		_ = on(make([]byte, 640))
		<-ctx.Done()
		return ctx.Err()
	})
	p, _ := New(Config{
		Service: svc, InputSampleRate: 16000,
		Sink: func([]byte) error { return nil },
	})
	p.Start(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.Speak("x"); err != nil {
			t.Errorf("Speak: %v", err)
		}
	}()
	time.Sleep(20 * time.Millisecond)
	p.Interrupt()
	wg.Wait()
	close(block)
	if p.IsPlaying() {
		t.Fatal("IsPlaying should be false after Interrupt")
	}
	p.Stop()
}

func TestPipelineServiceError(t *testing.T) {
	svc := ServiceFunc(func(_ context.Context, _ string, _ func([]byte) error) error {
		return errors.New("vendor down")
	})
	p, _ := New(Config{Service: svc, InputSampleRate: 16000, Sink: func([]byte) error { return nil }})
	p.Start(context.Background())
	if err := p.Speak("x"); err == nil {
		t.Fatal("want error")
	}
	p.Stop()
}

func TestPipelineStopCancelsInFlight(t *testing.T) {
	svc := ServiceFunc(func(ctx context.Context, _ string, _ func([]byte) error) error {
		<-ctx.Done()
		return ctx.Err()
	})
	p, _ := New(Config{Service: svc, InputSampleRate: 16000, Sink: func([]byte) error { return nil }})
	p.Start(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Speak("x")
	}()
	time.Sleep(10 * time.Millisecond)
	p.Stop()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop did not cancel Speak")
	}
}
