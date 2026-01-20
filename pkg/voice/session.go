package voice

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voice/errhandler"
	"github.com/LingByte/SoulNexus/pkg/voice/factory"
	"github.com/LingByte/SoulNexus/pkg/voice/filter"
	"github.com/LingByte/SoulNexus/pkg/voice/llm"
	"github.com/LingByte/SoulNexus/pkg/voice/message"
	"github.com/LingByte/SoulNexus/pkg/voice/state"
	"github.com/LingByte/SoulNexus/pkg/voice/tts"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// AudioSampleRate 音频采样率，用于计算音频大小
	AudioSampleRate = 32000
	// ASRUsageRecordTimeout ASR使用量记录超时时间
	ASRUsageRecordTimeout = 5 * time.Second
	// ASRConnectionWaitDelay ASR连接建立后的等待时间
	ASRConnectionWaitDelay = 500 * time.Millisecond
)

// Session 语音会话实现
type Session struct {
	config        *SessionConfig
	ctx           context.Context
	cancel        context.CancelFunc
	stateManager  *state.Manager
	errorHandler  *errhandler.Handler
	asrService    *asr.Service
	ttsService    *tts.Service
	llmService    *llm.Service
	messageWriter *message.Writer
	processor     *message.Processor
	vadDetector   *VADDetector // VAD 检测器用于 barge-in
	mu            sync.RWMutex
	active        bool
}

