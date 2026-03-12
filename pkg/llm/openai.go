package llm

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type LLMHandler struct {
	client          *openai.Client
	ctx             context.Context
	systemMsg       string
	defaultModel    string // 默认模型
	baseURL         string // Store base URL for logging
	mutex           sync.Mutex
	messages        []openai.ChatCompletionMessage
	hangupChan      chan struct{}
	interruptCh     chan struct{}
	functionManager *FunctionToolManager
	lastUsage       openai.Usage
	lastUsageValid  bool
}

type QueryOptions struct {
	Model               string                               // chat model
	MaxTokens           *int                                 // max tokens contains request and response
	MaxCompletionTokens *int                                 // max completion tokens
	Temperature         *float32                             // temperature
	TopP                *float32                             // top p
	FrequencyPenalty    *float32                             // reduce repeat content
	PresencePenalty     *float32                             // add new topic
	Stop                []string                             // if generate this word will stop generation
	N                   *int                                 // generate N kind of results
	LogitBias           map[string]int                       // force to increase or decrease the frequency of words
	User                string                               // user flag
	Stream              bool                                 // is stream
	ResponseFormat      *openai.ChatCompletionResponseFormat // format of response
	Seed                *int                                 //  seed
	EnableJSONOutput    bool                                 // 是否启用JSON格式化输出

	// Optional context for logging (used by ChatSessionLog)
	UserID       *uint  // 用户ID（可选，用于记录日志）
	AssistantID  *int64 // 助手ID（可选，用于记录日志）
	CredentialID *uint  // 凭证ID（可选，用于记录日志）
	SessionID    string // 会话ID（可选，用于记录日志）
	ChatType     string // 聊天类型（可选，用于记录日志）
}

// ToolCallInfo contains information about a tool call
type ToolCallInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// LLMUsageInfo contains comprehensive LLM call information for signal emission
type LLMUsageInfo struct {
	// Request Information
	Model               string
	MaxTokens           *int
	MaxCompletionTokens *int
	Temperature         *float32
	TopP                *float32
	FrequencyPenalty    *float32
	PresencePenalty     *float32
	Stop                []string
	N                   *int
	LogitBias           map[string]int
	User                string
	Stream              bool
	ResponseFormat      *openai.ChatCompletionResponseFormat
	Seed                *int

	// Response Information
	ResponseID       string
	Object           string
	Created          int64
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int

	// Context Information
	SystemPrompt string
	MessageCount int

	// Timing Information
	StartTime time.Time `json:"startTime"` // 调用开始时间
	EndTime   time.Time `json:"endTime"`   // 调用结束时间
	Duration  int64     `json:"duration"`  // 调用持续时间（毫秒）

	// Tool Call Information
	HasToolCalls  bool           `json:"hasToolCalls"`        // 是否调用了工具
	ToolCallCount int            `json:"toolCallCount"`       // 工具调用数量
	ToolCalls     []ToolCallInfo `json:"toolCalls,omitempty"` // 工具调用详情

	// Optional context for logging (used by ChatSessionLog)
	UserID       *uint  // 用户ID（可选，用于记录日志）
	AssistantID  *int64 // 助手ID（可选，用于记录日志）
	CredentialID *uint  // 凭证ID（可选，用于记录日志）
	SessionID    string // 会话ID（可选，用于记录日志）
	ChatType     string // 聊天类型（可选，用于记录日志）
}

// NewLLMHandler creates a new LLM handler
// If logger is nil, uses the global logger from pkg/logger
func NewLLMHandler(ctx context.Context, apiKey, baseURL, systemPrompt string) *LLMHandler {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := openai.NewClientWithConfig(config)
	// Create system message
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// Create function tool manager
	functionManager := NewFunctionToolManager()

	return &LLMHandler{
		client:          client,
		systemMsg:       systemPrompt,
		baseURL:         baseURL,
		ctx:             ctx,
		messages:        messages,
		hangupChan:      make(chan struct{}),
		interruptCh:     make(chan struct{}, 1),
		functionManager: functionManager,
	}
}

func (h *LLMHandler) Query(text, model string) (string, error) {
	// 如果没有指定模型，使用默认模型
	if model == "" {
		model = h.defaultModel
	}
	return h.QueryWithOptions(text, QueryOptions{Model: model, Temperature: Float32Ptr(0.7)})
}

