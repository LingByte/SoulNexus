package task

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var graphStore graph.Store
var graphProcessorEnabled bool

// InitGraphProcessor 初始化图处理器
func InitGraphProcessor(store graph.Store, enabled bool) {
	graphStore = store
	graphProcessorEnabled = enabled
	if enabled {
		logger.Info("Graph processor initialized and enabled")
	} else {
		logger.Info("Graph processor initialized but disabled")
	}
}

// ProcessConversationAsync 异步处理对话记录，提取知识并存储到图数据库
func ProcessConversationAsync(db *gorm.DB, assistantID int64, sessionID string, userID uint) {
	if !graphProcessorEnabled || graphStore == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := processConversation(ctx, db, assistantID, sessionID, userID); err != nil {
			logger.Error("Failed to process conversation for graph",
				zap.Int64("assistantID", assistantID),
				zap.String("sessionID", sessionID),
				zap.Error(err))
		}
	}()
}

// processConversation 处理对话记录的核心逻辑
func processConversation(ctx context.Context, db *gorm.DB, assistantID int64, sessionID string, userID uint) error {
	// 1. 获取对话记录
	logs, err := models.GetChatSessionLogsBySession(db, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get conversation logs: %w", err)
	}

	if len(logs) == 0 {
		logger.Warn("No conversation logs found", zap.String("sessionID", sessionID))
		return nil
	}

	// 2. 获取助手信息
	var assistant models.Assistant
	if err := db.First(&assistant, assistantID).Error; err != nil {
		return fmt.Errorf("failed to get assistant: %w", err)
	}

	// 如果该助手未开启图记忆功能，则直接跳过（不写入 Neo4j）
	if !assistant.EnableGraphMemory {
		logger.Info("Graph memory is disabled for assistant, skip graph processing",
			zap.Int64("assistantID", assistantID),
			zap.String("sessionID", sessionID))
		return nil
	}

	// 3. 获取助手的 LLM 配置（用于总结对话）
	// 通过 assistant 的 ApiKey 和 ApiSecret 查找对应的 UserCredential
	var credential models.UserCredential
	if assistant.ApiKey != "" && assistant.ApiSecret != "" {
		if err := db.Where("api_key = ? AND api_secret = ? AND llm_provider != ''", assistant.ApiKey, assistant.ApiSecret).First(&credential).Error; err != nil {
			logger.Warn("No LLM credential found for assistant's API key/secret",
				zap.String("apiKey", assistant.ApiKey),
				zap.Error(err))
			// 如果没有找到对应的 LLM 凭证，仍然可以存储基础信息
			return storeBasicConversationInfo(ctx, assistantID, sessionID, userID, assistant.Name, logs)
		}
	} else {
		// 如果 assistant 没有配置 ApiKey/ApiSecret，回退到使用 userID 查找
		if err := db.Where("user_id = ? AND llm_provider != ''", userID).First(&credential).Error; err != nil {
			logger.Warn("No LLM credential found for conversation summary", zap.Error(err))
			// 如果没有 LLM 凭证，仍然可以存储基础信息
			return storeBasicConversationInfo(ctx, assistantID, sessionID, userID, assistant.Name, logs)
		}
	}
	// 4. 构建对话文本用于 LLM 总结
	conversationText := buildConversationText(logs)

	// 5. 调用 LLM 进行总结
	summary, relevant, err := summarizeConversation(ctx, &credential, conversationText, assistant, logs, userID, sessionID)
	if err != nil {
		logger.Warn("Failed to summarize conversation with LLM, storing basic info", zap.Error(err))
		// LLM 总结失败时，仍然存储基础信息
		return storeBasicConversationInfo(ctx, assistantID, sessionID, userID, assistant.Name, logs)
	}

	// 6. 如果对话与助手定位不相关，则不存储到图数据库
	if !relevant {
		logger.Info("Conversation is not relevant to assistant, skipping graph storage",
			zap.Int64("assistantID", assistantID),
			zap.String("sessionID", sessionID),
			zap.String("summary", summary.Summary))
		return nil
	}

	// 7. 存储到图数据库
	return graphStore.ProcessConversation(ctx, assistantID, sessionID, summary)
}

