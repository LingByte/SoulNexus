package qiniu

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVideoCensorRequest_Marshal(t *testing.T) {
	req := VideoCensorRequest{
		Data: VideoCensorData{
			URI: "https://mars-assets.qnssl.com/scene.mp4",
			ID:  "video_censor_test",
		},
		Params: VideoCensorParams{
			Scenes: []string{"pulp", "terror", "politician"},
			CutParam: &VideoCutParam{
				IntervalMsecs: 5000,
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Verify JSON contains necessary fields
	jsonStr := string(jsonData)
	if !contains(jsonStr, "scene.mp4") {
		t.Error("URI was not serialized correctly")
	}
	if !contains(jsonStr, "video_censor_test") {
		t.Error("ID was not serialized correctly")
	}
	if !contains(jsonStr, "pulp") {
		t.Error("Scenes was not serialized correctly")
	}
	if !contains(jsonStr, "5000") {
		t.Error("IntervalMsecs was not serialized correctly")
	}
}

func TestNewVideoCensorClient(t *testing.T) {
	client := NewVideoCensorClient("test_access_key", "test_secret_key")
	if client.AccessKey != "test_access_key" {
		t.Error("AccessKey was set incorrectly")
	}
	if client.SecretKey != "test_secret_key" {
		t.Error("SecretKey was set incorrectly")
	}
	if client.Host != VideoCensorHost {
		t.Errorf("Host was set incorrectly: expected %s, got %s", VideoCensorHost, client.Host)
	}
	if client.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}

func TestGetCensorResult_EmptyJobID(t *testing.T) {
	client := NewVideoCensorClient("test_access_key", "test_secret_key")
	_, err := client.GetCensorResult("")
	if err == nil {
		t.Error("Expected error when jobID is empty")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
