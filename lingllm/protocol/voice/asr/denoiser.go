package asr

import (
	"context"
	"fmt"
)

// Denoiser performs optional uplink PCM processing (RNNoise, WebRTC AEC3,
// hardware AEC, ...). Implementations must not mutate the input slice and
// may return the input unchanged when they have nothing to do.
type Denoiser interface {
	Process(pcm []byte) []byte
}

// DenoiserComponent applies a Denoiser in the ASR input chain. Place it after
// decode/PCM input and before VAD so barge-in still sees processed mic energy.
type DenoiserComponent struct {
	dn Denoiser
}

// NewDenoiserComponent creates a denoiser stage. dn must be non-nil.
func NewDenoiserComponent(dn Denoiser) *DenoiserComponent {
	return &DenoiserComponent{dn: dn}
}

// Name returns the component identifier.
func (d *DenoiserComponent) Name() string { return "denoiser" }

// Process runs uplink PCM through the configured denoiser.
func (d *DenoiserComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	_ = ctx
	pcm, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if len(pcm) == 0 || d == nil || d.dn == nil {
		return pcm, true, nil
	}
	return d.dn.Process(pcm), true, nil
}