// NewSession 创建新的语音会话
func NewSession(config *SessionConfig) (*Session, error) {
	if config == nil {
		return nil, errhandler.NewRecoverableError("Session", "配置不能为空", nil)
	}

	if config.Conn == nil {
		return nil, errhandler.NewRecoverableError("Session", "WebSocket连接不能为空", nil)
	}

	if config.Logger == nil {
		config.Logger = zap.L()
	}

	if config.Context == nil {
		config.Context = context.Background()
	}

	ctx, cancel := context.WithCancel(config.Context)

	// 创建状态管理器
	stateManager := state.NewManager()

	// 创建错误处理器
	errorHandler := errhandler.NewHandler(config.Logger)

	// 创建 VAD 检测器，使用配置中的参数
	vadDetector := NewVADDetector()
	vadDetector.SetLogger(config.Logger) // 设置日志记录器
	if config.EnableVAD {
		vadDetector.SetEnabled(true)
		vadDetector.SetThreshold(config.VADThreshold)
		// 如果阈值很低（<200），自动降低连续帧数要求以提高灵敏度
		consecutiveFrames := config.VADConsecutiveFrames
		if config.VADThreshold < 200 && consecutiveFrames > 1 {
			consecutiveFrames = 1
			config.Logger.Info("VAD阈值较低，自动降低连续帧数要求以提高灵敏度",
				zap.Float64("threshold", config.VADThreshold),
				zap.Int("originalFrames", config.VADConsecutiveFrames),
				zap.Int("adjustedFrames", consecutiveFrames),
			)
		}
		vadDetector.SetConsecutiveFrames(consecutiveFrames)
	} else {
		vadDetector.SetEnabled(false)
	}

	// 创建服务工厂
	transcriberFactory := recognizer.GetGlobalFactory()
	serviceFactory := factory.NewServiceFactory(transcriberFactory, config.Logger)

	// 创建消息写入器
	messageWriter := message.NewWriter(config.Conn, config.Logger)

	// 创建ASR服务
	transcriber, err := serviceFactory.CreateASR(config.Credential, config.Language)
	if err != nil {
		cancel()
		return nil, errhandler.NewRecoverableError("Session", "创建ASR服务失败", err)
	}

	asrService := asr.NewService(
		ctx,
		config.Credential,
		config.Language,
		transcriber,
		errorHandler,
		config.Logger,
	)

	// 如果提供了连接池，设置到ASR服务
	if config.ASRPool != nil {
		asrService.SetPool(config.ASRPool)
	}

	// 创建TTS服务
	synthesizer, err := serviceFactory.CreateTTS(config.Credential, config.Speaker)
	if err != nil {
		cancel()
		return nil, errhandler.NewRecoverableError("Session", "创建TTS服务失败", err)
	}

	ttsService := tts.NewService(
		ctx,
		config.Credential,
		config.Speaker,
		synthesizer,
		errorHandler,
		config.Logger,
	)

	// 创建LLM服务
	llmProvider, err := serviceFactory.CreateLLM(ctx, config.Credential, config.SystemPrompt)
	if err != nil {
		cancel()
		return nil, errhandler.NewRecoverableError("Session", "创建LLM服务失败", err)
	}

	llmService := llm.NewService(
		ctx,
		config.Credential,
		config.SystemPrompt,
		config.LLMModel,
		config.Temperature,
		config.MaxTokens,
		llmProvider,
		errorHandler,
		config.Logger,
	)

	// 创建过滤词管理器
	filterManager, err := filter.NewManager(config.Logger)
	if err != nil {
		config.Logger.Warn("创建过滤词管理器失败，将不使用过滤功能", zap.Error(err))
		filterManager = nil
	}

	// 创建消息处理器
	processor := message.NewProcessor(
		stateManager,
		llmService,
		ttsService,
		messageWriter,
		errorHandler,
		config.Logger,
		synthesizer,   // 传递synthesizer以获取音频格式
		filterManager, // 传递过滤词管理器
	)

	// 设置ASR回调
	asrService.SetCallbacks(
		func(text string, isLast bool, duration time.Duration, uuid string) {
			// 记录ASR使用量
			if isLast && config.DB != nil && config.Credential != nil && duration > 0 {
				go recordASRUsage(ctx, config, duration, uuid, config.Logger)
			}

			// 处理ASR结果
			incremental := stateManager.UpdateASRText(text, isLast)
			if incremental != "" {
				processor.ProcessASRResult(ctx, incremental)
			}
		},
		func(err error) {
			classified := errorHandler.HandleError(err, "ASR")
			if classifiedErr, ok := classified.(*errhandler.Error); ok && classifiedErr.Type == errhandler.ErrorTypeFatal {
				stateManager.SetFatalError(true)
				messageWriter.SendError("ASR错误: "+err.Error(), true)
			}
		},
	)

	session := &Session{
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		stateManager:  stateManager,
		errorHandler:  errorHandler,
		asrService:    asrService,
		ttsService:    ttsService,
		llmService:    llmService,
		messageWriter: messageWriter,
		processor:     processor,
		vadDetector:   vadDetector,
		active:        false,
	}

	return session, nil
}

// Start 启动会话
func (s *Session) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return nil
	}

	// 连接ASR服务
	if err := s.asrService.Connect(); err != nil {
		return errhandler.NewRecoverableError("Session", "连接ASR服务失败", err)
	}

	// 等待ASR连接建立
	time.Sleep(ASRConnectionWaitDelay)

	// 发送连接成功消息
	if err := s.messageWriter.SendConnected(); err != nil {
		s.config.Logger.Error("发送连接成功消息失败", zap.Error(err))
	}

	s.active = true

	// 启动消息处理循环
	go s.messageLoop()

	return nil
}

// Stop 停止会话
func (s *Session) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	s.cancel()

	// 断开ASR服务
	if s.asrService != nil {
		s.asrService.Disconnect()
	}

	// 关闭TTS服务
	if s.ttsService != nil {
		s.ttsService.Close()
	}

	// 关闭LLM服务
	if s.llmService != nil {
		s.llmService.Close()
	}

	// 关闭消息写入器
	if s.messageWriter != nil {
		s.messageWriter.Close()
	}

	// 清空状态
	if s.stateManager != nil {
		s.stateManager.Clear()
	}

	s.active = false

	return nil
}

