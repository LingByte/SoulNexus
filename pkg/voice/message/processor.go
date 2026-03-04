package message

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/voice/errhandler"
	"github.com/LingByte/SoulNexus/pkg/voice/filter"
	llmv3 "github.com/LingByte/SoulNexus/pkg/voice/llm"
	"github.com/LingByte/SoulNexus/pkg/voice/state"
	"github.com/LingByte/SoulNexus/pkg/voice/tts"
	"go.uber.org/zap"
)

const (
	// MaxMessageHistory 最大消息历史数量，防止内存无限增长
	MaxMessageHistory = 100
)

// Processor 消息处理器
type Processor struct {
	stateManager  *state.Manager
	llmService    *llmv3.Service
	ttsService    *tts.Service
	writer        *Writer
	errorHandler  *errhandler.Handler
	filterManager *filter.Manager // 过滤词管理器
	logger        *zap.Logger
	mu            sync.Mutex
	messages      []llm.Message
	synthesizer   synthesizer.SynthesisService // 用于获取音频格式
}

// NewProcessor 创建消息处理器
func NewProcessor(
	stateManager *state.Manager,
	llmService *llmv3.Service,
	ttsService *tts.Service,
	writer *Writer,
	errorHandler *errhandler.Handler,
	logger *zap.Logger,
	synthesizer synthesizer.SynthesisService,
	filterManager *filter.Manager,
) *Processor {
	return &Processor{
		stateManager:  stateManager,
		llmService:    llmService,
		ttsService:    ttsService,
		writer:        writer,
		errorHandler:  errorHandler,
		filterManager: filterManager,
		logger:        logger,
		messages:      make([]llm.Message, 0),
		synthesizer:   synthesizer,
	}
}

// ProcessASRResult 处理ASR识别结果
func (p *Processor) ProcessASRResult(ctx context.Context, text string) {
	if text == "" {
		return
	}

	// 只检查致命错误，不检查processing状态
	// 因为每个完整句子都应该立即处理，不应该等待
	if p.stateManager.IsFatalError() {
		p.logger.Debug("致命错误状态，忽略ASR结果")
		return
	}

	// 如果 TTS 正在播放，取消 TTS 播放（用户打断）
	if p.stateManager.IsTTSPlaying() {
		p.logger.Info("ASR检测到用户说话，中断TTS播放",
			zap.String("user_text", text),
		)
		// 取消当前TTS播放
		p.stateManager.CancelTTS()
		// 设置 TTS 播放状态为 false
		p.stateManager.SetTTSPlaying(false)
	}

	// 如果正在处理 LLM，取消当前的处理，优先处理新的请求
	if p.stateManager.IsProcessing() {
		p.logger.Debug("检测到新的完整句子，取消当前处理以处理新请求",
			zap.String("new_text", text),
		)
		// 重置处理状态，允许处理新请求
		p.stateManager.SetProcessing(false)
	}

	// 提前发送ASR结果给前端，不阻塞后续处理
	if err := p.writer.SendASRResult(text); err != nil {
		p.logger.Error("发送ASR结果失败", zap.Error(err))
		// 发送失败不影响后续处理
	}

	// 检查是否在过滤词黑名单中（在发送ASR结果之后，避免阻塞）
	if p.filterManager != nil && p.filterManager.IsFiltered(text) {
		// 记录被过滤的词（累计计数）
		p.filterManager.RecordFiltered(text)
		count := p.filterManager.GetFilteredCount(text)

		p.logger.Debug("ASR结果被过滤词黑名单过滤，不发送给LLM",
			zap.String("text", text),
			zap.Int("filtered_count", count),
		)
		// 已发送ASR结果，但不调用LLM和TTS
		return
	}

	// 异步处理文本（调用LLM和TTS），不阻塞ASR结果返回
	go p.processText(ctx, text)
}