// QueryWithOptions 支持完整的参数控制
func (h *LLMHandler) QueryWithOptions(text string, options QueryOptions) (string, error) {
	// Record start time for timing statistics
	startTime := time.Now()

	h.mutex.Lock()

	// Clean up any incomplete tool calls before starting new query
	// This prevents errors from previous failed tool call processing
	h.cleanupIncompleteToolCalls()

	// Add user message to history
	h.messages = append(h.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: text,
	})

	logger.Debug("Added user message to history",
		zap.String("user_text", text),
		zap.Int("total_messages", len(h.messages)),
	)

	h.mutex.Unlock()

	// Get all available function tools
	tools := h.functionManager.GetTools()

	// Loop to handle tool calls - continue until we get a final text response
	maxIterations := 10 // Prevent infinite loops
	var finalResponse string
	var finalUsage openai.Usage
	var finalResponseID string
	var finalObject string
	var finalCreated int64
	var finalFinishReason string

	// Track tool calls across all iterations
	var allToolCalls []ToolCallInfo

	h.mutex.Lock()
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Log message history for debugging
		logger.Debug("Building LLM request",
			zap.Int("iteration", iteration),
			zap.Int("message_count", len(h.messages)),
			zap.String("model", options.Model),
		)
		// Log last few messages for debugging
		startIdx := 0
		if len(h.messages) > 3 {
			startIdx = len(h.messages) - 3
		}
		for i := startIdx; i < len(h.messages); i++ {
			msg := h.messages[i]
			contentPreview := msg.Content
			if len(contentPreview) > 50 {
				contentPreview = contentPreview[:50] + "..."
			}
			logger.Debug("Message in history",
				zap.Int("index", i),
				zap.String("role", msg.Role),
				zap.String("content_preview", contentPreview),
			)
		}

		// Construct the OpenAI request with current messages
		// 确保所有消息的 Content 字段都是有效的字符串类型（避免 DashScope 等兼容模式 API 报错）
		// DashScope 兼容模式要求 Content 必须是 string 或 array，不能是 null 或 object
		sanitizedMessages := make([]openai.ChatCompletionMessage, 0, len(h.messages))
		for _, msg := range h.messages {
			// 创建一个新的消息副本，确保 Content 字段是字符串类型
			sanitizedMsg := openai.ChatCompletionMessage{
				Role:      msg.Role,
				Content:   "", // 初始化为空字符串
				ToolCalls: msg.ToolCalls,
			}

			// 确保 Content 是字符串类型（不能是 nil）
			if msg.Content != "" {
				sanitizedMsg.Content = msg.Content
			} else if msg.Role == openai.ChatMessageRoleSystem {
				// System 消息的 Content 不能为空，至少需要一个占位符
				sanitizedMsg.Content = "You are a helpful assistant."
			}

			// 复制 ToolCallID（如果有）
			if msg.ToolCallID != "" {
				sanitizedMsg.ToolCallID = msg.ToolCallID
			}

			sanitizedMessages = append(sanitizedMessages, sanitizedMsg)
		}

		request := openai.ChatCompletionRequest{
			Model:    options.Model,
			Messages: sanitizedMessages,
			Tools:    tools,
		}

		// Apply optional parameters if provided
		if options.MaxTokens != nil {
			request.MaxTokens = *options.MaxTokens
		}
		if options.MaxCompletionTokens != nil {
			request.MaxCompletionTokens = *options.MaxCompletionTokens
		}
		if options.Temperature != nil {
			request.Temperature = *options.Temperature
		}
		if options.TopP != nil {
			request.TopP = *options.TopP
		}
		if options.FrequencyPenalty != nil {
			request.FrequencyPenalty = *options.FrequencyPenalty
		}
		if options.PresencePenalty != nil {
			request.PresencePenalty = *options.PresencePenalty
		}
		if len(options.Stop) > 0 {
			request.Stop = options.Stop
		}
		if options.N != nil {
			request.N = *options.N
		}
		if options.LogitBias != nil {
			request.LogitBias = options.LogitBias
		}
		if options.User != "" {
			request.User = options.User
		}
		if options.ResponseFormat != nil {
			request.ResponseFormat = options.ResponseFormat
		}
		if options.Seed != nil {
			request.Seed = options.Seed
		}
		// 处理 JSON 格式化输出
		if options.EnableJSONOutput {
			request.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			}
		}
		request.Stream = options.Stream

		// Set default model if not provided
		if request.Model == "" {
			request.Model = openai.GPT4o
		}

		// 调试：打印第一条消息的 Content 类型和内容（用于排查 DashScope 兼容模式问题）
		if len(request.Messages) > 0 {
			firstMsg := request.Messages[0]
			logger.Debug("First message details",
				zap.String("role", firstMsg.Role),
				zap.String("content_type", fmt.Sprintf("%T", firstMsg.Content)),
				zap.String("content_preview", func() string {
					if len(firstMsg.Content) > 100 {
						return firstMsg.Content[:100] + "..."
					}
					return firstMsg.Content
				}()),
				zap.Bool("content_is_empty", firstMsg.Content == ""),
			)
		}

		// Send the request to OpenAI
		logger.Info("Sending request to LLM API",
			zap.String("base_url", h.baseURL),
			zap.String("model", request.Model),
			zap.Int("message_count", len(request.Messages)),
			zap.Int("iteration", iteration),
		)

		response, err := h.client.CreateChatCompletion(h.ctx, request)
		if err != nil {
			h.mutex.Unlock()
			logger.Error("LLM API call failed",
				zap.String("base_url", h.baseURL),
				zap.String("model", request.Model),
				zap.Error(err),
			)
			return "", fmt.Errorf("error querying LLM API: %w", err)
		}

		logger.Info("Received response from LLM API",
			zap.String("response_id", response.ID),
			zap.String("object", response.Object),
			zap.Int("choices_count", len(response.Choices)),
		)

		// Validate response
		if len(response.Choices) == 0 {
			h.mutex.Unlock()
			return "", fmt.Errorf("no choices in response")
		}

		// Get the response message
		message := response.Choices[0].Message

		// Log the response for debugging
		logger.Info("LLM response received",
			zap.String("role", message.Role),
			zap.String("content", message.Content),
			zap.Int("content_length", len(message.Content)),
			zap.Int("tool_calls", len(message.ToolCalls)),
			zap.String("finish_reason", string(response.Choices[0].FinishReason)),
			zap.Int("message_count", len(h.messages)),
		)

		// Check if response content is suspiciously similar to user input
		if len(h.messages) > 0 {
			lastUserMsg := ""
			for i := len(h.messages) - 1; i >= 0; i-- {
				if h.messages[i].Role == openai.ChatMessageRoleUser {
					lastUserMsg = h.messages[i].Content
					break
				}
			}
			if lastUserMsg != "" && message.Content == lastUserMsg {
				logger.Error("CRITICAL: LLM response exactly matches user input - API may not be working correctly",
					zap.String("user_input", lastUserMsg),
					zap.String("response", message.Content),
					zap.String("base_url", h.baseURL),
					zap.String("model", request.Model),
				)
				h.mutex.Unlock()
				return "", fmt.Errorf("LLM API returned user input as response - possible API configuration issue. User: %s, Response: %s", lastUserMsg, message.Content)
			}
		}

		// Accumulate usage
		finalUsage.PromptTokens += response.Usage.PromptTokens
		finalUsage.CompletionTokens += response.Usage.CompletionTokens
		finalUsage.TotalTokens += response.Usage.TotalTokens

		// Capture response metadata
		if response.ID != "" {
			finalResponseID = response.ID
		}
		if response.Object != "" {
			finalObject = response.Object
		}
		if response.Created != 0 {
			finalCreated = response.Created
		}
		if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
			finalFinishReason = string(response.Choices[0].FinishReason)
		}

		// Handle tool calls if any
		if len(message.ToolCalls) > 0 {
			logger.Info("Tool calls detected", zap.Int("count", len(message.ToolCalls)))

			// Collect tool call information for statistics
			for _, toolCall := range message.ToolCalls {
				allToolCalls = append(allToolCalls, ToolCallInfo{
					ID:        toolCall.ID,
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				})
			}

			// Add assistant message with tool calls to history
			h.messages = append(h.messages, message)

			// Track which tool calls we've processed
			processedToolCallIDs := make(map[string]bool)

			for _, toolCall := range message.ToolCalls {
				// Handle all function calls through the function manager
				result, err := h.functionManager.HandleToolCall(toolCall)
				if err != nil {
					logger.Error("Failed to handle tool call",
						zap.String("tool", toolCall.Function.Name),
						zap.Error(err))
					// Add error result to conversation
					h.messages = append(h.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    fmt.Sprintf("Error: %v", err),
						ToolCallID: toolCall.ID,
					})
				} else {
					logger.Info("Tool call result",
						zap.String("tool", toolCall.Function.Name),
						zap.String("result", result))
					// Add tool result to conversation history
					h.messages = append(h.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    result,
						ToolCallID: toolCall.ID,
					})
				}
				processedToolCallIDs[toolCall.ID] = true
			}

			// Verify all tool calls were processed
			for _, toolCall := range message.ToolCalls {
				if !processedToolCallIDs[toolCall.ID] {
					logger.Warn("Tool call was not processed, adding error response",
						zap.String("toolCallID", toolCall.ID))
					h.messages = append(h.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "Error: Tool call was not processed",
						ToolCallID: toolCall.ID,
					})
				}
			}

			// Update request messages for next iteration
			request.Messages = h.messages
			// Continue loop to get final response
			continue
		}

		// If no tool calls, add assistant message and we have the final response
		// Validate that we have content
		if message.Content == "" {
			logger.Warn("Empty response content from LLM",
				zap.String("finish_reason", string(response.Choices[0].FinishReason)),
				zap.Int("message_count", len(h.messages)),
			)
			// Don't add empty message to history, but still return error
			h.mutex.Unlock()
			return "", fmt.Errorf("empty response content from LLM")
		}

		// Check if response is just echoing user input (safety check)
		if len(h.messages) > 0 {
			lastUserMsg := ""
			for i := len(h.messages) - 1; i >= 0; i-- {
				if h.messages[i].Role == openai.ChatMessageRoleUser {
					lastUserMsg = h.messages[i].Content
					break
				}
			}
			if lastUserMsg != "" && message.Content == lastUserMsg {
				logger.Warn("LLM response matches user input exactly - possible echo issue",
					zap.String("content", message.Content),
					zap.Int("message_count", len(h.messages)),
				)
				// Still add to history but log warning
			}
		}

		h.messages = append(h.messages, message)
		finalResponse = message.Content
		break
	}

	// Check if we hit max iterations without getting a final response
	if finalResponse == "" {
		h.mutex.Unlock()
		// Clean up incomplete tool calls before returning error
		h.cleanupIncompleteToolCalls()
		return "", fmt.Errorf("max iterations reached without final response, possible incomplete tool calls")
	}

	h.mutex.Unlock()

	// Record end time and calculate duration
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// Process the response and emit signal for async token usage recording
	h.lastUsage = finalUsage
	h.lastUsageValid = true

	// Emit signal for async token usage recording with comprehensive information
	usageInfo := &LLMUsageInfo{
		// Request Information
		Model:               options.Model,
		MaxTokens:           options.MaxTokens,
		MaxCompletionTokens: options.MaxCompletionTokens,
		Temperature:         options.Temperature,
		TopP:                options.TopP,
		FrequencyPenalty:    options.FrequencyPenalty,
		PresencePenalty:     options.PresencePenalty,
		Stop:                options.Stop,
		N:                   options.N,
		LogitBias:           options.LogitBias,
		User:                options.User,
		Stream:              options.Stream,
		ResponseFormat:      options.ResponseFormat,
		Seed:                options.Seed,

		// Response Information
		ResponseID:       finalResponseID,
		Object:           finalObject,
		Created:          finalCreated,
		FinishReason:     finalFinishReason,
		PromptTokens:     finalUsage.PromptTokens,
		CompletionTokens: finalUsage.CompletionTokens,
		TotalTokens:      finalUsage.TotalTokens,

		// Context Information
		SystemPrompt: h.systemMsg,
		MessageCount: len(h.messages),

		// Timing Information
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,

		// Tool Call Information
		HasToolCalls:  len(allToolCalls) > 0,
		ToolCallCount: len(allToolCalls),
		ToolCalls:     allToolCalls,

		// Optional context for logging
		UserID:       options.UserID,
		AssistantID:  options.AssistantID,
		CredentialID: options.CredentialID,
		SessionID:    options.SessionID,
		ChatType:     options.ChatType,
	}

	utils.Sig().Emit(constants.LLMUsage, usageInfo, text, finalResponse)

	return finalResponse, nil
}

