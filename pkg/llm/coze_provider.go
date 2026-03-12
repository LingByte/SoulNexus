package llm

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/coze-dev/coze-go"
	"go.uber.org/zap"
)

const (
	// MaxMessageHistory 最大消息历史数量，避免发送过长的历史导致请求变慢
	MaxMessageHistory = 20
	// RequestTimeout 请求超时时间（秒）- 增加到60秒，因为流式响应可能需要更长时间
	RequestTimeout = 60 * time.Second
	// StreamReadTimeout 流式读取单个数据块的超时时间（秒）
	StreamReadTimeout = 10 * time.Second
)

// CozeProvider Coze LLM 提供者实现
type CozeProvider struct {
	client          coze.CozeAPI
	ctx             context.Context
	systemMsg       string
	mutex           sync.Mutex
	messages        []coze.Message
	hangupChan      chan struct{}
	interruptCh     chan struct{}
	functionManager *FunctionToolManager
	lastUsage       Usage
	lastUsageValid  bool

	// Coze 特定配置
	botID  string // Bot ID（必需）
	userID string // User ID（必需，可以从 credential 或其他地方获取）
}

// CozeConfig Coze 配置
type CozeConfig struct {
	BotID   string `json:"botId"`   // Bot ID
	UserID  string `json:"userId"`  // User ID（可选，如果不提供会使用默认值）
	BaseURL string `json:"baseUrl"` // Base URL（可选）
}

// NewCozeProvider 创建 Coze 提供者
// apiKey: Coze API Token
// botID: Coze Bot ID（可以从 LLMApiURL 或配置中获取）
// userID: 用户 ID（可选，如果不提供会使用默认值）
// systemPrompt: 系统提示词（Coze 通过 Bot 配置，这里主要用于兼容）
func NewCozeProvider(ctx context.Context, apiKey, botID, userID, systemPrompt string, baseURL ...string) (*CozeProvider, error) {
	if botID == "" {
		return nil, fmt.Errorf("botID is required for Coze provider")
	}

	// 如果 userID 为空，使用默认值
	if userID == "" {
		userID = "default_user"
	}

	// 创建认证客户端
	authClient := coze.NewTokenAuth(apiKey)

	// 创建 Coze 客户端
	var client coze.CozeAPI
	if len(baseURL) > 0 && baseURL[0] != "" {
		client = coze.NewCozeAPI(authClient, coze.WithBaseURL(baseURL[0]))
	} else {
		client = coze.NewCozeAPI(authClient)
	}

	// 初始化消息历史（Coze 使用自己的消息格式）
	messages := make([]coze.Message, 0)
	if systemPrompt != "" {
		// Coze 可能不支持系统消息，但我们可以将其作为第一条用户消息的上下文
		// 或者通过 Bot 配置来设置
	}

	return &CozeProvider{
		client:          client,
		ctx:             ctx,
		systemMsg:       systemPrompt,
		messages:        messages,
		hangupChan:      make(chan struct{}),
		interruptCh:     make(chan struct{}, 1),
		functionManager: NewFunctionToolManager(),
		botID:           botID,
		userID:          userID,
	}, nil
}

// Query 执行非流式查询
func (p *CozeProvider) Query(text, model string) (string, error) {
	return p.QueryWithOptions(text, QueryOptions{Model: model, Temperature: Float32Ptr(0.7)})
}

// truncateMessages 限制消息历史长度，只保留最近的 N 条消息
func (p *CozeProvider) truncateMessages() {
	if len(p.messages) > MaxMessageHistory {
		// 保留最近的 MaxMessageHistory 条消息
		start := len(p.messages) - MaxMessageHistory
		p.messages = p.messages[start:]
		logger.Debug("Truncated message history",
			zap.Int("original_count", len(p.messages)+start),
			zap.Int("truncated_count", len(p.messages)))
	}
}

