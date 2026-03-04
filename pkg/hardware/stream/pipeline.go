package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TTSPipelineConfig 配置
type TTSPipelineConfig struct {
	TTSService          TTSService
	SendCallback        func(data []byte) error // 发送音频数据的回调
	GetPendingCountFunc func() int              // 获取待发送包数量的回调
	TargetSampleRate    int                     // 目标采样率，默认 16000
	FrameDuration       time.Duration           // 帧时长，默认 60ms
	RecordCallback      func(data []byte) error // 录音回调（用于记录AI音频）
	Logger              *zap.Logger
}

// TTSPipeline TTS 流式处理管道
type TTSPipeline struct {
	segmenter           *TextSegmenter
	ttsService          TTSService
	audioSender         *AudioSender
	audioCh             chan AudioFrame
	currentPlayID       string
	globalSeq           uint32 // 全局 sequence
	seqMu               sync.Mutex
	mu                  sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	logger              *zap.Logger
	onCompleteFunc      func() // 播放完成回调
	completeMu          sync.Mutex
	completionCancel    context.CancelFunc // 用于取消上一个 waitForCompletion
	completionCancelMu  sync.Mutex
	getPendingCountFunc func() int              // 获取待发送包数量的回调
	recordCallback      func(data []byte) error // 录音回调
}

// NewTTSPipeline 创建 TTS 管道
func NewTTSPipeline(config *TTSPipelineConfig) (*TTSPipeline, error) {
	if config.TTSService == nil {
		return nil, fmt.Errorf("TTSService is required")
	}
	if config.SendCallback == nil {
		return nil, fmt.Errorf("SendCallback is required")
	}
	if config.TargetSampleRate == 0 {
		config.TargetSampleRate = 16000
	}
	if config.FrameDuration == 0 {
		config.FrameDuration = 60 * time.Millisecond
	}
	if config.Logger == nil {
		config.Logger, _ = zap.NewDevelopment()
	}

	audioCh := make(chan AudioFrame, 20)

	audioSender, err := NewAudioSender(
		audioCh,
		config.TargetSampleRate,
		config.FrameDuration,
		config.SendCallback,
		config.GetPendingCountFunc,
		config.Logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio sender: %w", err)
	}

	pipeline := &TTSPipeline{
		ttsService:          config.TTSService,
		audioSender:         audioSender,
		audioCh:             audioCh,
		currentPlayID:       uuid.New().String(),
		logger:              config.Logger,
		getPendingCountFunc: config.GetPendingCountFunc,
		recordCallback:      config.RecordCallback,
	}

	// 创建 Segmenter，传入 TTS 处理回调
	pipeline.segmenter = NewTextSegmenterWithCallback(
		pipeline.processTTSSegment,
		config.Logger,
	)

	return pipeline, nil
}

// Start 启动管道
func (p *TTSPipeline) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// 设置 Segmenter 的 PlayID
	p.segmenter.SetPlayID(p.currentPlayID)

	// 启动组件
	if err := p.segmenter.Start(p.ctx); err != nil {
		return fmt.Errorf("failed to start segmenter: %w", err)
	}

	if err := p.audioSender.Start(p.ctx); err != nil {
		return fmt.Errorf("failed to start audio sender: %w", err)
	}

	p.logger.Info("TTSPipeline started",
		zap.String("play_id", p.currentPlayID))

	return nil
}

// Stop 停止管道
func (p *TTSPipeline) Stop() error {
	// 取消所有协程
	p.cancel()

	// 取消 waitForCompletion 协程
	p.completionCancelMu.Lock()
	if p.completionCancel != nil {
		p.completionCancel()
	}
	p.completionCancelMu.Unlock()

	// 停止所有组件
	p.segmenter.Stop()
	p.audioSender.Stop()

	// 关闭 channel
	close(p.audioCh)

	p.logger.Info("TTSPipeline stopped")
	return nil
}

// OnLLMToken 处理 LLM 输出的 token
func (p *TTSPipeline) OnLLMToken(token string) {
	p.segmenter.OnToken(token)
}

// processTTSSegment 处理文本片段的 TTS 合成（替代 TTSWorkerPool）
func (p *TTSPipeline) processTTSSegment(segment TextSegment) {
	p.logger.Info("Synthesizing segment",
		zap.String("text", segment.Text),
		zap.String("play_id", segment.PlayID))

	const frameSizeBytes = 1920 // 60ms @ 16kHz, 16bit, mono
	buffer := make([]byte, 0, frameSizeBytes*2)

	err := p.ttsService.SynthesizeStream(segment.Text, func(pcmData []byte) error {
		buffer = append(buffer, pcmData...)
		for len(buffer) >= frameSizeBytes {
			frameData := make([]byte, frameSizeBytes)
			copy(frameData, buffer[:frameSizeBytes])
			buffer = buffer[frameSizeBytes:]

			// 使用全局 sequence
			p.seqMu.Lock()
			sequence := p.globalSeq
			p.globalSeq++
			p.seqMu.Unlock()

			frame := AudioFrame{
				Data:       frameData,
				SampleRate: 16000,
				Channels:   1,
				PlayID:     segment.PlayID,
				Sequence:   sequence,
			}

			// 记录AI音频
			if p.recordCallback != nil {
				if err := p.recordCallback(frameData); err != nil {
					p.logger.Warn("Recording AI audio failed", zap.Error(err))
				}
			}

			select {
			case p.audioCh <- frame:
			case <-p.ctx.Done():
				return nil
			}
		}
		return nil
	})

	if err != nil {
		p.logger.Error("TTS synthesis failed",
			zap.String("text", segment.Text),
			zap.Error(err))
		return
	}

	// 处理剩余数据
	if len(buffer) > 0 {
		p.seqMu.Lock()
		sequence := p.globalSeq
		p.globalSeq++
		p.seqMu.Unlock()

		frame := AudioFrame{
			Data:       buffer,
			SampleRate: 16000,
			Channels:   1,
			PlayID:     segment.PlayID,
			Sequence:   sequence,
		}

		// 记录AI音频
		if p.recordCallback != nil {
			if err := p.recordCallback(buffer); err != nil {
				p.logger.Warn("Recording AI audio failed", zap.Error(err))
			}
		}

		select {
		case p.audioCh <- frame:
		case <-p.ctx.Done():
		}
	}
}

