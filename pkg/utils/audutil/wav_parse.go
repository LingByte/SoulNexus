// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import (
	"encoding/binary"
	"fmt"
	"time"
)

// parsePCM16WAV extracts PCM16 LE samples from a minimal RIFF/WAVE file.
func parsePCM16WAV(wav []byte) (sampleRate, channels int, pcm []byte, err error) {
	if len(wav) < 44 {
		return 0, 0, nil, fmt.Errorf("audutil: wav too short")
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0, 0, nil, fmt.Errorf("audutil: not RIFF/WAVE")
	}
	i := 12
	var data []byte
	for i+8 <= len(wav) {
		chunkID := string(wav[i : i+4])
		chunkSize := int(binary.LittleEndian.Uint32(wav[i+4 : i+8]))
		if chunkSize < 0 {
			break
		}
		payloadStart := i + 8
		payloadEnd := payloadStart + chunkSize
		if payloadEnd > len(wav) {
			break
		}
		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				audioFmt := binary.LittleEndian.Uint16(wav[payloadStart : payloadStart+2])
				channels = int(binary.LittleEndian.Uint16(wav[payloadStart+2 : payloadStart+4]))
				sampleRate = int(binary.LittleEndian.Uint32(wav[payloadStart+4 : payloadStart+8]))
				bits := binary.LittleEndian.Uint16(wav[payloadStart+14 : payloadStart+16])
				if audioFmt != 1 || bits != 16 {
					return 0, 0, nil, fmt.Errorf("audutil: only PCM16 WAV supported")
				}
			}
		case "data":
			data = wav[payloadStart:payloadEnd]
		}
		advance := 8 + chunkSize
		if chunkSize%2 == 1 {
			advance++
		}
		i += advance
	}
	if sampleRate <= 0 || channels < 1 || len(data) == 0 {
		return 0, 0, nil, fmt.Errorf("audutil: incomplete wav")
	}
	return sampleRate, channels, data, nil
}

// SplitStereoWAVChannels splits stereo WAV into left (caller) and right (agent) mono PCM16 LE.
// Mono WAV returns all samples on left; right is nil.
func SplitStereoWAVChannels(wav []byte) (left, right []byte, sampleRate int, err error) {
	sampleRate, channels, pcm, err := parsePCM16WAV(wav)
	if err != nil {
		return nil, nil, 0, err
	}
	if channels <= 1 {
		out := make([]byte, len(pcm))
		copy(out, pcm)
		return out, nil, sampleRate, nil
	}
	frameSize := channels * 2
	left = make([]byte, 0, len(pcm)/channels)
	right = make([]byte, 0, len(pcm)/channels)
	for off := 0; off+frameSize <= len(pcm); off += frameSize {
		left = append(left, pcm[off], pcm[off+1])
		if channels >= 2 {
			right = append(right, pcm[off+2], pcm[off+3])
		}
	}
	return left, right, sampleRate, nil
}

// TrimPCMS16FromTime drops leading mono PCM16 LE samples for skip duration.
func TrimPCMS16FromTime(pcm []byte, sampleRate int, skip time.Duration) []byte {
	if len(pcm) < 2 || sampleRate <= 0 || skip <= 0 {
		return pcm
	}
	skipSamples := int(skip.Seconds() * float64(sampleRate))
	if skipSamples <= 0 {
		return pcm
	}
	skipBytes := skipSamples * 2
	if skipBytes >= len(pcm) {
		return nil
	}
	out := make([]byte, len(pcm)-skipBytes)
	copy(out, pcm[skipBytes:])
	return out
}

// MonoPCM16ToWAV wraps mono PCM16 LE bytes into a playable WAV blob.
func MonoPCM16ToWAV(pcm []byte, sampleRate int) ([]byte, error) {
	if len(pcm) < 2 || sampleRate <= 0 {
		return nil, fmt.Errorf("audutil: empty pcm for wav")
	}
	if len(pcm)%2 != 0 {
		pcm = pcm[:len(pcm)-1]
	}
	samples := make([]int16, len(pcm)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	return pcm16MonoToWav(samples, sampleRate), nil
}
