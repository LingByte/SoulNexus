package conversation

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LingByte/SoulNexus/pkg/media"
)

func loadWAVAsPCM16Mono(path string, targetSampleRate int) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pcm, sampleRate, channels, bitsPerSample, err := parseWAVPCM(raw)
	if err != nil {
		return nil, err
	}
	mono16, err := toMonoPCM16(pcm, channels, bitsPerSample)
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

func parseWAVPCM(raw []byte) ([]byte, int, int, int, error) {
	if len(raw) < 44 {
		return nil, 0, 0, 0, fmt.Errorf("invalid wav: too short")
	}
	if string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return nil, 0, 0, 0, fmt.Errorf("invalid wav: missing RIFF/WAVE")
	}
	offset := 12
	sampleRate, channels, bits := 0, 0, 0
	var pcm []byte
	for offset+8 <= len(raw) {
		chunkID := string(raw[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(raw[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(raw) {
			return nil, 0, 0, 0, fmt.Errorf("invalid wav: chunk overflow")
		}
		chunk := raw[offset : offset+chunkSize]
		switch chunkID {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, 0, 0, 0, fmt.Errorf("invalid wav: short fmt chunk")
			}
			audioFormat := binary.LittleEndian.Uint16(chunk[0:2])
			if audioFormat != 1 { // PCM only
				return nil, 0, 0, 0, fmt.Errorf("unsupported wav format: %d", audioFormat)
			}
			channels = int(binary.LittleEndian.Uint16(chunk[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(chunk[4:8]))
			bits = int(binary.LittleEndian.Uint16(chunk[14:16]))
		case "data":
			pcm = chunk
		}
		offset += chunkSize
		if chunkSize%2 == 1 && offset < len(raw) {
			offset++ // word alignment
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

func toMonoPCM16(pcm []byte, channels, bitsPerSample int) ([]byte, error) {
	switch bitsPerSample {
	case 16:
		if channels <= 1 {
			return pcm, nil
		}
		if channels == 2 {
			return stereo16ToMono16(pcm), nil
		}
		return nil, fmt.Errorf("unsupported wav channels: %d", channels)
	case 8:
		return u8ToMono16(pcm, channels), nil
	default:
		return nil, fmt.Errorf("unsupported wav bit depth: %d", bitsPerSample)
	}
}

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

func u8ToMono16(pcm []byte, channels int) []byte {
	if channels < 1 {
		channels = 1
	}
	samples := len(pcm) / channels
	out := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sum := 0
		for c := 0; c < channels; c++ {
			v := int(pcm[i*channels+c]) - 128
			sum += v
		}
		avg := sum / channels
		s16 := int16(avg << 8)
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(s16))
	}
	return out
}
