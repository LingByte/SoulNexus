package audio

import "github.com/LingByte/SoulNexus/pkg/utils/common"

// LoadWAVAsPCM16Mono reads a PCM WAV file and returns mono s16le PCM at targetSampleRate.
func LoadWAVAsPCM16Mono(path string, targetSampleRate int) ([]byte, error) {
	return common.LoadWAVAsPCM16Mono(path, targetSampleRate)
}

// LoadWAVAsPCM16FromBytes parses WAV bytes and returns mono s16le PCM at targetSampleRate.
func LoadWAVAsPCM16FromBytes(raw []byte, targetSampleRate int) ([]byte, error) {
	return common.LoadWAVAsPCM16FromBytes(raw, targetSampleRate)
}