// HandleAudio 处理音频数据
func (s *Session) HandleAudio(data []byte) error {
	s.mu.RLock()
	active := s.active
	vadDetector := s.vadDetector
	ttsPlaying := s.stateManager.IsTTSPlaying()
	s.mu.RUnlock()

	if !active {
		return errhandler.NewRecoverableError("Session", "会话未激活", nil)
	}

	// 致命错误时忽略音频
	if s.stateManager.IsFatalError() {
		return nil
	}

	// 如果 TTS 正在播放，使用 VAD 检测 barge-in
	if ttsPlaying {
		// 检测用户是否说话（barge-in）
		if vadDetector.CheckBargeIn(data, true) {
			s.config.Logger.Info("检测到用户说话，中断 TTS")
			// 取消 TTS 播放
			s.stateManager.CancelTTS()
			// 设置 TTS 播放状态为 false
			s.stateManager.SetTTSPlaying(false)
			// 继续处理音频（用户开始说话了）
			return s.asrService.SendAudio(data)
		}
		// TTS 播放中且未检测到用户说话，忽略音频输入
		return nil
	}

	// TTS 未播放，正常处理音频
	return s.asrService.SendAudio(data)
}

// HandleText 处理文本消息
func (s *Session) HandleText(data []byte) error {
	s.mu.RLock()
	active := s.active
	processor := s.processor
	ctx := s.ctx
	s.mu.RUnlock()

	if !active {
		return errhandler.NewRecoverableError("Session", "会话未激活", nil)
	}

	processor.HandleTextMessage(ctx, data)
	return nil
}

// IsActive 检查会话是否活跃
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// messageLoop 消息处理循环
func (s *Session) messageLoop() {
	defer func() {
		// 当消息循环退出时，触发优雅关闭
		s.config.Logger.Info("消息循环退出，触发会话关闭")
		s.cancel()
	}()

	for {
		select {
		case <-s.ctx.Done():
			s.config.Logger.Info("消息循环退出")
			return
		default:
		}

		messageType, message, err := s.config.Conn.ReadMessage()
		if err != nil {
			// 检查是否是正常的关闭错误
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				s.config.Logger.Debug("WebSocket连接正常关闭", zap.Error(err))
			} else {
				s.config.Logger.Debug("读取WebSocket消息失败", zap.Error(err))
			}
			// 取消context，触发优雅关闭
			s.cancel()
			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			// 音频消息
			if err := s.HandleAudio(message); err != nil {
				s.config.Logger.Warn("处理音频消息失败", zap.Error(err))
			}
		case websocket.TextMessage:
			// 文本消息
			if err := s.HandleText(message); err != nil {
				s.config.Logger.Warn("处理文本消息失败", zap.Error(err))
			}
		}
	}
}

// recordASRUsage 记录ASR使用量
func recordASRUsage(ctx context.Context, config *SessionConfig, duration time.Duration, uuid string, logger *zap.Logger) {
	// 创建带超时的上下文
	recordCtx, cancel := context.WithTimeout(ctx, ASRUsageRecordTimeout)
	defer cancel()

	// 计算音频大小
	audioSize := int64(duration.Seconds() * AudioSampleRate)

	// 准备助手ID
	var assistantID *uint
	if config.AssistantID > 0 {
		aid := uint(config.AssistantID)
		assistantID = &aid
	}

	// 获取组织ID（如果助手属于组织）
	var groupID *uint
	if assistantID != nil && config.DB != nil {
		var assistant models.Assistant
		if err := config.DB.WithContext(recordCtx).Where("id = ?", *assistantID).First(&assistant).Error; err == nil {
			groupID = assistant.GroupID
		} else if err != gorm.ErrRecordNotFound {
			logger.Warn("查询助手信息失败", zap.Error(err))
		}
	}

	// 记录使用量
	if err := models.RecordASRUsage(
		config.DB,
		config.Credential.UserID,
		config.Credential.ID,
		assistantID,
		groupID,
		uuid,
		int(duration.Seconds()),
		audioSize,
	); err != nil {
		logger.Warn("记录ASR使用量失败", zap.Error(err))
	}
}
