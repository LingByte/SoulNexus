package protocol

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/LingByte/SoulNexus/pkg/hardware/sessions"
	"github.com/LingByte/SoulNexus/pkg/hardware/stream"
	"github.com/LingByte/SoulNexus/pkg/hardware/tools"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/voiceclone"
	"github.com/LingByte/SoulNexus/pkg/voiceprint"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HardwareSessionOption hardware session options
type HardwareSessionOption struct {
	Conn                 *websocket.Conn
	Logger               *zap.Logger
	AssistantID          uint
	Credential           *models.UserCredential
	LLMModel             string
	SystemPrompt         string
	MaxToken             int
	Speaker              string
	EnableVAD            bool
	VADThreshold         float64
	VADConsecutiveFrames int
	DB                   *gorm.DB
	VoiceprintService    *voiceprint.Service
	DeviceID             *string
	MacAddress           string
	VoiceCloneID         *int
}

// HardwareSession hardware session
type HardwareSession struct {
	sessionID         string
	config            *HardwareSessionOption
	conn              *websocket.Conn
	cancel            context.CancelFunc
	writer            *HardwareWriter
	asrPipeline       *sessions.ASRPipeline
	stateManager      *sessions.ASRStateManager
	llmService        *tools.LLMService
	ttsPipeline       *stream.TTSPipeline
	speakerManager    *tools.SpeakerManager
	voiceprintTool    *tools.VoiceprintIdentifyTool // voice print tools
	ctx               context.Context
	logger            *zap.Logger
	mu                sync.RWMutex
	active            bool
	goodbyePending    bool
	llmProcessing     bool
	currentSpeakerID  string                                  // current speaker ID, for switch speaker
	ttsServiceCache   map[string]synthesizer.SynthesisService // TTS service cache：speakerID -> service
	db                *gorm.DB
	voiceprintService *voiceprint.Service
	callRecording     *models.CallRecording
	recorder          *AudioRecorder
	conversationTurns []models.ConversationTurn // 对话轮次列表
	currentTurnID     int                       // 当前轮次ID
}

