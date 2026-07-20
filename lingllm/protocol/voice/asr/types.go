package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Voice ASR Protocol — universal audio processing pipeline for ASR.
//
// Provides a pluggable, chain-of-responsibility architecture for audio
// processing. Audio flows through a series of processors (decode, VAD, echo
// filter, etc.) before reaching ASR recognition.
//
// Architecture:
// audio input -> decode (optional) -> VAD -> echo filter -> ASR -> output text

import (
	"context"
	"sync"
	"time"
)

// PipelineComponent is a single step in the audio processing pipeline.
// Implementations MUST be safe for concurrent calls.
type PipelineComponent interface {
	// Name returns the component's identifier (e.g., "vad", "echo_filter", "asr_input")
	Name() string
	// Process handles one audio frame and returns the result.
	// data: input data (typically []byte for audio or string for text)
	// Returns: (output data, shouldContinue, error)
	// If shouldContinue is false, the pipeline stops processing this frame.
	Process(ctx context.Context, data interface{}) (interface{}, bool, error)
}

// Pipeline is the main audio processing pipeline.
type Pipeline interface {
	// Process processes one audio frame through the entire pipeline.
	Process(ctx context.Context, data interface{}) (interface{}, error)
	// ProcessOutput runs recognized text through output stages and fires callbacks.
	ProcessOutput(ctx context.Context, text string, isFinal bool)
	// SetOutputCallback sets the callback for final output.
	SetOutputCallback(callback func(text string, isFinal bool))
	// SetPCMAudioCallback sets the callback for PCM audio recording.
	SetPCMAudioCallback(callback func(data []byte) error)
	// SetBargeInCallback sets the callback for barge-in detection.
	SetBargeInCallback(callback func())
	// SetTTSPlaying sets the TTS playing state.
	SetTTSPlaying(playing bool)
	// IsTTSPlaying checks if TTS is playing.
	IsTTSPlaying() bool
	// ClearTTSState clears the TTS state.
	ClearTTSState()
	// ResetState resets the pipeline state for a new conversation round.
	ResetState()
	// GetMetrics returns the pipeline performance metrics.
	GetMetrics() *Metrics
	// Close tears down the pipeline and releases resources.
	Close() error
}

// Metrics contains ASR pipeline performance metrics.
type Metrics struct {
	mu              sync.RWMutex
	FirstPacketTime time.Time     // First audio packet time
	LastPacketTime  time.Time     // Last audio packet time
	ASRFirstResult  time.Time     // First ASR result time
	ASRLatency      time.Duration // ASR latency (from last packet to first result)
	TotalAudioBytes int           // Total audio bytes (PCM)
	AudioDuration   time.Duration // Total audio duration
	RTF             float64       // Real-Time Factor (processing time / audio duration)
}

// StandardPipeline implements Pipeline with a chain of components.
type StandardPipeline struct {
	inputStages  []PipelineComponent
	outputStages []PipelineComponent
	onOutput     func(text string, isFinal bool)
	onPCMAudio   func(data []byte) error
	metrics      *Metrics
	ttsPlaying   bool
	mu           sync.RWMutex
}

// NewStandardPipeline creates a new standard ASR processing pipeline.
func NewStandardPipeline(
	inputStages []PipelineComponent,
	outputStages []PipelineComponent,
) (*StandardPipeline, error) {
	if len(inputStages) == 0 {
		return nil, ErrEmptyInputStages
	}

	return &StandardPipeline{
		inputStages:  inputStages,
		outputStages: outputStages,
		metrics:      &Metrics{},
	}, nil
}

// Process processes one audio frame through the entire pipeline.
func (p *StandardPipeline) Process(ctx context.Context, data interface{}) (interface{}, error) {
	// Update metrics
	p.metrics.mu.Lock()
	if p.metrics.FirstPacketTime.IsZero() {
		p.metrics.FirstPacketTime = time.Now()
	}
	p.metrics.LastPacketTime = time.Now()
	p.metrics.mu.Unlock()

	current := data
	var pcmData []byte // Save PCM data after decoding

	// Process through input stages
	for i, stage := range p.inputStages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, shouldContinue, err := stage.Process(ctx, current)
		if err != nil {
			return nil, err
		}

		if !shouldContinue {
			return nil, nil
		}

		// Save PCM data after first input stage (decode or pcm_input)
		if i == 0 {
			if data, ok := result.([]byte); ok {
				pcmData = data
				// Update metrics
				p.metrics.mu.Lock()
				p.metrics.TotalAudioBytes += len(pcmData)
				p.metrics.mu.Unlock()
			}
		}

		current = result
	}

	// Call PCM audio recording callback
	if pcmData != nil && p.onPCMAudio != nil {
		if err := p.onPCMAudio(pcmData); err != nil {
			// Log but don't fail
			_ = err
		}
	}

	return current, nil
}

