package asr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/lingllm/media"
)

// VADComponent performs energy-based barge-in detection during downlink playback.
// It must sit before EchoFilterComponent so it analyzes raw microphone PCM.
type VADComponent struct {
	mu                      sync.RWMutex
	enabled                 bool
	threshold               float64
	adaptiveThreshold       float64
	consecutiveFramesNeeded int
	frameCounter            int
	lastLogTime             time.Time
	noiseLevel              float64
	noiseSamples            []float64
	maxNoiseSamples         int

	gate            *PlaybackGate
	bargeInCallback func()
	logger          func(string)

	// armed prevents repeated barge-in fires until playback window closes.
	armed bool
}

// VADConfig contains configuration for VAD component.
type VADConfig struct {
	Enabled                 bool
	Threshold               float64
	ConsecutiveFramesNeeded int
	MaxNoiseSamples         int
}

// DefaultVADConfig returns general-purpose VAD settings (not barge-in tuned).
func DefaultVADConfig() VADConfig {
	return VADConfig{
		Enabled:                 true,
		Threshold:               1500.0,
		ConsecutiveFramesNeeded: 1,
		MaxNoiseSamples:         20,
	}
}

// DefaultBargeInVADConfig returns thresholds calibrated for interrupting TTS
// on uncancelled speakers.
func DefaultBargeInVADConfig() VADConfig {
	return VADConfig{
		Enabled:                 true,
		Threshold:               5500.0,
		ConsecutiveFramesNeeded: 8,
		MaxNoiseSamples:         20,
	}
}

// NewVADComponent creates a VAD stage. gate may be nil (detection disabled).
func NewVADComponent(config VADConfig, gate *PlaybackGate) *VADComponent {
	if config.Threshold == 0 {
		config.Threshold = 1500.0
	}
	if config.ConsecutiveFramesNeeded == 0 {
		config.ConsecutiveFramesNeeded = 1
	}
	if config.MaxNoiseSamples == 0 {
		config.MaxNoiseSamples = 20
	}
	return &VADComponent{
		enabled:                 config.Enabled,
		threshold:               config.Threshold,
		consecutiveFramesNeeded: config.ConsecutiveFramesNeeded,
		noiseSamples:            make([]float64, 0, config.MaxNoiseSamples),
		maxNoiseSamples:         config.MaxNoiseSamples,
		lastLogTime:             time.Now(),
		gate:                    gate,
	}
}

// Name returns the component name.
func (v *VADComponent) Name() string { return "vad" }

// Process analyzes PCM for barge-in; audio passes through unchanged.
func (v *VADComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if len(pcmData) < 2 {
		return pcmData, true, nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.enabled || v.gate == nil {
		return pcmData, true, nil
	}

	inWindow := v.gate.IsBargeInWindow()
	if !inWindow {
		v.frameCounter = 0
		v.armed = false
		return pcmData, true, nil
	}

	rms := media.RMSPCM16LE(pcmData)
	v.updateNoiseFloor(rms)
	effective := v.effectiveThreshold()

	if rms > effective {
		v.frameCounter++
		if v.frameCounter >= v.consecutiveFramesNeeded && !v.armed {
			v.armed = true
			v.frameCounter = 0
			if v.logger != nil {
				v.logger(fmt.Sprintf("[VAD] barge-in: rms=%.0f threshold=%.0f", rms, effective))
			}
			if v.bargeInCallback != nil {
				// Fire without holding the lock.
				cb := v.bargeInCallback
				go cb()
			}
		}
	} else {
		v.frameCounter = 0
	}

	return pcmData, true, nil
}

func (v *VADComponent) updateNoiseFloor(rms float64) {
	if rms >= 350 {
		return
	}
	v.noiseSamples = append(v.noiseSamples, rms)
	if len(v.noiseSamples) > v.maxNoiseSamples {
		v.noiseSamples = v.noiseSamples[1:]
	}
	var sum float64
	for _, s := range v.noiseSamples {
		sum += s
	}
	if len(v.noiseSamples) == 0 {
		return
	}
	v.noiseLevel = sum / float64(len(v.noiseSamples))
	v.adaptiveThreshold = v.noiseLevel * 4.0
	if v.adaptiveThreshold < 180 {
		v.adaptiveThreshold = 180
	}
	if v.adaptiveThreshold > v.threshold {
		v.adaptiveThreshold = v.threshold
	}
}

func (v *VADComponent) effectiveThreshold() float64 {
	// Never lower the configured barge-in threshold via adaptive floor —
	// that made TTS echo / line noise trigger false interrupts.
	if v.adaptiveThreshold > v.threshold {
		return v.adaptiveThreshold
	}
	return v.threshold
}

// SetBargeInCallback sets the callback invoked on barge-in (edge-triggered per window).
func (v *VADComponent) SetBargeInCallback(callback func()) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.bargeInCallback = callback
}

// SetLogger sets the logging callback.
func (v *VADComponent) SetLogger(callback func(string)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.logger = callback
}

// SetEnabled enables or disables VAD.
func (v *VADComponent) SetEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = enabled
	if !enabled {
		v.frameCounter = 0
		v.armed = false
	}
}

// SetThreshold sets the RMS energy threshold.
func (v *VADComponent) SetThreshold(threshold float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = threshold
}

// SetConsecutiveFrames sets frames required before barge-in fires.
func (v *VADComponent) SetConsecutiveFrames(frames int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.consecutiveFramesNeeded = frames
}

// Reset clears internal state between turns.
func (v *VADComponent) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.frameCounter = 0
	v.armed = false
}
