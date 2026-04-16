package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

var llmListenerDB *gorm.DB

// InitLLMListenerWithDB Initialize LLM usage listener (with database connection)
func InitLLMListenerWithDB(db *gorm.DB) {
	llmListenerDB = db
	initSignalConnections(db)
}

// 初始化信号连接
func initSignalConnections(db *gorm.DB) {
	// LLMUsage 信号连接
	utils.Sig().Connect(constants.SignalLLMUsage, func(sender any, params ...any) {
		usageInfo, ok := sender.(map[string]interface{})
		if !ok {
			return
		}

		requestID := asString(usageInfo["request_id"])
		if requestID == "" {
			return
		}

		// 检查是否已存在相同 request_id 的记录
		var existingUsage models.LLMUsage
		if err := db.Where("request_id = ?", requestID).First(&existingUsage).Error; err == nil {
			return
		}

		usage := &models.LLMUsage{
			ID:          utils.SnowflakeUtil.GenID(),
			RequestID:   requestID,
			SessionID:   asString(usageInfo["session_id"]),
			UserID:      asString(usageInfo["user_id"]),
			Provider:    asString(usageInfo["provider"]),
			Model:       asString(usageInfo["model"]),
			BaseURL:     asString(usageInfo["base_url"]),
			RequestType: asString(usageInfo["request_type"]),

			// Token统计
			InputTokens:  asInt(usageInfo["input_tokens"]),
			OutputTokens: asInt(usageInfo["output_tokens"]),
			TotalTokens:  asInt(usageInfo["total_tokens"]),

			// 性能指标
			LatencyMs:   asInt64(usageInfo["latency_ms"]),
			TTFTMs:      asInt64(usageInfo["ttft_ms"]),
			TPS:         asFloat64(usageInfo["tps"]),
			QueueTimeMs: asInt64(usageInfo["queue_time_ms"]),

			// 请求响应内容
			RequestContent:  asString(usageInfo["request_content"]),
			ResponseContent: asString(usageInfo["response_content"]),

			// 请求元信息
			UserAgent:  asString(usageInfo["user_agent"]),
			IPAddress:  asString(usageInfo["ip_address"]),
			StatusCode: asInt(usageInfo["status_code"]),

			// 错误信息
			Success:      asBool(usageInfo["success"]),
			ErrorCode:    asString(usageInfo["error_code"]),
			ErrorMessage: asString(usageInfo["error_message"]),

			// 时间戳
			RequestedAt:  toTimeFromMillis(usageInfo["requested_at"]),
			StartedAt:    toTimeFromMillis(usageInfo["started_at"]),
			FirstTokenAt: toTimeFromMillis(usageInfo["first_token_at"]),
			CompletedAt:  toTimeFromMillis(usageInfo["completed_at"]),
		}
		_ = db.Create(usage).Error
	})

	// SessionCreated 信号连接
	utils.Sig().Connect(llm.SignalSessionCreated, func(sender any, params ...any) {
		if len(params) < 1 {
			return
		}
		data, ok := params[0].(llm.SessionCreatedData)
		if !ok {
			return
		}

		sessionID := data.SessionID
		if sessionID == "" {
			sessionID = utils.SnowflakeUtil.GenID()
		}

		session := &models.ChatSession{
			ID:           sessionID,
			UserID:       data.UserID,
			Title:        data.Title,
			Provider:     data.Provider,
			Model:        data.Model,
			SystemPrompt: data.SystemPrompt,
			Status:       "active",
			CreatedAt:    toTimeFromMillis(data.CreatedAt),
			UpdatedAt:    toTimeFromMillis(data.CreatedAt),
		}
		_ = db.Where("id = ?", sessionID).Assign(session).FirstOrCreate(&models.ChatSession{}).Error
	})

	// MessageCreated 信号连接
	utils.Sig().Connect(llm.SignalMessageCreated, func(sender any, params ...any) {
		if len(params) < 1 {
			return
		}
		data, ok := params[0].(llm.MessageCreatedData)
		if !ok {
			return
		}
		message := &models.ChatMessage{
			ID:         utils.SnowflakeUtil.GenID(),
			SessionID:  data.SessionID,
			Role:       data.Role,
			Content:    data.Content,
			TokenCount: data.TokenCount,
			Model:      data.Model,
			Provider:   data.Provider,
			RequestID:  data.RequestID,
			CreatedAt:  toTimeFromMillis(data.CreatedAt),
			UpdatedAt:  toTimeFromMillis(data.CreatedAt),
		}
		_ = db.Create(message).Error
	})
}

// 辅助函数
func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

var asFloat64 = func(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func asInt(v interface{}) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case float32:
		return int(t)
	default:
		return 0
	}
}

func asInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}

func asBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func toTimeFromMillis(v interface{}) time.Time {
	ms := asInt64(v)
	if ms <= 0 {
		return time.Now()
	}
	return time.Unix(ms/1000, 0)
}
