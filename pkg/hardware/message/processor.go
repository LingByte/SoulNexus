package message

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/hardware/audio"
	"github.com/LingByte/SoulNexus/pkg/hardware/errhandler"
	"github.com/LingByte/SoulNexus/pkg/hardware/filter"
	llmv2 "github.com/LingByte/SoulNexus/pkg/hardware/llm"
	"github.com/LingByte/SoulNexus/pkg/hardware/state"
	"github.com/LingByte/SoulNexus/pkg/hardware/tts"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"go.uber.org/zap"
)

// Processor 消息处理器
type Processor struct {
	stateManager  *state.Manager
	llmService    *llmv2.Service
	ttsService    *tts.Service
	writer        *Writer
	errorHandler  *errhandler.Handler
	filterManager *filter.Manager
	audioManager  *audio.Manager
	logger        *zap.Logger
	mu            sync.RWMutex
	messages      []llm.Message
	synthesizer   synthesizer.SynthesisService // 用于获取音频格式

	// OPUS编码相关（用于硬件协议）
	audioFormat string
	sampleRate  int
	channels    int
	opusEncoder media.EncoderFunc // PCM -> OPUS (for TTS)
}

// NewProcessor 创建消息处理器
func NewProcessor(
	stateManager *state.Manager,
	llmService *llmv2.Service,
	ttsService *tts.Service,
	writer *Writer,
	errorHandler *errhandler.Handler,
	logger *zap.Logger,
	synthesizer synthesizer.SynthesisService,
	filterManager *filter.Manager,
	audioManager *audio.Manager,
) *Processor {
	return &Processor{
		stateManager:  stateManager,
		llmService:    llmService,
		ttsService:    ttsService,
		writer:        writer,
		errorHandler:  errorHandler,
		filterManager: filterManager,
		audioManager:  audioManager,
		logger:        logger,
		messages:      make([]llm.Message, 0),
		synthesizer:   synthesizer,
		audioFormat:   "opus",
		sampleRate:    16000,
		channels:      1,
	}
}

// SetAudioConfig 设置音频配置（用于OPUS编码）
func (p *Processor) SetAudioConfig(audioFormat string, sampleRate, channels int, opusEncoder media.EncoderFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.audioFormat = audioFormat
	p.sampleRate = sampleRate
	p.channels = channels
	p.opusEncoder = opusEncoder
}

// ProcessASRResult 处理ASR识别结果
func (p *Processor) ProcessASRResult(ctx context.Context, text string) {
	if text == "" {
		return
	}

	// 检查致命错误
	if p.stateManager.IsFatalError() {
		p.logger.Debug("致命错误状态，忽略ASR结果")
		return
	}

	// 如果 TTS 正在播放，取消 TTS 播放（用户打断）
	if p.stateManager.IsTTSPlaying() {
		p.logger.Info("ASR检测到用户说话，中断TTS播放",
			zap.String("user_text", text),
		)
		// 先取消 TTS context，停止音频合成和发送
		p.stateManager.CancelTTS()
		// 然后设置 TTS 播放状态为 false
		p.stateManager.SetTTSPlaying(false)
		// 发送 TTS 结束消息，通知前端停止播放
		if err := p.writer.SendTTSEnd(); err != nil {
			p.logger.Warn("发送TTS结束消息失败", zap.Error(err))
		}
	}

	// 提前发送ASR结果给前端，不阻塞后续处理
	if err := p.writer.SendASRResult(text); err != nil {
		p.logger.Error("发送ASR结果失败", zap.Error(err))
		// 发送失败不影响后续处理
	}

	// 检查是否在过滤词黑名单中
	if p.filterManager != nil && p.filterManager.IsFiltered(text) {
		p.filterManager.RecordFiltered(text)
		count := p.filterManager.GetFilteredCount(text)

		p.logger.Debug("ASR结果被过滤词黑名单过滤，不发送给LLM",
			zap.String("text", text),
			zap.Int("filtered_count", count),
		)
		// 已发送ASR结果，但不调用LLM和TTS
		return
	}

	// 如果正在处理 LLM，取消当前的处理，优先处理新请求
	if p.stateManager.IsProcessing() {
		p.logger.Debug("检测到新的完整句子，取消当前处理以处理新请求",
			zap.String("new_text", text),
		)
		// 重置处理状态，允许处理新请求
		p.stateManager.SetProcessing(false)
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

	// 再次检查状态
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
	// 限制消息历史大小
	const maxMessageHistory = 100
	if len(p.messages) > maxMessageHistory {
		keepCount := maxMessageHistory / 2
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
	// 添加"。。。。。。"以保持与hardware包的一致性
	if err := p.writer.SendLLMResponse(response + "。。。。。。"); err != nil {
		p.logger.Error("发送LLM响应失败", zap.Error(err))
	}

	// 合成TTS（在goroutine中异步执行，不阻塞）
	p.logger.Info("准备启动TTS合成", zap.String("text", response))
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.logger.Error("TTS合成发生panic", zap.Any("panic", r))
			}
		}()
		p.synthesizeTTS(ctx, response)
	}()
}

