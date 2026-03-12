package qiniu

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestAudioCensorRequest_Marshal(t *testing.T) {
	req := AudioCensorRequest{
		Data: AudioCensorData{
			URI: "http://xxx.mp3",
			ID:  "audio-censor-demo",
		},
		Params: AudioCensorParams{
			Scenes:   []string{"antispam"},
			HookURL:  "http://xxx.com",
			HookAuth: false,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Verify JSON contains necessary fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "xxx.mp3") {
		t.Error("URI was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "audio-censor-demo") {
		t.Error("ID was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "antispam") {
		t.Error("Scenes was not serialized correctly")
	}
}

func TestNewAudioCensorClient(t *testing.T) {
	client := NewAudioCensorClient("test_access_key", "test_secret_key")
	if client.AccessKey != "test_access_key" {
		t.Error("AccessKey was set incorrectly")
	}
	if client.SecretKey != "test_secret_key" {
		t.Error("SecretKey was set incorrectly")
	}
	if client.Host != AudioCensorHost {
		t.Errorf("Host was set incorrectly: expected %s, got %s", AudioCensorHost, client.Host)
	}
	if client.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}

func TestGetCensorResult_EmptyTaskID(t *testing.T) {
	ak := utils.GetEnv("QINIU_ACCESS_KEY")
	sk := utils.GetEnv("QINIU_SECRET_KEY")
	if ak == "" || sk == "" {
		t.Skip("QINIU_ACCESS_KEY or QINIU_SECRET_KEY not set, skipping test")
	}
	client := NewAudioCensorClient(ak, sk)
	_, err := client.GetCensorResult("")
	if err == nil {
		t.Error("Expected error when taskID is empty")
	}
}
