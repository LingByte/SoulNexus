package webrtc

import (
	"context"
	"sync"
	"testing"
	"time"

	pionmedia "github.com/pion/webrtc/v3/pkg/media"
)

type fakeSampleWriter struct {
	mu    sync.Mutex
	n     int
	times []time.Time
}

func (f *fakeSampleWriter) WriteSample(pionmedia.Sample) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	f.times = append(f.times, time.Now())
	return nil
}

func TestDownlinkPacer_PacesAfterKickBurst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &fakeSampleWriter{}
	var mu sync.Mutex
	p := newDownlinkPacer(ctx, w, &mu, 20*time.Millisecond, 16)
	defer p.Close()

	const n = 8
	start := time.Now()
	for i := 0; i < n; i++ {
		if err := p.Enqueue([]byte{byte(i)}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}
	deadline := time.After(2 * time.Second)
	for {
		w.mu.Lock()
		got := w.n
		w.mu.Unlock()
		if got >= n {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("wrote %d samples, want %d", got, n)
		case <-time.After(5 * time.Millisecond):
		}
	}
	elapsed := time.Since(start)
	// kickBurst=5 immediate, then paced @20ms.
	if elapsed < 50*time.Millisecond {
		t.Fatalf("elapsed %v too fast — pacing after kick-burst failed", elapsed)
	}
	w.mu.Lock()
	times := append([]time.Time(nil), w.times...)
	w.mu.Unlock()
	if len(times) < 5 {
		t.Fatal("need at least 5 writes")
	}
	burstSpan := times[4].Sub(times[0])
	if burstSpan > 50*time.Millisecond {
		t.Fatalf("kick-burst span %v, want first 5 packets nearly immediate", burstSpan)
	}
}

func TestDownlinkPacer_FlushDropsQueued(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &fakeSampleWriter{}
	var mu sync.Mutex
	p := newDownlinkPacer(ctx, w, &mu, 20*time.Millisecond, 32)
	defer p.Close()

	for i := 0; i < 20; i++ {
		_ = p.Enqueue([]byte{byte(i)})
	}
	p.Flush()
	time.Sleep(80 * time.Millisecond)
	w.mu.Lock()
	n := w.n
	w.mu.Unlock()
	if n > 4 {
		t.Fatalf("after Flush wrote %d samples; expected most queued frames dropped", n)
	}
}
