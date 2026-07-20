package recognizer

import (
	"testing"
)

func TestConfigCreation(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Auth: AuthConfig{
			ResourceId: "test-resource",
			AccessKey:  "test-key",
			AppKey:     "test-app",
		},
		User: UserConfig{
			UID:      "user-123",
			Platform: "web",
		},
		Audio: AudioConfig{
			Format:  "pcm",
			Codec:   "raw",
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
	}

	if config.URL == "" {
		t.Error("Config URL should not be empty")
	}

	if config.Auth.AccessKey == "" {
		t.Error("Config Auth.AccessKey should not be empty")
	}

	if config.Audio.Rate != 16000 {
		t.Errorf("Audio.Rate = %d, want 16000", config.Audio.Rate)
	}
}

func TestAudioConfig(t *testing.T) {
	audio := AudioConfig{
		Format:  "pcm",
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

func TestAuthConfig(t *testing.T) {
	auth := AuthConfig{
		ResourceId: "resource-123",
		AccessKey:  "access-key",
		AppKey:     "app-key",
	}

	if auth.ResourceId == "" {
		t.Error("ResourceId should not be empty")
	}

	if auth.AccessKey == "" {
		t.Error("AccessKey should not be empty")
	}
}
