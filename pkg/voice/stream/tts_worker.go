package stream

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"go.uber.org/zap"
)

// TTSService TTS 合成服务接口
type TTSService interface {
	// SynthesizeStream 流式合成音频
	// callback: 每生成一帧音频就调用一次
	SynthesizeStream(text string, callback func([]byte) error) error
}

// TTSServiceAdapter 将现有的 synthesizer.SynthesisService 适配为 stream.TTSService
type TTSServiceAdapter struct {
	service synthesizer.SynthesisService
}

// NewTTSServiceAdapter 创建 TTS 服务适配器
func NewTTSServiceAdapter(service synthesizer.SynthesisService) TTSService {
	return &TTSServiceAdapter{
		service: service,
	}
}

// SynthesizeStream 流式合成音频
func (a *TTSServiceAdapter) SynthesizeStream(text string, callback func([]byte) error) error {
	// 创建一个 handler 来接收流式音频数据
	handler := &streamHandler{
		callback: callback,
	}

	// 调用现有的 TTS 服务
	ctx := context.Background()
	err := a.service.Synthesize(ctx, handler, text)
	if err != nil {
		return fmt.Errorf("TTS synthesis failed: %w", err)
	}

	return nil
}

// streamHandler 实现 synthesizer.SynthesisHandler 接口
type streamHandler struct {
	callback func([]byte) error
}

func (h *streamHandler) OnMessage(data []byte) {
	if h.callback != nil {
		if err := h.callback(data); err != nil {
			// 错误处理
			fmt.Printf("Callback error: %v\n", err)
		}
	}
}

func (h *streamHandler) OnTimestamp(timestamp synthesizer.SentenceTimestamp) {
	// 可选：处理时间戳信息
}

// AudioFrame 音频帧（PCM 格式）
type AudioFrame struct {
	Data       []byte
	SampleRate int
	Channels   int
	PlayID     string
	Sequence   uint32
}

// TTSWorkerPool TTS 工作池
type TTSWorkerPool struct {
	inputCh      <-chan TextSegment
	outputCh     chan<- AudioFrame
	ttsService   TTSService
	workers      int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	logger       *zap.Logger
	sequenceMu   sync.Mutex   // 保护全局 sequence
	globalSeq    uint32       // 全局 sequence，确保音频帧顺序
	ttsServiceMu sync.RWMutex // 保护 ttsService 的并发访问
}

// NewTTSWorkerPool 创建 TTS 工作池
func NewTTSWorkerPool(
	inputCh <-chan TextSegment,
	outputCh chan<- AudioFrame,
	ttsService TTSService,
	workers int,
	logger *zap.Logger,
) *TTSWorkerPool {
	return &TTSWorkerPool{
		inputCh:    inputCh,
		outputCh:   outputCh,
		ttsService: ttsService,
		workers:    workers,
		logger:     logger,
	}
}

// Start 启动工作池
func (p *TTSWorkerPool) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("TTSWorkerPool started", zap.Int("workers", p.workers))
	return nil
}

// Stop 停止工作池
func (p *TTSWorkerPool) Stop() error {
	p.cancel()
	p.wg.Wait()
	p.logger.Info("TTSWorkerPool stopped")
	return nil
}

// worker 工作协程
func (p *TTSWorkerPool) worker(id int) {
	defer p.wg.Done()
	p.logger.Info("TTS worker started", zap.Int("worker_id", id))
	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("TTS worker stopped", zap.Int("worker_id", id))
			return
		case segment := <-p.inputCh:
			p.synthesize(id, segment)
		}
	}
}

// synthesize 合成音频
func (p *TTSWorkerPool) synthesize(workerID int, segment TextSegment) {
	p.logger.Info("Synthesizing segment",
		zap.Int("worker_id", workerID),
		zap.String("text", segment.Text),
		zap.String("play_id", segment.PlayID))

	const frameSizeBytes = 1920 // 60ms @ 16kHz, 16bit, mono = 16000 * 0.06 * 2 = 1920 bytes
	// 预分配足够大的buffer，避免频繁扩容（假设平均每个字符生成约100字节音频）
	estimatedSize := len(segment.Text) * 100
	if estimatedSize < frameSizeBytes*2 {
		estimatedSize = frameSizeBytes * 2
	}
	buffer := make([]byte, 0, estimatedSize)

	// 获取 TTS 服务（线程安全）
	p.ttsServiceMu.RLock()
	ttsService := p.ttsService
	p.ttsServiceMu.RUnlock()

	err := ttsService.SynthesizeStream(segment.Text, func(pcmData []byte) error {
		buffer = append(buffer, pcmData...)
		for len(buffer) >= frameSizeBytes {
			frameData := make([]byte, frameSizeBytes)
			copy(frameData, buffer[:frameSizeBytes])
			buffer = buffer[frameSizeBytes:]

			// 使用全局 sequence，确保顺序
			p.sequenceMu.Lock()
			sequence := p.globalSeq
			p.globalSeq++
			p.sequenceMu.Unlock()

			frame := AudioFrame{
				Data:       frameData,
				SampleRate: 16000,
				Channels:   1,
				PlayID:     segment.PlayID,
				Sequence:   sequence,
			}
			select {
			case p.outputCh <- frame:
			case <-p.ctx.Done():
				return nil
			}
		}
		return nil
	})
	if err != nil {
		p.logger.Error("TTS synthesis failed",
			zap.Int("worker_id", workerID),
			zap.String("text", segment.Text),
			zap.Error(err))
		return
	}
	if len(buffer) > 0 {
		// 使用全局 sequence
		p.sequenceMu.Lock()
		sequence := p.globalSeq
		p.globalSeq++
		p.sequenceMu.Unlock()

		frame := AudioFrame{
			Data:       buffer,
			SampleRate: 16000,
			Channels:   1,
			PlayID:     segment.PlayID,
			Sequence:   sequence,
		}
		select {
		case p.outputCh <- frame:
		case <-p.ctx.Done():
		}
	}
}

// UpdateTTSService 动态更新 TTS 服务（用于发音人切换）
func (p *TTSWorkerPool) UpdateTTSService(newService TTSService) {
	p.logger.Info("更新 TTS 服务")
	// 使用写锁保护 TTS 服务更新
	p.ttsServiceMu.Lock()
	p.ttsService = newService
	p.ttsServiceMu.Unlock()
	p.logger.Info("TTS 服务更新完成")
}
