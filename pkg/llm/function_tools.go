package llm

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// FunctionToolCallback 定义函数工具回调类型
// 第二个参数 llmService 可用于获取上下文信息（如客户端IP）
type FunctionToolCallback func(args map[string]interface{}, llmService interface{}) (string, error)

// FunctionToolDefinition 定义函数工具的结构
type FunctionToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Callback    FunctionToolCallback
}

// FunctionToolManager 管理所有Function Tools
type FunctionToolManager struct {
	tools      map[string]*FunctionToolDefinition
	llmService interface{} // 引用 LLMService，用于工具回调中获取上下文信息
}

// NewFunctionToolManager 创建新的Function Tool管理器
func NewFunctionToolManager() *FunctionToolManager {
	manager := &FunctionToolManager{
		tools: make(map[string]*FunctionToolDefinition),
	}

	// 注册默认的工具
	manager.registerDefaultTools()

	return manager
}

// SetLLMService 设置 LLMService 引用
func (m *FunctionToolManager) SetLLMService(service interface{}) {
	m.llmService = service
}

// GetLLMService 获取 LLMService 引用
func (m *FunctionToolManager) GetLLMService() interface{} {
	return m.llmService
}

// RegisterTool 注册新的Function Tool
func (m *FunctionToolManager) RegisterTool(name, description string, parameters json.RawMessage, callback FunctionToolCallback) {
	m.tools[name] = &FunctionToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Callback:    callback,
	}
	logger.Info("Function tool registered", zap.String("tool", name))
}

// RegisterToolDefinition 通过定义结构注册工具
func (m *FunctionToolManager) RegisterToolDefinition(def *FunctionToolDefinition) {
	m.tools[def.Name] = def
	logger.Info("Function tool registered", zap.String("tool", def.Name))
}

// GetTools 获取所有可用的Function Tools定义
func (m *FunctionToolManager) GetTools() []openai.Tool {
	tools := make([]openai.Tool, 0, len(m.tools))
	for _, def := range m.tools {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			},
		})
	}
	return tools
}

// HandleToolCall 处理工具调用
func (m *FunctionToolManager) HandleToolCall(toolCall openai.ToolCall) (string, error) {
	def, exists := m.tools[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("unknown function tool: %s", toolCall.Function.Name)
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool call arguments: %w", err)
	}

	// 执行回调（传入 llmService）
	result, err := def.Callback(args, m.llmService)
	if err != nil {
		logger.Error("Tool call failed",
			zap.String("tool", toolCall.Function.Name),
			zap.Error(err))
		return "", err
	}

	logger.Info("Tool call completed successfully",
		zap.String("tool", toolCall.Function.Name),
		zap.String("result", result))

	return result, nil
}

// GetTool 获取指定名称的工具定义
func (m *FunctionToolManager) GetTool(name string) (*FunctionToolDefinition, bool) {
	def, exists := m.tools[name]
	return def, exists
}

// ListTools 列出所有已注册的工具名称
func (m *FunctionToolManager) ListTools() []string {
	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

// registerDefaultTools 注册默认的工具
func (m *FunctionToolManager) registerDefaultTools() {
	// 默认不注册任何工具，工具可以通过RegisterTool或RegisterToolDefinition动态注册
}