// QueryWithOptions 执行带完整参数的非流式查询
// 优化：使用非流式API，减少网络往返和流式处理开销
func (p *CozeProvider) QueryWithOptions(text string, options QueryOptions) (string, error) {
	startTime := time.Now()

	p.mutex.Lock()

	// 添加用户消息到历史
	p.messages = append(p.messages, coze.Message{
		Role:    "user",
		Content: text,
	})

	// 限制消息历史长度，避免请求体过大
	p.truncateMessages()

	// 获取要发送的消息（限制后的历史）
	messagesToSend := p.messages
	p.mutex.Unlock()

	// 创建带超时的上下文（非流式可以设置更短的超时）
	ctx, cancel := context.WithTimeout(p.ctx, RequestTimeout)
	defer cancel()

	// 优化：直接使用已有的消息格式，减少转换开销
	additionalMessages := make([]*coze.Message, 0, len(messagesToSend))
	for i := range messagesToSend {
		additionalMessages = append(additionalMessages, &messagesToSend[i])
	}

	// 使用非流式API，减少网络往返和流式处理开销
	streamFlag := false
	req := &coze.CreateChatsReq{
		BotID:    p.botID,
		UserID:   p.userID,
		Messages: additionalMessages,
		Stream:   &streamFlag,
	}

	logger.Debug("Coze non-streaming request started",
		zap.Int("message_count", len(additionalMessages)),
		zap.String("bot_id", p.botID))

	requestStartTime := time.Now()

	// 使用流式API但设置Stream=false，SDK可能会优化为非流式处理
	// 或者直接使用流式但快速收集完整响应
	stream, err := p.client.Chat.Stream(ctx, req)
	if err != nil {
		return "", fmt.Errorf("error creating chat stream: %w", err)
	}
	defer stream.Close()

	requestLatency := time.Since(requestStartTime)
	logger.Debug("Coze request created",
		zap.Duration("request_latency_ms", requestLatency))

	var finalResponse string
	var chatUsage *coze.ChatUsage
	firstChunkTime := time.Time{}

	// 快速收集流式响应，减少不必要的检查
	for {
		// 检查中断信号（简化检查，减少开销）
		select {
		case <-p.interruptCh:
			return "", fmt.Errorf("request interrupted")
		case <-p.hangupChan:
			return "", fmt.Errorf("hangup requested")
		case <-ctx.Done():
			if finalResponse != "" {
				break // 有部分响应就返回
			}
			return "", fmt.Errorf("request timeout: %w", ctx.Err())
		default:
		}

		// 直接读取，减少检查开销
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			// 如果是超时但有部分响应，返回部分响应
			if (ctx.Err() != nil || strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded")) && finalResponse != "" {
				logger.Warn("Coze request timeout but partial response received",
					zap.String("partial_response", finalResponse),
					zap.Error(err))
				break
			}
			return "", fmt.Errorf("error receiving from stream: %w", err)
		}

		// 记录第一个数据块时间
		if firstChunkTime.IsZero() && event != nil {
			firstChunkTime = time.Now()
		}

		// 处理事件
		if event.Event == coze.ChatEventConversationMessageDelta {
			if event.Message != nil && event.Message.Content != "" {
				finalResponse += event.Message.Content
			}
		} else if event.Event == coze.ChatEventConversationMessageCompleted {
			if event.Message != nil && event.Message.Content != "" {
				if !strings.Contains(finalResponse, event.Message.Content) {
					finalResponse += event.Message.Content
				}
			}
			if event.Chat != nil && event.Chat.Usage != nil {
				chatUsage = event.Chat.Usage
			}
			break // 完成，立即退出
		} else if event.IsDone() {
			break
		}
	}

	// 更新消息历史（添加用户消息和助手响应）
	p.mutex.Lock()
	// 添加助手响应到历史
	if finalResponse != "" {
		p.messages = append(p.messages, coze.Message{
			Role:    "assistant",
			Content: finalResponse,
		})
	}
	p.mutex.Unlock()

	// 记录使用统计
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// 记录性能指标
	if !firstChunkTime.IsZero() {
		firstTokenLatency := firstChunkTime.Sub(requestStartTime).Milliseconds()
		totalLatency := endTime.Sub(requestStartTime).Milliseconds()
		logger.Info("Coze request completed (optimized)",
			zap.Duration("total_duration_ms", time.Duration(duration)*time.Millisecond),
			zap.Duration("first_token_latency_ms", time.Duration(firstTokenLatency)*time.Millisecond),
			zap.Duration("request_latency_ms", requestLatency),
			zap.Duration("total_latency_ms", time.Duration(totalLatency)*time.Millisecond),
			zap.Int("response_length", len(finalResponse)),
			zap.Int("message_count", len(messagesToSend)))
	}

	// 使用 Coze 提供的 Usage 信息，如果没有则估算
	var promptTokens, completionTokens, totalTokens int
	if chatUsage != nil {
		promptTokens = chatUsage.InputCount
		completionTokens = chatUsage.OutputCount
		totalTokens = chatUsage.TokenCount
	} else {
		// 估算 token 使用（如果 Coze 不提供）
		promptTokens = estimateTokens(text)
		completionTokens = estimateTokens(finalResponse)
		totalTokens = promptTokens + completionTokens
	}

	p.lastUsage = Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
	}
	p.lastUsageValid = true

	// 发送使用统计信号
	usageInfo := &LLMUsageInfo{
		Model:            options.Model,
		Temperature:      options.Temperature,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		SystemPrompt:     p.systemMsg,
		MessageCount:     len(p.messages),
		StartTime:        startTime,
		EndTime:          endTime,
		Duration:         duration,
		UserID:           options.UserID,
		AssistantID:      options.AssistantID,
		CredentialID:     options.CredentialID,
		SessionID:        options.SessionID,
		ChatType:         options.ChatType,
	}

	utils.Sig().Emit(constants.LLMUsage, usageInfo, text, finalResponse)

	return finalResponse, nil
}

