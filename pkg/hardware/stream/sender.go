package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"go.uber.org/zap"
)

// OpusFrame Opus 编码后的音频帧
type OpusFrame struct {
	Data     []byte
	PlayID   string
	Sequence uint32 // 使用 uint32 避免溢出
	Duration time.Duration
}

// AudioSender 音频发送器（包含编码、缓冲、网络发送）
type AudioSender struct {
	inputCh             <-chan AudioFrame
	encoder             media.EncoderFunc
	buffer              []OpusFrame
	bufferMu            sync.Mutex
	ctx                 context.Context
	cancel              context.CancelFunc
	logger              *zap.Logger
	sendCallback        func(data []byte) error // 发送音频数据的回调
	getPendingCountFunc func() int              // 获取待发送包数量的回调
}

// NewAudioSender 创建音频发送器
func NewAudioSender(
	inputCh <-chan AudioFrame,
	targetSampleRate int,
	frameDuration time.Duration,
	sendCallback func(data []byte) error,
	getPendingCountFunc func() int,
	logger *zap.Logger,
) (*AudioSender, error) {
	opusEncoder, err := encoder.CreateEncode(
		media.CodecConfig{
			Codec:         "opus",
			SampleRate:    targetSampleRate,
			Channels:      1,
			BitDepth:      16,
			FrameDuration: "60ms",
		},
		media.CodecConfig{
			Codec:      "pcm",
			SampleRate: targetSampleRate,
			Channels:   1,
			BitDepth:   16,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	return &AudioSender{
		inputCh:             inputCh,
		encoder:             opusEncoder,
		buffer:              make([]OpusFrame, 0, 100), // 预分配更大容量，避免频繁扩容
		sendCallback:        sendCallback,
		getPendingCountFunc: getPendingCountFunc,
		logger:              logger,
	}, nil
}

// Start 启动音频发送器
func (s *AudioSender) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 启动输入处理
	go s.inputLoop()

	// 启动输出处理（定时发送）
	go s.outputLoop()
	return nil
}

// Stop 停止音频发送器
func (s *AudioSender) Stop() error {
	s.cancel()
	s.logger.Info("AudioSender stopped")
	return nil
}

// inputLoop 输入处理循环
func (s *AudioSender) inputLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return

		case frame := <-s.inputCh:
			s.processFrame(frame)
		}
	}
}

// processFrame 处理音频帧（编码 + 缓冲）
func (s *AudioSender) processFrame(frame AudioFrame) {
	pcmData := frame.Data
	packets, err := s.encoder(&media.AudioPacket{Payload: pcmData})
	if err != nil {
		s.logger.Error("Opus encoding failed", zap.Error(err))
		return
	}

	if len(packets) == 0 {
		return
	}

	audioPacket := packets[0].(*media.AudioPacket)
	opusData := audioPacket.Payload
	opusFrame := OpusFrame{
		Data:     opusData,
		PlayID:   frame.PlayID,
		Sequence: frame.Sequence,
	}

	// 写入缓冲区（带背压控制）
	s.writeToBuffer(opusFrame)
}

// writeToBuffer 写入缓冲区
func (s *AudioSender) writeToBuffer(frame OpusFrame) {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()

	s.buffer = append(s.buffer, frame)
}

// outputLoop 输出处理循环
func (s *AudioSender) outputLoop() {
	s.logger.Info("[AudioSender] ========== outputLoop 启动 ==========")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("[AudioSender] outputLoop 退出")
			return

		default:
			s.sendFrame()
		}
	}
}

// sendFrame 发送一帧音频（阻塞式，等待发送完成）
func (s *AudioSender) sendFrame() {
	s.bufferMu.Lock()

	// 检查缓冲区
	if len(s.buffer) == 0 {
		s.bufferMu.Unlock()
		return
	}
	frame := s.buffer[0]
	s.buffer = s.buffer[1:]
	s.bufferMu.Unlock()

	// 通过回调发送到网络
	err := s.sendCallback(frame.Data)
	if err != nil {
		s.logger.Error("Network send failed", zap.Error(err))
		return
	}
}

// GetPendingCount 获取待发送的数据包数量
func (s *AudioSender) GetPendingCount() int {
	if s.getPendingCountFunc != nil {
		return s.getPendingCountFunc()
	}
	return 0
}

// GetBufferLevel 获取当前缓冲区帧数
func (s *AudioSender) GetBufferLevel() int {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()
	return len(s.buffer)
}

// Reset 重置状态
func (s *AudioSender) Reset() {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()

	s.buffer = s.buffer[:0]
	s.logger.Info("AudioSender reset")
}
