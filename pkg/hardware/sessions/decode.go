package sessions

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/LingByte/SoulNexus/pkg/media"
	"go.uber.org/zap"
)

// OpusDecodeComponent opus decode component decode_time 86.291µs, total_time 86.542µs
type OpusDecodeComponent struct {
	logger  *zap.Logger
	decoder media.EncoderFunc
}

func (s *OpusDecodeComponent) Name() string {
	return constants.COMPONENT_OPUS_DECODE
}

func (s *OpusDecodeComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	opusData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("invalid data type: expected []byte")
	}
	packets, err := s.decoder(&media.AudioPacket{Payload: opusData})
	if err != nil {
		s.logger.Error("[OpusDecode] 解码失败", zap.Error(err))
		return nil, false, fmt.Errorf("opus decode error: %w", err)
	}
	if len(packets) == 0 {
		return nil, false, nil
	}
	audioPacket, ok := packets[0].(*media.AudioPacket)
	if !ok {
		return nil, false, fmt.Errorf("decoder returned invalid packet type")
	}
	return audioPacket.Payload, true, nil
}
