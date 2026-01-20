package tts

import (
	"context"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/hardware/errhandler"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"go.uber.org/zap"
)

// Service TTS服务实现
type Service struct {
	ctx          context.Context
	credential   *models.UserCredential
	speaker      string
	synthesizer  synthesizer.SynthesisService
	errorHandler *errhandler.Handler
	logger       *zap.Logger
	mu           sync.RWMutex
	closed       bool
}

// NewService 创建TTS服务
func NewService(
	ctx context.Context,
	credential *models.UserCredential,
	speaker string,
	synthesizer synthesizer.SynthesisService,
	errorHandler *errhandler.Handler,
	logger *zap.Logger,
) *Service {
	return &Service{
		ctx:          ctx,
		credential:   credential,
		speaker:      speaker,
		synthesizer:  synthesizer,
		errorHandler: errorHandler,
		logger:       logger,
	}
}

// Synthesize 合成语音
func (s *Service) Synthesize(ctx context.Context, text string) (<-chan []byte, error) {
	s.mu.RLock()
	closed := s.closed
	synthesizer := s.synthesizer
	s.mu.RUnlock()

	if closed || synthesizer == nil {
		return nil, errhandler.NewRecoverableError("TTS", "服务已关闭", nil)
	}

	if text == "" {
		return nil, errhandler.NewRecoverableError("TTS", "文本为空", nil)
	}

	// 创建音频通道（增大缓冲区，避免阻塞）
	audioChan := make(chan []byte, 50)

	// 创建SynthesisHandler
	handler := &synthesisHandler{
		audioChan: audioChan,
		ctx:       ctx,
	}

	// 在goroutine中合成
	go func() {
		defer close(audioChan)

		err := synthesizer.Synthesize(ctx, handler, text)
		if err != nil {
			classified := s.errorHandler.Classify(err, "TTS")
			s.logger.Error("TTS合成失败", zap.Error(classified))
			// 发送错误信号
			select {
			case <-ctx.Done():
			case audioChan <- nil: // nil表示错误
			}
		}
	}()

	return audioChan, nil
}

// synthesisHandler 实现 SynthesisHandler 接口
type synthesisHandler struct {
	audioChan chan []byte
	ctx       context.Context
}

func (h *synthesisHandler) OnMessage(data []byte) {
	select {
	case <-h.ctx.Done():
		return
	case h.audioChan <- data:
		// 成功发送
	default:
		// 通道满了，记录警告但不阻塞（避免TTS合成被阻塞）
		// 这种情况应该很少发生，因为缓冲区已经足够大
	}
}

func (h *synthesisHandler) OnTimestamp(timestamp synthesizer.SentenceTimestamp) {
	// 暂时不处理时间戳
}

// Close 关闭服务
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if s.synthesizer != nil {
		s.synthesizer.Close()
	}

	s.closed = true
	return nil
}