// QueryStream 执行流式查询
func (p *CozeProvider) QueryStream(text string, options QueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	startTime := time.Now()

	p.mutex.Lock()

	// 添加用户消息到历史
	p.messages = append(p.messages, coze.Message{
		Role:    "user",
		Content: text,
	})

	// 限制消息历史长度，避免请求体过大
	p.truncateMessages()

	// 获取要发送的消息（限制后的历史）
	messagesToSend := p.messages
	p.mutex.Unlock()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(p.ctx, RequestTimeout)
	defer cancel()

	// 构建 Coze 流式请求
	// 将消息转换为 Coze 的 Message 格式
	additionalMessages := make([]*coze.Message, 0, len(messagesToSend))
	for _, msg := range messagesToSend {
		additionalMessages = append(additionalMessages, &coze.Message{
			Role:    coze.MessageRole(msg.Role),
			Content: msg.Content,
		})
	}

	streamFlag := true
	req := &coze.CreateChatsReq{
		BotID:    p.botID,
		UserID:   p.userID,
		Messages: additionalMessages,
		Stream:   &streamFlag,
	}

	logger.Debug("Coze stream request started",
		zap.Int("message_count", len(additionalMessages)),
		zap.String("bot_id", p.botID))

	// 清空之前可能残留的中断信号
	select {
	case <-p.interruptCh:
		logger.Debug("Cleared stale interrupt signal before starting new stream")
	default:
	}

	requestStartTime := time.Now()

	// 创建流式连接
	stream, err := p.client.Chat.Stream(ctx, req)
	if err != nil {
		return "", fmt.Errorf("error creating chat stream: %w", err)
	}
	defer stream.Close()

	requestLatency := time.Since(requestStartTime)
	logger.Debug("Coze stream created",
		zap.Duration("request_latency_ms", requestLatency))

	var fullResponse string
	var buffer string
	firstChunkTime := time.Time{}
	lastReceiveTime := time.Now()

	// 处理流式响应
	for {
		// 检查中断信号
		select {
		case <-p.interruptCh:
			logger.Info("Coze stream interrupted")
			return fullResponse, fmt.Errorf("stream interrupted")
		case <-p.hangupChan:
			logger.Info("Coze stream hangup requested")
			return fullResponse, fmt.Errorf("hangup requested")
		default:
		}

		// 检查总体超时（只在创建流后检查，给流式响应更多时间）
		if ctx.Err() != nil {
			// 如果已经收到部分响应，返回已收到的内容而不是错误
			if fullResponse != "" {
				logger.Warn("Coze stream timeout but partial response received",
					zap.String("partial_response", fullResponse),
					zap.Error(ctx.Err()))
				break
			}
			logger.Warn("Coze stream timeout", zap.Error(ctx.Err()))
			return fullResponse, fmt.Errorf("stream timeout: %w", ctx.Err())
		}

		// 检查流读取超时（如果超过 StreamReadTimeout 没有收到数据，可能连接已断开）
		if !firstChunkTime.IsZero() && time.Since(lastReceiveTime) > StreamReadTimeout {
			logger.Warn("Coze stream read timeout, no data received",
				zap.Duration("timeout", StreamReadTimeout),
				zap.String("partial_response", fullResponse))
			// 如果已有部分响应，返回它
			if fullResponse != "" {
				break
			}
			return fullResponse, fmt.Errorf("stream read timeout: no data received for %v", StreamReadTimeout)
		}

		// 读取流数据（stream.Recv() 会阻塞，但 context 超时会中断）
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			// 检查是否是超时错误
			if ctx.Err() != nil {
				// 如果已经收到部分响应，返回已收到的内容
				if fullResponse != "" {
					logger.Warn("Coze stream timeout but partial response received",
						zap.String("partial_response", fullResponse),
						zap.Error(ctx.Err()))
					break
				}
				return fullResponse, fmt.Errorf("stream receive timeout: %w", ctx.Err())
			}
			// 检查是否是网络超时错误
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
				// 如果已经收到部分响应，返回已收到的内容
				if fullResponse != "" {
					logger.Warn("Coze stream network timeout but partial response received",
						zap.String("partial_response", fullResponse),
						zap.Error(err))
					break
				}
				return fullResponse, fmt.Errorf("stream network timeout: %w", err)
			}
			return fullResponse, fmt.Errorf("error receiving from stream: %w", err)
		}

		// 更新最后接收时间
		lastReceiveTime = time.Now()

		// 记录第一个数据块到达的时间（首字延迟）
		if firstChunkTime.IsZero() && event != nil {
			firstChunkTime = time.Now()
			firstTokenLatency := firstChunkTime.Sub(requestStartTime)
			logger.Debug("Coze first token received (stream)",
				zap.Duration("first_token_latency_ms", firstTokenLatency))
		}

		// 处理事件
		// Coze SDK 的事件类型是 ChatEvent
		if event.Event == coze.ChatEventConversationMessageDelta {
			// 增量消息
			if event.Message != nil && event.Message.Content != "" {
				content := event.Message.Content
				buffer += content
				fullResponse += content

				// 通过回调发送增量内容
				if callback != nil {
					if err := callback(content, false); err != nil {
						logger.Error("Failed to process stream segment", zap.Error(err))
					}
				}
			}
		} else if event.Event == coze.ChatEventConversationMessageCompleted {
			// 消息完成
			if event.Message != nil && event.Message.Content != "" {
				content := event.Message.Content
				if !strings.Contains(fullResponse, content) {
					fullResponse += content
					if callback != nil {
						if err := callback(content, false); err != nil {
							logger.Error("Failed to process final stream segment", zap.Error(err))
						}
					}
				}
			}
			if callback != nil {
				if err := callback("", true); err != nil {
					logger.Error("Failed to send completion signal", zap.Error(err))
				}
			}
			break
		} else if event.IsDone() {
			// 流结束
			if callback != nil {
				if err := callback("", true); err != nil {
					logger.Error("Failed to send completion signal", zap.Error(err))
				}
			}
			break
		}
	}

	// 更新消息历史（从流式响应中获取完整消息）
	// 注意：Coze 流式响应可能不包含完整消息历史，需要单独获取
	p.mutex.Lock()
	// 添加助手响应到历史
	p.messages = append(p.messages, coze.Message{
		Role:    "assistant",
		Content: fullResponse,
	})
	p.mutex.Unlock()

	// 记录使用统计
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// 记录性能指标
	if !firstChunkTime.IsZero() {
		firstTokenLatency := firstChunkTime.Sub(requestStartTime).Milliseconds()
		totalLatency := endTime.Sub(requestStartTime).Milliseconds()
		logger.Info("Coze stream request completed",
			zap.Duration("total_duration_ms", time.Duration(duration)*time.Millisecond),
			zap.Duration("first_token_latency_ms", time.Duration(firstTokenLatency)*time.Millisecond),
			zap.Duration("total_latency_ms", time.Duration(totalLatency)*time.Millisecond),
			zap.Int("response_length", len(fullResponse)),
			zap.Int("message_count", len(messagesToSend)))
	}

	promptTokens := estimateTokens(text)
	completionTokens := estimateTokens(fullResponse)
	totalTokens := promptTokens + completionTokens

	p.lastUsage = Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
	}
	p.lastUsageValid = true

	// 发送使用统计信号
	usageInfo := &LLMUsageInfo{
		Model:            options.Model,
		Temperature:      options.Temperature,
		Stream:           true,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		SystemPrompt:     p.systemMsg,
		MessageCount:     len(p.messages),
		StartTime:        startTime,
		EndTime:          endTime,
		Duration:         duration,
		UserID:           options.UserID,
		AssistantID:      options.AssistantID,
		CredentialID:     options.CredentialID,
		SessionID:        options.SessionID,
		ChatType:         options.ChatType,
	}

	utils.Sig().Emit(constants.LLMUsage, usageInfo, text, fullResponse)

	return fullResponse, nil
}

