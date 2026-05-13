package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voice"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var voiceUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 1024, // 1MB读缓冲区，支持大音频数据
	WriteBufferSize: 1024 * 1024, // 1MB写缓冲区，支持大音频数据
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// HandleWebSocketVoice 处理浏览器直连 SoulNexus 的旧版 WebSocket 语音（ASR/TTS/LLM 在 API 进程内）。
// 推荐架构：浏览器连接 cmd/voice 的 xiaozhi WebSocket（与 WebRTC 同源），鉴权经 ?payload= 透传到 /ws/call，
// SoulNexus 仅作对话面；本入口保留兼容或内网工具调用。
func (h *Handlers) HandleWebSocketVoice(c *gin.Context) {
	// 获取参数
	apiKey := c.Query("apiKey")
	apiSecret := c.Query("apiSecret")
	assistantIDStr := c.Query("agentId")
	language := c.Query("language")
	if language == "" {
		language = "zh-cn"
	}
	speaker := c.Query("speaker")
	if speaker == "" {
		speaker = "101016"
	}

	// 验证参数
	if apiKey == "" || apiSecret == "" {
		response.Fail(c, "缺少apiKey或apiSecret参数", nil)
		return
	}

	assistantID, err := strconv.Atoi(assistantIDStr)
	if err != nil || assistantID <= 0 {
		response.Fail(c, "无效的助手ID", nil)
		return
	}

	// 验证凭证
	cred, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, apiKey, apiSecret)
	if err != nil {
		response.Fail(c, "Database error: "+err.Error(), nil)
		return
	}
	if cred == nil {
		response.Fail(c, "Invalid credentials", nil)
		return
	}

	// 获取助手配置
	var assistant models.Agent
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		response.Fail(c, "获取助手配置失败: "+err.Error(), nil)
		return
	}
	if assistant.ID == 0 {
		response.Fail(c, "助手不存在", nil)
		return
	}

	// 升级为WebSocket连接
	conn, err := voiceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		response.Fail(c, "Failed to upgrade connection", nil)
		return
	}

	// 使用助手配置中的参数
	systemPrompt := assistant.SystemPrompt
	temperature := assistant.Temperature
	if assistant.Speaker != "" {
		speaker = assistant.Speaker
	}

	handler := voice.NewHardwareHandler(h.db, logger.Lg)
	handler.HandlerHardwareWebsocket(c.Request.Context(), &voice.HardwareOptions{
		Conn:                 conn,
		AgentID:              uint(assistantID),
		Language:             language,
		Speaker:              speaker,
		Temperature:          float64(temperature),
		SystemPrompt:         systemPrompt,
		UserID:               cred.CreatedBy,
		MacAddress:           "",
		LLMModel:             assistant.LLMModel,
		Credential:           cred,
		EnableVAD:            assistant.EnableVAD,
		VADThreshold:         assistant.VADThreshold,
		VADConsecutiveFrames: assistant.VADConsecutiveFrames,
		VoiceCloneID:         assistant.VoiceCloneID,
		LowLatency:           true,
	})
}

// HandleHardwareWebSocketVoice 处理硬件WebSocket语音连接
// 从Header中获取Device-Id（MAC地址），查询设备绑定的助手，动态获取配置
func (h *Handlers) HandleHardwareWebSocketVoice(c *gin.Context) {
	// 从Header获取Device-Id（MAC地址），与xiaozhi-esp32兼容
	deviceID := c.GetHeader("Device-Id")
	if deviceID == "" {
		// 如果Header中没有，尝试从URL查询参数获取（xiaozhi-esp32兼容）
		deviceID = c.Query("device-id")
	}

	logger.Info("硬件WebSocket连接请求",
		zap.String("deviceID", deviceID),
		zap.String("path", c.Request.URL.Path),
		zap.String("remoteAddr", c.Request.RemoteAddr),
		zap.String("userAgent", c.Request.UserAgent()))

	sess, status, msg := h.resolveSoulnexusHardwareSession(deviceID)
	if sess == nil {
		if deviceID == "" {
			logger.Warn("硬件WebSocket连接缺少Device-Id参数",
				zap.String("path", c.Request.URL.Path),
				zap.String("headers", fmt.Sprintf("%v", c.Request.Header)))
		}
		c.JSON(status, gin.H{
			"code": 500,
			"msg":  msg,
			"data": nil,
		})
		c.Abort()
		return
	}
	device := sess.Device
	assistant := sess.Assistant
	cred := sess.Credential
	assistantID := sess.AssistantID

	// 升级为WebSocket连接
	logger.Info("准备升级WebSocket连接",
		zap.String("deviceID", deviceID),
		zap.Int64("assistantID", int64(assistantID)))
	conn, err := voiceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket升级失败",
			zap.String("deviceID", deviceID),
			zap.Error(err))
		return
	}
	logger.Info("WebSocket连接已建立",
		zap.String("deviceID", deviceID),
		zap.Int64("assistantID", int64(assistantID)))

	// 创建WebSocket处理器
	handler := voice.NewHardwareHandler(h.db, logger.Lg)

	ctx := context.Background()

	handler.HandlerHardwareWebsocket(ctx, &voice.HardwareOptions{
		Conn:                 conn,
		AgentID:              assistantID,
		DeviceID:             &device.ID,
		Language:             sess.Language,
		Speaker:              sess.Speaker,
		Temperature:          sess.Temperature,
		SystemPrompt:         sess.SystemPrompt,
		UserID:               device.CreatedBy,
		MacAddress:           device.MacAddress,
		LLMModel:             sess.LLMModel,
		Credential:           cred,
		EnableVAD:            sess.EnableVAD,
		VADThreshold:         sess.VADThreshold,
		VADConsecutiveFrames: sess.VADConsecutiveFrames,
		VoiceCloneID:         assistant.VoiceCloneID,
	})
}
