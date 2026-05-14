// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package audioasset loads short PCM/WAV assets (ringback, hold music,
// IVR prompts) into the same PCM16 LE mono format VoiceServer's bridge
// uses internally. It exists so any of the three transports (SIP /
// xiaozhi / WebRTC) can play a canned clip without each one re-
// implementing WAV parsing + resample + downmix logic.
//
// Source format: RIFF/WAVE PCM, 8-bit unsigned or 16-bit signed,
// 1 or 2 channels, any sample rate.
// Output format: PCM16 LE mono at the caller's requested sample rate.
//
// Why not pkg/media/wav: pkg/media is the codec framing layer (per-
// frame encode/decode hot path). Asset loading is one-shot at startup
// or on TTS hold, never in a per-packet loop, so it lives in pkg/voice
// alongside the recorder which is the closest cousin in scope.
package audioasset

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/media"
)

// LoadWAVAsPCM16Mono reads a PCM WAV file from disk and returns mono
// PCM16 LE at targetSampleRate. Returns an error if the file is not a
// canonical PCM WAV (compressed forms like A-law or MP3-in-WAV are
// not supported — those should be transcoded by the asset pipeline,
// not the runtime).
//
// Use case: load `scripts/hold/please_wait.wav` once at startup, hand
// the bytes to TTS pipeline's Sink during reconnect / IVR / hold.
func LoadWAVAsPCM16Mono(path string, targetSampleRate int) ([]byte, error) {
	clean := filepath.Clean(path)
	raw, err := os.ReadFile(clean)
	if err != nil {
		return nil, err
	}
	return LoadWAVAsPCM16FromBytes(raw, targetSampleRate)
}

// LoadWAVAsPCM16FromBytes is the in-memory variant of
// LoadWAVAsPCM16Mono. Useful for go:embed fixtures and unit tests.
func LoadWAVAsPCM16FromBytes(raw []byte, targetSampleRate int) ([]byte, error) {
	pcm, sampleRate, channels, bits, err := parseWAVPCM(raw)
	if err != nil {
		return nil, err
	}
	mono16, err := toMonoPCM16(pcm, channels, bits)
	if err != nil {
		return nil, err
	}
	if targetSampleRate <= 0 {
		targetSampleRate = 16000
	}
	if sampleRate > 0 && sampleRate != targetSampleRate {
		out, err := media.ResamplePCM(mono16, sampleRate, targetSampleRate)
		if err != nil {
			return nil, fmt.Errorf("resample wav %d->%d: %w", sampleRate, targetSampleRate, err)
		}
		mono16 = out
	}
	return mono16, nil
}

// parseWAVPCM walks RIFF chunks looking for fmt + data. Tolerates
// extra chunks (LIST/INFO/cue/labl ...) by skipping them. Honours
// the WAV word-alignment rule (odd-sized chunks have a pad byte).
func parseWAVPCM(raw []byte) (pcm []byte, sampleRate, channels, bits int, err error) {
	if len(raw) < 44 {
		return nil, 0, 0, 0, fmt.Errorf("invalid wav: too short (%d bytes)", len(raw))
	}
	if string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return nil, 0, 0, 0, fmt.Errorf("invalid wav: missing RIFF/WAVE magic")
	}
	offset := 12
	for offset+8 <= len(raw) {
		chunkID := string(raw[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(raw[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(raw) {
			return nil, 0, 0, 0, fmt.Errorf("invalid wav: chunk %q overflows", chunkID)
		}
		chunk := raw[offset : offset+chunkSize]
		switch strings.TrimRight(chunkID, " ") {
		case "fmt":
			if len(chunk) < 16 {
				return nil, 0, 0, 0, fmt.Errorf("invalid wav: short fmt chunk")
			}
			audioFormat := binary.LittleEndian.Uint16(chunk[0:2])
			if audioFormat != 1 {
				// 0xFFFE WAVEFORMATEXTENSIBLE could carry PCM but we keep
				// the parser strict — encode-side compatibility is the
				// asset pipeline's problem, not the runtime's.
				return nil, 0, 0, 0, fmt.Errorf("unsupported wav format: 0x%04x (need PCM)", audioFormat)
			}
			channels = int(binary.LittleEndian.Uint16(chunk[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(chunk[4:8]))
			bits = int(binary.LittleEndian.Uint16(chunk[14:16]))
		case "data":
			pcm = chunk
		}
		offset += chunkSize
		if chunkSize%2 == 1 && offset < len(raw) {
			offset++
		}
	}
	if len(pcm) == 0 {
		return nil, 0, 0, 0, fmt.Errorf("invalid wav: no data chunk")
	}
	if channels < 1 {
		channels = 1
	}
	if bits == 0 {
		bits = 16
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	return pcm, sampleRate, channels, bits, nil
}

// toMonoPCM16 normalises any (8/16-bit, 1/2-channel) PCM to mono 16-bit
// little-endian. Stereo is downmixed via simple L+R averaging — fine
// for prompts, not appropriate for music mastering.
func toMonoPCM16(pcm []byte, channels, bitsPerSample int) ([]byte, error) {
	switch bitsPerSample {
	case 16:
		switch channels {
		case 1:
			return pcm, nil
		case 2:
			return stereo16ToMono16(pcm), nil
		default:
			return nil, fmt.Errorf("unsupported wav channels: %d", channels)
		}
	case 8:
		return u8ToMono16(pcm, channels), nil
	default:
		return nil, fmt.Errorf("unsupported wav bit depth: %d", bitsPerSample)
	}
}

// stereo16ToMono16 averages L and R per sample frame.
func stereo16ToMono16(pcm []byte) []byte {
	if len(pcm) < 4 {
		return nil
	}
	samples := len(pcm) / 4
	out := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		base := i * 4
		l := int16(binary.LittleEndian.Uint16(pcm[base : base+2]))
		r := int16(binary.LittleEndian.Uint16(pcm[base+2 : base+4]))
		m := int16((int32(l) + int32(r)) / 2)
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(m))
	}
	return out
}

// u8ToMono16 promotes 8-bit unsigned PCM (centred at 128) to int16.
// Multi-channel input is averaged first then promoted; matches what
// most ringback/IVR prompts in the wild produce.
func u8ToMono16(pcm []byte, channels int) []byte {
	if channels < 1 {
		channels = 1
	}
	samples := len(pcm) / channels
	out := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sum := 0
		for c := 0; c < channels; c++ {
			sum += int(pcm[i*channels+c]) - 128
		}
		avg := sum / channels
		s16 := int16(avg << 8)
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(s16))
	}
	return out
}
