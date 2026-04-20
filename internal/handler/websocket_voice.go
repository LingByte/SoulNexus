package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voice"
	"github.com/LingByte/SoulNexus/pkg/voice/constants"
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

// HandleWebSocketVoice 处理通用WebSocket语音连接
func (h *Handlers) HandleWebSocketVoice(c *gin.Context) {
	// 获取参数
	apiKey := c.Query("apiKey")
	apiSecret := c.Query("apiSecret")
	assistantIDStr := c.Query("assistantId")
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
	var assistant models.Assistant
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

	// 如果开启了图记忆功能，则尝试从 Neo4j 中获取该用户的长期偏好主题，并拼接到系统提示词中
	if config.GlobalConfig.Services.GraphMemory.Neo4j.Enabled && assistant.EnableGraphMemory {
		if store := graph.GetDefaultStore(); store != nil {
			// 通过凭证反查用户
			var user models.User
			if err := h.db.First(&user, cred.UserID).Error; err == nil {
				ctx := c.Request.Context()
				if userCtx, err := store.GetUserContext(ctx, user.ID, int64(assistantID)); err == nil {
					if len(userCtx.Topics) > 0 {
						// 构建一段自然语言描述用户长期偏好
						preferenceText := fmt.Sprintf("该用户在历史对话中经常讨论这些主题：%s。请在回答时优先从这些兴趣和习惯的角度来组织内容，让风格尽量贴近他的偏好。",
							strings.Join(userCtx.Topics, "、"))
						if systemPrompt == "" {
							systemPrompt = preferenceText
						} else {
							systemPrompt = systemPrompt + "\n\n" + preferenceText
						}
					}
				}
			}
		}
	}
	handler := voice.NewHardwareHandler(h.db, logger.Lg)
	handler.HandlerHardwareWebsocket(c.Request.Context(), &voice.HardwareOptions{
		Conn:                 conn,
		AssistantID:          uint(assistantID),
		Language:             language,
		Speaker:              speaker,
		Temperature:          float64(temperature),
		SystemPrompt:         systemPrompt,
		KnowledgeKey:         "",
		UserID:               cred.UserID,
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

	if deviceID == "" {
		// WebSocket升级前返回错误
		logger.Warn("硬件WebSocket连接缺少Device-Id参数",
			zap.String("path", c.Request.URL.Path),
			zap.String("headers", fmt.Sprintf("%v", c.Request.Header)))
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "缺少Device-Id参数",
			"data": nil,
		})
		c.Abort()
		return
	}

	// 根据Device-Id查询设备
	device, err := models.GetDeviceByMacAddress(h.db, deviceID)
	if err != nil || device == nil {
		// 设备不存在或未激活
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "设备未激活，请先激活设备",
			"data": nil,
		})
		c.Abort()
		return
	}

	// 检查设备是否绑定了助手
	if device.AssistantID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "设备未绑定助手",
			"data": nil,
		})
		c.Abort()
		return
	}

	assistantID := *device.AssistantID

	// 获取助手配置
	var assistant models.Assistant
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "获取助手配置失败: " + err.Error(),
			"data": nil,
		})
		c.Abort()
		return
	}
	if assistant.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "助手不存在",
			"data": nil,
		})
		c.Abort()
		return
	}

	// 使用 Assistant 的 ApiKey 和 ApiSecret 获取用户凭证
	if assistant.ApiKey == "" || assistant.ApiSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "助手未配置API凭证",
			"data": nil,
		})
		c.Abort()
		return
	}

	cred, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, assistant.ApiKey, assistant.ApiSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "获取凭证失败: " + err.Error(),
			"data": nil,
		})
		c.Abort()
		return
	}
	if cred == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "无效的API凭证",
			"data": nil,
		})
		c.Abort()
		return
	}

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

	// 使用助手配置中的参数
	language := "zh-cn"
	speaker := assistant.Speaker
	if speaker == "" {
		speaker = "502007"
	}
	systemPrompt := assistant.SystemPrompt
	temperature := assistant.Temperature

	// Get LLM model from assistant, fallback to default
	llmModel := assistant.LLMModel
	if llmModel == "" {
		llmModel = "deepseek-v3.1" // Default model
	}

	// 创建WebSocket处理器
	handler := voice.NewHardwareHandler(h.db, logger.Lg)

	ctx := context.Background()

	// 使用常量中的VAD配置（覆盖数据库值）
	// 数据库中的值太低，导致Barge-in过于敏感
	vadThreshold := constants.DefaultVADThreshold
	vadConsecutiveFrames := constants.DefaultVADConsecutiveFrames

	handler.HandlerHardwareWebsocket(ctx, &voice.HardwareOptions{
		Conn:                 conn,
		AssistantID:          assistantID,
		DeviceID:             &device.ID,
		Language:             language,
		Speaker:              speaker,
		Temperature:          float64(temperature),
		SystemPrompt:         systemPrompt,
		KnowledgeKey:         "",
		UserID:               device.UserID,
		MacAddress:           device.MacAddress,
		LLMModel:             llmModel,
		Credential:           cred,
		EnableVAD:            assistant.EnableVAD,
		VADThreshold:         vadThreshold,
		VADConsecutiveFrames: vadConsecutiveFrames,
		VoiceCloneID:         assistant.VoiceCloneID,
	})
}
