package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/task"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var llmListenerDB *gorm.DB

// InitLLMListenerWithDB Initialize LLM usage listener (with database connection)
func InitLLMListenerWithDB(db *gorm.DB) {
	llmListenerDB = db
	utils.Sig().Connect(constants.LLMUsage, func(sender any, params ...any) {
		usageInfo, ok := sender.(map[string]interface{})
		if !ok {
			logger.Warn("LLM usage signal: invalid sender type")
			return
		}

		// Get parameters
		var userInput, aiResponse string
		if len(params) >= 1 {
			if input, ok := params[0].(string); ok {
				userInput = input
			}
		}
		if len(params) >= 2 {
			if response, ok := params[1].(string); ok {
				aiResponse = response
			}
		}

		logger.Info("LLM Token Usage",
			zap.String("model", toString(usageInfo["model"])),
			zap.Int("promptTokens", toInt(usageInfo["input_tokens"])),
			zap.Int("completionTokens", toInt(usageInfo["output_tokens"])),
			zap.Int("totalTokens", toInt(usageInfo["total_tokens"])),
			zap.String("user", toString(usageInfo["user_id"])),
			zap.String("userInput", userInput),
			zap.String("aiResponse", aiResponse),
			zap.Int64("duration", toInt64(usageInfo["latency_ms"])),
		)

		logger.Info("=== LLM Usage Details ===",
			zap.String("Model", toString(usageInfo["model"])),
			zap.Int("Prompt Tokens", toInt(usageInfo["input_tokens"])),
			zap.Int("Completion Tokens", toInt(usageInfo["output_tokens"])),
			zap.Int("Total Tokens", toInt(usageInfo["total_tokens"])),
			zap.String("User ID", toString(usageInfo["user_id"])),
			zap.Int64("Duration (ms)", toInt64(usageInfo["latency_ms"])),
		)

		// If there is a database connection and necessary context information, save to ChatSessionLog
		userID := uint(toInt64(usageInfo["user_id"]))
		assistantIDVal := uint(toInt64(usageInfo["assistant_id"]))
		if llmListenerDB != nil && userID > 0 && assistantIDVal > 0 {
			go func() {
				// Generate or use provided sessionID
				sessionID := toString(usageInfo["session_id"])
				if sessionID == "" {
					sessionID = fmt.Sprintf("session_%d_%d", userID, time.Now().Unix())
				}

				// Determine chat type
				chatType := toString(usageInfo["chat_type"])
				if chatType == "" {
					chatType = models.ChatTypeText // Default to text chat
				}

				// Calculate duration (milliseconds to seconds, if 0 use default value)
				duration := int(toInt64(usageInfo["latency_ms"]) / 1000)
				if duration == 0 {
					duration = 1
				}

				// Save chat log
				_, err := models.CreateChatSessionLogWithUsage(
					llmListenerDB,
					userID,
					int64(assistantIDVal),
					chatType,
					sessionID,
					userInput,
					aiResponse,
					"", // audioURL
					duration,
					&models.LLMUsage{
						Model:            toString(usageInfo["model"]),
						PromptTokens:     toInt(usageInfo["input_tokens"]),
						CompletionTokens: toInt(usageInfo["output_tokens"]),
						TotalTokens:      toInt(usageInfo["total_tokens"]),
					},
				)
				if err != nil {
					logger.Error("Failed to save chat log", zap.Error(err))
				} else {
					logger.Info("Chat log saved", zap.String("sessionID", sessionID))

					// Trigger async graph processing for conversation
					// This will summarize the conversation and store knowledge in Neo4j
					task.ProcessConversationAsync(
						llmListenerDB,
						int64(assistantIDVal),
						sessionID,
						userID,
					)
				}

				// Record LLM usage in billing system
				var credentialID uint
				var assistantID *uint

				// Prioritize CredentialID from usageInfo
				var groupID *uint
				credID := uint(toInt64(usageInfo["credential_id"]))
				if credID > 0 {
					credentialID = credID
				} else if assistantIDVal > 0 {
					// If no CredentialID, try to get credential ID from assistant
					aid := assistantIDVal
					assistantID = &aid

					// Get credential ID and group ID from assistant (if assistant is associated with a credential)
					var assistant models.Assistant
					if err := llmListenerDB.Where("id = ? AND user_id = ?", *assistantID, userID).
						First(&assistant).Error; err == nil {
						if assistant.GroupID != nil {
							groupID = assistant.GroupID
						}
					}
				}

				// Record LLM usage
				if err := models.RecordLLMUsage(
					llmListenerDB,
					userID,
					credentialID,
					assistantID,
					groupID,
					sessionID,
					toString(usageInfo["model"]),
					toInt(usageInfo["input_tokens"]),
					toInt(usageInfo["output_tokens"]),
					toInt(usageInfo["total_tokens"]),
				); err != nil {
					logger.Warn("Failed to record LLM usage", zap.Error(err))
				}
			}()
		}
	})
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toInt(v interface{}) int {
	return int(toInt64(v))
}

func toInt64(v interface{}) int64 {
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
