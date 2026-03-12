package protocol

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/LingByte/SoulNexus/pkg/hardware/sessions"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

// handleText handler text message
func (s *HardwareSession) handleText(data []byte) error {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if !active {
		err := fmt.Errorf("[Session] session is not active")
		s.logSessionError("SESSION", "WARN", "SESSION_INACTIVE", "会话未激活", "", "Attempted to handle text message on inactive session")
		return err
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		s.logger.Warn("解析文本消息失败", zap.Error(err))
		s.logSessionError("MESSAGE", "WARN", "JSON_PARSE_ERROR", fmt.Sprintf("文本消息解析失败: %v", err), "", "Failed to parse JSON message")
		return nil
	}
	msgType, ok := msg["type"].(string)
	if !ok {
		s.logSessionError("MESSAGE", "WARN", "INVALID_MESSAGE_TYPE", "消息类型无效", "", "Message type field is missing or invalid")
		return nil
	}
	switch msgType {
	case constants.MessageTypeHello:
		s.handleHelloMessage(msg)
	case constants.MessageTypeListen:
		s.handleListenMessage(msg)
	case constants.MessageTypeAbort:
		s.handleAbortMessage()
	case constants.MessageTypePing:
		s.handlePingMessage()
	default:
		s.logger.Warn(fmt.Sprintf("[Session] --- 未知消息类型：%s", msgType))
		s.logSessionError("MESSAGE", "WARN", "UNKNOWN_MESSAGE_TYPE", fmt.Sprintf("未知消息类型: %s", msgType), "", "Received unknown message type")
	}
	return nil
}

// handlePingMessage 处理 ping 消息
// 消息格式：{"type":"ping"}
func (s *HardwareSession) handlePingMessage() {
	if err := s.writer.SendPong(); err != nil {
		s.logger.Error("[Session] send pong message failed", zap.Error(err))
		s.logSessionError("COMMUNICATION", "ERROR", "PONG_SEND_ERROR", fmt.Sprintf("发送 Pong 消息失败: %v", err), "", "Failed to send pong message")
	}
}

// handleListenMessage 处理拾音模式消息
// 消息格式：{"type":"listen","state":"start|stop|detect","mode":"auto|manual"}
func (s *HardwareSession) handleListenMessage(msg map[string]interface{}) {
	state, _ := msg["state"].(string)
	mode, _ := msg["mode"].(string)
	s.logger.Info("[Session] handle listen message",
		zap.String("state", state),
		zap.String("mode", mode))
	if s.voiceprintTool != nil {
		s.voiceprintTool.ClearIdentification()
		s.logger.Info("[Session] 已清除之前的声纹识别结果")
		// 标记新句子开始，清空当前句子的音频缓冲区
		s.voiceprintTool.StartNewSentence()
		s.logger.Info("[Session] 已标记新句子开始")
	}
	s.mu.Lock()
	s.currentSpeakerID = ""
	s.mu.Unlock()
}

// handleAbortMessage 处理中断消息
// 消息格式：{"type":"abort"}
func (s *HardwareSession) handleAbortMessage() {
	s.logger.Info("[Session] 收到中断请求，停止 LLM 和 TTS")
	if s.llmService != nil {
		s.llmService.GetProvider().Interrupt()
		s.logger.Info("[Session] LLM 已中断")
	}
	if s.ttsPipeline != nil {
		s.ttsPipeline.Interrupt()
		s.logger.Info("[Session] TTS 已中断")
	}
	if s.writer != nil {
		s.writer.ClearTTSQueue()
	}
	if s.writer != nil {
		s.writer.ResetTTSFlowControl()
	}
	if s.asrPipeline != nil {
		s.asrPipeline.SetTTSPlaying(false)
	}
	if s.writer != nil {
		if err := s.writer.SendTTSEnd(); err != nil {
			s.logger.Error("[Session] 发送 TTS 结束消息失败", zap.Error(err))
			s.logSessionError("COMMUNICATION", "ERROR", "TTS_END_SEND_ERROR", fmt.Sprintf("发送 TTS 结束消息失败: %v", err), "", "Failed to send TTS end message")
		} else {
			s.logger.Info("[Session] 已发送 TTS 结束消息")
		}
	}
	if s.writer != nil {
		if err := s.writer.SendAbortConfirmation(); err != nil {
			s.logger.Error("[Session] 发送中断确认消息失败", zap.Error(err))
			s.logSessionError("COMMUNICATION", "ERROR", "ABORT_CONFIRM_SEND_ERROR", fmt.Sprintf("发送中断确认消息失败: %v", err), "", "Failed to send abort confirmation message")
		} else {
			s.logger.Info("[Session] 已发送中断确认消息")
		}
	}
}

