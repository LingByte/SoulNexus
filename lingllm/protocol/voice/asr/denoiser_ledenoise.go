package asr

import (
	"context"
	"fmt"

	"github.com/LingByte/lingllm/media/denoise"
)

// LedenoiseDenoiser wraps media/denoise.Ledenoise as asr.Denoiser.
type LedenoiseDenoiser struct {
	inner *denoise.Ledenoise
}

// NewLedenoiseDenoiser creates a native RNNoise denoise instance.
func NewLedenoiseDenoiser(sampleRate int) (*LedenoiseDenoiser, error) {
	d, err := denoise.NewLedenoise(sampleRate)
	if err != nil {
		return nil, err
	}
	return &LedenoiseDenoiser{inner: d}, nil
}

func (d *LedenoiseDenoiser) Process(pcm []byte) []byte {
	if d == nil || d.inner == nil {
		return pcm
	}
	return d.inner.Process(pcm)
}

func (d *LedenoiseDenoiser) Close() error {
	if d == nil || d.inner == nil {
		return nil
	}
	return d.inner.Close()
}

// LedenoiseDenoiserComponent adapts Ledenoise for the ASR component chain.
type LedenoiseDenoiserComponent struct {
	dn *LedenoiseDenoiser
}

func NewLedenoiseDenoiserComponent(config interface{}) (*LedenoiseDenoiserComponent, error) {
	sr := 16000
	switch c := config.(type) {
	case *SimpleDenoiserConfig:
		if c != nil && c.SampleRate > 0 {
			sr = c.SampleRate
		}
	case int:
		if c > 0 {
			sr = c
		}
	case map[string]any:
		if v, ok := c["sampleRate"].(float64); ok && v > 0 {
			sr = int(v)
		}
	}
	dn, err := NewLedenoiseDenoiser(sr)
	if err != nil {
		return nil, fmt.Errorf("ledenoise component: %w", err)
	}
	return &LedenoiseDenoiserComponent{dn: dn}, nil
}

func (c *LedenoiseDenoiserComponent) Name() string { return "ledenoise" }

func (c *LedenoiseDenoiserComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	_ = ctx
	pcm, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if c == nil || c.dn == nil || len(pcm) == 0 {
		return pcm, true, nil
	}
	return c.dn.Process(pcm), true, nil
}
