package sessions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"go.uber.org/zap"
)

// pipeline architecture for asr handler
// handle data by: audio input -> VAD -> echo filter -> ASR -> sensitive filter -> output text

// Pipeline pipeline abstract interface
type Pipeline interface {
	Process(ctx context.Context, data interface{}) (interface{}, error)
}

// PipelineComponent pipeline component abstract interface
type PipelineComponent interface {
	Name() string
	Process(ctx context.Context, data interface{}) (interface{}, bool, error)
}

// PipelineMetrics ASR Pipeline 性能指标
type PipelineMetrics struct {
	mu              sync.RWMutex
	FirstPacketTime time.Time     // 第一个音频包时间
	LastPacketTime  time.Time     // 最后一个音频包时间
	ASRFirstResult  time.Time     // 第一个 ASR 结果时间
	ASRLatency      time.Duration // ASR 延迟（从最后一个包到第一个结果）
	TotalAudioBytes int           // 总音频字节数（PCM）
	AudioDuration   time.Duration // 音频总时长
	RTF             float64       // Real-Time Factor (处理时间/音频时长)
}

type ASRPipelineOption struct {
	Asr                  recognizer.TranscribeService
	SampleRate           int
	Channels             int
	FrameDuration        string
	EnableVAD            bool
	VADThreshold         float64
	VADConsecutiveFrames int
	VADServiceURL        string // VAD 服务 URL（例如 http://localhost:7073）
	VADSessionID         string // VAD 会话 ID
}

func (ao *ASRPipelineOption) String() string {
	return fmt.Sprintf("ASRPipelineOption{Asr: %s, SampleRate: %d, Channels: %d, FrameDuration: %s}", ao.Asr, ao.SampleRate, ao.Channels, ao.FrameDuration)
}

// ASRPipeline Global ASR pipeline
type ASRPipeline struct {
	inputStages     []PipelineComponent
	Asr             recognizer.TranscribeService
	outputStages    []PipelineComponent
	onOutput        func(text string, isFinal bool)
	onPCMAudio      func(data []byte) error // PCM 音频记录回调
	logger          *zap.Logger
	metrics         *PipelineMetrics
	ttsPlaying      bool
	mu              sync.RWMutex
	vadComponent    *VADComponent      // VAD 组件引用（用于设置回调）
	asrOption       *ASRPipelineOption // 保存 ASR 配置用于重连
	reconnecting    bool               // 是否正在重连
	reconnectCancel context.CancelFunc // 重连取消函数
	ctx             context.Context    // Pipeline 上下文
}

// NewASRPipeline Create a new ASR pipeline
func NewASRPipeline(option *ASRPipelineOption, logger *zap.Logger) (*ASRPipeline, error) {
	if option == nil || option.Asr == nil {
		return nil, fmt.Errorf("invalid option or invalid asr service")
	}
	pipeline := &ASRPipeline{
		Asr:       option.Asr,
		logger:    logger,
		metrics:   &PipelineMetrics{},
		asrOption: option, // 保存配置用于重连
	}
	decode, err := encoder.CreateDecode(
		media.CodecConfig{
			Codec:         "opus",
			SampleRate:    option.SampleRate,
			Channels:      option.Channels,
			BitDepth:      16,
			FrameDuration: option.FrameDuration,
		},
		media.CodecConfig{
			Codec:         "pcm",
			SampleRate:    option.SampleRate,
			Channels:      option.Channels,
			BitDepth:      16,
			FrameDuration: option.FrameDuration,
		},
	)
	if err != nil {
		return nil, err
	}

	pipeline.inputStages = []PipelineComponent{
		&OpusDecodeComponent{logger: logger, decoder: decode},
	}
	if option.EnableVAD {
		vadServiceURL := option.VADServiceURL
		if vadServiceURL == "" {
			vadServiceURL = "http://localhost:7073" // 默认 URL
		}
		vadSessionID := option.VADSessionID
		if vadSessionID == "" {
			vadSessionID = "default" // 默认会话 ID
		}
		vadComponent, err := NewVADComponent(
			option.EnableVAD,
			logger,
			vadServiceURL,
			vadSessionID,
			option.VADThreshold,
		)
		if err != nil {
			logger.Error("[ASR Pipeline] VAD 组件初始化失败，禁用 VAD",
				zap.Error(err),
				zap.String("service_url", vadServiceURL))
		} else {
			vadComponent.SetTTSPlayingCallback(func() bool {
				return pipeline.IsTTSPlaying()
			})
			if option.VADConsecutiveFrames > 0 {
				vadComponent.SetConsecutiveFrames(option.VADConsecutiveFrames)
			}
			pipeline.inputStages = append(pipeline.inputStages, vadComponent)
			pipeline.vadComponent = vadComponent
		}
	} else {
		logger.Info("[ASR Pipeline] VAD 已禁用")
	}
	// 在 VAD 之后添加回声过滤组件
	// 当 TTS 播放时，将音频替换为静音帧，防止 ASR 识别 AI 自己的声音
	pipeline.inputStages = append(pipeline.inputStages, NewEchoFilterComponent(logger, pipeline))

	pipeline.inputStages = append(pipeline.inputStages, &ASRInputComponent{
		asr:      pipeline.Asr,
		logger:   logger,
		metrics:  pipeline.metrics,
		pipeline: pipeline,
	})
	pipeline.outputStages = []PipelineComponent{
		&FilterWordComponent{
			logger:      logger,
			filterWords: make(map[string]bool),
		},
	}
	pipeline.Asr.Init(
		pipeline.onASRResult,
		pipeline.onASRError,
	)
	return pipeline, nil
}