// ProcessOutput processes output through output stages.
func (p *StandardPipeline) ProcessOutput(ctx context.Context, text string, isFinal bool) {
	// Update metrics
	p.metrics.mu.Lock()
	if p.metrics.ASRFirstResult.IsZero() {
		p.metrics.ASRFirstResult = time.Now()
		if !p.metrics.LastPacketTime.IsZero() {
			p.metrics.ASRLatency = p.metrics.ASRFirstResult.Sub(p.metrics.LastPacketTime)
		}
	}
	p.metrics.mu.Unlock()

	current := interface{}(text)

	// Process through output stages
	for _, stage := range p.outputStages {
		result, shouldContinue, err := stage.Process(ctx, current)
		if err != nil {
			// Log but don't fail
			_ = err
			return
		}

		if !shouldContinue {
			return
		}

		current = result
	}

	// Call output callback
	if filteredText, ok := current.(string); ok {
		if p.onOutput != nil {
			p.onOutput(filteredText, isFinal)
		}
	}
}

// SetOutputCallback sets the callback for final output.
func (p *StandardPipeline) SetOutputCallback(callback func(text string, isFinal bool)) {
	p.onOutput = callback
}

// SetPCMAudioCallback sets the callback for PCM audio recording.
func (p *StandardPipeline) SetPCMAudioCallback(callback func(data []byte) error) {
	p.onPCMAudio = callback
}

// SetBargeInCallback sets the callback for barge-in detection on all VAD stages.
func (p *StandardPipeline) SetBargeInCallback(callback func()) {
	for _, stage := range p.inputStages {
		if vad, ok := stage.(*VADComponent); ok {
			vad.SetBargeInCallback(callback)
		}
	}
}

// WirePlaybackGate attaches a shared gate to VAD and echo-filter stages.
func (p *StandardPipeline) WirePlaybackGate(gate *PlaybackGate) {
	for _, stage := range p.inputStages {
		switch s := stage.(type) {
		case *VADComponent:
			s.gate = gate
		case *EchoFilterComponent:
			s.gate = gate
		}
	}
}

// SetTTSPlaying sets the TTS playing state.
func (p *StandardPipeline) SetTTSPlaying(playing bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ttsPlaying = playing
}

// IsTTSPlaying checks if TTS is playing.
func (p *StandardPipeline) IsTTSPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ttsPlaying
}

// ClearTTSState clears the TTS state.
func (p *StandardPipeline) ClearTTSState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ttsPlaying = false
}

// ResetState resets the pipeline state for a new conversation round.
func (p *StandardPipeline) ResetState() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.FirstPacketTime = time.Time{}
	p.metrics.LastPacketTime = time.Time{}
	p.metrics.ASRFirstResult = time.Time{}
	p.metrics.ASRLatency = 0
	p.metrics.TotalAudioBytes = 0
	p.metrics.AudioDuration = 0
	p.metrics.RTF = 0
}

// GetMetrics returns the pipeline performance metrics.
func (p *StandardPipeline) GetMetrics() *Metrics {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	// Calculate audio duration (assuming 16kHz, mono, 16-bit)
	if p.metrics.TotalAudioBytes > 0 {
		p.metrics.AudioDuration = time.Duration(p.metrics.TotalAudioBytes/2/16000) * time.Second
	}

	// Calculate RTF (Real-Time Factor) = ASR latency / audio duration
	if p.metrics.AudioDuration > 0 && p.metrics.ASRLatency > 0 {
		p.metrics.RTF = float64(p.metrics.ASRLatency) / float64(p.metrics.AudioDuration)
	}

	return p.metrics
}

// Close tears down the pipeline.
func (p *StandardPipeline) Close() error {
	// Close all components that implement Closer
	for _, stage := range p.inputStages {
		if closer, ok := stage.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				return err
			}
		}
	}

	for _, stage := range p.outputStages {
		if closer, ok := stage.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}