// QueryStream processes the LLM response as a stream and calls the callback for each segment
// callback: func(segment string, isComplete bool) error
// - segment: the text segment received
// - isComplete: true if this is the final segment
func (h *LLMHandler) QueryStream(text string, options QueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	// Record start time for timing statistics
	startTime := time.Now()

	h.mutex.Lock()

	// Clean up any incomplete tool calls before starting new query
	// This prevents errors from previous failed tool call processing
	h.cleanupIncompleteToolCalls()

	// Add user message to history
	h.messages = append(h.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: text,
	})

	// Get all available function tools
	tools := h.functionManager.GetTools()

	// Construct the OpenAI request with all available options
	request := openai.ChatCompletionRequest{
		Model:    options.Model,
		Messages: h.messages,
		Stream:   true, // Force stream mode
		Tools:    tools,
	}

	// Apply optional parameters if provided
	if options.MaxTokens != nil {
		request.MaxTokens = *options.MaxTokens
	}

	if options.MaxCompletionTokens != nil {
		request.MaxCompletionTokens = *options.MaxCompletionTokens
	}

	if options.Temperature != nil {
		request.Temperature = *options.Temperature
	}

	if options.TopP != nil {
		request.TopP = *options.TopP
	}

	if options.FrequencyPenalty != nil {
		request.FrequencyPenalty = *options.FrequencyPenalty
	}

	if options.PresencePenalty != nil {
		request.PresencePenalty = *options.PresencePenalty
	}

	if len(options.Stop) > 0 {
		request.Stop = options.Stop
	}

	if options.N != nil {
		request.N = *options.N
	}

	if options.LogitBias != nil {
		request.LogitBias = options.LogitBias
	}

	if options.User != "" {
		request.User = options.User
	}

	if options.ResponseFormat != nil {
		request.ResponseFormat = options.ResponseFormat
	}

	if options.Seed != nil {
		request.Seed = options.Seed
	}

	// 处理 JSON 格式化输出
	if options.EnableJSONOutput {
		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	// Set default model if not provided
	if request.Model == "" {
		request.Model = openai.GPT4o
	}

	// Include usage in stream
	request.StreamOptions = &openai.StreamOptions{
		IncludeUsage: true,
	}

	// Generate a unique ID for this stream
	streamID := fmt.Sprintf("stream-%s", uuid.New().String())
	logger.Info("Starting LLM stream", zap.String("streamID", streamID))

	// 清空之前可能残留的中断信号
	select {
	case <-h.interruptCh:
		logger.Debug("Cleared stale interrupt signal before starting new stream")
	default:
	}

	// Create stream
	stream, err := h.client.CreateChatCompletionStream(h.ctx, request)
	if err != nil {
		h.mutex.Unlock()
		return "", fmt.Errorf("error creating chat completion stream: %w", err)
	}

	h.mutex.Unlock()
	defer stream.Close()

	// Buffer to collect text
	fullResponse := ""
	var finishReason string
	var responseID string
	var created int64
	var object string

	// Collect tool calls from stream
	var collectedToolCalls []openai.ToolCall
	toolCallMap := make(map[int]*openai.ToolCall) // Index -> ToolCall

	// Track tool calls for statistics
	var allToolCalls []ToolCallInfo

	// Process the stream of responses
	for {
		// Check for interrupt or hangup signals (non-blocking)
		select {
		case <-h.interruptCh:
			logger.Info("LLM stream interrupted", zap.String("streamID", streamID))
			return fullResponse, fmt.Errorf("stream interrupted")
		case <-h.hangupChan:
			logger.Info("LLM stream hangup requested", zap.String("streamID", streamID))
			return fullResponse, fmt.Errorf("hangup requested")
		default:
			// Continue to receive stream data
		}

		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Stream closed normally
				break
			}
			return fullResponse, fmt.Errorf("error receiving from stream: %w", err)
		}

		// Capture response metadata
		if response.ID != "" {
			responseID = response.ID
		}
		if response.Object != "" {
			object = response.Object
		}
		if response.Created != 0 {
			created = response.Created
		}

		// Capture token usage if provided in this chunk
		if response.Usage != nil {
			h.mutex.Lock()
			h.lastUsage = *response.Usage
			h.lastUsageValid = true
			h.mutex.Unlock()
		}

		// Get finish reason and handle tool calls
		if len(response.Choices) > 0 {
			if response.Choices[0].FinishReason != "" {
				finishReason = string(response.Choices[0].FinishReason)
			}

			// Collect tool calls from stream (they come in chunks)
			if len(response.Choices[0].Delta.ToolCalls) > 0 {
				for _, deltaToolCall := range response.Choices[0].Delta.ToolCalls {
					// Index is a pointer, need to dereference it
					if deltaToolCall.Index == nil {
						continue
					}
					idx := *deltaToolCall.Index

					if toolCallMap[idx] == nil {
						toolCallMap[idx] = &openai.ToolCall{
							ID:   deltaToolCall.ID,
							Type: deltaToolCall.Type,
							Function: openai.FunctionCall{
								Name:      deltaToolCall.Function.Name,
								Arguments: deltaToolCall.Function.Arguments,
							},
						}
					} else {
						// Append to existing tool call
						if deltaToolCall.Function.Name != "" {
							toolCallMap[idx].Function.Name = deltaToolCall.Function.Name
						}
						if deltaToolCall.Function.Arguments != "" {
							toolCallMap[idx].Function.Arguments += deltaToolCall.Function.Arguments
						}
					}
				}
			}

			// Process content if available - 直接透传，不做缓冲
			if response.Choices[0].Delta.Content != "" {
				content := response.Choices[0].Delta.Content
				fullResponse += content

				// 直接调用 callback，不等待标点符号
				if callback != nil {
					if err := callback(content, false); err != nil {
						logger.Error("Failed to process stream segment", zap.Error(err))
					}
				}
			}
		}
	}

	// Convert tool call map to slice, sorted by index
	maxIdx := 0
	for idx := range toolCallMap {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	for i := 0; i <= maxIdx; i++ {
		if toolCall, exists := toolCallMap[i]; exists {
			collectedToolCalls = append(collectedToolCalls, *toolCall)
		}
	}

	// Handle tool calls if any were collected
	h.mutex.Lock()
	if len(collectedToolCalls) > 0 {
		logger.Info("Tool calls detected in stream", zap.Int("count", len(collectedToolCalls)))

		// Collect tool call information for statistics
		for _, toolCall := range collectedToolCalls {
			allToolCalls = append(allToolCalls, ToolCallInfo{
				ID:        toolCall.ID,
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			})
		}

		// Add assistant message with tool calls to history
		h.messages = append(h.messages, openai.ChatCompletionMessage{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   fullResponse,
			ToolCalls: collectedToolCalls,
		})

		// Process each tool call
		for _, toolCall := range collectedToolCalls {
			// Handle all function calls through the function manager
			result, err := h.functionManager.HandleToolCall(toolCall)
			if err != nil {
				logger.Error("Failed to handle tool call",
					zap.String("tool", toolCall.Function.Name),
					zap.Error(err))
				// Add error result to conversation
				h.messages = append(h.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    fmt.Sprintf("Error: %v", err),
					ToolCallID: toolCall.ID,
				})
			} else {
				logger.Info("Tool call result",
					zap.String("tool", toolCall.Function.Name),
					zap.String("result", result))
				// Add tool result to conversation history
				h.messages = append(h.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}
		}

		// Update request and make another call to get final response
		request.Messages = h.messages
		request.Stream = false      // Use non-streaming for final response
		request.StreamOptions = nil // Clear stream options for non-streaming request
		h.mutex.Unlock()

		// Make another call to get final response after tool execution
		finalResp, err := h.client.CreateChatCompletion(h.ctx, request)
		if err != nil {
			return fullResponse, fmt.Errorf("error getting final response after tool call: %w", err)
		}

		finalResponse := ""
		if len(finalResp.Choices) > 0 {
			finalResponse = finalResp.Choices[0].Message.Content
			// Add final response to history
			h.mutex.Lock()
			h.messages = append(h.messages, finalResp.Choices[0].Message)
			h.mutex.Unlock()
		}

		// Record end time and calculate duration
		endTime := time.Now()
		duration := endTime.Sub(startTime).Milliseconds()

		// Update usage from final response
		if len(finalResp.Choices) > 0 {
			h.mutex.Lock()
			h.lastUsage.PromptTokens += finalResp.Usage.PromptTokens
			h.lastUsage.CompletionTokens += finalResp.Usage.CompletionTokens
			h.lastUsage.TotalTokens += finalResp.Usage.TotalTokens
			h.lastUsageValid = true
			h.mutex.Unlock()
		}

		// Emit signal for async token usage recording with tool call information
		usageInfo := &LLMUsageInfo{
			// Request Information
			Model:               request.Model,
			MaxTokens:           options.MaxTokens,
			MaxCompletionTokens: options.MaxCompletionTokens,
			Temperature:         options.Temperature,
			TopP:                options.TopP,
			FrequencyPenalty:    options.FrequencyPenalty,
			PresencePenalty:     options.PresencePenalty,
			Stop:                options.Stop,
			N:                   options.N,
			LogitBias:           options.LogitBias,
			User:                options.User,
			Stream:              true,
			ResponseFormat:      options.ResponseFormat,
			Seed:                options.Seed,

			// Response Information
			ResponseID:       responseID,
			Object:           object,
			Created:          created,
			FinishReason:     finishReason,
			PromptTokens:     h.lastUsage.PromptTokens,
			CompletionTokens: h.lastUsage.CompletionTokens,
			TotalTokens:      h.lastUsage.TotalTokens,

			// Context Information
			SystemPrompt: h.systemMsg,
			MessageCount: len(h.messages),

			// Timing Information
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  duration,

			// Tool Call Information
			HasToolCalls:  len(allToolCalls) > 0,
			ToolCallCount: len(allToolCalls),
			ToolCalls:     allToolCalls,

			// Optional context for logging
			UserID:       options.UserID,
			AssistantID:  options.AssistantID,
			CredentialID: options.CredentialID,
			SessionID:    options.SessionID,
			ChatType:     options.ChatType,
		}

		utils.Sig().Emit(constants.LLMUsage, usageInfo, text, fullResponse+finalResponse)

		// Stream the final response through callback
		if callback != nil {
			if err := callback(finalResponse, false); err != nil {
				logger.Error("Failed to process final response segment", zap.Error(err))
			}
			if err := callback("", true); err != nil {
				logger.Error("Failed to send completion signal", zap.Error(err))
			}
		}

		return fullResponse + finalResponse, nil
	}

	// No tool calls, add assistant's complete response to conversation history
	h.messages = append(h.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: fullResponse,
	})

	// Record end time and calculate duration
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// Emit signal for async token usage recording
	if h.lastUsageValid {
		usageInfo := &LLMUsageInfo{
			// Request Information
			Model:               request.Model,
			MaxTokens:           options.MaxTokens,
			MaxCompletionTokens: options.MaxCompletionTokens,
			Temperature:         options.Temperature,
			TopP:                options.TopP,
			FrequencyPenalty:    options.FrequencyPenalty,
			PresencePenalty:     options.PresencePenalty,
			Stop:                options.Stop,
			N:                   options.N,
			LogitBias:           options.LogitBias,
			User:                options.User,
			Stream:              true, // Always true for stream
			ResponseFormat:      options.ResponseFormat,
			Seed:                options.Seed,

			// Response Information
			ResponseID:       responseID,
			Object:           object,
			Created:          created,
			FinishReason:     finishReason,
			PromptTokens:     h.lastUsage.PromptTokens,
			CompletionTokens: h.lastUsage.CompletionTokens,
			TotalTokens:      h.lastUsage.TotalTokens,

			// Context Information
			SystemPrompt: h.systemMsg,
			MessageCount: len(h.messages),

			// Timing Information
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  duration,

			// Tool Call Information
			HasToolCalls:  false,
			ToolCallCount: 0,
			ToolCalls:     nil,

			// Optional context for logging
			UserID:       options.UserID,
			AssistantID:  options.AssistantID,
			CredentialID: options.CredentialID,
			SessionID:    options.SessionID,
			ChatType:     options.ChatType,
		}

		utils.Sig().Emit(constants.LLMUsage, usageInfo, text, fullResponse)
	}

	logger.Info("LLM stream completed",
		zap.String("streamID", streamID),
		zap.Int("responseLength", len(fullResponse)),
		zap.Int("totalTokens", h.lastUsage.TotalTokens),
	)

	h.mutex.Unlock()

	// 发送完成信号给 callback
	if callback != nil {
		if err := callback("", true); err != nil {
			logger.Error("Failed to send completion signal", zap.Error(err))
		}
	}

	return fullResponse, nil
}