// buildConversationText 构建对话文本
func buildConversationText(logs []models.ChatSessionLog) string {
	var sb strings.Builder
	for i, log := range logs {
		sb.WriteString(fmt.Sprintf("轮次 %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("用户: %s\n", log.UserMessage))
		sb.WriteString(fmt.Sprintf("助手: %s\n\n", log.AgentMessage))
	}
	return sb.String()
}

// summarizeConversation 使用 LLM 总结对话
// 返回: summary, relevant, error
func summarizeConversation(ctx context.Context, credential *models.UserCredential, conversationText string, assistant models.Assistant, logs []models.ChatSessionLog, userID uint, sessionID string) (*graph.ConversationSummary, bool, error) {
	// 创建 LLM provider
	// 传入一个非空的系统提示词，避免 DashScope 等兼容模式 API 报错（Content 不能为空）
	systemPrompt := "You are a helpful assistant for analyzing conversations."
	llmProvider, err := llm.NewLLMProvider(ctx, credential.LLMProvider, credential.LLMApiKey, credential.LLMApiURL, systemPrompt)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create LLM provider: %w", err)
	}
	defer llmProvider.Hangup()

	// 构建总结 prompt
	prompt := buildSummaryPrompt(conversationText, assistant)

	// 调用 LLM
	temp := float32(0.4) // 降低温度以获得更稳定的总结
	options := llm.QueryOptions{
		Model:       assistant.LLMModel,
		Temperature: &temp,
		MaxTokens:   intPtr(2000),
		Stream:      false,
	}

	if assistant.LLMModel == "" {
		// 如果没有指定模型，使用默认模型
		options.Model = "gpt-4o-mini"
	}

	response, err := llmProvider.QueryWithOptions(prompt, options)
	if err != nil {
		return nil, false, fmt.Errorf("failed to query LLM: %w", err)
	}

	// 解析 LLM 返回的 JSON
	summary, relevant, err := parseSummaryResponse(response, assistant.ID, assistant.Name, userID, sessionID, logs)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse summary response: %w", err)
	}

	return summary, relevant, nil
}

// buildSummaryPrompt 构建总结 prompt
func buildSummaryPrompt(conversationText string, assistant models.Assistant) string {
	// 构建助手上下文信息
	assistantContext := fmt.Sprintf("助手名称: %s", assistant.Name)
	if assistant.Description != "" {
		assistantContext += fmt.Sprintf("\n助手描述: %s", assistant.Description)
	}
	if assistant.SystemPrompt != "" {
		assistantContext += fmt.Sprintf("\n系统提示词: %s", assistant.SystemPrompt)
	}
	if assistant.PersonaTag != "" {
		assistantContext += fmt.Sprintf("\n角色标签: %s", assistant.PersonaTag)
	}

	return fmt.Sprintf(`你是一个专业的对话分析助手。请分析以下对话内容，提取关键信息并返回 JSON 格式的结果。

助手上下文信息：
%s

对话内容：
%s

重要提示：
1. 请首先判断对话内容是否与助手的系统提示词、描述和角色定位相关
2. 如果对话内容与助手定位完全无关（例如：用户只是闲聊、测试、或者讨论与助手职责无关的话题），请在 JSON 中设置 "relevant": false，并返回空的 topics、intents 和 knowledge 数组
3. 只有当对话内容与助手定位相关时，才提取和存储知识信息

请按照以下 JSON 格式返回分析结果：
{
  "relevant": true,  // 对话是否与助手定位相关，如果不相关请设置为 false
  "summary": "对话的简要总结（100-200字，如果不相关可以简要说明原因）",
  "topics": ["主题1", "主题2", ...],  // 讨论的主要主题，最多10个（如果不相关则为空数组）
  "intents": ["意图1", "意图2", ...],  // 用户的主要意图，最多5个（如果不相关则为空数组）
  "knowledge": [
    {
      "content": "知识点内容",
      "category": "知识点类别（如：事实、方法、概念等）",
      "source": "conversation",
      "relatedTopics": ["相关主题1", "相关主题2"]
    }
  ]  // 从对话中提取的重要知识点，最多10个（如果不相关则为空数组）
}

要求：
1. 严格判断对话相关性：如果对话与助手的系统提示词、描述、角色定位完全无关，必须设置 "relevant": false
2. topics 应该是具体的主题名称，如"机器学习"、"Python编程"等
3. intents 应该是用户的意图，如"学习新知识"、"解决问题"、"获取建议"等
4. knowledge 应该是有价值的知识点，避免过于琐碎的信息
5. 只返回 JSON，不要包含其他文字说明`, assistantContext, conversationText)
}

