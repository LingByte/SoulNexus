package providers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/protocol"
	"go.uber.org/zap"
)

// LLMFunctionToolCallback receives parsed tool args; owner is the ChatLLM session.
type LLMFunctionToolCallback func(args map[string]interface{}, owner interface{}) (string, error)

// LLMFunctionToolDefinition describes one registered function tool.
type LLMFunctionToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Callback    LLMFunctionToolCallback
}

type llmFunctionToolManager struct {
	tools map[string]*LLMFunctionToolDefinition
	owner interface{}
}

func newLLMFunctionToolManager() *llmFunctionToolManager {
	return &llmFunctionToolManager{tools: make(map[string]*LLMFunctionToolDefinition)}
}

func (m *llmFunctionToolManager) setOwner(owner interface{}) { m.owner = owner }

func (m *llmFunctionToolManager) registerTool(name, description string, parameters json.RawMessage, callback LLMFunctionToolCallback) {
	if strings.TrimSpace(name) == "" {
		logger.Warn("function tool registration skipped", zap.Error(utils.ErrEmptyToolName))
		return
	}
	if _, exists := m.tools[name]; exists {
		logger.Warn("function tool already registered", zap.String("tool", name), zap.Error(utils.ErrToolAlreadyRegistered))
		return
	}
	m.tools[name] = &LLMFunctionToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Callback:    callback,
	}
	logger.Info("Function tool registered", zap.String("tool", name))
}

func (m *llmFunctionToolManager) registerToolDefinition(def *LLMFunctionToolDefinition) {
	m.tools[def.Name] = def
	logger.Info("Function tool registered", zap.String("tool", def.Name))
}

func (m *llmFunctionToolManager) protocolTools() []protocol.Tool {
	tools := make([]protocol.Tool, 0, len(m.tools))
	for _, def := range m.tools {
		params := map[string]interface{}{"type": "object"}
		if len(def.Parameters) > 0 {
			_ = json.Unmarshal(def.Parameters, &params)
		}
		tools = append(tools, protocol.Tool{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  params,
		})
	}
	return tools
}

func (m *llmFunctionToolManager) executeProtocolTool(toolName string, args json.RawMessage) (string, error) {
	def, ok := m.tools[toolName]
	if !ok {
		return "", fmt.Errorf("%w: %s", utils.ErrToolNotFound, toolName)
	}
	var parsed map[string]interface{}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("%w: %v", utils.ErrInvalidToolParams, err)
		}
	}
	result, err := def.Callback(parsed, m.owner)
	if err != nil {
		logger.Error("Tool call failed", zap.String("tool", toolName), zap.Error(err))
		return "", fmt.Errorf("%w: %v", utils.ErrToolExecutionFailed, err)
	}
	logger.Info("Tool call completed successfully",
		zap.String("tool", toolName),
		zap.String("result", truncateToolResultLog(result, 400)),
	)
	return result, nil
}

func truncateToolResultLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("…(%d bytes)", len(s))
}

func (m *llmFunctionToolManager) listTools() []string {
	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}