// synthesizeTTS 合成TTS
func (p *Processor) synthesizeTTS(ctx context.Context, text string) {
	if text == "" {
		p.logger.Warn("TTS文本为空，跳过合成")
		return
	}

	p.logger.Info("开始TTS合成", zap.String("text", text))

	// 设置TTS播放状态
	p.stateManager.SetTTSPlaying(true)
	defer func() {
		p.stateManager.SetTTSPlaying(false)
		p.logger.Info("TTS播放结束")
		// 发送TTS结束消息
		if err := p.writer.SendTTSEnd(); err != nil {
			p.logger.Error("发送TTS结束消息失败", zap.Error(err))
		}
	}()

	// 获取音频格式并发送TTS开始消息
	if p.synthesizer == nil {
		p.logger.Error("TTS合成器未初始化，无法合成语音")
		return
	}

	format := p.synthesizer.Format()
	p.logger.Info("发送TTS开始消息",
		zap.Int("sampleRate", format.SampleRate),
		zap.Int("channels", format.Channels),
		zap.Int("bitDepth", format.BitDepth),
	)
	if err := p.writer.SendTTSStart(format); err != nil {
		p.logger.Error("发送TTS开始消息失败", zap.Error(err))
		return
	}

	// 重置TTS流控状态（新的TTS会话开始）
	p.writer.ResetTTSFlowControl()

	// 创建TTS上下文
	ttsCtx, ttsCancel := context.WithCancel(ctx)
	defer ttsCancel()

	// 设置TTS上下文到状态管理器
	p.stateManager.SetTTSCtx(ttsCtx, ttsCancel)

	// 合成语音
	// 在文本末尾添加"。。。。。。"（6个中文句号），让TTS生成更多音频数据
	// 这样可以确保硬件端有足够的音频数据，避免提前停止
	ttsText := text + "。。。。。。"
	p.logger.Debug("TTS合成文本", zap.String("original", text), zap.String("withPadding", ttsText))

	audioChan, err := p.ttsService.Synthesize(ttsCtx, ttsText)
	if err != nil {
		p.logger.Error("TTS合成失败", zap.Error(err))
		p.handleServiceError(err, "TTS")
		return
	}

	p.logger.Info("TTS合成已启动，等待音频数据")

	// 发送音频数据
	p.mu.RLock()
	audioFormat := p.audioFormat
	opusEncoder := p.opusEncoder
	sampleRate := p.sampleRate
	channels := p.channels
	p.mu.RUnlock()

	var pcmBuffer []byte // 累积PCM数据（用于OPUS编码）
	var totalBytesReceived int
	var frameCount int

	for {
		select {
		case <-ttsCtx.Done():
			p.logger.Info("TTS合成被取消（context done）")
			return
		case data, ok := <-audioChan:
			if !ok {
				p.logger.Info("TTS音频通道已关闭，发送剩余数据",
					zap.Int("totalBytes", totalBytesReceived),
					zap.Int("frameCount", frameCount),
					zap.Int("bufferSize", len(pcmBuffer)),
				)
				// 发送缓冲区剩余数据
				if audioFormat == "opus" && opusEncoder != nil && len(pcmBuffer) > 0 {
					p.sendRemainingOPUSFrames(pcmBuffer, opusEncoder, sampleRate, channels)
				}
				// 发送填充帧，确保硬件播放完整
				if audioFormat == "opus" && opusEncoder != nil {
					p.logger.Info("发送填充帧，确保硬件播放完整")
					p.sendPaddingFrames(opusEncoder, sampleRate, channels)
				}
				// 等待硬件播放完缓冲区
				// 1800ms (1.8秒) 确保硬件端有足够时间播放完所有音频
				waitDuration := 300 * time.Millisecond
				p.logger.Info("等待硬件播放完缓冲区", zap.Duration("wait", waitDuration))
				time.Sleep(waitDuration)
				p.logger.Info("TTS合成完成")
				return
			}
			if data == nil {
				// 错误信号
				p.logger.Warn("收到TTS错误信号（nil数据）")
				return
			}

			totalBytesReceived += len(data)

			// 记录TTS输出到音频管理器（用于回声消除）
			if p.audioManager != nil {
				p.audioManager.RecordTTSOutput(data)
			}

			// 如果是OPUS格式，需要编码PCM -> OPUS
			if audioFormat == "opus" && opusEncoder != nil {
				// 将新数据追加到缓冲区
				pcmBuffer = append(pcmBuffer, data...)

				// 计算每帧的字节数（60ms @ sampleRate, channels, 16-bit）
				frameSize := sampleRate * 60 / 1000 * channels * 2

				// 逐帧编码和发送
				for len(pcmBuffer) >= frameSize {
					// 检查 context 是否被取消
					select {
					case <-ttsCtx.Done():
						p.logger.Info("TTS合成被取消，停止发送音频")
						return
					default:
					}

					// 取出一帧数据
					frameData := pcmBuffer[:frameSize]
					pcmBuffer = pcmBuffer[frameSize:]

					// 检测静音帧并处理
					frameData = p.processSilentFrame(frameData)

					// 编码这一帧
					audioFrame := &media.AudioPacket{Payload: frameData}
					frames, err := opusEncoder(audioFrame)
					if err != nil {
						p.logger.Error("OPUS编码失败", zap.Error(err))
						continue
					}

					if len(frames) > 0 {
						if af, ok := frames[0].(*media.AudioPacket); ok {
							// 发送编码后的OPUS数据（带流控）
							frameCount++
							// 使用固定延迟（60ms）发送，避免长时间播放时时间同步累积误差导致发送过快
							if err := p.writer.SendTTSAudioWithFlowControl(af.Payload, 60, 60); err != nil {
								p.logger.Error("发送TTS音频失败", zap.Error(err))
								return
							}
							// 每10帧记录一次
							if frameCount%10 == 0 {
								p.logger.Debug("已发送TTS音频帧",
									zap.Int("frameCount", frameCount),
									zap.Int("opusSize", len(af.Payload)),
									zap.Int("totalBytes", totalBytesReceived),
								)
							}
						}
					}
				}
			} else {
				// PCM格式，直接发送（带流控）
				// 检查 context 是否被取消
				select {
				case <-ttsCtx.Done():
					p.logger.Info("TTS合成被取消，停止发送音频")
					return
				default:
				}
				// 使用固定延迟（60ms）发送，避免长时间播放时时间同步累积误差导致发送过快
				if err := p.writer.SendTTSAudioWithFlowControl(data, 60, 60); err != nil {
					p.logger.Error("发送TTS音频失败", zap.Error(err))
					return
				}
			}
		}
	}
}