func NewHardwareSession(ctx context.Context, hardwareConfig *HardwareSessionOption) *HardwareSession {
	if hardwareConfig.Logger == nil {
		hardwareConfig.Logger = zap.L()
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	writer := NewHardwareWriter(sessionCtx, hardwareConfig.Conn, hardwareConfig.Logger)
	llmService, err := tools.NewLLMService(&tools.LLMConfig{
		APIKey:       hardwareConfig.Credential.LLMApiKey,
		BaseURL:      hardwareConfig.Credential.LLMApiURL,
		Provider:     hardwareConfig.Credential.LLMProvider,
		SystemPrompt: fmt.Sprintf(constants.LLMSystemPromptTemplate, hardwareConfig.SystemPrompt, hardwareConfig.MaxToken),
		MaxTokens:    hardwareConfig.MaxToken,
	}, hardwareConfig.Logger)
	if err != nil {
		logger.Fatal("创建 LLM 服务失败", zap.Error(err))
	}
	tools.RegisterBuiltinTools(llmService)
	speakerManager := tools.NewSpeakerManager(hardwareConfig.Logger)
	tools.RegisterSpeakerTool(llmService, speakerManager)
	ttsProvider := hardwareConfig.Credential.GetTTSProvider()
	ttsConfig := make(synthesizer.TTSCredentialConfig)
	ttsConfig["provider"] = ttsProvider
	if hardwareConfig.Credential.TtsConfig != nil {
		for key, value := range hardwareConfig.Credential.TtsConfig {
			ttsConfig[key] = value
		}
	}
	if _, exists := ttsConfig["voiceType"]; !exists {
		ttsConfig["voiceType"] = hardwareConfig.Speaker
	}

	var ttsService synthesizer.SynthesisService
	if hardwareConfig.VoiceCloneID != nil && *hardwareConfig.VoiceCloneID > 0 {
		voiceClone, voiceCloneErr := models.GetVoiceCloneByID(hardwareConfig.DB, int64(*hardwareConfig.VoiceCloneID))
		if voiceCloneErr == nil && voiceClone != nil {
			cloneFactory := voiceclone.NewFactory()
			cloneConfig := &voiceclone.Config{
				Provider: voiceclone.Provider(voiceClone.Provider),
				Options:  make(map[string]interface{}),
			}
			if voiceClone.Provider == "xunfei" {
				if hardwareConfig.Credential.AsrConfig != nil {
					if appID, ok := hardwareConfig.Credential.AsrConfig["appId"].(string); ok {
						cloneConfig.Options["app_id"] = appID
					}
				}
				cloneConfig.Options["api_key"] = hardwareConfig.Credential.LLMApiKey
				if wsAppID := os.Getenv("XUNFEI_WS_APP_ID"); wsAppID != "" {
					cloneConfig.Options["ws_app_id"] = wsAppID
				}
				if wsAPIKey := os.Getenv("XUNFEI_WS_API_KEY"); wsAPIKey != "" {
					cloneConfig.Options["ws_api_key"] = wsAPIKey
				}
				if wsAPISecret := os.Getenv("XUNFEI_WS_API_SECRET"); wsAPISecret != "" {
					cloneConfig.Options["ws_api_secret"] = wsAPISecret
				}
			} else if voiceClone.Provider == "volcengine" {
				if hardwareConfig.Credential.TtsConfig != nil {
					if appID, ok := hardwareConfig.Credential.TtsConfig["appId"].(string); ok {
						cloneConfig.Options["app_id"] = appID
					}
				}
				cloneConfig.Options["token"] = hardwareConfig.Credential.LLMApiKey
			}

			cloneService, cloneErr := cloneFactory.CreateService(cloneConfig)
			if cloneErr == nil && cloneService != nil {
				// 使用克隆音色服务
				ttsService = voiceclone.NewVoiceCloneSynthesisService(cloneService, voiceClone.AssetID)
				hardwareConfig.Logger.Info("[Session] 使用克隆音色进行TTS合成",
					zap.Int("voiceCloneID", *hardwareConfig.VoiceCloneID),
					zap.String("provider", voiceClone.Provider),
					zap.String("assetID", voiceClone.AssetID))
			} else {
				// 克隆音色创建失败，降级到普通TTS
				hardwareConfig.Logger.Warn("[Session] 克隆音色服务创建失败，降级到普通TTS",
					zap.Int("voiceCloneID", *hardwareConfig.VoiceCloneID),
					zap.String("provider", voiceClone.Provider),
					zap.Error(cloneErr))
				ttsService, err = synthesizer.NewSynthesisServiceFromCredential(ttsConfig)
			}
		} else {
			// 克隆音色不存在，降级到普通TTS
			hardwareConfig.Logger.Warn("[Session] 克隆音色不存在，降级到普通TTS",
				zap.Int("voiceCloneID", *hardwareConfig.VoiceCloneID),
				zap.Error(voiceCloneErr))
			ttsService, err = synthesizer.NewSynthesisServiceFromCredential(ttsConfig)
		}
	} else {
		// 没有指定克隆音色，使用普通TTS
		ttsService, err = synthesizer.NewSynthesisServiceFromCredential(ttsConfig)
	}

	if err != nil {
		logger.Fatal("创建 TTS 服务失败", zap.Error(err))
	}
	var session *HardwareSession
	var sessionRef *HardwareSession
	pipeline, err := stream.NewTTSPipeline(&stream.TTSPipelineConfig{
		TTSService: stream.NewTTSServiceAdapter(ttsService),
		SendCallback: func(data []byte) error {
			return writer.SendTTSAudioWithFlowControl(data, 60, 60)
		},
		RecordCallback: func(data []byte) error {
			if sessionRef != nil && sessionRef.recorder != nil {
				return sessionRef.recorder.WriteAudio(data)
			}
			return nil
		},
		GetPendingCountFunc: func() int { return writer.GetBinaryChannelLength() },
		TargetSampleRate:    16000,
		FrameDuration:       60 * time.Millisecond,
		Logger:              hardwareConfig.Logger,
	})
	if err != nil {
		logger.Fatal("创建管道失败", zap.Error(err))
	}
	if err := pipeline.Start(sessionCtx); err != nil {
		logger.Fatal("启动 TTS Pipeline 失败", zap.Error(err))
	}
	sessionID := fmt.Sprintf("%s%d_%d", constants.HARDWARE_SESSION_PREFIX, hardwareConfig.AssistantID, time.Now().UnixNano())
	session = &HardwareSession{
		sessionID:         sessionID,
		config:            hardwareConfig,
		logger:            hardwareConfig.Logger,
		active:            false,
		writer:            writer,
		ctx:               sessionCtx,
		cancel:            cancel,
		conn:              hardwareConfig.Conn,
		stateManager:      sessions.NewASRStateManager(),
		llmService:        llmService,
		ttsPipeline:       pipeline,
		speakerManager:    speakerManager,
		ttsServiceCache:   make(map[string]synthesizer.SynthesisService),
		db:                hardwareConfig.DB,
		voiceprintService: hardwareConfig.VoiceprintService,
		callRecording: &models.CallRecording{
			UserID:      hardwareConfig.Credential.UserID,
			AssistantID: hardwareConfig.AssistantID,
			SessionID:   sessionID,
			DeviceID: func() string {
				if hardwareConfig.DeviceID != nil {
					return *hardwareConfig.DeviceID
				}
				return ""
			}(),
			MacAddress:  hardwareConfig.MacAddress,
			CallType:    "voice",
			CallStatus:  "ongoing",
			StartTime:   time.Now(),
			EndTime:     time.Now(), // 初始化为当前时间，后续会更新
			LLMModel:    hardwareConfig.LLMModel,
			TTSProvider: ttsProvider,
			ASRProvider: hardwareConfig.Credential.GetASRProvider(),
		},
	}
	if hardwareConfig.Speaker != "" {
		session.callRecording.AddSpeaker(hardwareConfig.Speaker)
	}
	if hardwareConfig.DB != nil {
		if err := models.CreateCallRecording(hardwareConfig.DB, session.callRecording); err != nil {
			hardwareConfig.Logger.Error("[Session] 初始化时保存通话记录失败", zap.Error(err))
		} else {
			hardwareConfig.Logger.Info(fmt.Sprintf("[Session] 通话记录已创建 SessionID:%s, RecordingID:%d", sessionID, uint64(session.callRecording.ID)))
		}
	}
	recordingDir := fmt.Sprintf("recordings/%d/%d", hardwareConfig.Credential.UserID, hardwareConfig.AssistantID)
	recordingFile := fmt.Sprintf("%s/%s.wav", recordingDir, sessionID)
	if recorder, err := NewAudioRecorder(recordingFile, 16000, 1, 16, hardwareConfig.Logger); err != nil {
		hardwareConfig.Logger.Warn("[Session] 初始化录音器失败", zap.Error(err))
	} else {
		session.recorder = recorder
	}
	speakerManager.SetSwitchCallback(func(speakerID string) error {
		return session.switchSpeaker(speakerID, ttsProvider, ttsConfig)
	})
	tools.RegisterGoodbyeTool(llmService, func() error {
		return session.handleGoodbye()
	})
	pipeline.SetOnComplete(func() {
		session.onTTSComplete()
	})
	if hardwareConfig.DB != nil && hardwareConfig.VoiceprintService != nil {
		assistantIDStr := fmt.Sprintf("%d", hardwareConfig.AssistantID)
		session.voiceprintTool = tools.NewVoiceprintIdentifyTool(
			hardwareConfig.DB,
			hardwareConfig.VoiceprintService,
			assistantIDStr,
			hardwareConfig.Logger,
		)
		tools.RegisterVoiceprintIdentifyTool(llmService, session.voiceprintTool)
	}
	sessionRef = session
	go session.preloadCommonSpeakers(ttsConfig)
	return session
}

// preloadCommonSpeakers 预加载常用发音人的 TTS 服务
func (s *HardwareSession) preloadCommonSpeakers(baseConfig synthesizer.TTSCredentialConfig) {
	speakersToPreload := []struct {
		id       string
		name     string
		priority string
	}{
		{"101040", "智瑜（四川话）", "high"},
		{"101019", "智瑜（粤语）", "high"},
		{"101004", "智云（男声，成熟）", "medium"},
		{"101007", "智娜（女声，知性）", "medium"},
	}
	preloadedCount := 0
	for _, speaker := range speakersToPreload {
		currentSpeaker := s.config.Speaker
		var currentVoiceType int64
		fmt.Sscanf(currentSpeaker, "%d", &currentVoiceType)
		if fmt.Sprintf("%d", currentVoiceType) == speaker.id {
			s.logger.Debug("[Session] 跳过当前使用的发音人",
				zap.String("speaker_id", speaker.id),
				zap.String("name", speaker.name))
			continue
		}
		newConfig := make(synthesizer.TTSCredentialConfig)
		for k, v := range baseConfig {
			newConfig[k] = v
		}
		var voiceType int64
		fmt.Sscanf(speaker.id, "%d", &voiceType)
		newConfig["voiceType"] = voiceType
		startTime := time.Now()
		ttsService, err := synthesizer.NewSynthesisServiceFromCredential(newConfig)
		if err != nil {
			s.logger.Warn("[Session] 预加载发音人失败",
				zap.String("speaker_id", speaker.id),
				zap.String("name", speaker.name),
				zap.String("priority", speaker.priority),
				zap.Error(err))
			continue
		}
		s.mu.Lock()
		s.ttsServiceCache[speaker.id] = ttsService
		s.mu.Unlock()

		elapsed := time.Since(startTime)
		preloadedCount++
		s.logger.Info("[Session] 预加载发音人成功",
			zap.String("speaker_id", speaker.id),
			zap.String("name", speaker.name),
			zap.String("priority", speaker.priority),
			zap.Duration("耗时", elapsed))
	}
}

func (s *HardwareSession) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return nil
	}
	s.active = true

	// 设置设备在线状态为 true
	if s.db != nil && s.config.MacAddress != "" {
		if err := models.UpdateDeviceOnlineStatus(s.db, s.config.MacAddress, true); err != nil {
			s.logger.Warn("[Session] 更新设备在线状态失败", zap.Error(err), zap.String("macAddress", s.config.MacAddress))
		} else {
			s.logger.Info("[Session] 设备在线状态已更新为 true", zap.String("macAddress", s.config.MacAddress))
		}
	}

	go s.messageLoop()
	return nil
}

