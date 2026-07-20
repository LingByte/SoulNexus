package synthesizer

import (
	"testing"
)

func TestMinimaxOptionGetProvider(t *testing.T) {
	option := &MinimaxOption{
		APIKey: "test-key",
	}

	provider := option.GetProvider()
	if provider != ProviderMinimax {
		t.Errorf("GetProvider() = %v, want %v", provider, ProviderMinimax)
	}
}

func TestNewMinimaxOption(t *testing.T) {
	option := NewMinimaxOption("test-api-key")

	if option.APIKey != "test-api-key" {
		t.Errorf("APIKey = %s, want test-api-key", option.APIKey)
	}

	if option.Model != MinimaxSpeech25TurboPreview {
		t.Errorf("Model = %s, want %s", option.Model, MinimaxSpeech25TurboPreview)
	}

	if option.VoiceID != "male-qn-qingse" {
		t.Errorf("VoiceID = %s, want male-qn-qingse", option.VoiceID)
	}
}

func TestMinimaxOptionDefaults(t *testing.T) {
	option := NewMinimaxOption("key")

	if option.SpeedRatio != 1.0 {
		t.Errorf("SpeedRatio = %f, want 1.0", option.SpeedRatio)
	}

	if option.Volume != 1.0 {
		t.Errorf("Volume = %f, want 1.0", option.Volume)
	}

	if option.Pitch != 0.0 {
		t.Errorf("Pitch = %f, want 0.0", option.Pitch)
	}

	if option.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", option.SampleRate)
	}

	if option.Channels != 1 {
		t.Errorf("Channels = %d, want 1", option.Channels)
	}
}

func TestMinimaxOptionString(t *testing.T) {
	option := NewMinimaxOption("key")
	str := option.String()

	if str == "" {
		t.Error("String() should not return empty string")
	}

	if !contains(str, "MinimaxOption") {
		t.Errorf("String() should contain 'MinimaxOption', got %s", str)
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
