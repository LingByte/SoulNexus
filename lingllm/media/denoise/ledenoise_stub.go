//go:build !ledenoise

package denoise

import "errors"

var errLedenoiseNotBuilt = errors.New("ledenoise: rebuild with -tags ledenoise (and make build-ledenoise)")

func LedenoiseEnabled() bool { return false }

func LedenoiseVersion() string { return "" }

// Ledenoise stub when native lib is not linked.
type Ledenoise struct{}

func NewLedenoise(_ int) (*Ledenoise, error) {
	return nil, errLedenoiseNotBuilt
}

func (d *Ledenoise) Process(pcm []byte) []byte { return pcm }

func (d *Ledenoise) Close() error { return nil }