// RegisterFunctionTool 注册函数工具
// 注意：Coze 的工具调用方式可能与 OpenAI 不同，这里提供基本实现
func (p *CozeProvider) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	var params json.RawMessage
	if parameters != nil {
		if raw, ok := parameters.(json.RawMessage); ok {
			params = raw
		} else {
			bytes, _ := json.Marshal(parameters)
			params = json.RawMessage(bytes)
		}
	}
	p.functionManager.RegisterTool(name, description, params, callback)
}

// RegisterFunctionToolDefinition 通过定义结构注册工具
func (p *CozeProvider) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	p.functionManager.RegisterToolDefinition(def)
}

// GetFunctionTools 获取所有可用的函数工具
func (p *CozeProvider) GetFunctionTools() []interface{} {
	// Coze 的工具格式可能不同，这里返回空列表
	// 实际使用时需要根据 Coze 的 API 格式进行转换
	return []interface{}{}
}

// ListFunctionTools 列出所有已注册的工具名称
func (p *CozeProvider) ListFunctionTools() []string {
	return p.functionManager.ListTools()
}

// GetLastUsage 获取最后一次调用的使用统计信息
func (p *CozeProvider) GetLastUsage() (Usage, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.lastUsage, p.lastUsageValid
}

// ResetMessages 重置对话历史
func (p *CozeProvider) ResetMessages() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = make([]coze.Message, 0)
}

// SetSystemPrompt 设置系统提示词
func (p *CozeProvider) SetSystemPrompt(systemPrompt string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.systemMsg = systemPrompt
	// 注意：Coze 的系统提示词通常在 Bot 配置中设置
}

// GetMessages 获取当前对话历史
func (p *CozeProvider) GetMessages() []Message {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	messages := make([]Message, len(p.messages))
	for i, msg := range p.messages {
		messages[i] = Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	return messages
}

// Interrupt 中断当前请求
func (p *CozeProvider) Interrupt() {
	select {
	case p.interruptCh <- struct{}{}:
	default:
	}
}

// Hangup 挂断（清理资源）
func (p *CozeProvider) Hangup() {
	close(p.hangupChan)
}

// estimateTokens 估算 token 数量（简单方法）
func estimateTokens(text string) int {
	// 简单的估算：中文字符按 2 tokens 计算，英文单词按 1.3 tokens 计算
	// 这是一个粗略的估算，实际应该使用更精确的方法
	chineseChars := 0
	englishWords := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseChars++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishWords++
		}
	}

	// 简单估算
	return chineseChars*2 + englishWords/4
}
