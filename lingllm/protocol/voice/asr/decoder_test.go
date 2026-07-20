package asr

import (
	"context"
	"testing"
)

func TestDecoderComponentCreation(t *testing.T) {
	// Use PCMU instead of OPUS as OPUS requires build tag
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
		FrameDuration:    "20ms",
	}
	decoder, err := NewDecoderComponent(config)

	if err != nil {
		t.Errorf("NewDecoderComponent failed: %v", err)
	}

	if decoder == nil {
		t.Error("DecoderComponent should not be nil")
	}

	if decoder.Name() != "decoder_pcmu_to_pcm" {
		t.Errorf("Name = %s, want 'decoder_pcmu_to_pcm'", decoder.Name())
	}
}

func TestDecoderComponentCustomConfig(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
		FrameDuration:    "20ms",
	}
	decoder, err := NewDecoderComponent(config)

	if err != nil {
		t.Errorf("NewDecoderComponent failed: %v", err)
	}

	if decoder.Name() != "decoder_pcmu_to_pcm" {
		t.Errorf("Name = %s, want 'decoder_pcmu_to_pcm'", decoder.Name())
	}

	retrievedConfig := decoder.GetConfig()
	if retrievedConfig.SourceCodec != "pcmu" {
		t.Errorf("SourceCodec = %s, want 'pcmu'", retrievedConfig.SourceCodec)
	}

	if retrievedConfig.SourceSampleRate != 8000 {
		t.Errorf("SourceSampleRate = %d, want 8000", retrievedConfig.SourceSampleRate)
	}
}

func TestDecoderComponentDefaultConfig(t *testing.T) {
	config := DefaultDecoderConfig()

	// DefaultDecoderConfig defaults to opus, but we just verify the structure
	if config.SourceCodec == "" {
		t.Error("SourceCodec should not be empty")
	}

	if config.SourceSampleRate == 0 {
		t.Error("SourceSampleRate should not be 0")
	}

	if config.TargetSampleRate == 0 {
		t.Error("TargetSampleRate should not be 0")
	}

	if config.TargetChannels == 0 {
		t.Error("TargetChannels should not be 0")
	}
}

func TestDecoderComponentInvalidDataType(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
	}
	decoder, _ := NewDecoderComponent(config)

	_, _, err := decoder.Process(context.Background(), "invalid")
	if err == nil {
		t.Error("Process with invalid data type should return error")
	}
}

func TestDecoderComponentEmptyData(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
	}
	decoder, _ := NewDecoderComponent(config)

	_, _, err := decoder.Process(context.Background(), []byte{})
	if err == nil {
		t.Error("Process with empty data should return error")
	}
}

func TestDecoderComponentSetLogger(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
	}
	decoder, _ := NewDecoderComponent(config)

	logCount := 0
	decoder.SetLogger(func(msg string) {
		logCount++
	})

	// Logger should be set
	if logCount != 0 {
		t.Error("Logger should not be called during SetLogger")
	}
}

func TestDecoderComponentGetConfig(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
		FrameDuration:    "20ms",
	}
	decoder, _ := NewDecoderComponent(config)

	retrievedConfig := decoder.GetConfig()

	if retrievedConfig.SourceCodec != config.SourceCodec {
		t.Errorf("SourceCodec mismatch: got %s, want %s", retrievedConfig.SourceCodec, config.SourceCodec)
	}

	if retrievedConfig.SourceSampleRate != config.SourceSampleRate {
		t.Errorf("SourceSampleRate mismatch: got %d, want %d", retrievedConfig.SourceSampleRate, config.SourceSampleRate)
	}

	if retrievedConfig.TargetSampleRate != config.TargetSampleRate {
		t.Errorf("TargetSampleRate mismatch: got %d, want %d", retrievedConfig.TargetSampleRate, config.TargetSampleRate)
	}
}

func TestDecoderComponentUnsupportedCodec(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "unsupported_codec",
		SourceSampleRate: 16000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
	}

	_, err := NewDecoderComponent(config)
	if err == nil {
		t.Error("NewDecoderComponent with unsupported codec should return error")
	}
}

func TestDecoderComponentDefaultsApplied(t *testing.T) {
	config := DecoderConfig{
		SourceCodec:      "pcmu",
		SourceSampleRate: 8000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
	}
	decoder, err := NewDecoderComponent(config)

	if err != nil {
		t.Errorf("NewDecoderComponent with config should work: %v", err)
	}

	retrievedConfig := decoder.GetConfig()

	if retrievedConfig.SourceCodec != "pcmu" {
		t.Errorf("SourceCodec = %s, want 'pcmu'", retrievedConfig.SourceCodec)
	}

	if retrievedConfig.SourceSampleRate != 8000 {
		t.Errorf("SourceSampleRate = %d, want 8000", retrievedConfig.SourceSampleRate)
	}

	if retrievedConfig.TargetSampleRate != 16000 {
		t.Errorf("TargetSampleRate = %d, want 16000", retrievedConfig.TargetSampleRate)
	}
}

func TestDecoderComponentName(t *testing.T) {
	testCases := []struct {
		codec    string
		expected string
	}{
		{"opus", "decoder_opus_to_pcm"},
		{"pcmu", "decoder_pcmu_to_pcm"},
		{"pcma", "decoder_pcma_to_pcm"},
		{"g722", "decoder_g722_to_pcm"},
	}

	for _, tc := range testCases {
		config := DecoderConfig{
			SourceCodec:      tc.codec,
			SourceSampleRate: 16000,
			SourceChannels:   1,
			TargetSampleRate: 16000,
			TargetChannels:   1,
		}

		decoder, err := NewDecoderComponent(config)
		if err != nil {
			// Skip if codec not supported
			continue
		}

		if decoder.Name() != tc.expected {
			t.Errorf("Name for codec %s = %s, want %s", tc.codec, decoder.Name(), tc.expected)
		}
	}
}
