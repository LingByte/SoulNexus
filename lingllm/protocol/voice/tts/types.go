package tts

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// TTSService defines the interface for text-to-speech synthesis.
type TTSService interface {
	// Synthesize synthesizes text to audio and calls the callback for each audio chunk.
	// The callback receives PCM audio data (typically 16-bit mono at 16kHz).
	Synthesize(ctx context.Context, text string, callback func([]byte) error) error
}

// TTSPipelineComponent defines the interface for TTS pipeline components.
type TTSPipelineComponent interface {
	// Name returns the component name.
	Name() string

	// Process processes data through the component.
	// For text processors: input is string, output is string
	// For audio processors: input is []byte, output is []byte
	// Returns (processedData, shouldContinue, error)
	Process(ctx context.Context, data interface{}) (interface{}, bool, error)
}

// TextSegment represents a text segment for TTS synthesis.
type TextSegment struct {
	Text      string
	IsFinal   bool
	Timestamp time.Time
	PlayID    string
}

// AudioFrame represents a frame of audio data.
type AudioFrame struct {
	Data       []byte
	SampleRate int
	Channels   int
	PlayID     string
	Sequence   uint32
}

// TTSPipelineConfig contains configuration for the TTS pipeline.
type TTSPipelineConfig struct {
	// TTSService: the TTS service to use for synthesis
	TTSService TTSService
	// OutputCodec: output audio codec (e.g., "opus", "pcm")
	OutputCodec string
	// TargetSampleRate: target sample rate for output audio (default: 16000)
	TargetSampleRate int
	// FrameDuration: frame duration for audio processing (default: 60ms)
	FrameDuration time.Duration
	// TextProcessors: optional text processing components (e.g., text normalization)
	TextProcessors []TTSPipelineComponent
	// AudioProcessors: optional audio processing components (e.g., encoding)
	AudioProcessors []TTSPipelineComponent
	// SendCallback: callback for sending synthesized audio
	SendCallback func(data []byte) error
	// RecordCallback: optional callback for recording synthesized audio
	RecordCallback func(data []byte) error
	// Logger: optional logging callback
	Logger func(string)
	// PaceRealtime sleeps between frames so playback matches wall-clock (required for RTP/VoIP).
	PaceRealtime bool
}

// DefaultTTSPipelineConfig returns default TTS pipeline configuration.
func DefaultTTSPipelineConfig(ttsService TTSService) TTSPipelineConfig {
	return TTSPipelineConfig{
		TTSService:       ttsService,
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		FrameDuration:    60 * time.Millisecond,
		TextProcessors:   []TTSPipelineComponent{},
		AudioProcessors:  []TTSPipelineComponent{},
	}
}

// TTSPipeline manages text-to-speech synthesis with pluggable components.
type TTSPipeline struct {
	mu                 sync.RWMutex
	config             TTSPipelineConfig
	ttsService         TTSService
	textProcessors     []TTSPipelineComponent
	audioProcessors    []TTSPipelineComponent
	currentPlayID      string
	globalSeq          uint32
	seqMu              sync.Mutex
	ctx                context.Context
	cancel             context.CancelFunc
	logger             func(string)
	onCompleteFunc     func()
	completeMu         sync.Mutex
	completionCancel   context.CancelFunc
	completionCancelMu sync.Mutex

	speakMu        sync.Mutex
	speakCtx       context.Context
	speakCancel    context.CancelFunc
	playing        atomic.Bool
	firstFrameHook atomic.Pointer[func()]

	paceMu      sync.Mutex
	nextFrameAt time.Time
}

// NewTTSPipeline creates a new TTS pipeline with the given configuration.
func NewTTSPipeline(config TTSPipelineConfig) (*TTSPipeline, error) {
	if config.TTSService == nil {
		return nil, ErrTTSServiceRequired
	}

	if config.SendCallback == nil {
		return nil, ErrSendCallbackRequired
	}

	if config.TargetSampleRate == 0 {
		config.TargetSampleRate = 16000
	}

	if config.FrameDuration == 0 {
		config.FrameDuration = 60 * time.Millisecond
	}

	if config.OutputCodec == "" {
		config.OutputCodec = "pcm"
	}

	return &TTSPipeline{
		config:          config,
		ttsService:      config.TTSService,
		textProcessors:  config.TextProcessors,
		audioProcessors: config.AudioProcessors,
		logger:          config.Logger,
	}, nil
}

// Start starts the TTS pipeline.
func (p *TTSPipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ctx, p.cancel = context.WithCancel(ctx)

	if p.logger != nil {
		p.logger("[TTS Pipeline] Started")
	}

	return nil
}

// Stop stops the TTS pipeline.
func (p *TTSPipeline) Stop() error {
	p.Interrupt()
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	p.completionCancelMu.Lock()
	if p.completionCancel != nil {
		p.completionCancel()
	}
	p.completionCancelMu.Unlock()

	if p.logger != nil {
		p.logger("[TTS Pipeline] Stopped")
	}

	return nil
}

// Synthesize synthesizes text to audio using the provided context.
func (p *TTSPipeline) Synthesize(ctx context.Context, text string) error {
	if p == nil {
		return errors.New("tts: nil pipeline")
	}
	if p.logger != nil {
		p.logger("[TTS Pipeline] Synthesizing: " + text)
	}
	return p.synthesize(ctx, text)
}

// SetOnCompleteFunc sets the completion callback.
func (p *TTSPipeline) SetOnCompleteFunc(callback func()) {
	p.completeMu.Lock()
	defer p.completeMu.Unlock()
	p.onCompleteFunc = callback
}

// SetLogger sets the logging callback.
func (p *TTSPipeline) SetLogger(callback func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.logger = callback
}

// GetConfig returns the current pipeline configuration.
func (p *TTSPipeline) GetConfig() TTSPipelineConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// Close closes the TTS pipeline.
func (p *TTSPipeline) Close() error {
	return p.Stop()
}
