package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voice/constants"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HeaderSoulnexusVoiceSecret is sent by cmd/voice when calling the binding API.
// Must match SoulNexus env LINGECHO_HARDWARE_BINDING_SECRET when that env is set.
// The header value (X-Lingecho-Voice-Secret) and env name are preserved for
// backward compatibility with deployed VoiceServer instances.
const HeaderSoulnexusVoiceSecret = "X-Lingecho-Voice-Secret"

// soulnexusHardwareSession holds everything needed for in-process hardware WS
// (pkg/voice) or for exposing dialog-plane credentials to VoiceServer.
type soulnexusHardwareSession struct {
	Device               *models.Device
	Assistant            models.Agent
	Credential           *models.UserCredential
	AssistantID          uint
	Language             string
	Speaker              string
	SystemPrompt         string
	Temperature          float64
	LLMModel             string
	EnableVAD            bool
	VADThreshold         float64
	VADConsecutiveFrames int
}

// resolveSoulnexusHardwareSession loads device + assistant + credential for a
// hardware Device-Id (MAC). msg is a short Chinese message suitable for JSON / logs.
func (h *Handlers) resolveSoulnexusHardwareSession(deviceID string) (*soulnexusHardwareSession, int, string) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, http.StatusBadRequest, "缺少Device-Id参数"
	}

	device, err := models.GetDeviceByMacAddress(h.db, deviceID)
	if err != nil || device == nil {
		return nil, http.StatusBadRequest, "设备未激活，请先激活设备"
	}
	if device.AgentID == nil {
		return nil, http.StatusBadRequest, "设备未绑定助手"
	}
	assistantID := *device.AgentID

	var assistant models.Agent
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		return nil, http.StatusBadRequest, "获取助手配置失败: " + err.Error()
	}
	if assistant.ID == 0 {
		return nil, http.StatusBadRequest, "助手不存在"
	}
	if assistant.ApiKey == "" || assistant.ApiSecret == "" {
		return nil, http.StatusBadRequest, "助手未配置API凭证"
	}

	cred, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, assistant.ApiKey, assistant.ApiSecret)
	if err != nil {
		return nil, http.StatusInternalServerError, "获取凭证失败: " + err.Error()
	}
	if cred == nil {
		return nil, http.StatusBadRequest, "无效的API凭证"
	}

	language := "zh-cn"
	speaker := assistant.Speaker
	if speaker == "" {
		speaker = "502007"
	}
	llmModel := assistant.LLMModel
	if llmModel == "" {
		llmModel = "deepseek-v3.1"
	}

	vadThreshold := constants.DefaultVADThreshold
	vadConsecutiveFrames := constants.DefaultVADConsecutiveFrames

	return &soulnexusHardwareSession{
		Device:               device,
		Assistant:            assistant,
		Credential:           cred,
		AssistantID:          assistantID,
		Language:             language,
		Speaker:              speaker,
		SystemPrompt:         assistant.SystemPrompt,
		Temperature:          float64(assistant.Temperature),
		LLMModel:             llmModel,
		EnableVAD:            assistant.EnableVAD,
		VADThreshold:         vadThreshold,
		VADConsecutiveFrames: vadConsecutiveFrames,
	}, 0, ""
}

// HandleSoulnexusHardwareBinding returns dialog-plane payload (apiKey, apiSecret, agentId)
// for VoiceServer to merge into ?payload= on the xiaozhi adapter, matching the browser path.
//
// Security: when LINGECHO_HARDWARE_BINDING_SECRET is set in the environment, requests must
// send HeaderSoulnexusVoiceSecret with the same value (typically only cmd/voice on a private network).
func (h *Handlers) HandleSoulnexusHardwareBinding(c *gin.Context) {
	wantSecret := strings.TrimSpace(utils.GetEnv("LINGECHO_HARDWARE_BINDING_SECRET"))
	if wantSecret != "" {
		if c.GetHeader(HeaderSoulnexusVoiceSecret) != wantSecret {
			response.FailWithCode(c, 403, "unauthorized", nil)
			return
		}
	}

	deviceID := c.GetHeader("Device-Id")
	if deviceID == "" {
		deviceID = c.Query("device-id")
	}

	logger.Info("SoulNexus hardware binding request",
		zap.String("deviceID", deviceID),
		zap.String("path", c.Request.URL.Path))

	sess, status, msg := h.resolveSoulnexusHardwareSession(deviceID)
	if sess == nil {
		c.JSON(status, gin.H{
			"code": 500,
			"msg":  msg,
			"data": nil,
		})
		c.Abort()
		return
	}

	agentIDStr := strconv.FormatInt(sess.Assistant.ID, 10)
	response.Success(c, "ok", gin.H{
		"payload": gin.H{
			"apiKey":    sess.Assistant.ApiKey,
			"apiSecret": sess.Assistant.ApiSecret,
			"agentId":   agentIDStr,
		},
	})
}
