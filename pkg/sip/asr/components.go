package asr

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
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
		return nil, fmt.Errorf("sip/asr: CreateDecode: %w", err)
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
		return nil, false, fmt.Errorf("sip/asr: %s: expected []byte, got %T", s.name, data)
	}
	if len(raw) == 0 {
		return nil, false, nil
	}
	packets, err := s.decoder(&media.AudioPacket{Payload: raw})
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("sip/asr decode failed", zap.Error(err))
		}
		return nil, false, err
	}
	if len(packets) == 0 {
		return nil, false, nil
	}
	ap, ok := packets[0].(*media.AudioPacket)
	if !ok {
		return nil, false, fmt.Errorf("sip/asr: decoder returned invalid packet type")
	}
	return ap.Payload, true, nil
}
