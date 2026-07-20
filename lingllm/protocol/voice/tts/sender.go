package tts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/media/encoder"
)

// EncodedFrame represents an encoded audio frame.
type EncodedFrame struct {
	Data     []byte
	PlayID   string
	Sequence uint32
}

// AudioSenderConfig contains configuration for the audio sender.
type AudioSenderConfig struct {
	// OutputCodec: output codec (e.g., "opus", "pcm")
	OutputCodec string
	// TargetSampleRate: target sample rate (default: 16000)
	TargetSampleRate int
	// FrameDuration: frame duration (default: 60ms)
	FrameDuration time.Duration
	// SendCallback: callback for sending encoded audio
	SendCallback func(data []byte) error
	// GetPendingCountFunc: optional callback to get pending packet count
	GetPendingCountFunc func() int
	// Logger: optional logging callback
	Logger func(string)
}

// AudioSender handles audio encoding and sending.
type AudioSender struct {
	mu       sync.RWMutex
	config   AudioSenderConfig
	encoder  media.EncoderFunc
	buffer   []EncodedFrame
	bufferMu sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	logger   func(string)
	notifyCh chan struct{}
}

// NewAudioSender creates a new audio sender.
func NewAudioSender(config AudioSenderConfig) (*AudioSender, error) {
	if config.SendCallback == nil {
		return nil, ErrSendCallbackRequired
	}

	if config.TargetSampleRate == 0 {
		config.TargetSampleRate = 16000
	}

	if config.FrameDuration == 0 {
		config.FrameDuration = 60 * time.Millisecond
	}

	codec := strings.ToLower(strings.TrimSpace(config.OutputCodec))
	if codec == "" {
		codec = "pcm"
	}

	var encoderFunc media.EncoderFunc
	var err error

	// Create encoder if not PCM
	if codec == "opus" {
		encoderFunc, err = encoder.CreateEncode(
			media.CodecConfig{
				Codec:         "opus",
				SampleRate:    config.TargetSampleRate,
				Channels:      1,
				BitDepth:      16,
				FrameDuration: "60ms",
			},
			media.CodecConfig{
				Codec:      "pcm",
				SampleRate: config.TargetSampleRate,
				Channels:   1,
				BitDepth:   16,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create opus encoder: %w", err)
		}
	}

	return &AudioSender{
		config:   config,
		encoder:  encoderFunc,
		buffer:   make([]EncodedFrame, 0, 100),
		logger:   config.Logger,
		notifyCh: make(chan struct{}, 1),
	}, nil
}

// Start starts the audio sender.
func (s *AudioSender) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start input processing goroutine
	go s.outputLoop()

	if s.logger != nil {
		s.logger("[AudioSender] Started")
	}

	return nil
}

// Stop stops the audio sender.
func (s *AudioSender) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	if s.logger != nil {
		s.logger("[AudioSender] Stopped")
	}

	return nil
}

// ProcessFrame processes a PCM audio frame (encode + buffer).
func (s *AudioSender) ProcessFrame(frame AudioFrame) error {
	s.mu.RLock()
	codec := s.config.OutputCodec
	s.mu.RUnlock()

	pcmData := frame.Data
	encodedData := pcmData

	// Encode if not PCM
	if codec == "opus" {
		packets, err := s.encoder(&media.AudioPacket{Payload: pcmData})
		if err != nil {
			if s.logger != nil {
				s.logger(fmt.Sprintf("[AudioSender] Encoding error: %v", err))
			}
			return fmt.Errorf("encoding failed: %w", err)
		}

		if len(packets) == 0 {
			return nil
		}

		audioPacket, ok := packets[0].(*media.AudioPacket)
		if !ok {
			return fmt.Errorf("invalid packet type")
		}
		encodedData = audioPacket.Payload
	}

	encodedFrame := EncodedFrame{
		Data:     encodedData,
		PlayID:   frame.PlayID,
		Sequence: frame.Sequence,
	}

	s.writeToBuffer(encodedFrame)
	return nil
}

// writeToBuffer writes a frame to the buffer.
func (s *AudioSender) writeToBuffer(frame EncodedFrame) {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()

	s.buffer = append(s.buffer, frame)

	select {
	case s.notifyCh <- struct{}{}:
	default:
	}
}

// outputLoop processes and sends buffered frames.
func (s *AudioSender) outputLoop() {
	for {
		select {
		case <-s.ctx.Done():
			if s.logger != nil {
				s.logger("[AudioSender] Output loop stopped")
			}
			return

		case <-s.notifyCh:
			for s.sendFrame() {
			}
		}
	}
}

// sendFrame sends a single frame from the buffer.
func (s *AudioSender) sendFrame() bool {
	s.bufferMu.Lock()

	if len(s.buffer) == 0 {
		s.bufferMu.Unlock()
		return false
	}

	frame := s.buffer[0]
	s.buffer = s.buffer[1:]
	s.bufferMu.Unlock()

	// Send via callback
	s.mu.RLock()
	sendCallback := s.config.SendCallback
	s.mu.RUnlock()

	if err := sendCallback(frame.Data); err != nil {
		if s.logger != nil {
			s.logger(fmt.Sprintf("[AudioSender] Send error: %v", err))
		}
		return false
	}

	return true
}

// GetBufferLevel returns the current buffer level.
func (s *AudioSender) GetBufferLevel() int {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()
	return len(s.buffer)
}

// GetPendingCount returns the pending packet count.
func (s *AudioSender) GetPendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config.GetPendingCountFunc != nil {
		return s.config.GetPendingCountFunc()
	}
	return 0
}

// Reset resets the sender state.
func (s *AudioSender) Reset() {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()

	s.buffer = s.buffer[:0]

	if s.logger != nil {
		s.logger("[AudioSender] Reset")
	}
}

// SetOutputCodec sets the output codec.
func (s *AudioSender) SetOutputCodec(codec string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "" {
		codec = "pcm"
	}

	if s.config.OutputCodec == codec {
		return nil
	}

	// Create new encoder if needed
	var encoderFunc media.EncoderFunc
	var err error

	if codec == "opus" {
		encoderFunc, err = encoder.CreateEncode(
			media.CodecConfig{
				Codec:         "opus",
				SampleRate:    s.config.TargetSampleRate,
				Channels:      1,
				BitDepth:      16,
				FrameDuration: "60ms",
			},
			media.CodecConfig{
				Codec:      "pcm",
				SampleRate: s.config.TargetSampleRate,
				Channels:   1,
				BitDepth:   16,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create encoder: %w", err)
		}
	}

	s.config.OutputCodec = codec
	s.encoder = encoderFunc

	s.bufferMu.Lock()
	s.buffer = s.buffer[:0]
	s.bufferMu.Unlock()

	if s.logger != nil {
		s.logger(fmt.Sprintf("[AudioSender] Output codec changed to: %s", codec))
	}

	return nil
}

// SetLogger sets the logging callback.
func (s *AudioSender) SetLogger(callback func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = callback
}

// Close closes the audio sender.
func (s *AudioSender) Close() error {
	return s.Stop()
}
