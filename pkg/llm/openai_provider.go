package llm

import (
	"context"
	"encoding/json"
)

// OpenAIProvider 包装现有的 LLMHandler，实现 LLMProvider 接口
type OpenAIProvider struct {
	handler *LLMHandler
}

// NewOpenAIProvider 创建 OpenAI 提供者
func NewOpenAIProvider(ctx context.Context, apiKey, baseURL, systemPrompt string) *OpenAIProvider {
	return &OpenAIProvider{
		handler: NewLLMHandler(ctx, apiKey, baseURL, systemPrompt),
	}
}

// Query 执行非流式查询
func (p *OpenAIProvider) Query(text, model string) (string, error) {
	return p.handler.Query(text, model)
}

// QueryWithOptions 执行带完整参数的非流式查询
func (p *OpenAIProvider) QueryWithOptions(text string, options QueryOptions) (string, error) {
	return p.handler.QueryWithOptions(text, options)
}

// QueryStream 执行流式查询
func (p *OpenAIProvider) QueryStream(text string, options QueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	return p.handler.QueryStream(text, options, callback)
}

// RegisterFunctionTool 注册函数工具
func (p *OpenAIProvider) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
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
func (p *OpenAIProvider) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	p.handler.RegisterFunctionToolDefinition(def)
}

// GetFunctionTools 获取所有可用的函数工具
func (p *OpenAIProvider) GetFunctionTools() []interface{} {
	tools := p.handler.GetFunctionTools()
	result := make([]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = tool
	}
	return result
}

// ListFunctionTools 列出所有已注册的工具名称
func (p *OpenAIProvider) ListFunctionTools() []string {
	return p.handler.ListFunctionTools()
}

// GetLastUsage 获取最后一次调用的使用统计信息
func (p *OpenAIProvider) GetLastUsage() (Usage, bool) {
	usage, valid := p.handler.GetLastUsage()
	if !valid {
		return Usage{}, false
	}
	return Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}, true
}

// ResetMessages 重置对话历史
func (p *OpenAIProvider) ResetMessages() {
	p.handler.ResetMessages()
}

// SetSystemPrompt 设置系统提示词
func (p *OpenAIProvider) SetSystemPrompt(systemPrompt string) {
	p.handler.SetSystemPrompt(systemPrompt)
}

// SetModel 设置模型
func (p *OpenAIProvider) SetModel(model string) {
	p.handler.SetModel(model)
}

// GetMessages 获取当前对话历史
func (p *OpenAIProvider) GetMessages() []Message {
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
func (p *OpenAIProvider) Interrupt() {
	select {
	case p.handler.interruptCh <- struct{}{}:
	default:
	}
}

// Hangup 挂断（清理资源）
func (p *OpenAIProvider) Hangup() {
	close(p.handler.hangupChan)
}