// onASRResult ASR 结果回调
func (p *ASRPipeline) onASRResult(text string, isFinal bool, duration time.Duration, uuid string) {
	p.logger.Info("[Pipeline] onASRResult 被调用",
		zap.String("text", text),
		zap.Bool("isFinal", isFinal),
		zap.Duration("duration", duration),
		zap.String("uuid", uuid))
	if text == "" {
		p.logger.Debug("[Pipeline] 跳过空文本")
		return
	}
	p.metrics.mu.Lock()
	if p.metrics.ASRFirstResult.IsZero() {
		p.metrics.ASRFirstResult = time.Now()
		if !p.metrics.LastPacketTime.IsZero() {
			p.metrics.ASRLatency = p.metrics.ASRFirstResult.Sub(p.metrics.LastPacketTime)
		}
	}
	p.metrics.mu.Unlock()
	ctx := context.Background()
	current := interface{}(text)
	for _, stage := range p.outputStages {
		result, shouldContinue, err := stage.Process(ctx, current)
		if err != nil {
			p.logger.Error("[Pipeline] 输出阶段失败",
				zap.String("stage", stage.Name()),
				zap.Error(err),
			)
			return
		}
		if !shouldContinue {
			p.logger.Info("[Pipeline] 输出被过滤",
				zap.String("stage", stage.Name()),
				zap.String("text", text),
			)
			return
		}
		current = result
	}
	filteredText, ok := current.(string)
	if !ok {
		p.logger.Error("[Pipeline] 输出阶段返回了无效的数据类型")
		return
	}
	if p.onOutput != nil {
		p.onOutput(filteredText, isFinal)
	} else {
		p.logger.Warn("[Pipeline] 输出回调未设置！")
	}
}

// onASRError ASR 错误回调
func (p *ASRPipeline) onASRError(err error, isFatal bool) {
	p.logger.Error("[Pipeline] ASR 错误", zap.Error(err), zap.Bool("fatal", isFatal))
}

// ProcessInput 处理输入（音频）
func (p *ASRPipeline) ProcessInput(ctx context.Context, audioData []byte) error {
	p.metrics.mu.Lock()
	if p.metrics.FirstPacketTime.IsZero() {
		p.metrics.FirstPacketTime = time.Now()
	}
	p.metrics.LastPacketTime = time.Now()
	p.metrics.mu.Unlock()

	current := interface{}(audioData)
	var pcmData []byte // 保存解码后的 PCM 数据

	for i, stage := range p.inputStages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		result, shouldContinue, err := stage.Process(ctx, current)
		if err != nil {
			p.logger.Error("[Pipeline] 输入阶段失败",
				zap.String("stage", stage.Name()),
				zap.Error(err),
			)
			return err
		}

		if !shouldContinue {
			return nil
		}

		// 在 OpusDecodeComponent 之后保存 PCM 数据
		if i == 0 && stage.Name() == constants.COMPONENT_OPUS_DECODE {
			if data, ok := result.([]byte); ok {
				pcmData = data
			}
		}

		current = result
	}

	// 调用 PCM 音频记录回调
	if pcmData != nil && p.onPCMAudio != nil {
		if err := p.onPCMAudio(pcmData); err != nil {
			p.logger.Warn("[Pipeline] PCM 音频记录失败", zap.Error(err))
		}
	}

	return nil
}

// SetOutputCallback 设置输出回调
func (p *ASRPipeline) SetOutputCallback(callback func(text string, isFinal bool)) {
	p.onOutput = callback
}

// SetPCMAudioCallback 设置 PCM 音频记录回调
func (p *ASRPipeline) SetPCMAudioCallback(callback func(data []byte) error) {
	p.onPCMAudio = callback
}

// SetBargeInCallback 设置 Barge-in 回调（当 VAD 检测到用户说话时触发）
func (p *ASRPipeline) SetBargeInCallback(callback func()) {
	if p.vadComponent != nil {
		p.vadComponent.SetBargeInCallback(callback)
		p.logger.Info("[ASR Pipeline] Barge-in 回调已设置")
	}
}

// SetTTSPlaying 设置TTS播放状态
func (p *ASRPipeline) SetTTSPlaying(playing bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ttsPlaying = playing
}