// messageLoop message handler loop
func (s *HardwareSession) messageLoop() {
	defer func() {
		s.logger.Info("[Session] --- 消息循环退出，触发会话关闭")
		if err := s.Stop(); err != nil {
			s.logger.Error("[Session] --- 会话关闭失败", zap.Error(err))
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("[Session] --- Context 已取消，退出消息循环")
			return
		default:
		}
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				s.logger.Info("[Session] WebSocket 连接正常关闭", zap.Error(err))
			} else {
				s.logger.Error("[Session] 读取 WebSocket 消息失败", zap.Error(err))
				s.logSessionError("WEBSOCKET", "ERROR", "WS_READ_ERROR", err.Error(), "", "Failed to read WebSocket message")
			}
			return
		}
		switch messageType {
		case websocket.BinaryMessage:
			if err := s.handleAudio(message); err != nil {
				s.logger.Warn("[Session] 处理音频消息失败", zap.Error(err))
				s.logSessionError("AUDIO", "WARN", "AUDIO_PROCESS_ERROR", err.Error(), "", "Failed to process audio message")
			}
		case websocket.TextMessage:
			if err := s.handleText(message); err != nil {
				s.logger.Warn("[Session] 处理文本消息失败", zap.Error(err))
				s.logSessionError("TEXT", "WARN", "TEXT_PROCESS_ERROR", err.Error(), "", "Failed to process text message")
			}
		}
	}
}

