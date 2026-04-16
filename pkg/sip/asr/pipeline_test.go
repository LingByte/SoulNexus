package asr

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
)

type fakeASR struct {
	mu       sync.Mutex
	onResult func(text string, isFinal bool, duration time.Duration, uuid string)
	onErr    func(err error, fatal bool)
	sent     int
}

func (f *fakeASR) Init(onResult recognizer.TranscribeResult, onError recognizer.ProcessError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onResult = onResult
	f.onErr = onError
}

func (f *fakeASR) Vendor() string { return "fake" }
func (f *fakeASR) ConnAndReceive(_ string) error { return nil }
func (f *fakeASR) Activity() bool { return true }
func (f *fakeASR) RestartClient() {}

func (f *fakeASR) SendAudioBytes(pcmData []byte) error {
	f.mu.Lock()
	f.sent += len(pcmData)
	cb := f.onResult
	f.mu.Unlock()

	// Emit one interim result once we receive audio.
	if cb != nil && len(pcmData) > 0 {
		cb("hello", false, 0, "uuid")
		cb("hello world", true, 0, "uuid")
	}
	return nil
}

func (f *fakeASR) SendEnd() error  { return nil }
func (f *fakeASR) StopConn() error { return nil }

var _ recognizer.TranscribeService = (*fakeASR)(nil)

func TestPipeline_ProcessPCM_EmitsText(t *testing.T) {
	asrSvc := &fakeASR{}
	p, err := New(Options{ASR: asrSvc, SampleRate: 16000, Channels: 1})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	var got []string
	var mu sync.Mutex
	done := make(chan struct{})
	p.SetTextCallback(func(text string, isFinal bool) {
		mu.Lock()
		got = append(got, text)
		if isFinal {
			close(done)
		}
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := p.ProcessPCM(ctx, make([]byte, 320)); err != nil {
		t.Fatalf("ProcessPCM failed: %v", err)
	}

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("timeout waiting for final result")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) < 2 {
		t.Fatalf("expected >=2 results, got=%v", got)
	}
	if got[len(got)-1] != "hello world" {
		t.Fatalf("unexpected final text: %q", got[len(got)-1])
	}

	m := p.GetMetrics()
	if m.TotalAudioBytes == 0 {
		t.Fatalf("expected metrics to record audio bytes")
	}
}

