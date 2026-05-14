package dsp

import (
	"encoding/binary"
	"math"
	"testing"
)

// helper: encode int16 samples as little-endian PCM
func pcmLE(samples ...int16) []byte {
	buf := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	return buf
}

func TestRMSPCM16LE_Empty(t *testing.T) {
	if got := RMSPCM16LE(nil); got != 0 {
		t.Fatalf("nil buf: want 0, got %v", got)
	}
	if got := RMSPCM16LE([]byte{}); got != 0 {
		t.Fatalf("empty buf: want 0, got %v", got)
	}
	if got := RMSPCM16LE([]byte{0x01}); got != 0 {
		t.Fatalf("single byte: want 0, got %v", got)
	}
}

func TestRMSPCM16LE_Silence(t *testing.T) {
	buf := pcmLE(0, 0, 0, 0, 0, 0, 0, 0)
	if got := RMSPCM16LE(buf); got != 0 {
		t.Fatalf("silence rms: want 0, got %v", got)
	}
}

func TestRMSPCM16LE_DCConstant(t *testing.T) {
	// 8 samples all = 1000 → RMS = 1000
	buf := pcmLE(1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000)
	got := RMSPCM16LE(buf)
	if math.Abs(got-1000) > 1e-9 {
		t.Fatalf("dc=1000: want 1000, got %v", got)
	}
}

func TestRMSPCM16LE_KnownSquareWave(t *testing.T) {
	// alternating ±2000 → RMS = 2000
	buf := pcmLE(2000, -2000, 2000, -2000, 2000, -2000)
	got := RMSPCM16LE(buf)
	if math.Abs(got-2000) > 1e-9 {
		t.Fatalf("square ±2000: want 2000, got %v", got)
	}
}

func TestRMSPCM16LE_OddByteCount_DropsLast(t *testing.T) {
	// 4 valid samples + 1 stray byte (should be ignored)
	buf := append(pcmLE(100, 100, 100, 100), 0xFF)
	got := RMSPCM16LE(buf)
	if math.Abs(got-100) > 1e-9 {
		t.Fatalf("odd-byte buf: want 100, got %v", got)
	}
}

func TestRMSPCM16LE_NegativeAndPositiveSymmetric(t *testing.T) {
	// −1000 and +1000 → RMS = 1000
	buf := pcmLE(-1000, 1000)
	got := RMSPCM16LE(buf)
	if math.Abs(got-1000) > 1e-9 {
		t.Fatalf("symmetric: want 1000, got %v", got)
	}
}