// Stop stop hardware session
func (s *HardwareSession) Stop() error {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return nil
	}
	s.active = false
	ttsPipeline := s.ttsPipeline
	writer := s.writer
	conn := s.conn
	macAddress := s.config.MacAddress
	db := s.db
	s.mu.Unlock()

	if ttsPipeline != nil {
		if err := ttsPipeline.Stop(); err != nil {
			s.logger.Error("[Session] --- 停止 TTS Pipeline 失败", zap.Error(err))
		}
	}
	s.cancel()
	if writer != nil {
		writer.Close()
	}
	if conn != nil {
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
			s.logger.Debug("[Session] --- 发送WebSocket关闭消息失败", zap.Error(err))
		}
		if err := conn.Close(); err != nil {
			s.logger.Debug("[Session] --- 关闭WebSocket连接时出错", zap.Error(err))
		} else {
			s.logger.Debug("[Session] --- WebSocket连接已关闭")
		}
	}
	if db != nil && macAddress != "" {
		if err := models.UpdateDeviceOnlineStatus(db, macAddress, false); err != nil {
			s.logger.Warn("[Session] 更新设备在线状态失败", zap.Error(err), zap.String("macAddress", macAddress))
		} else {
			s.logger.Info("[Session] 设备在线状态已更新为 false", zap.String("macAddress", macAddress))
		}
	}
	s.saveCallRecording()
	return nil
}

