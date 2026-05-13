// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audioasset

import (
	"encoding/binary"
	"testing"
)

// makeWAV builds a minimal canonical PCM WAV in memory for tests.
// channels=1|2, bits=8|16, samples per channel = pcm/(channels*bits/8).
func makeWAV(sampleRate, channels, bits int, pcm []byte) []byte {
	const fmtChunkSize = 16
	byteRate := sampleRate * channels * bits / 8
	blockAlign := channels * bits / 8
	totalSize := 4 + (8 + fmtChunkSize) + (8 + len(pcm))

	var b []byte
	b = append(b, "RIFF"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(totalSize))
	b = append(b, "WAVE"...)
	b = append(b, "fmt "...)
	b = binary.LittleEndian.AppendUint32(b, uint32(fmtChunkSize))
	b = binary.LittleEndian.AppendUint16(b, 1) // PCM
	b = binary.LittleEndian.AppendUint16(b, uint16(channels))
	b = binary.LittleEndian.AppendUint32(b, uint32(sampleRate))
	b = binary.LittleEndian.AppendUint32(b, uint32(byteRate))
	b = binary.LittleEndian.AppendUint16(b, uint16(blockAlign))
	b = binary.LittleEndian.AppendUint16(b, uint16(bits))
	b = append(b, "data"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(pcm)))
	b = append(b, pcm...)
	return b
}

func TestLoadWAV_Mono16_PassesThrough(t *testing.T) {
	in := []byte{0x10, 0x20, 0x30, 0x40} // 2 samples
	wav := makeWAV(16000, 1, 16, in)
	out, err := LoadWAVAsPCM16FromBytes(wav, 16000)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(out) != string(in) {
		t.Fatalf("mono passthrough mismatch: %v vs %v", out, in)
	}
}

func TestLoadWAV_Stereo16_Downmixes(t *testing.T) {
	// L=100, R=300 → mono = 200 → bytes 0xC8 0x00
	pcm := []byte{
		0x64, 0x00, 0x2C, 0x01, // sample 1: L=100, R=300
	}
	wav := makeWAV(16000, 2, 16, pcm)
	out, err := LoadWAVAsPCM16FromBytes(wav, 16000)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 bytes mono, got %d", len(out))
	}
	v := int16(binary.LittleEndian.Uint16(out))
	if v != 200 {
		t.Fatalf("downmix=%d, want 200", v)
	}
}

func TestLoadWAV_PromotesU8(t *testing.T) {
	// u8 centred at 128: value 144 → diff 16 → s16 = 16 << 8 = 4096
	wav := makeWAV(16000, 1, 8, []byte{144})
	out, err := LoadWAVAsPCM16FromBytes(wav, 16000)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	v := int16(binary.LittleEndian.Uint16(out))
	if v != 4096 {
		t.Fatalf("u8→s16=%d, want 4096", v)
	}
}

func TestLoadWAV_Resamples(t *testing.T) {
	// 8 kHz mono, 8 samples → resample to 16 kHz should grow.
	pcm := make([]byte, 16) // 8 int16 samples = 16 bytes
	wav := makeWAV(8000, 1, 16, pcm)
	out, err := LoadWAVAsPCM16FromBytes(wav, 16000)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out) <= len(pcm) {
		t.Fatalf("resample 8k→16k expected growth, got %d (in %d)", len(out), len(pcm))
	}
}

func TestLoadWAV_RejectsBad(t *testing.T) {
	if _, err := LoadWAVAsPCM16FromBytes(nil, 16000); err == nil {
		t.Fatal("expected error for nil")
	}
	if _, err := LoadWAVAsPCM16FromBytes([]byte("nope"), 16000); err == nil {
		t.Fatal("expected error for short blob")
	}
}