// processText 处理文本（调用LLM和TTS）
// 注意：此方法在goroutine中异步执行，减少锁持有时间
func (p *Processor) processText(ctx context.Context, text string) {
	// 设置处理状态
	p.stateManager.SetProcessing(true)
	defer p.stateManager.SetProcessing(false)

	// 再次检查状态（可能在goroutine启动期间状态已改变）
	if p.stateManager.IsFatalError() {
		p.logger.Debug("致命错误状态，取消处理")
		return
	}

	// 添加用户消息（最小化锁持有时间）
	userMsg := llm.Message{
		Role:    "user",
		Content: text,
	}
	p.mu.Lock()
	p.messages = append(p.messages, userMsg)
	// 限制消息历史大小（在锁内快速完成）
	if len(p.messages) > MaxMessageHistory {
		keepCount := MaxMessageHistory / 2
		p.messages = p.messages[len(p.messages)-keepCount:]
		p.logger.Debug("消息历史超过限制，已清理旧消息",
			zap.Int("kept", keepCount),
		)
	}
	p.mu.Unlock()

	// 调用LLM（在锁外执行，不阻塞其他操作）
	response, err := p.llmService.Query(ctx, text)
	if err != nil {
		p.handleServiceError(err, "LLM")
		return
	}

	if response == "" {
		p.logger.Warn("LLM返回空响应")
		return
	}

	// 添加助手回复（最小化锁持有时间）
	assistantMsg := llm.Message{
		Role:    "assistant",
		Content: response,
	}
	p.mu.Lock()
	p.messages = append(p.messages, assistantMsg)
	p.mu.Unlock()

	// 发送LLM响应给前端（在锁外执行）
	if err := p.writer.SendLLMResponse(response); err != nil {
		p.logger.Error("发送LLM响应失败", zap.Error(err))
	}

	// 合成TTS（异步执行，不阻塞）
	p.synthesizeTTS(ctx, response)
}

// synthesizeTTS 合成TTS
func (p *Processor) synthesizeTTS(ctx context.Context, text string) {
	if text == "" {
		return
	}

	// 设置TTS播放状态
	p.stateManager.SetTTSPlaying(true)
	defer func() {
		p.stateManager.SetTTSPlaying(false)
		// 发送TTS结束消息
		if err := p.writer.SendTTSEnd(); err != nil {
			p.logger.Error("发送TTS结束消息失败", zap.Error(err))
		}
	}()

	// 获取音频格式并发送TTS开始消息
	if p.synthesizer != nil {
		format := p.synthesizer.Format()
		if err := p.writer.SendTTSStart(format); err != nil {
			p.logger.Error("发送TTS开始消息失败", zap.Error(err))
			return
		}
	}

	// 创建TTS上下文
	ttsCtx, ttsCancel := context.WithCancel(ctx)
	defer ttsCancel()

	// 设置TTS上下文到状态管理器
	p.stateManager.SetTTSCtx(ttsCtx, ttsCancel)

	// 合成语音
	audioChan, err := p.ttsService.Synthesize(ttsCtx, text)
	if err != nil {
		p.handleServiceError(err, "TTS")
		return
	}

	// 发送音频数据
	for {
		select {
		case <-ttsCtx.Done():
			return
		case data, ok := <-audioChan:
			if !ok {
				// 音频数据发送完毕，等待足够长的时间确保所有数据都已写入WebSocket
				// 增加延迟到 500ms，确保二进制数据通过网络传输完成
				time.Sleep(500 * time.Millisecond)
				return
			}
			if data == nil {
				// 错误信号
				return
			}
			if err := p.writer.SendTTSAudio(data); err != nil {
				p.logger.Error("发送TTS音频失败", zap.Error(err))
				return
			}
		}
	}
}

// HandleTextMessage 处理文本消息
func (p *Processor) HandleTextMessage(ctx context.Context, data []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		p.logger.Warn("解析文本消息失败", zap.Error(err))
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		p.logger.Warn("消息类型无效")
		return
	}

	switch msgType {
	case "new_session":
		// 新会话，清空消息历史
		p.mu.Lock()
		p.messages = make([]llm.Message, 0)
		p.mu.Unlock()
		p.logger.Info("新会话开始")

	case "text":
		// 文本消息，直接处理
		text, ok := msg["text"].(string)
		if !ok {
			p.logger.Warn("文本消息格式无效")
			return
		}
		p.processText(ctx, text)
	}
}

// Clear 清空消息历史
func (p *Processor) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = make([]llm.Message, 0)
}

// handleServiceError 统一处理服务错误
// 返回true表示是致命错误，调用者应该立即返回
func (p *Processor) handleServiceError(err error, serviceName string) bool {
	if err == nil {
		return false
	}

	classified := p.errorHandler.HandleError(err, serviceName)
	isFatal := false
	if classifiedErr, ok := classified.(*errhandler.Error); ok {
		isFatal = classifiedErr.Type == errhandler.ErrorTypeFatal
		if isFatal {
			p.stateManager.SetFatalError(true)
		}
	}
	p.writer.SendError(serviceName+"处理失败: "+err.Error(), isFatal)
	return isFatal
}
