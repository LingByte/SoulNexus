package volcdialogue

import (
	"encoding/json"
	"testing"

	"github.com/LingByte/lingllm/realtime"
)

func TestMarshalParseStartConnection(t *testing.T) {
	raw, err := marshalJSONEvent(eventStartConnection, "", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := parseFrame(raw)
	if err != nil {
		t.Fatal(err)
	}
	if f.event != eventStartConnection {
		t.Fatalf("event=%d", f.event)
	}
}

func TestMarshalParseAudioTask(t *testing.T) {
	pcm := make([]byte, 640)
	for i := range pcm {
		pcm[i] = byte(i % 256)
	}
	raw, err := marshalAudioTask("sess-1", pcm)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parseFrame(raw)
	if err != nil {
		t.Fatal(err)
	}
	if f.event != eventTaskRequest {
		t.Fatalf("event=%d", f.event)
	}
	if f.sessionID != "sess-1" {
		t.Fatalf("session=%q", f.sessionID)
	}
	if len(f.payload) != len(pcm) {
		t.Fatalf("payload len=%d want %d", len(f.payload), len(pcm))
	}
}

func TestBuildStartSessionJSON(t *testing.T) {
	p := startSessionPayload{
		ASR: asrPayload{Format: "pcm", Rate: 16000, Bits: 16, Channel: 1},
		TTS: ttsPayload{
			Speaker:     "zh_female_vv_jupiter_bigtts",
			AudioConfig: audioConfig{Channel: 1, Format: "pcm_s16le", SampleRate: 24000},
		},
		Dialog: dialogPayload{
			BotName: "test", SystemRole: "hi", Extra: map[string]any{"model": "1.2.1.1"},
		},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatal("invalid json")
	}
}

func TestNewRequiresCredentials(t *testing.T) {
	_, err := New(map[string]any{"provider": ProviderSlug}, realtime.Options{
		OnEvent: func(realtime.Event) {},
	})
	if err == nil {
		t.Fatal("expected error without credentials")
	}
}