// switchSpeaker 切换发音人
func (s *HardwareSession) switchSpeaker(speakerID string, ttsProvider string, baseConfig synthesizer.TTSCredentialConfig) error {
	s.logger.Info("[Session] 开始切换发音人",
		zap.String("speaker_id", speakerID),
		zap.String("provider", ttsProvider))
	if s.callRecording != nil {
		s.callRecording.AddSpeaker(speakerID)
		if s.db != nil {
			if err := s.db.Model(s.callRecording).Update("speakers", s.callRecording.Speakers).Error; err != nil {
				s.logger.Warn("[Session] 更新发音人列表失败", zap.Error(err), zap.String("speaker_id", speakerID))
			}
		}
	}

	// 检查缓存中是否已有该发音人的 TTS 服务
	s.mu.RLock()
	cachedService, exists := s.ttsServiceCache[speakerID]
	s.mu.RUnlock()

	if exists {
		// 使用缓存的服务，立即切换
		s.ttsPipeline.UpdateTTSService(stream.NewTTSServiceAdapter(cachedService))
		s.logger.Info("[Session] 使用缓存的 TTS 服务，发音人切换成功",
			zap.String("speaker_id", speakerID))
		return nil
	}

	// 缓存中没有，异步创建新的 TTS 服务
	go func() {
		startTime := time.Now()

		// 创建新的 TTS 配置
		newConfig := make(synthesizer.TTSCredentialConfig)
		for k, v := range baseConfig {
			newConfig[k] = v
		}

		// 更新 voiceType
		var voiceType int64
		fmt.Sscanf(speakerID, "%d", &voiceType)
		newConfig["voiceType"] = voiceType

		// 创建新的 TTS 服务
		newTTSService, err := synthesizer.NewSynthesisServiceFromCredential(newConfig)
		if err != nil {
			s.logger.Error("[Session] 创建新 TTS 服务失败",
				zap.Error(err),
				zap.String("speaker_id", speakerID))
			return
		}

		// 保存到缓存
		s.mu.Lock()
		s.ttsServiceCache[speakerID] = newTTSService
		s.mu.Unlock()

		// 动态更新 TTS 服务
		s.ttsPipeline.UpdateTTSService(stream.NewTTSServiceAdapter(newTTSService))

		elapsed := time.Since(startTime)
		s.logger.Info("[Session] 发音人切换成功（新建服务）",
			zap.String("speaker_id", speakerID),
			zap.Duration("耗时", elapsed))
	}()
	return nil
}

