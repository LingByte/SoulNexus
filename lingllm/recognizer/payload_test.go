package recognizer

import (
	"encoding/json"
	"testing"
)

func TestUserMeta(t *testing.T) {
	user := UserMeta{
		UID:        "user-123",
		DID:        "device-456",
		Platform:   "web",
		SDKVersion: "1.0.0",
		APPVersion: "2.0.0",
	}

	if user.UID != "user-123" {
		t.Errorf("UID = %s, want user-123", user.UID)
	}

	if user.Platform != "web" {
		t.Errorf("Platform = %s, want web", user.Platform)
	}
}

func TestAudioMeta(t *testing.T) {
	audio := AudioMeta{
		Format:  "pcm",
		Codec:   "raw",
		Rate:    16000,
		Bits:    16,
		Channel: 1,
	}

	if audio.Format != "pcm" {
		t.Errorf("Format = %s, want pcm", audio.Format)
	}

	if audio.Rate != 16000 {
		t.Errorf("Rate = %d, want 16000", audio.Rate)
	}
}

func TestRequestPayload(t *testing.T) {
	payload := RequestPayload{
		User: UserMeta{
			UID: "user-123",
		},
		Audio: AudioMeta{
			Rate: 16000,
		},
		Request: RequestMeta{
			ModelName: "default",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("JSON marshal failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled payload should not be empty")
	}
}

func TestNewFullClientRequest(t *testing.T) {
	config := &Config{
		User: UserConfig{
			UID: "user-123",
		},
		Audio: AudioConfig{
			Format:  "pcm",
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
		Request: RequestConfig{
			ModelName: "default",
		},
	}

	request := NewFullClientRequest(config)
	if len(request) == 0 {
		t.Error("NewFullClientRequest should return non-empty bytes")
	}
}

func TestNewAudioOnlyRequest(t *testing.T) {
	segment := []byte{0x01, 0x02, 0x03}

	// Test with positive sequence
	request := NewAudioOnlyRequest(1, segment)
	if len(request) == 0 {
		t.Error("NewAudioOnlyRequest should return non-empty bytes")
	}

	// Test with negative sequence (end frame)
	request = NewAudioOnlyRequest(-1, segment)
	if len(request) == 0 {
		t.Error("NewAudioOnlyRequest with negative seq should return non-empty bytes")
	}
}
