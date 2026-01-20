package listeners

import (
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/task"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/llm"
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
		// Type assertion to get usage information
		usageInfo, ok := sender.(*llm.LLMUsageInfo)
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
			zap.String("model", usageInfo.Model),
			zap.Int("promptTokens", usageInfo.PromptTokens),
			zap.Int("completionTokens", usageInfo.CompletionTokens),
			zap.Int("totalTokens", usageInfo.TotalTokens),
			zap.String("user", usageInfo.User),
			zap.String("userInput", userInput),
			zap.String("aiResponse", aiResponse),
			zap.Int64("duration", usageInfo.Duration),
			zap.Bool("hasToolCalls", usageInfo.HasToolCalls),
			zap.Int("toolCallCount", usageInfo.ToolCallCount),
		)

		logger.Info("=== LLM Usage Details ===",
			zap.String("Model", usageInfo.Model),
			zap.Int("Prompt Tokens", usageInfo.PromptTokens),
			zap.Int("Completion Tokens", usageInfo.CompletionTokens),
			zap.Int("Total Tokens", usageInfo.TotalTokens),
			zap.String("User ID", usageInfo.User),
			zap.Time("Start Time", usageInfo.StartTime),
			zap.Time("End Time", usageInfo.EndTime),
			zap.Int64("Duration (ms)", usageInfo.Duration),
			zap.Bool("Has Tool Calls", usageInfo.HasToolCalls),
			zap.Int("Tool Call Count", usageInfo.ToolCallCount),
		)

		// Log tool calls if any
		if usageInfo.HasToolCalls && len(usageInfo.ToolCalls) > 0 {
			for i, toolCall := range usageInfo.ToolCalls {
				logger.Info("=== Tool Call ===",
					zap.Int("Index", i+1),
					zap.String("ID", toolCall.ID),
					zap.String("Name", toolCall.Name),
					zap.String("Arguments", toolCall.Arguments),
				)
			}
		}

		// If there is a database connection and necessary context information, save to ChatSessionLog
		if llmListenerDB != nil && usageInfo.UserID != nil && usageInfo.AssistantID != nil {
			go func() {
				// Convert to rtcmedia.LLMUsage
				llmUsage := models.ConvertLLMUsageInfoToLLMUsage(usageInfo)
				if llmUsage == nil {
					logger.Warn("Failed to convert LLM Usage")
					return
				}

				// Generate or use provided sessionID
				sessionID := usageInfo.SessionID
				if sessionID == "" {
					sessionID = fmt.Sprintf("session_%d_%d", *usageInfo.UserID, time.Now().Unix())
				}

				// Determine chat type
				chatType := usageInfo.ChatType
				if chatType == "" {
					chatType = models.ChatTypeText // Default to text chat
				}

				// Calculate duration (milliseconds to seconds, if 0 use default value)
				duration := int(usageInfo.Duration / 1000)
				if duration == 0 && !usageInfo.StartTime.IsZero() && !usageInfo.EndTime.IsZero() {
					duration = int(usageInfo.EndTime.Sub(usageInfo.StartTime).Seconds())
				}

				// Save chat log
				_, err := models.CreateChatSessionLogWithUsage(
					llmListenerDB,
					*usageInfo.UserID,
					*usageInfo.AssistantID,
					chatType,
					sessionID,
					userInput,
					aiResponse,
					"", // audioURL
					duration,
					llmUsage,
				)
				if err != nil {
					logger.Error("Failed to save chat log", zap.Error(err))
				} else {
					logger.Info("Chat log saved", zap.String("sessionID", sessionID))

					// Trigger async graph processing for conversation
					// This will summarize the conversation and store knowledge in Neo4j
					task.ProcessConversationAsync(
						llmListenerDB,
						*usageInfo.AssistantID,
						sessionID,
						*usageInfo.UserID,
					)
				}

				// Record LLM usage in billing system
				var credentialID uint
				var assistantID *uint

				// Prioritize CredentialID from usageInfo
				var groupID *uint
				if usageInfo.CredentialID != nil {
					credentialID = *usageInfo.CredentialID
				} else if usageInfo.AssistantID != nil {
					// If no CredentialID, try to get credential ID from assistant
					aid := uint(*usageInfo.AssistantID)
					assistantID = &aid

					// Get credential ID and group ID from assistant (if assistant is associated with a credential)
					var assistant models.Assistant
					if err := llmListenerDB.Where("id = ? AND user_id = ?", *assistantID, *usageInfo.UserID).
						First(&assistant).Error; err == nil {
						// Assistant may be associated with credentials in other ways, temporarily use 0 here
						// Subsequently, it can be obtained according to actual business logic
						// Get group ID from assistant if it's organization-shared
						if assistant.GroupID != nil {
							groupID = assistant.GroupID
						}
					}
				}

				// Record LLM usage
				if err := models.RecordLLMUsage(
					llmListenerDB,
					*usageInfo.UserID,
					credentialID,
					assistantID,
					groupID,
					sessionID,
					usageInfo.Model,
					usageInfo.PromptTokens,
					usageInfo.CompletionTokens,
					usageInfo.TotalTokens,
				); err != nil {
					logger.Warn("Failed to record LLM usage", zap.Error(err))
				}
			}()
		}
	})
}