// handleGoodbye 处理待机请求
func (s *HardwareSession) handleGoodbye() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Info("[Session] 收到待机请求，将在 TTS 播放完成后进入待机状态")
	s.goodbyePending = true
	return nil
}

// onTTSComplete TTS 播放完成回调
func (s *HardwareSession) onTTSComplete() {
	s.logger.Info("[TTS Pipeline] 播放完成，发送 TTS 结束消息")
	if s.asrPipeline != nil {
		s.asrPipeline.SetTTSPlaying(false)
	}
	if err := s.writer.SendTTSEnd(); err != nil {
		s.logger.Error("[TTS Pipeline] 发送 TTS 结束消息失败", zap.Error(err))
	}
	s.mu.RLock()
	goodbyePending := s.goodbyePending
	s.mu.RUnlock()
	if goodbyePending {
		s.logger.Info("[Session] 检测到待机请求，准备断开连接")
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.logger.Info("[Session] 执行待机断开连接")
			if err := s.Stop(); err != nil {
				s.logger.Error("[Session] 待机断开连接失败", zap.Error(err))
			} else {
				s.logger.Info("[Session] 待机断开连接成功，硬件将进入待机状态")
			}
		}()
	}
}

// logSessionError 记录会话中的错误
func (s *HardwareSession) logSessionError(errorType, errorLevel, errorCode, errorMsg, stackTrace, context string) {
	if s.db == nil || s.config.MacAddress == "" {
		s.logger.Warn("[Session] 无法记录错误：数据库或MAC地址为空", zap.String("errorMsg", errorMsg))
		return
	}
	deviceID := s.config.MacAddress
	if s.callRecording != nil && s.callRecording.DeviceID != "" {
		deviceID = s.callRecording.DeviceID
	}
	if err := models.LogDeviceError(s.db, deviceID, s.config.MacAddress, errorType, errorLevel, errorCode, errorMsg, stackTrace, context); err != nil {
		s.logger.Error("[Session] 记录设备错误失败", zap.Error(err), zap.String("errorMsg", errorMsg))
	} else {
		s.logger.Info("[Session] 设备错误已记录", zap.String("errorType", errorType), zap.String("errorLevel", errorLevel), zap.String("errorMsg", errorMsg))
	}
}