// parseSummaryResponse 解析 LLM 返回的总结
// 返回: summary, relevant, error
func parseSummaryResponse(response string, assistantID int64, assistantName string, userID uint, sessionID string, logs []models.ChatSessionLog) (*graph.ConversationSummary, bool, error) {
	// 尝试提取 JSON（可能 LLM 返回了 markdown 代码块）
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var result struct {
		Relevant  *bool             `json:"relevant"` // 使用指针，因为可能不存在（向后兼容）
		Summary   string            `json:"summary"`
		Topics    []string          `json:"topics"`
		Intents   []string          `json:"intents"`
		Knowledge []graph.Knowledge `json:"knowledge"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// 如果解析失败，尝试提取 JSON 部分
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}")
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				return nil, false, fmt.Errorf("failed to parse JSON: %w", err)
			}
		} else {
			return nil, false, fmt.Errorf("failed to find JSON in response: %w", err)
		}
	}

	// 判断是否相关（如果字段不存在，默认为 true，保持向后兼容）
	relevant := true
	if result.Relevant != nil {
		relevant = *result.Relevant
	}

	// 构建 Turn 列表
	turns := make([]graph.Turn, 0, len(logs))
	for i, log := range logs {
		turns = append(turns, graph.Turn{
			UserMessage:  log.UserMessage,
			AgentMessage: log.AgentMessage,
			Sequence:     i,
		})
	}

	summary := &graph.ConversationSummary{
		AssistantID:   assistantID,
		AssistantName: assistantName,
		UserID:        userID,
		SessionID:     sessionID,
		Summary:       result.Summary,
		Topics:        result.Topics,
		Intents:       result.Intents,
		Turns:         turns,
		Knowledge:     result.Knowledge,
	}

	return summary, relevant, nil
}

// storeBasicConversationInfo 存储基础对话信息（当 LLM 总结失败时）
func storeBasicConversationInfo(ctx context.Context, assistantID int64, sessionID string, userID uint, assistantName string, logs []models.ChatSessionLog) error {
	// 从对话中提取基础主题（简单的关键词提取）
	topics := extractBasicTopics(logs)

	// 构建基础总结
	turns := make([]graph.Turn, 0, len(logs))
	for i, log := range logs {
		turns = append(turns, graph.Turn{
			UserMessage:  log.UserMessage,
			AgentMessage: log.AgentMessage,
			Sequence:     i,
		})
	}

	summary := &graph.ConversationSummary{
		AssistantID:   assistantID,
		AssistantName: assistantName,
		UserID:        userID,
		SessionID:     sessionID,
		Summary:       fmt.Sprintf("包含 %d 轮对话", len(logs)),
		Topics:        topics,
		Intents:       []string{},
		Turns:         turns,
		Knowledge:     []graph.Knowledge{},
	}

	return graphStore.ProcessConversation(ctx, assistantID, sessionID, summary)
}

// extractBasicTopics 从对话中提取基础主题（简单实现）
func extractBasicTopics(logs []models.ChatSessionLog) []string {
	// 简单的关键词提取（实际可以更复杂）
	topics := make(map[string]bool)
	commonWords := map[string]bool{
		"的": true, "了": true, "是": true, "在": true, "有": true,
		"和": true, "与": true, "或": true, "但": true, "就": true,
		"这": true, "那": true, "你": true, "我": true, "他": true,
	}

	for _, log := range logs {
		// 简单的分词（实际应该使用专业分词工具）
		words := strings.FieldsFunc(log.UserMessage+log.AgentMessage, func(r rune) bool {
			return r == ' ' || r == '，' || r == '。' || r == '？' || r == '！' || r == '、'
		})

		for _, word := range words {
			word = strings.TrimSpace(word)
			if len(word) > 1 && !commonWords[word] {
				topics[word] = true
			}
		}
	}

	result := make([]string, 0, len(topics))
	for topic := range topics {
		if len(result) >= 10 { // 最多10个主题
			break
		}
		result = append(result, topic)
	}

	return result
}

// intPtr 返回 int 指针
func intPtr(i int) *int {
	return &i
}
