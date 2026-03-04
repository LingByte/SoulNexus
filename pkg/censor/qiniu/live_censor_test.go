package qiniu

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLiveCensorRequest_Marshal(t *testing.T) {
	req := LiveCensorRequest{
		Data: LiveCensorData{
			ID:  "live-censor",
			URI: "rtmp://pili-live-rtmp.qiniu.co/live/censor",
			Info: map[string]interface{}{
				"live_id": "example",
			},
		},
		Params: LiveCensorParams{
			HookURL:  "http://requestbin.fullcontact.com/1mhqoh41",
			HookAuth: true,
			Image: &LiveImageParams{
				IsOn:          true,
				Scenes:        []string{"pulp", "terror", "politician"},
				IntervalMsecs: 1000,
				Saver: &LiveSaverConfig{
					Bucket: "live",
					Prefix: "image/",
				},
				HookRule: 0,
			},
			Audio: &LiveAudioParams{
				IsOn:   true,
				Scenes: []string{"antispam"},
				Saver: &LiveSaverConfig{
					Bucket: "live",
					Prefix: "audio/",
				},
				HookRule: 0,
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Verify JSON contains necessary fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "live-censor") {
		t.Error("ID was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "rtmp://") {
		t.Error("URI was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "pulp") {
		t.Error("Image Scenes was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "antispam") {
		t.Error("Audio Scenes was not serialized correctly")
	}
}

func TestNewLiveCensorClient(t *testing.T) {
	client := NewLiveCensorClient("test_access_key", "test_secret_key")
	if client.AccessKey != "test_access_key" {
		t.Error("AccessKey was set incorrectly")
	}
	if client.SecretKey != "test_secret_key" {
		t.Error("SecretKey was set incorrectly")
	}
	if client.Host != LiveCensorHost {
		t.Errorf("Host was set incorrectly: expected %s, got %s", LiveCensorHost, client.Host)
	}
	if client.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}

func TestGetJobInfo_EmptyJobID(t *testing.T) {
	client := NewLiveCensorClient("test_access_key", "test_secret_key")
	_, err := client.GetJobInfo("")
	if err == nil {
		t.Error("Expected error when jobID is empty")
	}
}

func TestCloseJob_EmptyJobID(t *testing.T) {
	client := NewLiveCensorClient("test_access_key", "test_secret_key")
	_, err := client.CloseJob("")
	if err == nil {
		t.Error("Expected error when jobID is empty")
	}
}
