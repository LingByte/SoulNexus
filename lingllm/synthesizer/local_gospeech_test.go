package synthesizer

import (
	"testing"
	"time"
)

func TestLocalGoSpeechConfigGetProvider(t *testing.T) {
	config := &LocalGoSpeechConfig{
		Provider: "espeak",
	}

	provider := config.GetProvider()
	if provider != ProviderLocalGoSpeech {
		t.Errorf("GetProvider() = %v, want %v", provider, ProviderLocalGoSpeech)
	}
}

func TestNewLocalGoSpeechConfig(t *testing.T) {
	config := NewLocalGoSpeechConfig("espeak", "/path/to/model")

	if config == nil {
		t.Error("NewLocalGoSpeechConfig should not return nil")
	}

	if config.Provider != "espeak" {
		t.Errorf("Provider = %s, want espeak", config.Provider)
	}

	if config.ModelPath != "/path/to/model" {
		t.Errorf("ModelPath = %s, want /path/to/model", config.ModelPath)
	}

	if config.Language != "zh-CN" {
		t.Errorf("Language = %s, want zh-CN", config.Language)
	}

	if config.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", config.SampleRate)
	}
}

func TestLocalGoSpeechConfigDefaults(t *testing.T) {
	config := NewLocalGoSpeechConfig("festival", "")

	if config.Speaker != "default" {
		t.Errorf("Speaker = %s, want default", config.Speaker)
	}

	if config.Speed != 1.0 {
		t.Errorf("Speed = %f, want 1.0", config.Speed)
	}

	if config.Pitch != 1.0 {
		t.Errorf("Pitch = %f, want 1.0", config.Pitch)
	}

	if config.Volume != 1.0 {
		t.Errorf("Volume = %f, want 1.0", config.Volume)
	}

	if !config.EnableCache {
		t.Error("EnableCache should be true by default")
	}

	if config.CacheExpiry != 24*time.Hour {
		t.Errorf("CacheExpiry = %v, want 24h", config.CacheExpiry)
	}
}