// RegisterFunctionTool 注册新的Function Tool
func (h *LLMHandler) RegisterFunctionTool(name, description string, parameters json.RawMessage, callback FunctionToolCallback) {
	h.functionManager.RegisterTool(name, description, parameters, callback)
}

// RegisterFunctionToolDefinition 通过定义结构注册工具
func (h *LLMHandler) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	h.functionManager.RegisterToolDefinition(def)
}

// GetFunctionTools 获取所有可用的Function Tools
func (h *LLMHandler) GetFunctionTools() []openai.Tool {
	return h.functionManager.GetTools()
}

// ListFunctionTools 列出所有已注册的工具名称
func (h *LLMHandler) ListFunctionTools() []string {
	return h.functionManager.ListTools()
}

// GetLastUsage returns the usage information from the last API call
func (h *LLMHandler) GetLastUsage() (openai.Usage, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.lastUsage, h.lastUsageValid
}

// cleanupIncompleteToolCalls 清理未完成的 tool_calls
// 如果对话历史中有 assistant 消息包含 tool_calls，但后面没有对应的 tool 消息，则移除该 assistant 消息
func (h *LLMHandler) cleanupIncompleteToolCalls() {
	// Find assistant messages with tool_calls
	for i := len(h.messages) - 1; i >= 0; i-- {
		msg := h.messages[i]
		if msg.Role == openai.ChatMessageRoleAssistant && len(msg.ToolCalls) > 0 {
			// Check if all tool calls have responses
			toolCallIDs := make(map[string]bool)
			for _, toolCall := range msg.ToolCalls {
				toolCallIDs[toolCall.ID] = false
			}

			// Check messages after this assistant message
			for j := i + 1; j < len(h.messages); j++ {
				nextMsg := h.messages[j]
				if nextMsg.Role == openai.ChatMessageRoleTool && nextMsg.ToolCallID != "" {
					if _, exists := toolCallIDs[nextMsg.ToolCallID]; exists {
						toolCallIDs[nextMsg.ToolCallID] = true
					}
				}
			}

			// If any tool call is missing a response, remove the assistant message and all subsequent messages
			hasIncomplete := false
			for _, hasResponse := range toolCallIDs {
				if !hasResponse {
					hasIncomplete = true
					break
				}
			}

			if hasIncomplete {
				logger.Warn("Found incomplete tool calls, removing assistant message and subsequent messages",
					zap.Int("messageIndex", i),
					zap.Int("messagesToRemove", len(h.messages)-i))
				// Remove this assistant message and all subsequent messages
				h.messages = h.messages[:i]
				break
			}
		}
	}
}

// ResetMessages clears the conversation history
func (h *LLMHandler) ResetMessages() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.messages = []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: h.systemMsg,
		},
	}
}

// SetSystemPrompt 动态设置系统提示词
func (h *LLMHandler) SetSystemPrompt(systemPrompt string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.systemMsg = systemPrompt

	// 更新messages中的系统消息
	if len(h.messages) > 0 && h.messages[0].Role == openai.ChatMessageRoleSystem {
		h.messages[0].Content = systemPrompt
	} else {
		// 如果没有系统消息，添加一个
		systemMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		}
		h.messages = append([]openai.ChatCompletionMessage{systemMessage}, h.messages...)
	}
}

// SetModel 设置默认模型
func (h *LLMHandler) SetModel(model string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.defaultModel = model
}

// GetMessages returns the current conversation history
func (h *LLMHandler) GetMessages() []openai.ChatCompletionMessage {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Return a copy to prevent external modification
	messages := make([]openai.ChatCompletionMessage, len(h.messages))
	copy(messages, h.messages)
	return messages
}

func Float32Ptr(v float32) *float32 {
	return &v
}

func Float64Ptr(v float64) *float32 {
	val := float32(v)
	return &val
}

func IntPtr(v int) *int {
	return &v
}
