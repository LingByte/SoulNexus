package asr

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/lingllm/recognizer"
)

type fakeASR struct {
	mu       sync.Mutex
	onResult recognizer.SpeechRecognitionResult
	onErr    recognizer.RecognitionError
	sent     int
}

func (f *fakeASR) Init(onResult recognizer.SpeechRecognitionResult, onError recognizer.RecognitionError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onResult = onResult
	f.onErr = onError
}

func (f *fakeASR) Vendor() string                { return "fake" }
func (f *fakeASR) ConnAndReceive(_ string) error { return nil }
func (f *fakeASR) Activity() bool                { return true }
func (f *fakeASR) RestartClient()                {}

func (f *fakeASR) SendAudioBytes(pcmData []byte) error {
	f.mu.Lock()
	f.sent += len(pcmData)
	cb := f.onResult
	f.mu.Unlock()

	if cb != nil && len(pcmData) > 0 {
		cb("hello", false, 0, "uuid")
		cb("hello world", true, 0, "uuid")
	}
	return nil
}

func (f *fakeASR) SendEnd() error  { return nil }
func (f *fakeASR) StopConn() error { return nil }

var _ recognizer.SpeechRecognitionEngine = (*fakeASR)(nil)

func TestPipeline_ProcessPCM_EmitsText(t *testing.T) {
	fake := &fakeASR{}
	pipe, err := New(Options{ASR: fake, SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu    sync.Mutex
		texts []string
	)
	pipe.SetTextCallback(func(text string, isFinal bool) {
		mu.Lock()
		texts = append(texts, text)
		mu.Unlock()
	})

	ctx := context.Background()
	pcm := make([]byte, 3200) // 100ms @ 16kHz mono 16-bit
	if err := pipe.ProcessPCM(ctx, pcm); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(texts) == 0 {
		t.Fatal("expected at least one ASR callback")
	}
	if texts[len(texts)-1] != "hello world" {
		t.Fatalf("got %q, want %q", texts[len(texts)-1], "hello world")
	}
}

func TestPipeline_GetMetrics(t *testing.T) {
	fake := &fakeASR{}
	pipe, err := New(Options{ASR: fake, SampleRate: 16000})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	pcm := make([]byte, 640)
	_ = pipe.ProcessPCM(ctx, pcm)

	m := pipe.GetMetrics()
	if m.TotalAudioBytes != len(pcm) {
		t.Fatalf("TotalAudioBytes=%d want %d", m.TotalAudioBytes, len(pcm))
	}
	if m.ASRFirstResult.IsZero() {
		t.Fatal("expected ASRFirstResult to be set")
	}
	if m.ASRLatency < 0 {
		t.Fatalf("negative latency: %v", m.ASRLatency)
	}
	_ = time.Now() // keep time import used on older Go toolchains
}
