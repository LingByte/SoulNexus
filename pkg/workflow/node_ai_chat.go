package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
)

// AIChatNode 代表工作流中的AI对话节点
type AIChatNode struct {
	*Node
	Config *AIChatConfig
}

// AIChatConfig AI对话节点的配置
type AIChatConfig struct {
	// LLM提供者类型: openai, coze, ollama
	Provider string `json:"provider"`
	// API密钥
	APIKey string `json:"apiKey"`
	// API基础URL（可选，某些提供者需要）
	BaseURL string `json:"baseURL"`
	// 模型名称
	Model string `json:"model"`
	// 系统提示词
	SystemPrompt string `json:"systemPrompt"`
	// 用户输入的变量名（从工作流上下文中获取）
	InputVariable string `json:"inputVariable"`
	// 输出变量名（保存到工作流上下文）
	OutputVariable string `json:"outputVariable"`
	// 温度参数 (0-2)
	Temperature *float64 `json:"temperature,omitempty"`
	// 最大token数
	MaxTokens *int `json:"maxTokens,omitempty"`
	// 是否启用流式输出
	EnableStream bool `json:"enableStream"`
	// 对话历史是否持久化
	PersistHistory bool `json:"persistHistory"`
}

// Base 返回基础节点
func (a *AIChatNode) Base() *Node {
	return a.Node
}

// Run 执行AI对话节点
func (a *AIChatNode) Run(ctx *WorkflowContext) ([]string, error) {
	// 设置节点状态为运行中
	ctx.SetNodeStatus(a.ID, NodeStatusRunning, nil)

	// 准备输入
	inputs, err := a.PrepareInputs(ctx)
	if err != nil {
		ctx.SetNodeStatus(a.ID, NodeStatusFailed, err)
		return nil, fmt.Errorf("prepare inputs failed: %w", err)
	}

	// 执行AI对话
	outputs, err := a.executeAIChat(ctx, inputs)
	if err != nil {
		ctx.SetNodeStatus(a.ID, NodeStatusFailed, err)
		return nil, fmt.Errorf("execute AI chat failed: %w", err)
	}

	// 保存输出
	a.PersistOutputs(ctx, outputs)

	// 设置节点状态为完成
	ctx.SetNodeStatus(a.ID, NodeStatusCompleted, nil)

	// 返回下一个节点ID
	return a.NextNodes, nil
}

// executeAIChat 执行实际的AI对话逻辑
func (a *AIChatNode) executeAIChat(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	// 获取用户输入
	userInput := ""
	if a.Config.InputVariable != "" {
		if val, ok := inputs[a.Config.InputVariable]; ok {
			userInput = fmt.Sprintf("%v", val)
		}
	}

	if userInput == "" {
		return nil, fmt.Errorf("input variable '%s' not found or empty", a.Config.InputVariable)
	}

	// 创建LLM提供者
	provider, err := llm.NewLLMProvider(
		context.Background(),
		a.Config.Provider,
		a.Config.APIKey,
		a.Config.BaseURL,
		a.Config.SystemPrompt,
	)
	if err != nil {
		return nil, fmt.Errorf("create LLM provider failed: %w", err)
	}
	defer provider.Hangup()

	// 构建查询选项
	options := llm.QueryOptions{
		Model: a.Config.Model,
	}

	if a.Config.Temperature != nil {
		temp := float32(*a.Config.Temperature)
		options.Temperature = &temp
	}

	if a.Config.MaxTokens != nil {
		options.MaxTokens = a.Config.MaxTokens
	}

	// 执行查询
	var response string
	if a.Config.EnableStream {
		// 流式输出
		var fullResponse strings.Builder
		response, err = provider.QueryStream(userInput, options, func(segment string, isComplete bool) error {
			fullResponse.WriteString(segment)
			// 可以在这里添加流式处理的日志
			ctx.AddLog("info", fmt.Sprintf("Stream segment: %s", segment), a.ID, a.Name)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("query LLM failed: %w", err)
		}
	} else {
		// 非流式输出
		response, err = provider.QueryWithOptions(userInput, options)
		if err != nil {
			return nil, fmt.Errorf("query LLM failed: %w", err)
		}
	}

	// 获取使用统计信息
	usage, hasUsage := provider.GetLastUsage()
	if hasUsage {
		ctx.AddLog("info", fmt.Sprintf("LLM Usage - Prompt: %d, Completion: %d, Total: %d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens), a.ID, a.Name)
	}

	// 构建输出
	outputs := map[string]interface{}{
		a.Config.OutputVariable: response,
	}

	// 如果启用了历史持久化，保存对话历史
	if a.Config.PersistHistory {
		messages := provider.GetMessages()
		messagesJSON, _ := json.Marshal(messages)
		outputs["_chat_history"] = string(messagesJSON)
	}

	return outputs, nil
}

// NewAIChatNode 创建一个新的AI对话节点
func NewAIChatNode(node *Node, config *AIChatConfig) *AIChatNode {
	return &AIChatNode{
		Node:   node,
		Config: config,
	}
}

// ParseAIChatConfig 从节点属性中解析AI对话配置
func ParseAIChatConfig(properties map[string]interface{}) (*AIChatConfig, error) {
	configJSON, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("marshal properties failed: %w", err)
	}

	var config AIChatConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	// 验证必需字段
	if config.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if config.InputVariable == "" {
		return nil, fmt.Errorf("inputVariable is required")
	}
	if config.OutputVariable == "" {
		return nil, fmt.Errorf("outputVariable is required")
	}

	return &config, nil
}
