package llm

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/logger"
	"go.uber.org/zap"
)

// OllamaProvider Ollama LLM 提供者实现
// Ollama 使用 OpenAI 兼容的 API，所以可以复用 OpenAI 的实现
type OllamaProvider struct {
	handler *LLMHandler
	baseURL string
}

// NewOllamaProvider 创建 Ollama 提供者
// apiKey: Ollama 通常不需要 API Key，但为了兼容性可以传入空字符串
// baseURL: Ollama 服务的基础 URL，默认为 http://localhost:11434/v1
// systemPrompt: 系统提示词
func NewOllamaProvider(ctx context.Context, apiKey, baseURL, systemPrompt string) *OllamaProvider {
	// 如果 baseURL 为空，使用 Ollama 默认地址
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	} else {
		// 确保 baseURL 以 /v1 结尾（Ollama 的 OpenAI 兼容端点）
		baseURL = strings.TrimSuffix(baseURL, "/")
		if !strings.HasSuffix(baseURL, "/v1") {
			// 如果 URL 不包含 /v1，添加它
			if !strings.Contains(baseURL, "/v1") {
				baseURL = baseURL + "/v1"
			}
		}
	}

	logger.Info("Creating Ollama provider", zap.String("baseURL", baseURL))

	return &OllamaProvider{
		handler: NewLLMHandler(ctx, apiKey, baseURL, systemPrompt),
		baseURL: baseURL,
	}
}

// Query 执行非流式查询
func (p *OllamaProvider) Query(text, model string) (string, error) {
	return p.handler.Query(text, model)
}

// QueryWithOptions 执行带完整参数的非流式查询
func (p *OllamaProvider) QueryWithOptions(text string, options QueryOptions) (string, error) {
	return p.handler.QueryWithOptions(text, options)
}

// QueryStream 执行流式查询
func (p *OllamaProvider) QueryStream(text string, options QueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	return p.handler.QueryStream(text, options, callback)
}

// RegisterFunctionTool 注册函数工具
func (p *OllamaProvider) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	var params json.RawMessage
	if parameters != nil {
		if raw, ok := parameters.(json.RawMessage); ok {
			params = raw
		} else {
			bytes, _ := json.Marshal(parameters)
			params = json.RawMessage(bytes)
		}
	}
	p.handler.RegisterFunctionTool(name, description, params, callback)
}

// RegisterFunctionToolDefinition 通过定义结构注册工具
func (p *OllamaProvider) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	p.handler.RegisterFunctionToolDefinition(def)
}

// GetFunctionTools 获取所有可用的函数工具
func (p *OllamaProvider) GetFunctionTools() []interface{} {
	tools := p.handler.GetFunctionTools()
	result := make([]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = tool
	}
	return result
}

// ListFunctionTools 列出所有已注册的工具名称
func (p *OllamaProvider) ListFunctionTools() []string {
	return p.handler.ListFunctionTools()
}

// ResetMessages 重置对话历史
func (p *OllamaProvider) ResetMessages() {
	p.handler.ResetMessages()
}

// SetSystemPrompt 设置系统提示词
func (p *OllamaProvider) SetSystemPrompt(systemPrompt string) {
	p.handler.SetSystemPrompt(systemPrompt)
}

// GetMessages 获取当前对话历史
func (p *OllamaProvider) GetMessages() []Message {
	openaiMessages := p.handler.GetMessages()
	messages := make([]Message, len(openaiMessages))
	for i, msg := range openaiMessages {
		messages[i] = Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
		// 转换 ToolCalls
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			messages[i].ToolCalls = toolCalls
		}
	}
	return messages
}

// Interrupt 中断当前请求
func (p *OllamaProvider) Interrupt() {
	select {
	case p.handler.interruptCh <- struct{}{}:
	default:
	}
}

// Hangup 挂断（清理资源）
func (p *OllamaProvider) Hangup() {
	close(p.handler.hangupChan)
}
