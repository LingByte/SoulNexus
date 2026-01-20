package agent

import (
	"context"
	"fmt"
	"time"

	lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// ToolAgent 工具调用Agent，负责调用MCP工具
type ToolAgent struct {
	id          string
	name        string
	description string
	mcpServer   *lingechoMCP.MCPServer
	logger      *zap.Logger
}

// NewToolAgent 创建新的工具Agent
func NewToolAgent(mcpServer *lingechoMCP.MCPServer, logger *zap.Logger) *ToolAgent {
	return &ToolAgent{
		id:          "tool_agent",
		name:        "Tool Agent",
		description: "工具调用Agent，负责调用MCP工具执行各种操作",
		mcpServer:   mcpServer,
		logger:      logger,
	}
}

// ID 返回agent ID
func (a *ToolAgent) ID() string {
	return a.id
}

// Name 返回agent名称
func (a *ToolAgent) Name() string {
	return a.name
}

// Description 返回agent描述
func (a *ToolAgent) Description() string {
	return a.description
}

// Capabilities 返回agent能力
func (a *ToolAgent) Capabilities() []Capability {
	// 获取所有已注册的工具
	tools := a.mcpServer.GetRegisteredTools()
	caps := make([]Capability, 0, len(tools))

	for _, toolName := range tools {
		caps = append(caps, Capability{
			Name:        toolName,
			Description: fmt.Sprintf("调用MCP工具: %s", toolName),
			Type:        "tool_call",
		})
	}

	return caps
}

// CanHandle 判断是否能处理任务
func (a *ToolAgent) CanHandle(request *TaskRequest) bool {
	return request.Type == TaskTypeToolCall ||
		(request.Parameters != nil && request.Parameters["tool"] != nil)
}

// Process 处理任务
func (a *ToolAgent) Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	startTime := time.Now()

	// 获取工具名称
	toolName, ok := request.Parameters["tool"].(string)
	if !ok {
		return &TaskResponse{
			ID:        request.ID,
			Success:   false,
			Error:     "Tool name is required in parameters",
			CreatedAt: time.Now(),
		}, nil
	}

	// 获取工具参数
	toolArgs := make(map[string]interface{})
	if args, ok := request.Parameters["arguments"].(map[string]interface{}); ok {
		toolArgs = args
	} else if request.Parameters != nil {
		// 如果没有单独的arguments，使用整个parameters（排除tool字段）
		for k, v := range request.Parameters {
			if k != "tool" {
				toolArgs[k] = v
			}
		}
	}

	// 通过MCP服务器调用工具
	a.logger.Info("Tool call requested",
		zap.String("taskID", request.ID),
		zap.String("tool", toolName),
		zap.Any("arguments", toolArgs),
	)

	// 调用MCP工具
	result, err := a.mcpServer.CallToolInternal(ctx, toolName, toolArgs)
	if err != nil {
		a.logger.Error("Tool call failed",
			zap.String("tool", toolName),
			zap.Error(err),
		)
		return &TaskResponse{
			ID:        request.ID,
			Success:   false,
			Error:     fmt.Sprintf("Tool call failed: %v", err),
			CreatedAt: time.Now(),
		}, nil
	}

	// 提取结果文本
	resultText := ""
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			resultText += textContent.Text + "\n"
		}
	}

	return &TaskResponse{
		ID:      request.ID,
		Success: true,
		Content: resultText,
		Data: map[string]interface{}{
			"tool":   toolName,
			"result": result,
		},
		AgentID:        a.id,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
	}, nil
}

// Health 健康检查
func (a *ToolAgent) Health(ctx context.Context) error {
	if a.mcpServer == nil {
		return fmt.Errorf("MCP server is nil")
	}

	// 检查是否有注册的工具
	tools := a.mcpServer.GetRegisteredTools()
	if len(tools) == 0 {
		return fmt.Errorf("no tools registered")
	}

	return nil
}