// handleHelloMessage 处理hello消息
// {\"type\":\"hello\",\"version\":1,\"features\":{\"aec\":true,\"mcp\":true},\"transport\":\"websocket\",\"audio_params\":{\"format\":\"opus\",\"sample_rate\":16000,\"channels\":1,\"frame_duration\":60}}"}
func (s *HardwareSession) handleHelloMessage(msg map[string]interface{}) {
	audioFormat := "opus"
	sampleRate := 16000
	channels := 1
	frameDuration := "60ms"
	if audioParams, ok := msg["audio_params"].(map[string]interface{}); ok {
		if format, ok := audioParams["format"].(string); ok {
			audioFormat = format
		}
		if rate, ok := audioParams["sample_rate"].(float64); ok {
			sampleRate = int(rate)
		}
		if ch, ok := audioParams["channels"].(float64); ok {
			channels = int(ch)
		}
		if frameDur, ok := audioParams["frame_duration"].(float64); ok {
			frameDuration = fmt.Sprintf("%dms", int(frameDur))
		}
	}
	s.mu.Lock()
	asrProvider := s.config.Credential.GetASRProvider()
	factory := recognizer.GetGlobalFactory()
	asrConfig := make(map[string]interface{})
	asrConfig["provider"] = asrProvider
	asrConfig["language"] = "zh"
	for key, value := range s.config.Credential.AsrConfig {
		asrConfig[key] = value
	}
	config, err := recognizer.NewTranscriberConfigFromMap(asrProvider, asrConfig, "zh")
	if err != nil {
		s.mu.Unlock()
		s.logger.Error("创建 ASR 配置失败", zap.Error(err))
		s.logSessionError("ASR", "ERROR", "ASR_CONFIG_ERROR", fmt.Sprintf("创建 ASR 配置失败: %v", err), "", "Failed to create ASR configuration")
		return
	}
	asrService, err := factory.CreateTranscriber(config)
	if err != nil {
		s.mu.Unlock()
		s.logger.Error("创建 ASR 服务失败", zap.Error(err))
		s.logSessionError("ASR", "ERROR", "ASR_SERVICE_ERROR", fmt.Sprintf("创建 ASR 服务失败: %v", err), "", "Failed to create ASR service")
		return
	}

	pipeline, err := sessions.NewASRPipeline(&sessions.ASRPipelineOption{
		Asr:                  asrService,
		SampleRate:           sampleRate,
		Channels:             channels,
		FrameDuration:        frameDuration,
		EnableVAD:            s.config.EnableVAD,
		VADThreshold:         s.config.VADThreshold,
		VADConsecutiveFrames: s.config.VADConsecutiveFrames,
		VADServiceURL:        utils.GetEnv("VAD_SERVICE_URL"),
		VADSessionID:         fmt.Sprintf("user_%d", s.config.AssistantID),
	}, s.logger)
	if err != nil {
		s.mu.Unlock()
		s.logger.Error("创建ASRPipeline失败", zap.Error(err))
		s.logSessionError("ASR", "ERROR", "ASR_PIPELINE_ERROR", fmt.Sprintf("创建 ASR Pipeline 失败: %v", err), "", "Failed to create ASR pipeline")
		return
	}
	s.logger.Info(fmt.Sprintf(
		"Session create new Session sessionID:%s vad:%v, vadThreshold:%f, vadConsecultiveFrames:%d",
		s.sessionID, s.config.EnableVAD, s.config.VADThreshold, s.config.VADConsecutiveFrames))
	s.asrPipeline = pipeline

	// 设置 PCM 音频记录回调
	pipeline.SetPCMAudioCallback(func(data []byte) error {
		s.mu.RLock()
		recorder := s.recorder
		voiceprintTool := s.voiceprintTool
		s.mu.RUnlock()

		// 添加 PCM 音频到声纹识别工具
		if voiceprintTool != nil {
			voiceprintTool.AddAudioFrame(data)
		}

		if recorder != nil {
			return recorder.WriteAudio(data)
		}
		return nil
	})
	if s.config.EnableVAD {
		pipeline.SetBargeInCallback(func() {
			s.logger.Info("[Session] Barge-in 触发：用户说话，打断 TTS 和 LLM")
			if s.llmService != nil {
				s.llmService.GetProvider().Interrupt()
			}
			if s.ttsPipeline != nil {
				s.ttsPipeline.Interrupt()
			}
			if s.writer != nil {
				s.writer.ClearTTSQueue()
			}
			if s.writer != nil {
				s.writer.ResetTTSFlowControl()
			}
			if s.writer != nil {
				if err := s.writer.SendTTSEnd(); err != nil {
					s.logger.Error("[Session] Barge-in 发送 TTS 结束消息失败", zap.Error(err))
				} else {
					s.logger.Info("[Session] Barge-in 已发送 TTS 结束消息")
				}
			}
			// 延迟清除 TTS 播放状态给扬声器时间播放完缓冲区
			go func() {
				time.Sleep(300 * time.Millisecond) // 延迟 500ms
				if pipeline != nil {
					pipeline.ClearTTSState()
				}
			}()
		})
	}

	pipeline.SetOutputCallback(func(text string, isFinal bool) {
		incrementalText := s.stateManager.UpdateASRText(text, isFinal)
		if incrementalText == "" {
			return
		}
		s.logger.Info(fmt.Sprintf("[Session] --- 处理增量文本 [%s] 是否结束 [%v] ", incrementalText, isFinal))
		pipeline.PrintMetrics()
		if err := s.writer.SendASRResult(incrementalText); err != nil {
			s.logger.Error("[Session] 发送 ASR 结果失败", zap.Error(err))
			s.logSessionError("COMMUNICATION", "ERROR", "ASR_RESULT_SEND_ERROR", fmt.Sprintf("发送 ASR 结果失败: %v", err), "", "Failed to send ASR result")
		}

		s.mu.Lock()
		if s.llmProcessing {
			s.logger.Info("[Session] LLM 正在处理中，忽略新的 ASR 结果",
				zap.String("text", incrementalText))
			s.mu.Unlock()
			return
		}
		s.llmProcessing = true
		s.mu.Unlock()

		// 获取 ASR 时间指标
		metrics := pipeline.GetMetrics()
		asrStartTime := metrics.FirstPacketTime
		asrEndTime := time.Now()
		if asrStartTime.IsZero() {
			asrStartTime = asrEndTime
		}

		// 记录用户输入（在 LLM 开始处理时）
		s.recordUserInput(incrementalText, asrStartTime, asrEndTime)

		if err := s.writer.SendTTSStart(); err != nil {
			s.logger.Error("[Session] 发送 TTS 开始消息失败", zap.Error(err))
			s.logSessionError("COMMUNICATION", "ERROR", "TTS_START_SEND_ERROR", fmt.Sprintf("发送 TTS 开始消息失败: %v", err), "", "Failed to send TTS start message")
		}
		go func(text string) {
			receivedContent := false
			var llmResponse string // 收集完整的 LLM 回复
			var llmStartTime, llmEndTime time.Time
			var ttsStartTime, ttsEndTime time.Time

			defer func() {
				s.mu.Lock()
				s.llmProcessing = false
				s.mu.Unlock()
				s.logger.Info("[Session] LLM 处理完成，清除处理标志")
			}()
			var userContext string
			if isFinal && s.voiceprintTool != nil {
				identified, err := s.voiceprintTool.Identify(s.ctx)
				if err != nil {
					s.logger.Warn("[Session] 声纹识别失败",
						zap.Error(err))
					s.logSessionError("VOICEPRINT", "WARN", "VOICEPRINT_IDENTIFY_ERROR", fmt.Sprintf("声纹识别失败: %v", err), "", "Voiceprint identification failed")
				} else {
					s.logger.Info("[Session] 声纹识别成功",
						zap.String("speaker_id", identified.SpeakerID),
						zap.String("speaker_name", identified.SpeakerName),
						zap.Float64("confidence", identified.Confidence))
					s.mu.RLock()
					previousSpeakerID := s.currentSpeakerID
					s.mu.RUnlock()

					if identified.SpeakerID != previousSpeakerID {
						// 用户切换，更新 System Prompt
						s.logger.Info("[Session] 检测到用户切换",
							zap.String("previous_speaker", previousSpeakerID),
							zap.String("current_speaker", identified.SpeakerID))

						// 获取用户描述用于 LLM 上下文
						userContext = s.voiceprintTool.GetUserDescription()
						if userContext != "" {
							s.logger.Info("[Session] 已获取用户描述",
								zap.String("context", userContext))

							// 更新 LLM System Prompt 以包含用户信息
							currentPrompt := s.llmService.GetConfig().SystemPrompt
							// 添加明确的上下文说明，让 LLM 知道后面的信息是什么
							updatedPrompt := fmt.Sprintf("%s\n\n【当前用户信息】\n目前跟你对话的人是：%s\n用户背景：%s",
								currentPrompt,
								identified.SpeakerName,
								userContext)
							s.llmService.SetSystemPrompt(updatedPrompt)
							s.logger.Info("[Session] 已更新 LLM System Prompt 以包含用户信息",
								zap.String("speaker_name", identified.SpeakerName))
						}

						// 更新当前说话人 ID
						s.mu.Lock()
						s.currentSpeakerID = identified.SpeakerID
						s.mu.Unlock()
					} else {
						s.logger.Info("[Session] 同一用户继续说话，跳过 Prompt 更新",
							zap.String("speaker_id", identified.SpeakerID))
					}
				}
			}

			llmStartTime = time.Now()
			err := s.llmService.QueryStream(text, func(segment string, isComplete bool) error {
				if !isComplete {
					if !receivedContent {
						receivedContent = true
						ttsStartTime = time.Now()
						if s.asrPipeline != nil {
							s.asrPipeline.SetTTSPlaying(true)
							s.logger.Info("[Session] 已设置 TTS 播放状态（LLM 开始返回内容）")
						}
					}
					llmResponse += segment
					s.ttsPipeline.OnLLMToken(segment)
				} else {
					llmEndTime = time.Now()
					s.ttsPipeline.OnLLMComplete()
				}
				return nil
			}, llm.QueryOptions{
				MaxTokens: &s.config.MaxToken,
				Model:     "qwen-flash",
				Stream:    true,
			})

			// 记录 AI 回复
			if err == nil && receivedContent {
				ttsEndTime = time.Now()
				s.recordAIResponse(llmResponse, llmStartTime, llmEndTime, ttsStartTime, ttsEndTime)
			}

			if err != nil {
				s.logger.Error("[Session] LLM 查询失败", zap.Error(err))
				if !receivedContent {
					s.logger.Info("[Session] LLM 未返回内容就失败，清除 TTS 状态并发送 TTS 结束消息")
					if s.asrPipeline != nil {
						s.asrPipeline.SetTTSPlaying(false)
					}
					if sendErr := s.writer.SendTTSEnd(); sendErr != nil {
						s.logger.Error("[Session] 发送 TTS 结束消息失败", zap.Error(sendErr))
					}
				}
				if sendErr := s.writer.SendError(fmt.Sprintf("LLM 查询失败: %v", err), false); sendErr != nil {
					s.logger.Error("[Session] 发送错误消息失败", zap.Error(sendErr))
				}
			}
		}(incrementalText)
	})
	s.asrPipeline = pipeline
	if err := asrService.ConnAndReceive(s.writer.sessionID); err != nil {
		s.mu.Unlock()
		s.logger.Error("[Session] 连接 ASR 服务失败", zap.Error(err))
		return
	}
	s.mu.Unlock()
	var features map[string]interface{}
	if feat, ok := msg["features"].(map[string]interface{}); ok {
		features = feat
	}
	sessionID, err := s.writer.SendWelcome(audioFormat, sampleRate, channels, features)
	if err != nil {
		s.logger.Error("发送Welcome响应失败", zap.Error(err))
	} else {
		s.logger.Info(fmt.Sprintf(
			"[Session] --- 已发送Welcome响应 audioFormat:%s, sampleRate:%d, channel:%d, sessionId:%s",
			audioFormat, sampleRate, channels, sessionID))
	}
}