// processSilentFrame 处理静音帧（替换为低音量白噪声，避免DTX包）
func (p *Processor) processSilentFrame(frameData []byte) []byte {
	// 计算音频能量
	var energy int64
	for i := 0; i < len(frameData); i += 2 {
		sample := int16(frameData[i]) | (int16(frameData[i+1]) << 8)
		energy += int64(sample) * int64(sample)
	}
	avgEnergy := energy / int64(len(frameData)/2)

	// 如果是完全静音，替换为中等音量白噪声（±500范围）
	// 这样可以确保硬件端不会认为是静音而提前停止
	if avgEnergy == 0 {
		noiseData := make([]byte, len(frameData))
		seed := len(frameData)*7919 + 3571
		for i := 0; i < len(frameData); i += 2 {
			// 使用线性同余生成器生成高质量伪随机数
			seed = (seed*1103515245 + 12345) & 0x7fffffff
			noise := int16((seed % 1001) - 500) // ±500 范围，与填充帧一致
			noiseData[i] = byte(noise & 0xFF)
			noiseData[i+1] = byte((noise >> 8) & 0xFF)
		}
		return noiseData
	}

	return frameData
}

// sendRemainingOPUSFrames 发送缓冲区剩余的OPUS帧
func (p *Processor) sendRemainingOPUSFrames(pcmBuffer []byte, opusEncoder media.EncoderFunc, sampleRate, channels int) {
	frameSize := sampleRate * 60 / 1000 * channels * 2

	// 处理完整的帧
	for len(pcmBuffer) >= frameSize {
		frameData := pcmBuffer[:frameSize]
		pcmBuffer = pcmBuffer[frameSize:]

		frameData = p.processSilentFrame(frameData)

		audioFrame := &media.AudioPacket{Payload: frameData}
		frames, err := opusEncoder(audioFrame)
		if err != nil {
			p.logger.Error("编码剩余帧失败", zap.Error(err))
			continue
		}

		if len(frames) > 0 {
			if af, ok := frames[0].(*media.AudioPacket); ok {
				// 使用固定延迟（60ms）发送剩余帧，确保时序正确
				if err := p.writer.SendTTSAudioWithFlowControl(af.Payload, 60, 60); err != nil {
					p.logger.Error("发送剩余帧失败", zap.Error(err))
				}
			}
		}
	}

	// 处理最后的不完整帧（如果足够大）
	if len(pcmBuffer) >= 100 {
		// 填充到完整帧
		paddedBuffer := make([]byte, frameSize)
		copy(paddedBuffer, pcmBuffer)

		// 用最后一个样本填充
		if len(pcmBuffer) >= 2 {
			lastSample := []byte{pcmBuffer[len(pcmBuffer)-2], pcmBuffer[len(pcmBuffer)-1]}
			for i := len(pcmBuffer); i < frameSize; i += 2 {
				paddedBuffer[i] = lastSample[0]
				if i+1 < frameSize {
					paddedBuffer[i+1] = lastSample[1]
				}
			}
		}

		audioFrame := &media.AudioPacket{Payload: paddedBuffer}
		frames, err := opusEncoder(audioFrame)
		if err == nil && len(frames) > 0 {
			if af, ok := frames[0].(*media.AudioPacket); ok {
				// 使用固定延迟（60ms）发送不完整帧，确保时序正确
				p.writer.SendTTSAudioWithFlowControl(af.Payload, 60, 60)
			}
		}
	}
}

