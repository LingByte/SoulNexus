package tts

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type delayedMockTTS struct {
	delay time.Duration
}

func (m delayedMockTTS) Synthesize(ctx context.Context, text string, cb func([]byte) error) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return cb(make([]byte, 1920))
}

func TestSpeakerPipelineOverlap(t *testing.T) {
	var sent atomic.Int32
	p, err := NewTTSPipeline(TTSPipelineConfig{
		TTSService:   delayedMockTTS{delay: 40 * time.Millisecond},
		SendCallback: func([]byte) error { sent.Add(1); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	sp, err := NewSpeaker(SpeakerConfig{Pipeline: p, Prefetch: 2})
	if err != nil {
		t.Fatal(err)
	}
	sp.Start(context.Background())

	// Second segment should begin synthesis while the first is still playing.
	sp.Enqueue("segment one here", "u1", nil)
	sp.Enqueue("segment two here", "u1", nil)

	deadline := time.Now().Add(2 * time.Second)
	for sent.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if sent.Load() < 2 {
		t.Fatalf("expected 2 audio frames, got %d", sent.Load())
	}

	sp.Stop()
}

func TestSpeakerInterrupt(t *testing.T) {
	p, err := NewTTSPipeline(TTSPipelineConfig{
		TTSService:   delayedMockTTS{delay: 200 * time.Millisecond},
		SendCallback: func([]byte) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Start(context.Background())
	sp, _ := NewSpeaker(SpeakerConfig{Pipeline: p})
	sp.Start(context.Background())
	sp.Enqueue("hello", "u1", nil)
	time.Sleep(20 * time.Millisecond)
	sp.Interrupt()
	sp.Stop()
}

func TestSpeakerStreamPlayback(t *testing.T) {
	var mu sync.Mutex
	var started []string

	p, _ := NewTTSPipeline(TTSPipelineConfig{
		TTSService: mockTTSService{},
		SendCallback: func([]byte) error {
			return nil
		},
	})
	p.Start(context.Background())

	sp, _ := NewSpeaker(SpeakerConfig{
		Pipeline: p,
		OnStarted: func(id, text string, _ bool) {
			mu.Lock()
			started = append(started, text)
			mu.Unlock()
		},
	})
	sp.Start(context.Background())
	sp.Enqueue("你好。", "u1", nil)
	sp.Enqueue("今天不错！", "u1", nil)
	time.Sleep(100 * time.Millisecond)
	sp.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(started) == 0 {
		t.Fatal("expected at least one tts.started callback")
	}
}

type mockTTSService struct{}

func (mockTTSService) Synthesize(ctx context.Context, text string, cb func([]byte) error) error {
	return cb(make([]byte, 1920))
}
