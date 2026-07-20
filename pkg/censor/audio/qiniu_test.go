package audio

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestNewQiniuAudioCensor(t *testing.T) {
	c := NewQiniuAudioCensor("test-ak", "test-sk")
	if c.AccessKey != "test-ak" {
		t.Errorf("AccessKey = %q, want test-ak", c.AccessKey)
	}
	if c.SecretKey != "test-sk" {
		t.Errorf("SecretKey = %q, want test-sk", c.SecretKey)
	}
	if c.Host != AudioCensorHost {
		t.Errorf("Host = %q, want %q", c.Host, AudioCensorHost)
	}
	if c.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}

func TestAudioCensorRequest_Marshal(t *testing.T) {
	req := AudioCensorRequest{
		Data: AudioCensorData{
			URI: "https://example.com/audio.mp3",
			ID:  "audio-1",
		},
		Params: AudioCensorParams{
			Scenes: []string{"antispam"},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "https://example.com/audio.mp3") {
		t.Error("URI was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "antispam") {
		t.Error("scenes was not serialized correctly")
	}
}

func TestQiniuAudioCensor_SubmitCensorAudio_EmptyURL(t *testing.T) {
	c := NewQiniuAudioCensor("ak", "sk")
	_, err := c.SubmitCensorAudio("")
	if err == nil {
		t.Fatal("expected error for empty audioURL")
	}
}

func TestQiniuAudioCensor_GetCensorResult_EmptyTaskID(t *testing.T) {
	c := NewQiniuAudioCensor("ak", "sk")
	_, err := c.GetCensorResult("")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
	_, err = c.GetCensorResultFull("")
	if err == nil {
		t.Fatal("expected error for empty taskID in GetCensorResultFull")
	}
}

func TestQiniuAudioCensor_Network(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("QINIU_ACCESS_KEY or QINIU_SECRET_KEY not set")
	}
	c := NewQiniuAudioCensor(accessKey, secretKey)
	_, err := c.SubmitCensorAudio("https://example.com/audio.mp3")
	if err != nil {
		t.Logf("SubmitCensorAudio (network): %v", err)
	}
}
