package voice

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	siprtp "github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
)

// fakeASR satisfies recognizer.TranscribeService without any network.
type fakeASR struct {
	mu       sync.Mutex
	tr       recognizer.TranscribeResult
	er       recognizer.ProcessError
	conn     int32
	sent     int32
}

func (f *fakeASR) Init(tr recognizer.TranscribeResult, er recognizer.ProcessError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tr, f.er = tr, er
}
func (f *fakeASR) Vendor() string                 { return "fake" }
func (f *fakeASR) ConnAndReceive(string) error    { atomic.AddInt32(&f.conn, 1); return nil }
func (f *fakeASR) Activity() bool                 { return true }
func (f *fakeASR) RestartClient()                 {}
func (f *fakeASR) SendAudioBytes(b []byte) error  { atomic.AddInt32(&f.sent, int32(len(b))); return nil }
func (f *fakeASR) SendEnd() error                 { return nil }
func (f *fakeASR) StopConn() error                { return nil }

func pcmaOffer() []sdp.Codec {
	return []sdp.Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
		{PayloadType: 101, Name: "telephone-event", ClockRate: 8000},
	}
}

func TestAttach_ASRAndTTS(t *testing.T) {
	s, err := siprtp.NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	leg, err := session.NewMediaLeg(context.Background(), "c1", s, pcmaOffer(), session.MediaLegConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer leg.Stop()

	fa := &fakeASR{}
	ttsChunks := [][]byte{make([]byte, 1280)}

	att, err := Attach(context.Background(), leg, AttachConfig{
		ASR:           fa,
		ASRSampleRate: 8000,
		TTSService: tts.ServiceFunc(func(ctx context.Context, _ string, on func([]byte) error) error {
			for _, c := range ttsChunks {
				if err := on(c); err != nil {
					return err
				}
			}
			return nil
		}),
		TTSInputRate: 8000,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer att.Close()

	if att.ASR == nil || att.TTS == nil {
		t.Fatal("pipelines not attached")
	}

	// TTS path: Speak should deliver frames via MediaSession output queue without error.
	done := make(chan error, 1)
	go func() { done <- att.TTS.Speak("hi") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Speak: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Speak did not complete")
	}
}

func TestAttach_NilLeg(t *testing.T) {
	if _, err := Attach(context.Background(), nil, AttachConfig{}); err == nil {
		t.Fatal("want error on nil leg")
	}
}

func TestAttach_OnlyASR(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	defer s.Close()
	leg, _ := session.NewMediaLeg(context.Background(), "c1", s, pcmaOffer(), session.MediaLegConfig{})
	defer leg.Stop()
	att, err := Attach(context.Background(), leg, AttachConfig{ASR: &fakeASR{}})
	if err != nil {
		t.Fatal(err)
	}
	if att.ASR == nil || att.TTS != nil {
		t.Fatal("expected ASR-only wiring")
	}
	att.Close()
	att.Close() // idempotent
}
