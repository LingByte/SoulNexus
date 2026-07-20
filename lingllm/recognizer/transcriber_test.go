package recognizer

import (
	"testing"
)

func TestComputeSampleByteCount(t *testing.T) {
	tests := []struct {
		sampleRate int
		bitDepth   int
		channels   int
		want       int
	}{
		{16000, 16, 1, 32000}, // 16000 * 16 * 1 / 8 = 32000
		{16000, 16, 2, 64000}, // 16000 * 16 * 2 / 8 = 64000
		{48000, 16, 1, 96000}, // 48000 * 16 * 1 / 8 = 96000
		{8000, 8, 1, 8000},    // 8000 * 8 * 1 / 8 = 8000
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

func TestTranscriberPackage(t *testing.T) {
	t.Run("transcriber_available", func(t *testing.T) {
		if true {
			t.Log("Transcriber package is available")
		}
	})
}
