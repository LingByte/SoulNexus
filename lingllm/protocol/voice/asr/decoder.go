package asr

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/media/encoder"
)

// DecoderConfig contains configuration for the decoder component.
type DecoderConfig struct {
	// SourceCodec: source codec name (e.g., "opus", "pcmu", "pcma", "g722", "pcm")
	SourceCodec string
	// SourceSampleRate: source audio sample rate (e.g., 48000 for opus, 16000 for pcmu)
	SourceSampleRate int
	// SourceChannels: source audio channels (usually 1 or 2)
	SourceChannels int
	// TargetSampleRate: target PCM sample rate (e.g., 16000)
	TargetSampleRate int
	// TargetChannels: target PCM channels (usually 1)
	TargetChannels int
	// FrameDuration: frame duration string (e.g., "20ms", "40ms", "60ms")
	FrameDuration string
}

// DefaultDecoderConfig returns a default decoder configuration.
// Assumes OPUS input at 48kHz and outputs PCM at 16kHz mono.
func DefaultDecoderConfig() DecoderConfig {
	return DecoderConfig{
		SourceCodec:      "opus",
		SourceSampleRate: 48000,
		SourceChannels:   1,
		TargetSampleRate: 16000,
		TargetChannels:   1,
		FrameDuration:    "20ms",
	}
}

// DecoderComponent decodes compressed audio (e.g., Opus) to PCM.
// It uses the media/encoder package to handle codec-specific decoding.
type DecoderComponent struct {
	mu          sync.RWMutex
	config      DecoderConfig
	decoderFunc media.EncoderFunc
	logger      func(string)
}

// NewDecoderComponent creates a new decoder component with the given configuration.
func NewDecoderComponent(config DecoderConfig) (*DecoderComponent, error) {
	// Validate configuration
	if config.SourceCodec == "" {
		config.SourceCodec = "opus"
	}
	if config.SourceSampleRate == 0 {
		config.SourceSampleRate = 48000
	}
	if config.SourceChannels == 0 {
		config.SourceChannels = 1
	}
	if config.TargetSampleRate == 0 {
		config.TargetSampleRate = 16000
	}
	if config.TargetChannels == 0 {
		config.TargetChannels = 1
	}
	if config.FrameDuration == "" {
		config.FrameDuration = "20ms"
	}

	// Create source and target codec configs
	srcConfig := media.CodecConfig{
		Codec:         config.SourceCodec,
		SampleRate:    config.SourceSampleRate,
		Channels:      config.SourceChannels,
		FrameDuration: config.FrameDuration,
	}

	targetConfig := media.CodecConfig{
		Codec:      "pcm",
		SampleRate: config.TargetSampleRate,
		Channels:   config.TargetChannels,
	}

	// Create decoder function
	decoderFunc, err := encoder.CreateDecode(srcConfig, targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder for codec %q: %w", config.SourceCodec, err)
	}

	return &DecoderComponent{
		config:      config,
		decoderFunc: decoderFunc,
	}, nil
}

// Name returns the component name.
func (d *DecoderComponent) Name() string {
	return fmt.Sprintf("decoder_%s_to_pcm", d.config.SourceCodec)
}

// Process decodes compressed audio data to PCM.
// Returns (pcmData, shouldContinue, error)
func (d *DecoderComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	compressedData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}

	if len(compressedData) == 0 {
		return nil, false, fmt.Errorf("empty compressed data")
	}

	d.mu.RLock()
	decoderFunc := d.decoderFunc
	d.mu.RUnlock()

	if decoderFunc == nil {
		return nil, false, fmt.Errorf("decoder not initialized")
	}

	// Wrap the compressed data in an AudioPacket for the decoder
	inputPacket := &media.AudioPacket{
		Payload: compressedData,
	}

	// Decode the compressed data
	outputPackets, err := decoderFunc(inputPacket)
	if err != nil {
		if d.logger != nil {
			d.logger(fmt.Sprintf("[Decoder] decode error: %v", err))
		}
		return nil, false, fmt.Errorf("decode failed: %w", err)
	}

	if len(outputPackets) == 0 {
		if d.logger != nil {
			d.logger(fmt.Sprintf("[Decoder] no output packets from decoder"))
		}
		return nil, false, fmt.Errorf("decoder produced no output")
	}

	// Combine all output packets into a single PCM buffer
	var totalSize int
	for _, packet := range outputPackets {
		if audioPacket, ok := packet.(*media.AudioPacket); ok {
			totalSize += len(audioPacket.Payload)
		}
	}

	pcmData := make([]byte, 0, totalSize)
	for _, packet := range outputPackets {
		if audioPacket, ok := packet.(*media.AudioPacket); ok {
			pcmData = append(pcmData, audioPacket.Payload...)
		}
	}

	if d.logger != nil {
		d.logger(fmt.Sprintf("[Decoder] decoded %d bytes to %d bytes PCM", len(compressedData), len(pcmData)))
	}

	return pcmData, true, nil
}

// SetLogger sets the logging callback.
func (d *DecoderComponent) SetLogger(callback func(string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger = callback
}

// GetConfig returns the current decoder configuration.
func (d *DecoderComponent) GetConfig() DecoderConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}
