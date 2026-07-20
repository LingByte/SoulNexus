//go:build !ledenoise

package denoise

import "errors"

var errSNRNotBuilt = errors.New("ledenoise snr: rebuild with -tags ledenoise (and make build-ledenoise)")

// SNREnabled is false without the native staticlib.
func SNREnabled() bool { return false }

// LedenoiseSNR stub when native lib is not linked.
type LedenoiseSNR struct{}

func NewLedenoiseSNR(_ int) (*LedenoiseSNR, error) {
	return nil, errSNRNotBuilt
}

func (s *LedenoiseSNR) Process(_ []int16) (float64, bool) { return 0, false }

func (s *LedenoiseSNR) Close() error { return nil }
