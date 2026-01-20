package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// RegisterAgentTools 在MCP服务器上注册agent相关工具
func RegisterAgentTools(mcpServer *lingechoMCP.MCPServer, manager *Manager, logger *zap.Logger) {
	// 1. 列出所有agents
	mcpServer.RegisterTool(
		"list_agents",
		"列出所有可用的agents及其状态",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			agents := manager.ListAgents()
			statuses := manager.GetAllAgentStatuses()

			agentList := make([]map[string]interface{}, 0, len(agents))
			for _, agent := range agents {
				status := statuses[agent.ID()]
				agentInfo := map[string]interface{}{
					"id":           agent.ID(),
					"name":         agent.Name(),
					"description":  agent.Description(),
					"capabilities": agent.Capabilities(),
				}
				if status != nil {
					agentInfo["status"] = status.Status
					agentInfo["activeTasks"] = status.ActiveTasks
					agentInfo["health"] = status.Health
				}
				agentList = append(agentList, agentInfo)
			}

			return lingechoMCP.SuccessResponse(map[string]interface{}{
				"agents": agentList,
				"count":  len(agents),
			}), nil
		},
	)

	// 2. 获取agent状态
	mcpServer.RegisterTool(
		"get_agent_status",
		"获取指定agent的状态信息",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			agentID, err := lingechoMCP.SafeGetString(arguments, "agentId", true)
			if err != nil {
				return lingechoMCP.ErrorResponse(400, err.Error()), nil
			}

			status, err := manager.GetAgentStatus(agentID)
			if err != nil {
				return lingechoMCP.ErrorResponse(404, fmt.Sprintf("Agent not found: %s", agentID)), nil
			}

			return lingechoMCP.SuccessResponse(map[string]interface{}{
				"agentId":      status.ID,
				"name":         status.Name,
				"status":       status.Status,
				"activeTasks":  status.ActiveTasks,
				"totalTasks":   status.TotalTasks,
				"lastActivity": status.LastActivity,
				"health":       status.Health,
			}), nil
		},
		mcp.WithString(
			"agentId",
			mcp.Description("Agent ID"),
			mcp.Required(),
		),
	)

	// 3. 处理任务
	mcpServer.RegisterTool(
		"process_task",
		"通过agent系统处理任务",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			taskType, err := lingechoMCP.SafeGetString(arguments, "type", true)
			if err != nil {
				return lingechoMCP.ErrorResponse(400, err.Error()), nil
			}

			content, err := lingechoMCP.SafeGetString(arguments, "content", true)
			if err != nil {
				return lingechoMCP.ErrorResponse(400, err.Error()), nil
			}

			// 解析上下文（可选）
			var taskContext *TaskContext
			if contextStr, ok := arguments["context"].(string); ok && contextStr != "" {
				if err := json.Unmarshal([]byte(contextStr), &taskContext); err != nil {
					logger.Warn("Failed to parse context, using nil", zap.Error(err))
				}
			}

			// 解析参数（可选）
			var parameters map[string]interface{}
			if paramsStr, ok := arguments["parameters"].(string); ok && paramsStr != "" {
				if err := json.Unmarshal([]byte(paramsStr), &parameters); err != nil {
					logger.Warn("Failed to parse parameters, using empty map", zap.Error(err))
					parameters = make(map[string]interface{})
				}
			} else {
				parameters = make(map[string]interface{})
			}

			// 创建任务请求
			request := &TaskRequest{
				ID:         fmt.Sprintf("mcp_task_%d", time.Now().UnixNano()),
				Type:       taskType,
				Content:    content,
				Context:    taskContext,
				Parameters: parameters,
				CreatedAt:  time.Now(),
			}

			// 处理任务
			response, err := manager.Process(context.Background(), request)
			if err != nil {
				return lingechoMCP.ErrorResponse(500, fmt.Sprintf("Task processing failed: %v", err)), nil
			}

			return lingechoMCP.SuccessResponse(map[string]interface{}{
				"taskId":         response.ID,
				"success":        response.Success,
				"content":        response.Content,
				"data":           response.Data,
				"agentId":        response.AgentID,
				"processingTime": response.ProcessingTime.Milliseconds(),
				"error":          response.Error,
			}), nil
		},
		mcp.WithString(
			"type",
			mcp.Description("任务类型 (rag, graph_memory, tool_call, llm, general)"),
			mcp.Required(),
		),
		mcp.WithString(
			"content",
			mcp.Description("任务内容"),
			mcp.Required(),
		),
		mcp.WithString(
			"context",
			mcp.Description("任务上下文（JSON字符串）"),
		),
		mcp.WithString(
			"parameters",
			mcp.Description("任务参数（JSON字符串）"),
		),
	)

	// 4. 健康检查
	mcpServer.RegisterTool(
		"agent_health_check",
		"检查所有agents的健康状态",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			results := manager.HealthCheck(context.Background())

			healthStatus := make(map[string]interface{})
			for agentID, err := range results {
				if err != nil {
					healthStatus[agentID] = map[string]interface{}{
						"healthy": false,
						"error":   err.Error(),
					}
				} else {
					healthStatus[agentID] = map[string]interface{}{
						"healthy": true,
					}
				}
			}

			allHealthy := len(results) == 0
			return lingechoMCP.SuccessResponse(map[string]interface{}{
				"allHealthy": allHealthy,
				"status":     healthStatus,
			}), nil
		},
	)

	logger.Info("Agent tools registered in MCP server")
}
