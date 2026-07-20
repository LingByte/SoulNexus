//go:build !silero

package local

import "errors"

var errSileroNotBuilt = errors.New("silero vad: rebuild with -tags silero")

// SileroVAD is unavailable unless built with -tags silero.
type SileroVAD struct{}

// NewSileroVAD returns an error when Silero support was not compiled in.
func NewSileroVAD(_ int) (*SileroVAD, error) {
	return nil, errSileroNotBuilt
}

// IsSpeech is a no-op stub.
func (s *SileroVAD) IsSpeech(_ []byte) (bool, error) {
	return false, errSileroNotBuilt
}

// Close is a no-op stub.
func (s *SileroVAD) Close() error { return nil }