// IsTTSPlaying 检查TTS是否正在播放
func (p *ASRPipeline) IsTTSPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.ttsPlaying {
		return true
	}
	return false
}

// ClearTTSState 清除TTS状态
func (p *ASRPipeline) ClearTTSState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ttsPlaying = false
}

// ResetState 重置状态（用于新的对话轮次）
func (p *ASRPipeline) ResetState() {
	p.metrics.mu.Lock()
	p.metrics.FirstPacketTime = time.Time{}
	p.metrics.LastPacketTime = time.Time{}
	p.metrics.ASRFirstResult = time.Time{}
	p.metrics.ASRLatency = 0
	p.metrics.TotalAudioBytes = 0
	p.metrics.AudioDuration = 0
	p.metrics.RTF = 0
	p.metrics.mu.Unlock()
}

// GetMetrics 获取性能指标（线程安全）
func (p *ASRPipeline) GetMetrics() PipelineMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// 计算音频时长（假设 16kHz, 单声道, 16-bit）
	if p.metrics.TotalAudioBytes > 0 {
		p.metrics.AudioDuration = time.Duration(p.metrics.TotalAudioBytes/2/16000) * time.Second
	}

	// 计算 RTF (Real-Time Factor) RTF = ASR延迟 / 音频时长
	if p.metrics.AudioDuration > 0 && p.metrics.ASRLatency > 0 {
		p.metrics.RTF = float64(p.metrics.ASRLatency) / float64(p.metrics.AudioDuration)
	}

	return *p.metrics
}

// PrintMetrics 打印性能指标
func (p *ASRPipeline) PrintMetrics() {
	metrics := p.GetMetrics()

	p.logger.Info("=== ASR Pipeline 性能指标 ===")
	p.logger.Info("ASR 延迟:",
		zap.Duration("latency", metrics.ASRLatency),
		zap.String("说明", "从最后一个音频包到第一个识别结果"),
	)
	p.logger.Info("实时性能:",
		zap.Float64("rtf", metrics.RTF),
		zap.Duration("audio_duration", metrics.AudioDuration),
		zap.String("performance", getRTFDescription(metrics.RTF)),
	)
}

// getRTFDescription 获取 RTF 性能描述
func getRTFDescription(rtf float64) string {
	if rtf < 0.1 {
		return "优秀 (Excellent) - 实时性极佳"
	} else if rtf < 0.3 {
		return "良好 (Good) - 实时性好"
	} else if rtf < 0.5 {
		return "可接受 (Acceptable) - 接近实时"
	} else {
		return "需要优化 (Needs Optimization) - 延迟较高"
	}
}

// TriggerReconnect 触发 ASR 重连
func (p *ASRPipeline) TriggerReconnect() {
	p.mu.Lock()
	if p.reconnecting {
		p.logger.Debug("[ASR Pipeline] 已在重连中，跳过")
		p.mu.Unlock()
		return
	}
	p.reconnecting = true
	p.mu.Unlock()

	p.logger.Warn("[ASR Pipeline] 检测到 ASR 连接断开，开始重连...")

	// 取消之前的重连（如果有）
	if p.reconnectCancel != nil {
		p.reconnectCancel()
	}

	// 创建新的重连上下文
	ctx, cancel := context.WithCancel(context.Background())
	p.reconnectCancel = cancel

	// 重连逻辑
	go p.reconnectASR(ctx)
}

// reconnectASR ASR 重连逻辑（指数退避）
func (p *ASRPipeline) reconnectASR(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		p.reconnecting = false
		p.mu.Unlock()
	}()

	maxRetries := 5
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			p.logger.Info("[ASR Pipeline] 重连被取消")
			return
		default:
		}

		p.logger.Info("[ASR Pipeline] 尝试重连 ASR",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries))

		// 尝试重新连接
		dialogID := fmt.Sprintf("dialog_reconnect_%d", time.Now().Unix())
		err := p.Asr.ConnAndReceive(dialogID)
		if err == nil {
			p.logger.Info("[ASR Pipeline] ASR 重连成功")
			return
		}

		p.logger.Warn("[ASR Pipeline] ASR 重连失败",
			zap.Int("attempt", attempt),
			zap.Error(err))

		// 如果是最后一次尝试，放弃
		if attempt >= maxRetries {
			p.logger.Error("[ASR Pipeline] ASR 重连失败，已达最大重试次数")
			return
		}

		// 计算退避延迟（指数退避）
		delay := time.Duration(attempt) * baseDelay
		if delay > maxDelay {
			delay = maxDelay
		}

		p.logger.Info("[ASR Pipeline] 等待后重试",
			zap.Duration("delay", delay))

		// 等待后重试
		select {
		case <-ctx.Done():
			p.logger.Info("[ASR Pipeline] 重连被取消")
			return
		case <-time.After(delay):
		}
	}
}

// StopReconnect 停止重连
func (p *ASRPipeline) StopReconnect() {
	if p.reconnectCancel != nil {
		p.reconnectCancel()
	}
}
