package asr

import (
	"context"
	"fmt"
	"strings"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/media/encoder"
	"github.com/LingByte/lingllm/recognizer"
	"go.uber.org/zap"
)

const componentCodecDecode = "codec.decode"

// codecDecodeStage decodes RTP payload bytes (opus/pcmu/pcma/…) to PCM bytes for ASR.
// Mirrors hardware's OpusDecodeComponent pattern but uses generic CreateDecode.
type codecDecodeStage struct {
	name    string
	decoder media.EncoderFunc
	logger  *zap.Logger
}

func newCodecDecodeStage(src, dst media.CodecConfig, logger *zap.Logger) (*codecDecodeStage, error) {
	dec, err := encoder.CreateDecode(src, dst)
	if err != nil {
		return nil, fmt.Errorf("voice/asr: CreateDecode: %w", err)
	}
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	return &codecDecodeStage{
		name:    componentCodecDecode,
		decoder: dec,
		logger:  logger,
	}, nil
}

func (s *codecDecodeStage) Name() string { return s.name }

func (s *codecDecodeStage) Process(_ context.Context, data interface{}) (interface{}, bool, error) {
	raw, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("voice/asr: %s: expected []byte, got %T", s.name, data)
	}
	if len(raw) == 0 {
		return nil, false, nil
	}
	packets, err := s.decoder(&media.AudioPacket{Payload: raw})
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("voice/asr decode failed", zap.Error(err))
		}
		return nil, false, err
	}
	if len(packets) == 0 {
		return nil, false, nil
	}
	ap, ok := packets[0].(*media.AudioPacket)
	if !ok {
		return nil, false, fmt.Errorf("voice/asr: decoder returned invalid packet type")
	}
	return ap.Payload, true, nil
}

type asrInputStage struct {
	asr     recognizer.SpeechRecognitionEngine
	metrics *Metrics
	logger  *zap.Logger
}

func (s *asrInputStage) Name() string { return "asr.input" }

func (s *asrInputStage) Process(_ context.Context, data interface{}) (interface{}, bool, error) {
	pcm, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("voice/asr: invalid data type: %T", data)
	}
	if len(pcm) == 0 {
		return nil, false, nil
	}
	if s.metrics != nil {
		s.metrics.mu.Lock()
		s.metrics.TotalAudioBytes += len(pcm)
		s.metrics.mu.Unlock()
	}
	if err := s.asr.SendAudioBytes(pcm); err != nil {
		msg := err.Error()
		stopped := strings.Contains(msg, "recognizer not running") ||
			strings.Contains(msg, "recognizer is not running")
		if stopped {
			// QCloud (and similar) end the session after idle during long
			// LLM/tool/TTS turns; restart so the call can keep listening.
			s.asr.RestartClient()
			if retryErr := s.asr.SendAudioBytes(pcm); retryErr != nil {
				if s.logger != nil {
					s.logger.Warn("voice/asr send failed after restart", zap.Error(retryErr))
				}
				return nil, true, nil
			}
			if s.logger != nil {
				s.logger.Info("voice/asr recognizer restarted after idle stop")
			}
			return nil, true, nil
		}
		if s.logger != nil {
			s.logger.Warn("voice/asr send audio failed", zap.Error(err))
		}
		return nil, false, err
	}
	return nil, true, nil
}
