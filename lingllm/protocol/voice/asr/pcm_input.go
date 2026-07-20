package asr

import (
	"context"
	"fmt"
)

// PCMInputComponent handles raw PCM audio input.
type PCMInputComponent struct{}

// Name returns the component name.
func (p *PCMInputComponent) Name() string {
	return "pcm_input"
}

// Process passes through PCM data unchanged.
func (p *PCMInputComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}

	if len(pcmData) == 0 {
		return nil, false, fmt.Errorf("empty PCM data")
	}

	return pcmData, true, nil
}
