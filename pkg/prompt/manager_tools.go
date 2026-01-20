package prompt

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// serverProvider interface defines components that can provide server instances
type serverProvider interface {
	// WithContext injects server instance into the context
	withContext(ctx context.Context) context.Context
}

// MethodNameModifier 定义用于在上下文中修改方法名的函数类型。
// 这允许外部组件(如集成版)自定义方法名以进行监控。
type MethodNameModifier func(ctx context.Context, method, toolName string)

// toolManager 构建 MCP（Multi-Modal Communication Protocol）工具调用系统 的核心逻辑模块，目的是实现：
// 在服务器中注册各种「工具（Tool）」
// 客户端通过标准 JSON-RPC 接口调用这些工具
// 支持自动生成参数校验规则（OpenAPI schema）
// 支持工具列表展示、调用执行、错误处理、调用上下文控制等功能
type toolManager struct {
	// Registered tools
	tools map[string]*registeredTool

	// Mutex for concurrent access
	mu sync.RWMutex

	// Server provider for injecting server instance into context
	serverProvider serverProvider

	// Track insertion order of tools
	toolsOrder []string

	// Tool list filter function.
	toolListFilter ToolListFilter

	// Method name modifier for external customization.
	methodNameModifier MethodNameModifier
}

// newToolManager creates a tool manager
func newToolManager() *toolManager {
	return &toolManager{
		tools: make(map[string]*registeredTool),
	}
}

// withServerProvider sets the server provider
func (m *toolManager) withServerProvider(provider serverProvider) *toolManager {
	m.serverProvider = provider
	return m
}

// withToolListFilter sets the tool list filter.
func (m *toolManager) withToolListFilter(filter ToolListFilter) *toolManager {
	m.toolListFilter = filter
	return m
}

// withMethodNameModifier sets the method name modifier.
func (m *toolManager) withMethodNameModifier(modifier MethodNameModifier) *toolManager {
	m.methodNameModifier = modifier
	return m
}

// registerTool registers a tool
func (m *toolManager) registerTool(tool *Tool, handler toolHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tool == nil || tool.Name == "" {
		return
	}

	if _, exists := m.tools[tool.Name]; !exists {
		// Only add to order slice if it's a new tool
		m.toolsOrder = append(m.toolsOrder, tool.Name)
	}

	m.tools[tool.Name] = &registeredTool{
		Tool:    tool,
		Handler: handler,
	}
}

// getTool retrieves a tool by name
func (m *toolManager) getTool(name string) (*Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	registeredTool, ok := m.tools[name]
	if !ok {
		return nil, false
	}
	return registeredTool.Tool, true
}

// getTools gets all registered tools
func (m *toolManager) getTools(protocolVersion string) []*Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]*Tool, 0, len(m.tools))
	for _, registeredTool := range m.tools {
		if registeredTool != nil && registeredTool.Tool != nil {
			tools = append(tools, registeredTool.Tool)
		}
	}

	return tools
}

// handleListTools handles tools/list requests
func (m *toolManager) handleListTools(
	ctx context.Context,
	req *JSONRPCRequest,
	session utils.Session,
) (JSONRPCMessage, error) {
	// Get all tools
	toolPtrs := m.getTools("")

	// Apply filter if available.
	if m.toolListFilter != nil {
		toolPtrs = m.toolListFilter(ctx, toolPtrs)
	}

	// Convert []*mcp.Tool to []mcp.Tool
	tools := make([]Tool, len(toolPtrs))
	for i, toolPtr := range toolPtrs {
		if toolPtr != nil {
			tools[i] = *toolPtr
		}
	}

	// Format and return response
	result := ListToolsResult{
		Tools: tools,
	}

	return result, nil
}

// handleCallTool handles tools/call requests
func (m *toolManager) handleCallTool(
	ctx context.Context,
	req *JSONRPCRequest,
	session utils.Session,
) (JSONRPCMessage, error) {
	// Parse request parameters
	if req.Params == nil {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrMissingParams.Error(), nil), nil
	}

	// Convert params to map for easier access
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrInvalidParams.Error(), nil), nil
	}

	// Get tool name
	toolName, ok := paramsMap["name"].(string)
	if !ok || toolName == "" {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, "missing tool name", nil), nil
	}

	// Get tool
	registeredTool, ok := m.tools[toolName]
	if !ok {
		return newJSONRPCErrorResponse(
			req.ID,
			ErrCodeMethodNotFound,
			fmt.Sprintf("%v: %s", utils.ErrToolNotFound, toolName),
			nil,
		), nil
	}

	// Create tool call request
	toolReq := &CallToolRequest{}
	toolReq.Method = MethodToolsCall // Set method manually

	// Set up CallToolParams
	params := CallToolParams{
		Name: toolName,
	}

	// Get and validate tool arguments
	if args, ok := paramsMap["arguments"]; ok && args != nil {
		argsMap, ok := args.(map[string]interface{})
		if !ok {
			errMsg := fmt.Sprintf("%v: arguments must be an object, got %T", utils.ErrInvalidParams, args)
			return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, errMsg, nil), nil
		}
		params.Arguments = argsMap
	}

	toolReq.Params = params

	// Progress notification token (if any)
	if meta, ok := paramsMap["_meta"].(map[string]interface{}); ok {
		if progressToken, exists := meta["progressToken"]; exists {
			// Note: The current version of CallToolRequest doesn't fully implement Meta field
			// Future implementation should add toolReq.Meta = ... code
			_ = progressToken // Ignore progress token for now
		}
	}

	// Before calling the tool, inject server instance into context if server provider exists
	if m.serverProvider != nil {
		ctx = m.serverProvider.withContext(ctx)
	}

	// Modify method name for monitoring if modifier is available.
	if m.methodNameModifier != nil {
		m.methodNameModifier(ctx, MethodToolsCall, toolName)
	}

	// Execute tool
	result, err := registeredTool.Handler(ctx, toolReq)
	if err != nil {
		errMsg := fmt.Sprintf("tool execution failed (tool: %s): %v", registeredTool.Tool.Name, err)
		return newJSONRPCErrorResponse(req.ID, ErrCodeInternal, errMsg, nil), nil
	}

	return result, nil
}
