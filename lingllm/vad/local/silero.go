//go:build silero

package local

import (
	"fmt"
	"sync"

	"github.com/shenjinti/gosilero"
)

// SileroVAD wraps Silero voice-activity detection for uplink PCM16LE.
type SileroVAD struct {
	vad        *gosilero.VAD
	sampleRate int
	chunkSize  int
	mu         sync.Mutex
	buf        []byte
}

// NewSileroVAD creates a Silero VAD instance (requires -tags silero build).
func NewSileroVAD(sampleRate int) (*SileroVAD, error) {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	chunk := 512
	if sampleRate == 8000 {
		chunk = 256
	}
	v, err := gosilero.NewVAD(sampleRate, chunk)
	if err != nil {
		return nil, fmt.Errorf("silero vad: %w", err)
	}
	return &SileroVAD{vad: v, sampleRate: sampleRate, chunkSize: chunk}, nil
}

// IsSpeech returns true when the buffered PCM contains speech.
func (s *SileroVAD) IsSpeech(pcm []byte) (bool, error) {
	if s == nil || s.vad == nil || len(pcm) < 2 {
		return false, nil
	}
	bytesPerFrame := s.chunkSize * 2
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = append(s.buf, pcm...)
	for len(s.buf) >= bytesPerFrame {
		frame := s.buf[:bytesPerFrame]
		s.buf = s.buf[bytesPerFrame:]
		samples := pcm16ToFloat32(frame)
		prob, err := s.vad.Predict(samples)
		if err != nil {
			return false, err
		}
		if prob >= 0.5 {
			return true, nil
		}
	}
	return false, nil
}

// Close releases Silero resources.
func (s *SileroVAD) Close() error {
	if s == nil || s.vad == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vad.Free()
	s.vad = nil
	return nil
}

func pcm16ToFloat32(pcm []byte) []float32 {
	n := len(pcm) / 2
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		sample := int16(pcm[i*2]) | (int16(pcm[i*2+1]) << 8)
		out[i] = float32(sample) / 32768.0
	}
	return out
}
