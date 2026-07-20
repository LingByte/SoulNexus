package asr

import (
	"context"
	"fmt"
)

// EchoFilterComponent suppresses uplink audio while downlink TTS is active so ASR
// does not transcribe the AI's own voice. VAD for barge-in must run *before* this
// stage so it still sees raw microphone energy.
type EchoFilterComponent struct {
	gate *PlaybackGate // set at build time or via WirePlaybackGate
}

// NewEchoFilterComponent creates an echo suppressor backed by a PlaybackGate.
func NewEchoFilterComponent(gate *PlaybackGate) *EchoFilterComponent {
	return &EchoFilterComponent{gate: gate}
}

// Name returns the component name.
func (e *EchoFilterComponent) Name() string { return "echo_filter" }

// Process replaces PCM with silence while echo suppression is active.
func (e *EchoFilterComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if len(pcmData) == 0 {
		return pcmData, true, nil
	}

	if e.gate != nil && e.gate.IsEchoSuppressActive() {
		silentFrame := make([]byte, len(pcmData))
		return silentFrame, true, nil
	}
	return pcmData, true, nil
}