// sendPaddingFrames 发送填充帧，确保硬件播放完整
func (p *Processor) sendPaddingFrames(opusEncoder media.EncoderFunc, sampleRate, channels int) {
	frameSize := sampleRate * 60 / 1000 * channels * 2

	p.logger.Info("发送填充帧，确保音频完整播放", zap.Int("frameCount", 5))

	// 发送5帧填充帧（与hardware包保持一致）
	for i := 0; i < 5; i++ {
		// 生成中等音量白噪声帧（±500 范围）
		// 这个音量足够让硬件端识别为有效音频，不会提前停止
		paddingFrame := make([]byte, frameSize)
		for j := 0; j < frameSize; j += 2 {
			// 使用伪随机数生成白噪声（±500 范围）
			// 使用线性同余生成器生成高质量伪随机数，避免周期性模式
			seed := j*7919 + i*3571
			seed = (seed*1103515245 + 12345) & 0x7fffffff
			noise := int16((seed % 1001) - 500) // ±500 范围
			paddingFrame[j] = byte(noise & 0xFF)
			paddingFrame[j+1] = byte((noise >> 8) & 0xFF)
		}

		// 编码并发送
		audioFrame := &media.AudioPacket{Payload: paddingFrame}
		frames, err := opusEncoder(audioFrame)
		if err != nil {
			p.logger.Error("编码填充帧失败", zap.Error(err))
			break
		}

		if len(frames) > 0 {
			if af, ok := frames[0].(*media.AudioPacket); ok {
				// 使用固定延迟（60ms）发送填充帧，确保时序正确，避免声音混杂快速
				if err := p.writer.SendTTSAudioWithFlowControl(af.Payload, 60, 60); err != nil {
					p.logger.Error("发送填充帧失败", zap.Error(err))
					break
				}
				p.logger.Debug("填充帧已发送", zap.Int("frameIndex", i), zap.Int("opusSize", len(af.Payload)))
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

	case "ping":
		// 心跳消息，发送pong
		// 注意：这里不发送pong，由writer处理

	case "hello":
		// xiaozhi协议hello消息，由session处理
		p.logger.Debug("收到hello消息，由session处理")
	}
}

// Clear 清空消息历史
func (p *Processor) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = make([]llm.Message, 0)
}

// SetSynthesizer 设置合成器（用于重新初始化TTS服务时更新）
func (p *Processor) SetSynthesizer(synthesizer synthesizer.SynthesisService) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.synthesizer = synthesizer
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
