package qiniu

import (
	"encoding/json"
	"testing"
)

func TestImageCensorRequest_Marshal(t *testing.T) {
	req := ImageCensorRequest{
		Data: ImageCensorData{
			URI: "https://mars-assets.qnssl.com/resource/gogopher.jpg",
		},
		Params: ImageCensorParams{
			Scenes: []string{"pulp", "terror", "politician"},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	expected := `{"data":{"uri":"https://mars-assets.qnssl.com/resource/gogopher.jpg"},"params":{"scenes":["pulp","terror","politician"]}}`
	actual := string(jsonData)

	// Since JSON serialization order may differ, we compare after deserializing
	var req1, req2 ImageCensorRequest
	if err := json.Unmarshal([]byte(expected), &req1); err != nil {
		t.Fatalf("Failed to deserialize expected JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(actual), &req2); err != nil {
		t.Fatalf("Failed to deserialize actual JSON: %v", err)
	}

	if req1.Data.URI != req2.Data.URI {
		t.Errorf("URI mismatch: expected %s, got %s", req1.Data.URI, req2.Data.URI)
	}

	if len(req1.Params.Scenes) != len(req2.Params.Scenes) {
		t.Errorf("Scenes length mismatch: expected %d, got %d", len(req1.Params.Scenes), len(req2.Params.Scenes))
	}
}

func TestNewImageCensorClient(t *testing.T) {
	client := NewImageCensorClient("test_access_key", "test_secret_key")
	if client.AccessKey != "test_access_key" {
		t.Error("AccessKey was set incorrectly")
	}
	if client.SecretKey != "test_secret_key" {
		t.Error("SecretKey was set incorrectly")
	}
	if client.Host != ImageCensorHost {
		t.Errorf("Host was set incorrectly: expected %s, got %s", ImageCensorHost, client.Host)
	}
	if client.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}