// HandleAudio 处理音频数据
func (s *HardwareSession) handleAudio(data []byte) error {
	s.mu.RLock()
	active := s.active
	pipeline := s.asrPipeline
	s.mu.RUnlock()
	if !active {
		err := fmt.Errorf("[Session] 会话未激活")
		s.logSessionError("SESSION", "ERROR", "SESSION_INACTIVE", "会话未激活，无法处理音频", "", "Session is not active")
		return err
	}
	if pipeline == nil {
		err := fmt.Errorf("[Session] ASR Pipeline 未初始化")
		s.logSessionError("ASR", "ERROR", "PIPELINE_NOT_INITIALIZED", "ASR Pipeline 未初始化", "", "ASR Pipeline is not initialized")
		return err
	}

	// 检查音频数据有效性
	if len(data) == 0 {
		s.logSessionError("AUDIO", "WARN", "EMPTY_AUDIO_DATA", "接收到空音频数据", "", "Received empty audio data")
		return fmt.Errorf("empty audio data")
	}

	err := pipeline.ProcessInput(s.ctx, data)
	if err != nil {
		s.logger.Error("[Session] ASR 处理输入失败", zap.Error(err))
		s.logSessionError("ASR", "ERROR", "ASR_PROCESS_ERROR", fmt.Sprintf("ASR 处理失败: %v", err), "", "ASR processing failed")
		return err
	}
	return nil
}
