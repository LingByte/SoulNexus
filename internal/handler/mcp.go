package handlers

import (
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/service"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// MCPHandler MCP 处理器
type MCPHandler struct {
	manager *service.MCPManager
}

// NewMCPHandler 创建 MCP 处理器
func NewMCPHandler(manager *service.MCPManager) *MCPHandler {
	return &MCPHandler{manager: manager}
}

// ListMCPServers 列出 MCP 服务器
// GET /api/mcp/servers
func (h *MCPHandler) ListMCPServers(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	servers, err := h.manager.ListMCPServers(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", servers)
}

// GetMCPServer 获取 MCP 服务器详情
// GET /api/mcp/servers/:id
func (h *MCPHandler) GetMCPServer(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	server, err := h.manager.GetMCPServer(c.Request.Context(), user.ID, uint(serverID))
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", server)
}

// CreateMCPServer 创建 MCP 服务器
// POST /api/mcp/servers
func (h *MCPHandler) CreateMCPServer(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	var req models.CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	server, err := h.manager.InstallMCP(c.Request.Context(), user.ID, &req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "创建成功", server)
}

// UpdateMCPServer 更新 MCP 服务器配置
// PATCH /api/mcp/servers/:id
func (h *MCPHandler) UpdateMCPServer(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	var req models.UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	server, err := h.manager.UpdateMCPConfig(c.Request.Context(), user.ID, uint(serverID), &req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "更新成功", server)
}

// DeleteMCPServer 删除 MCP 服务器
// DELETE /api/mcp/servers/:id
func (h *MCPHandler) DeleteMCPServer(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	if err := h.manager.UninstallMCP(c.Request.Context(), user.ID, uint(serverID)); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "删除成功", nil)
}

// EnableMCPServer 启用 MCP 服务器
// POST /api/mcp/servers/:id/enable
func (h *MCPHandler) EnableMCPServer(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	if err := h.manager.EnableMCP(c.Request.Context(), user.ID, uint(serverID)); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "启用成功", nil)
}

// DisableMCPServer 禁用 MCP 服务器
// POST /api/mcp/servers/:id/disable
func (h *MCPHandler) DisableMCPServer(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	if err := h.manager.DisableMCP(c.Request.Context(), user.ID, uint(serverID)); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "禁用成功", nil)
}

// GetMCPTools 获取 MCP 工具列表
// GET /api/mcp/servers/:id/tools
func (h *MCPHandler) GetMCPTools(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	tools, err := h.manager.GetMCPTools(c.Request.Context(), user.ID, uint(serverID))
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", tools)
}

// CallMCPTool 调用 MCP 工具
// POST /api/mcp/servers/:id/call-tool
func (h *MCPHandler) CallMCPTool(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	var req models.CallMCPToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	result, err := h.manager.CallMCPTool(c.Request.Context(), user.ID, uint(serverID), &req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "调用成功", result)
}

// GetMCPLogs 获取 MCP 调用日志
// GET /api/mcp/servers/:id/logs
func (h *MCPHandler) GetMCPLogs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的服务器 ID", nil)
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	logs, err := h.manager.GetMCPLogs(c.Request.Context(), user.ID, uint(serverID), limit)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", logs)
}
