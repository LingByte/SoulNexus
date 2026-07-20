//go:build !levad

package local

import "errors"

var errLevadNotBuilt = errors.New("levad: rebuild with -tags levad (and make build-levad)")

func LevadEnabled() bool { return false }

func LevadVersion() string { return "" }

const (
	LevadKindSilero uint8 = 0
	LevadKindTen    uint8 = 1
)

// LevadVAD is unavailable unless built with -tags levad.
type LevadVAD struct{}

func NewLevadVAD(_ uint8, _ int, _ float32) (*LevadVAD, error) {
	return nil, errLevadNotBuilt
}

func NewTinySileroVAD(_ int) (*LevadVAD, error) {
	return nil, errLevadNotBuilt
}

func NewTinyTenVAD(_ int) (*LevadVAD, error) {
	return nil, errLevadNotBuilt
}

func (v *LevadVAD) IsSpeech(_ []byte) (bool, error) {
	return false, errLevadNotBuilt
}

func (v *LevadVAD) Close() error { return nil }
