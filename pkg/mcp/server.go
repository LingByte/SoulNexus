package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// ServerOption 是 server 包的 Option 类型别名
type ServerOption = server.ServerOption

// MCPServer 封装 MCP 服务器，提供工具注册功能
type MCPServer struct {
	server  *server.MCPServer
	logger  *zap.Logger
	tools   map[string]ToolHandler
	version string
	name    string
}

// Config MCP 服务器配置
type Config struct {
	Name                   string // 服务器名称，默认 "LingEcho/mcp"
	Version                string // 版本号，默认 "1.0.0"
	Logger                 *zap.Logger
	EnableLogging          bool // 是否启用 MCP 服务器日志，默认 true
	EnableToolCapabilities bool // 是否启用工具能力，默认 true
}

// NewMCPServer 创建新的 MCP 服务器实例
func NewMCPServer(cfg *Config) *MCPServer {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.Name == "" {
		cfg.Name = "LingEcho/mcp"
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if !cfg.EnableLogging {
		cfg.EnableLogging = true
	}
	if !cfg.EnableToolCapabilities {
		cfg.EnableToolCapabilities = true
	}

	// 创建 MCP 服务器选项
	opts := []ServerOption{
		server.WithToolCapabilities(cfg.EnableToolCapabilities),
		server.WithHooks(getHooks(cfg.Logger)),
	}

	if cfg.EnableLogging {
		opts = append(opts, server.WithLogging())
	}

	mcpServer := server.NewMCPServer(
		cfg.Name,
		cfg.Version,
		opts...,
	)

	return &MCPServer{
		server:  mcpServer,
		logger:  cfg.Logger,
		tools:   make(map[string]ToolHandler),
		version: cfg.Version,
		name:    cfg.Name,
	}
}

// GetServer 获取底层的 MCP 服务器实例
func (s *MCPServer) GetServer() *server.MCPServer {
	return s.server
}

// RegisterTool 注册一个工具
// name: 工具名称
// description: 工具描述
// handler: 工具处理函数
// params: 工具参数定义（使用 mcp.WithString, mcp.WithNumber 等）
func (s *MCPServer) RegisterTool(
	name string,
	description string,
	handler ToolHandler,
	params ...mcp.ToolOption,
) {
	// 保存处理器
	s.tools[name] = handler

	// 创建工具定义，合并描述和参数选项
	options := []mcp.ToolOption{
		mcp.WithDescription(description),
	}
	options = append(options, params...)
	tool := mcp.NewTool(name, options...)

	// 使用 SafeToolHandler 包装处理器，转换为 server.ToolHandlerFunc
	wrappedHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return SafeToolHandler(name, s.logger, handler)(ctx, request)
	}

	// 注册工具
	s.server.AddTool(tool, wrappedHandler)

	s.logger.Info("MCP 工具已注册",
		zap.String("name", name),
		zap.String("description", description),
	)
}

// RegisterToolWithSchema 使用自定义 schema 注册工具
// 注意：如果 mcp-go 库支持自定义 schema，可以使用此方法
func (s *MCPServer) RegisterToolWithSchema(
	name string,
	description string,
	handler ToolHandler,
	params ...mcp.ToolOption,
) {
	// 保存处理器
	s.tools[name] = handler

	// 创建工具定义
	options := []mcp.ToolOption{
		mcp.WithDescription(description),
	}
	options = append(options, params...)
	tool := mcp.NewTool(name, options...)

	// 使用 SafeToolHandler 包装处理器，转换为 server.ToolHandlerFunc
	wrappedHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return SafeToolHandler(name, s.logger, handler)(ctx, request)
	}

	// 注册工具
	s.server.AddTool(tool, wrappedHandler)

	s.logger.Info("MCP 工具已注册（自定义 schema）",
		zap.String("name", name),
		zap.String("description", description),
	)
}

// GetRegisteredTools 获取所有已注册的工具名称
func (s *MCPServer) GetRegisteredTools() []string {
	tools := make([]string, 0, len(s.tools))
	for name := range s.tools {
		tools = append(tools, name)
	}
	return tools
}

// CallToolInternal 内部调用工具（用于Agent系统）
func (s *MCPServer) CallToolInternal(ctx context.Context, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	handler, exists := s.tools[toolName]
	if !exists {
		return ErrorResponse(404, fmt.Sprintf("Tool %s not found", toolName)), nil
	}

	// 构建MCP请求
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}

	// 调用工具处理器
	result, err := SafeToolHandler(toolName, s.logger, handler)(ctx, request)
	if err != nil {
		return ErrorResponse(500, fmt.Sprintf("Tool execution failed: %v", err)), nil
	}

	return result, nil
}

// getHooks 创建 MCP 服务器钩子
func getHooks(logger *zap.Logger) *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		logger.Debug("工具调用完成",
			zap.Any("id", id),
			zap.String("tool", message.Params.Name),
		)
	})

	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		logger.Debug("工具调用开始",
			zap.Any("id", id),
			zap.String("tool", message.Params.Name),
		)
	})

	return hooks
}