// OnLLMComplete LLM 完成输出
func (p *TTSPipeline) OnLLMComplete() {
	p.logger.Info("[TTS Pipeline] ========== OnLLMComplete 被调用 ==========")

	p.segmenter.OnComplete()

	// 取消上一个 waitForCompletion 协程（如果存在）
	p.completionCancelMu.Lock()
	if p.completionCancel != nil {
		p.logger.Info("[TTS Pipeline] 取消上一个 waitForCompletion 协程")
		p.completionCancel()
	}

	// 创建新的 context 用于本次 waitForCompletion
	completionCtx, completionCancel := context.WithCancel(p.ctx)
	p.completionCancel = completionCancel
	p.completionCancelMu.Unlock()

	p.logger.Info("[TTS Pipeline] LLM 完成，启动 waitForCompletion 协程")

	// 启动监控协程，等待所有音频发送完成
	go p.waitForCompletion(completionCtx)

	p.logger.Info("[TTS Pipeline] waitForCompletion 协程已启动")
}

// waitForCompletion 等待所有音频发送完成
func (p *TTSPipeline) waitForCompletion(ctx context.Context) {
	p.logger.Info("[TTS Pipeline] ========== waitForCompletion 协程开始执行 ==========")

	// 先等待一段时间，给 TTS 合成留出时间
	time.Sleep(500 * time.Millisecond)
	p.logger.Info("[TTS Pipeline] 初始等待完成，开始检查队列状态")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Second) // 30秒超时
	emptyCount := 0                         // 连续空闲次数

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("[TTS Pipeline] waitForCompletion 被取消（可能是新的对话开始）")
			return
		case <-timeout:
			p.logger.Warn("[TTS Pipeline] 等待完成超时（30秒）")
			p.triggerComplete()
			return
		case <-ticker.C:
			bufferLevel := p.audioSender.GetBufferLevel()
			audioChLen := len(p.audioCh)
			pendingPackets := 0
			if p.getPendingCountFunc != nil {
				pendingPackets = p.getPendingCountFunc()
			}
			if audioChLen == 0 && bufferLevel == 0 && pendingPackets == 0 {
				emptyCount++
				p.logger.Info("[TTS Pipeline] 所有队列为空",
					zap.Int("emptyCount", emptyCount))
				// 连续 3 次检查都为空（300ms），认为播放完成
				if emptyCount >= 3 {
					p.triggerComplete()
					return
				}
			} else {
				emptyCount = 0 // 重置计数
			}
		}
	}
}

// triggerComplete 触发完成回调
func (p *TTSPipeline) triggerComplete() {
	p.logger.Info("[TTS Pipeline] ========== triggerComplete 被调用 ==========")

	p.completeMu.Lock()
	defer p.completeMu.Unlock()

	if p.onCompleteFunc != nil {
		p.logger.Info("[TTS Pipeline] 执行完成回调函数")
		p.onCompleteFunc()
		p.logger.Info("[TTS Pipeline] 完成回调函数执行完毕")
	} else {
		p.logger.Warn("[TTS Pipeline] 完成回调函数为 nil，无法执行")
	}
}

// SetOnComplete 设置完成回调
func (p *TTSPipeline) SetOnComplete(callback func()) {
	p.completeMu.Lock()
	defer p.completeMu.Unlock()
	p.onCompleteFunc = callback
}

// Interrupt 打断当前播放
func (p *TTSPipeline) Interrupt() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 生成新的 PlayID
	oldPlayID := p.currentPlayID
	p.currentPlayID = uuid.New().String()

	// 重置 sequence
	p.seqMu.Lock()
	p.globalSeq = 0
	p.seqMu.Unlock()

	p.logger.Info("Pipeline interrupted",
		zap.String("old_play_id", oldPlayID),
		zap.String("new_play_id", p.currentPlayID))

	// 重置所有组件
	p.segmenter.Reset()
	p.segmenter.SetPlayID(p.currentPlayID)
	p.audioSender.Reset()

	// 清空 channel
	for len(p.audioCh) > 0 {
		<-p.audioCh
	}
}

// GetCurrentPlayID 获取当前 PlayID
func (p *TTSPipeline) GetCurrentPlayID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentPlayID
}

// GetBufferLevel 获取当前缓冲区水位
func (p *TTSPipeline) GetBufferLevel() int {
	return p.audioSender.GetBufferLevel()
}

// PrintMetrics 打印性能指标（简化版）
func (p *TTSPipeline) PrintMetrics() {
	bufferLevel := p.audioSender.GetBufferLevel()

	p.logger.Info("=== TTS Pipeline 状态 ===",
		zap.Int("buffer_level", bufferLevel),
		zap.Int("audio_ch", len(p.audioCh)))
}

// UpdateTTSService 动态更新 TTS 服务（用于发音人切换）
func (p *TTSPipeline) UpdateTTSService(newService TTSService) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.logger.Info("Pipeline 开始更新 TTS 服务")
	p.ttsService = newService
	p.logger.Info("Pipeline TTS 服务更新完成")
}
