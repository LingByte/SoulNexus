package synthesizer

import (
	"testing"
	"time"
)

func TestComputeSampleByteCount(t *testing.T) {
	tests := []struct {
		sampleRate int
		bitDepth   int
		channels   int
		want       int
	}{
		{16000, 16, 1, 32000},  // 16000 * 16 * 1 / 8 = 32000
		{16000, 16, 2, 64000},  // 16000 * 16 * 2 / 8 = 64000
		{48000, 16, 1, 96000},  // 48000 * 16 * 1 / 8 = 96000
		{8000, 8, 1, 8000},     // 8000 * 8 * 1 / 8 = 8000
		{44100, 16, 2, 176400}, // 44100 * 16 * 2 / 8 = 176400
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ComputeSampleByteCount(tt.sampleRate, tt.bitDepth, tt.channels)
			if got != tt.want {
				t.Errorf("ComputeSampleByteCount(%d, %d, %d) = %d, want %d",
					tt.sampleRate, tt.bitDepth, tt.channels, got, tt.want)
			}
		})
	}
}

func TestNormalizeFramePeriod(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"20ms", true},
		{"50ms", true},
		{"100ms", true},
		{"5ms", false},     // Too small, should default to 20ms
		{"500ms", false},   // Too large, should default to 20ms
		{"invalid", false}, // Invalid, should default to 20ms
		{"", false},        // Empty, should default to 20ms
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeFramePeriod(tt.input)

			if tt.valid {
				// For valid inputs, check they're in reasonable range
				if got < 10*time.Millisecond || got > 300*time.Millisecond {
					t.Errorf("NormalizeFramePeriod(%s) = %v, outside valid range", tt.input, got)
				}
			} else {
				// For invalid inputs, should default to 20ms
				if got != 20*time.Millisecond {
					t.Errorf("NormalizeFramePeriod(%s) = %v, want 20ms", tt.input, got)
				}
			}
		})
	}
}

func TestWordStruct(t *testing.T) {
	word := Word{
		Confidence: 0.95,
		EndTime:    1000,
		StartTime:  500,
		Word:       "hello",
	}

	if word.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want 0.95", word.Confidence)
	}

	if word.Word != "hello" {
		t.Errorf("Word = %s, want hello", word.Word)
	}
}

func TestSentenceTimestamp(t *testing.T) {
	timestamp := SentenceTimestamp{
		Words: []Word{
			{Confidence: 0.9, EndTime: 100, StartTime: 0, Word: "hello"},
			{Confidence: 0.95, EndTime: 200, StartTime: 100, Word: "world"},
		},
	}

	if len(timestamp.Words) != 2 {
		t.Errorf("Words length = %d, want 2", len(timestamp.Words))
	}

	if timestamp.Words[0].Word != "hello" {
		t.Errorf("First word = %s, want hello", timestamp.Words[0].Word)
	}
}