// saveCallRecording 保存通话记录
func (s *HardwareSession) saveCallRecording() {
	s.logger.Info("[Session] saveCallRecording 被调用")

	if s.callRecording == nil {
		s.logger.Warn("[Session] callRecording 为 nil，无法保存")
		return
	}

	if s.db == nil {
		s.logger.Warn("[Session] db 为 nil，无法保存")
		return
	}

	s.callRecording.EndTime = time.Now()
	s.callRecording.Duration = int(s.callRecording.EndTime.Sub(s.callRecording.StartTime).Seconds())
	s.callRecording.CallStatus = "completed"

	s.logger.Info("[Session] 准备保存对话详情",
		zap.Int("conversationTurns", len(s.conversationTurns)))
	s.saveConversationDetails()
	// 处理录音数据
	if s.recorder != nil {
		// 关闭录音器
		if err := s.recorder.Stop(); err != nil {
			s.logger.Error("[Session] 关闭录音器失败", zap.Error(err))
		}

		// 获取本地文件路径
		filePath := s.recorder.GetFilePath()
		if filePath != "" {
			// 异步上传到存储服务
			go s.uploadRecordingFile(filePath)

			// 设置存储URL为本地路径（后续会更新为远程URL）
			s.callRecording.StorageURL = filePath
		}

		// 更新录音相关信息
		s.callRecording.AudioSize = s.recorder.GetTotalBytes()
		s.callRecording.Duration = s.recorder.GetDuration()
		s.callRecording.AudioFormat = "wav"
		s.callRecording.SampleRate = 16000
		s.callRecording.Channels = 1
	}

	s.logger.Info("[Session] 准备更新数据库",
		zap.Uint64("recordingID", uint64(s.callRecording.ID)),
		zap.String("sessionID", s.callRecording.SessionID))

	if err := s.db.Model(s.callRecording).Updates(map[string]interface{}{
		"end_time":             s.callRecording.EndTime,
		"duration":             s.callRecording.Duration,
		"call_status":          "completed",
		"speakers":             s.callRecording.Speakers,
		"audio_size":           s.callRecording.AudioSize,
		"audio_format":         s.callRecording.AudioFormat,
		"sample_rate":          s.callRecording.SampleRate,
		"channels":             s.callRecording.Channels,
		"storage_url":          s.callRecording.StorageURL,
		"conversation_details": s.callRecording.ConversationDetailsJSON,
	}).Error; err != nil {
		s.logger.Error("[Session] 更新通话记录失败", zap.Error(err))
		return
	}

	s.logger.Info("[Session] 通话记录已更新",
		zap.Uint64("recordingID", uint64(s.callRecording.ID)),
		zap.String("sessionID", s.callRecording.SessionID),
		zap.Int("duration", s.callRecording.Duration),
		zap.Int64("audioSize", s.callRecording.AudioSize),
		zap.String("storageURL", s.callRecording.StorageURL),
		zap.String("speakers", s.callRecording.Speakers))
}

// uploadRecordingFile 上传录音文件到存储服务
func (s *HardwareSession) uploadRecordingFile(filePath string) {
	if s.callRecording == nil {
		return
	}
	storageKey := fmt.Sprintf("recordings/%d/%d/%s.wav",
		s.callRecording.UserID,
		s.callRecording.AssistantID,
		s.callRecording.SessionID)
	s.logger.Info("[Session] 开始上传录音文件",
		zap.String("localPath", filePath),
		zap.String("storageKey", storageKey))
	wavData, err := os.ReadFile(filePath)
	if err != nil {
		s.logger.Error("[Session] 读取录音文件失败", zap.Error(err), zap.String("filePath", filePath))
		return
	}

	s.logger.Info("[Session] 录音文件已读取",
		zap.String("filePath", filePath),
		zap.Int("fileSize", len(wavData)))

	// 上传到存储服务
	result, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
		Bucket:   config.GlobalConfig.Services.Storage.Bucket,
		Data:     wavData,
		Filename: storageKey,
	})

	if err != nil {
		s.logger.Error("[Session] 上传录音文件失败", zap.Error(err), zap.String("storageKey", storageKey))
		return
	}

	s.logger.Info("[Session] 录音文件上传成功",
		zap.String("storageKey", storageKey),
		zap.String("storageURL", result.URL))

	// 更新数据库中的存储URL
	if err := s.db.Model(s.callRecording).Update("storage_url", result.URL).Error; err != nil {
		s.logger.Error("[Session] 更新存储URL失败", zap.Error(err))
		return
	}

	// 上传完成后删除本地文件
	if err := os.Remove(filePath); err != nil {
		s.logger.Warn("[Session] 删除本地录音文件失败", zap.Error(err), zap.String("filePath", filePath))
	} else {
		s.logger.Info("[Session] 本地录音文件已删除", zap.String("filePath", filePath))
	}
}

