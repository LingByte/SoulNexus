package dialog

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/tts"
)

type mockEngine struct {
	onResult func(text string, isFinal bool)
	onError  func(err error, fatal bool)
	ch       chan []byte
}

func (m *mockEngine) Start() error {
	m.ch = make(chan []byte, 8)
	go func() {
		for range m.ch {
			if m.onResult != nil {
				m.onResult("你好", true)
			}
		}
	}()
	return nil
}

func (m *mockEngine) Stop() {}

func (m *mockEngine) SendPCM(pcm []byte, _ bool) error {
	if m.ch != nil && len(pcm) > 0 {
		m.ch <- pcm
	}
	return nil
}

func (m *mockEngine) OnResult(cb func(text string, isFinal bool)) { m.onResult = cb }
func (m *mockEngine) OnError(cb func(err error, fatal bool))      { m.onError = cb }

type mockTTS struct{}

func (mockTTS) Synthesize(ctx context.Context, text string, cb func([]byte) error) error {
	return cb(make([]byte, 1920))
}

func TestSessionASRAndTTS(t *testing.T) {
	var mu sync.Mutex
	var events []Event
	var audioOut int

	sess, err := NewSession(context.Background(), Config{
		CallID:     "test-1",
		Meta:       StartMeta{PCMHz: 16000, Codec: "pcm"},
		Engine:     &mockEngine{},
		TTSService: mockTTS{},
		OnAudioOut: func([]byte) error {
			audioOut++
			return nil
		},
		OnEvent: func(ev Event) {
			mu.Lock()
			events = append(events, ev)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// ≥20ms @ 16kHz PCM16 (640 bytes) so uplink coalescer reaches ASR.
	pcm := make([]byte, 640)
	if err := sess.ProcessAudio(ctx, pcm); err != nil {
		t.Fatalf("ProcessAudio: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	sess.HandleCommand(Command{
		Type:        CmdTTSSpeak,
		CallID:      "test-1",
		Text:        "您好",
		UtteranceID: "u1",
	})
	time.Sleep(50 * time.Millisecond)

	sess.Close("test-done")

	mu.Lock()
	defer mu.Unlock()

	hasStarted := false
	hasASRFinal := false
	hasTTSEnded := false
	for _, ev := range events {
		switch ev.Type {
		case EvCallStarted:
			hasStarted = true
		case EvASRFinal:
			hasASRFinal = true
		case EvTTSEnded:
			hasTTSEnded = true
		}
	}
	if !hasStarted {
		t.Error("expected call.started event")
	}
	if !hasASRFinal {
		t.Error("expected asr.final event")
	}
	if !hasTTSEnded {
		t.Error("expected tts.ended event")
	}
	if audioOut == 0 {
		t.Error("expected downlink audio")
	}
}

func TestSessionSentenceBoundaryTriggersFinal(t *testing.T) {
	var mu sync.Mutex
	var finals []string

	eng := &mockEngine{}
	sess, err := NewSession(context.Background(), Config{
		CallID:     "boundary-final",
		Engine:     eng,
		TTSService: mockTTS{},
		OnAudioOut: func([]byte) error { return nil },
		OnEvent: func(ev Event) {
			if ev.Type == EvASRFinal {
				mu.Lock()
				finals = append(finals, ev.Text)
				mu.Unlock()
			}
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if eng.onResult == nil {
		t.Fatal("engine onResult not wired")
	}
	eng.onResult("你好，听得到吗？", false)

	mu.Lock()
	defer mu.Unlock()
	if len(finals) != 1 || finals[0] != "你好，听得到吗？" {
		t.Fatalf("expected one asr.final, got %v", finals)
	}
}

func TestSentenceFilterIntegration(t *testing.T) {
	f := asr.NewSentenceFilter(0)
	delta := f.Update("今天天气很好。", false)
	if delta == "" {
		t.Error("expected sentence-boundary partial emit")
	}
}

func TestSessionDefaultSentenceFilter(t *testing.T) {
	sess, err := NewSession(context.Background(), Config{
		CallID:     "sf-default",
		Engine:     &mockEngine{},
		TTSService: mockTTS{},
		OnAudioOut: func([]byte) error { return nil },
		OnEvent:    func(Event) {},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if sess.filter == nil {
		t.Fatal("expected default sentence filter")
	}
}

func TestSessionStreamSegmentation(t *testing.T) {
	var mu sync.Mutex
	var speaks []string

	sess, err := NewSession(context.Background(), Config{
		CallID:     "stream-1",
		Engine:     &mockEngine{},
		TTSService: mockTTS{},
		OnAudioOut: func([]byte) error { return nil },
		OnEvent: func(ev Event) {
			if ev.Type == EvTTSStarted {
				mu.Lock()
				speaks = append(speaks, ev.UtteranceID)
				mu.Unlock()
			}
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	sess.HandleCommand(Command{
		Type:        CmdTTSStream,
		CallID:      "stream-1",
		UtteranceID: "u-stream",
		Text:        "你好。",
	})
	sess.HandleCommand(Command{
		Type:        CmdTTSStream,
		CallID:      "stream-1",
		UtteranceID: "u-stream",
		Text:        "今天天气不错！",
		StreamEnd:   true,
	})
	time.Sleep(80 * time.Millisecond)
	sess.Close("test")

	mu.Lock()
	defer mu.Unlock()
	if len(speaks) == 0 {
		t.Fatal("expected at least one tts.started from stream segmentation")
	}
}

func TestSessionDenoiserInPipeline(t *testing.T) {
	stub := &stubDenoiser{}
	sess, err := NewSession(context.Background(), Config{
		CallID:     "denoise-1",
		Engine:     &mockEngine{},
		TTSService: mockTTS{},
		Denoiser:   stub,
		OnAudioOut: func([]byte) error { return nil },
		OnEvent:    func(Event) {},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	pcm := make([]byte, 640)
	_ = sess.ProcessAudio(ctx, pcm)
	time.Sleep(10 * time.Millisecond)
	sess.Close("test")
	if !stub.called {
		t.Error("expected denoiser to process uplink PCM")
	}
}

type stubDenoiser struct {
	called bool
}

func (s *stubDenoiser) Process(pcm []byte) []byte {
	s.called = true
	out := make([]byte, len(pcm))
	copy(out, pcm)
	return out
}

func TestTTSSpeakerInterrupt(t *testing.T) {
	p, err := tts.NewTTSPipeline(tts.TTSPipelineConfig{
		TTSService:   mockTTS{},
		SendCallback: func([]byte) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Start(context.Background())
	sp, _ := tts.NewSpeaker(tts.SpeakerConfig{Pipeline: p})
	sp.Start(context.Background())
	sp.Enqueue("hello", "u1", nil)
	sp.Interrupt()
	sp.Stop()
}
