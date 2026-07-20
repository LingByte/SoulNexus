package tenantcfg

import (
	"encoding/json"
	"testing"
)

func TestApplyTTSVoiceAliyunInvalidFallback(t *testing.T) {
	raw := []byte(`{"provider":"aliyun","apiKey":"sk-test"}`)
	out := ApplyTTSVoice(raw, "101001")
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	got, _ := m["voice"].(string)
	if got != "Cherry" {
		t.Fatalf("voice=%q want Cherry fallback", got)
	}
}

func TestApplyTTSVoiceAliyunValid(t *testing.T) {
	raw := []byte(`{"provider":"aliyun","apiKey":"sk-test"}`)
	out := ApplyTTSVoice(raw, "Serena")
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	got, _ := m["voice"].(string)
	if got != "Serena" {
		t.Fatalf("voice=%q", got)
	}
}

func TestApplyTTSVoiceFishAudioReferenceID(t *testing.T) {
	raw := []byte(`{"provider":"fishaudio","apiKey":"fa-test"}`)
	out := ApplyTTSVoice(raw, "ca3007f96ae7499ab87d27ea3599956a")
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	got, _ := m["reference_id"].(string)
	if got != "ca3007f96ae7499ab87d27ea3599956a" {
		t.Fatalf("reference_id=%q", got)
	}
	if _, ok := m["voice"]; ok {
		t.Fatalf("unexpected voice key: %#v", m["voice"])
	}
}