// recordUserInput 记录用户输入
func (s *HardwareSession) recordUserInput(asrText string, asrStartTime, asrEndTime time.Time) {
	if s.callRecording == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentTurnID++
	now := time.Now()

	// 计算 ASR 耗时
	asrDuration := asrEndTime.Sub(asrStartTime).Milliseconds()

	turn := models.ConversationTurn{
		TurnID:       s.currentTurnID,
		Timestamp:    now,
		Type:         "user",
		Content:      asrText,
		StartTime:    asrStartTime,
		EndTime:      asrEndTime,
		Duration:     asrDuration,
		ASRStartTime: &asrStartTime,
		ASREndTime:   &asrEndTime,
	}
	turn.ASRDuration = &asrDuration

	s.conversationTurns = append(s.conversationTurns, turn)

	s.logger.Info("[Session] 记录用户输入",
		zap.Int("turnID", s.currentTurnID),
		zap.String("text", asrText),
		zap.Int64("asrDuration", asrDuration))
}

// recordAIResponse 记录 AI 回复
func (s *HardwareSession) recordAIResponse(llmResponse string, llmStartTime, llmEndTime, ttsStartTime, ttsEndTime time.Time) {
	if s.callRecording == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentTurnID++
	turn := models.ConversationTurn{
		TurnID:       s.currentTurnID,
		Timestamp:    time.Now(),
		Type:         "ai",
		Content:      llmResponse,
		StartTime:    llmStartTime,
		EndTime:      ttsEndTime,
		Duration:     ttsEndTime.Sub(llmStartTime).Milliseconds(),
		LLMStartTime: &llmStartTime,
		LLMEndTime:   &llmEndTime,
		TTSStartTime: &ttsStartTime,
		TTSEndTime:   &ttsEndTime,
	}

	llmDuration := llmEndTime.Sub(llmStartTime).Milliseconds()
	turn.LLMDuration = &llmDuration
	ttsDuration := ttsEndTime.Sub(ttsStartTime).Milliseconds()
	turn.TTSDuration = &ttsDuration

	// 计算响应延迟（从用户说话结束到 AI 开始回复）
	if len(s.conversationTurns) > 0 {
		lastUserTurn := s.conversationTurns[len(s.conversationTurns)-1]
		if lastUserTurn.Type == "user" {
			responseDelay := llmStartTime.Sub(lastUserTurn.EndTime).Milliseconds()
			turn.ResponseDelay = &responseDelay

			// 计算总延迟（从用户说话结束到 AI 回复完成）
			totalDelay := ttsEndTime.Sub(lastUserTurn.EndTime).Milliseconds()
			turn.TotalDelay = &totalDelay
		}
	}

	s.conversationTurns = append(s.conversationTurns, turn)

	s.logger.Info("[Session] 记录 AI 回复",
		zap.Int("turnID", s.currentTurnID),
		zap.String("text", llmResponse),
		zap.Duration("duration", time.Duration(turn.Duration)*time.Millisecond))
}

// saveConversationDetails 保存对话详情到数据库
func (s *HardwareSession) saveConversationDetails() {
	if s.callRecording == nil || s.db == nil {
		return
	}

	s.mu.RLock()
	turns := make([]models.ConversationTurn, len(s.conversationTurns))
	copy(turns, s.conversationTurns)
	s.mu.RUnlock()

	details := &models.ConversationDetails{
		SessionID:  s.callRecording.SessionID,
		StartTime:  s.callRecording.StartTime,
		EndTime:    s.callRecording.EndTime,
		TotalTurns: len(turns),
		Turns:      turns,
	}

	// 统计用户和 AI 的轮次
	for _, turn := range turns {
		if turn.Type == "user" {
			details.UserTurns++
		} else if turn.Type == "ai" {
			details.AITurns++
		}
	}

	if err := s.callRecording.SetConversationDetails(details); err != nil {
		s.logger.Error("[Session] 设置对话详情失败", zap.Error(err))
		return
	}

	if err := s.db.Model(s.callRecording).Update("conversation_details", s.callRecording.ConversationDetailsJSON).Error; err != nil {
		s.logger.Error("[Session] 保存对话详情失败", zap.Error(err))
		return
	}

	s.logger.Info("[Session] 对话详情已保存",
		zap.Int("totalTurns", details.TotalTurns),
		zap.Int("userTurns", details.UserTurns),
		zap.Int("aiTurns", details.AITurns))
}
