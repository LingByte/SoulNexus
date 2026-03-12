package stream

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TextSegment 文本片段
type TextSegment struct {
	Text      string
	IsFinal   bool // 是否是最后一段
	Timestamp time.Time
	PlayID    string
}

// TextSegmenter text segmenter
type TextSegmenter struct {
	outputCh      chan<- TextSegment // 旧的 channel 模式（可选）
	outputFunc    func(TextSegment)  // 新的回调模式
	buffer        string
	lastUpdate    time.Time
	delayTimer    *time.Timer
	delayTimeout  time.Duration
	minChars      int
	maxChars      int
	currentPlayID string
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *zap.Logger
}

// NewTextSegmenter create text segmenter (channel mode)
func NewTextSegmenter(outputCh chan<- TextSegment, logger *zap.Logger) *TextSegmenter {
	return &TextSegmenter{
		outputCh:     outputCh,
		delayTimeout: 100 * time.Millisecond,
		minChars:     8,
		maxChars:     40,
		logger:       logger,
	}
}

// NewTextSegmenterWithCallback create text segmenter (callback mode)
func NewTextSegmenterWithCallback(outputFunc func(TextSegment), logger *zap.Logger) *TextSegmenter {
	return &TextSegmenter{
		outputFunc:   outputFunc,
		delayTimeout: 50 * time.Millisecond, // 减少延迟，从100ms降到50ms
		minChars:     15,                    // 提高第一段的最小字数，从8提高到15
		maxChars:     35,                    // 降低最大字数，从40降到35，更早触发分段
		logger:       logger,
	}
}

// Start 启动分段器
func (s *TextSegmenter) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.logger.Info("TextSegmenter started")
	return nil
}

// Stop 停止分段器
func (s *TextSegmenter) Stop() error {
	s.cancel()
	if s.delayTimer != nil {
		s.delayTimer.Stop()
	}
	s.logger.Info("TextSegmenter stopped")
	return nil
}

// OnToken 处理 LLM 输出的 token
func (s *TextSegmenter) OnToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer += token
	s.lastUpdate = time.Now()

	bufferLen := len([]rune(s.buffer))

	// 1. 遇到句号、感叹号、问号：立即分段
	if strings.HasSuffix(s.buffer, "。") ||
		strings.HasSuffix(s.buffer, "！") ||
		strings.HasSuffix(s.buffer, "？") {
		s.flush(false)
		return
	}

	// 2. 遇到逗号、顿号：如果已经积累了足够的字符，立即分段
	if strings.HasSuffix(s.buffer, "，") ||
		strings.HasSuffix(s.buffer, "、") {
		// 如果已经有15个字以上，立即分段（不等待）
		if bufferLen >= s.minChars {
			s.flush(false)
			return
		}
		// 否则等待一小段时间
		if s.delayTimer != nil {
			s.delayTimer.Stop()
		}
		s.delayTimer = time.AfterFunc(s.delayTimeout, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if time.Since(s.lastUpdate) >= s.delayTimeout {
				if len([]rune(s.buffer)) >= s.minChars {
					s.flush(false)
				}
			}
		})
		return
	}

	// 3. 达到最大字符数：强制分段
	if bufferLen >= s.maxChars {
		s.flush(false)
	}
}

// OnComplete LLM 完成输出
func (s *TextSegmenter) OnComplete() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.delayTimer != nil {
		s.delayTimer.Stop()
	}

	// 立即刷新最后一段
	s.flush(true)
}

// SetPlayID 设置当前 PlayID
func (s *TextSegmenter) SetPlayID(playID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPlayID = playID
}

// flush 刷新缓冲区（需要持有锁）
func (s *TextSegmenter) flush(isFinal bool) {
	if s.buffer == "" {
		return
	}

	segment := TextSegment{
		Text:      s.buffer,
		IsFinal:   isFinal,
		Timestamp: time.Now(),
		PlayID:    s.currentPlayID,
	}

	s.logger.Debug("Flushing segment",
		zap.String("text", s.buffer),
		zap.Bool("isFinal", isFinal),
		zap.Int("length", len([]rune(s.buffer))))

	// 支持两种模式：channel 或 callback
	if s.outputFunc != nil {
		// 回调模式：直接调用（同步）
		s.outputFunc(segment)
		s.buffer = ""
	} else if s.outputCh != nil {
		// Channel 模式：发送到 channel
		select {
		case s.outputCh <- segment:
			s.buffer = ""
		case <-s.ctx.Done():
			return
		}
	}
}

// Reset 重置分段器状态
func (s *TextSegmenter) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.delayTimer != nil {
		s.delayTimer.Stop()
		s.delayTimer = nil
	}
	s.buffer = ""
	s.lastUpdate = time.Time{}
}
