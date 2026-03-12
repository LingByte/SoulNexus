package service

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MCPManager MCP 管理服务
type MCPManager struct {
	db *gorm.DB
}

// NewMCPManager 创建 MCP 管理服务
func NewMCPManager(db *gorm.DB) *MCPManager {
	return &MCPManager{db: db}
}

// ListMCPServers 列出用户的 MCP 服务器
func (m *MCPManager) ListMCPServers(ctx context.Context, userID uint) ([]models.MCPServer, error) {
	var servers []models.MCPServer
	if err := m.db.WithContext(ctx).Where("user_id = ?", userID).Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// GetMCPServer 获取 MCP 服务器详情
func (m *MCPManager) GetMCPServer(ctx context.Context, userID uint, serverID uint) (*models.MCPServer, error) {
	var server models.MCPServer
	if err := m.db.WithContext(ctx).Where("id = ? AND user_id = ?", serverID, userID).First(&server).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// InstallMCP 安装 MCP 服务器
func (m *MCPManager) InstallMCP(ctx context.Context, userID uint, req *models.CreateMCPServerRequest) (*models.MCPServer, error) {
	// 转换参数
	var args datatypes.JSON
	if len(req.Args) > 0 {
		data, _ := json.Marshal(req.Args)
		args = data
	}

	var env datatypes.JSON
	if len(req.Env) > 0 {
		data, _ := json.Marshal(req.Env)
		env = data
	}

	var config datatypes.JSON
	if len(req.Config) > 0 {
		data, _ := json.Marshal(req.Config)
		config = data
	}

	server := &models.MCPServer{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Command:     req.Command,
		Args:        args,
		Env:         env,
		Config:      config,
		Protocol:    req.Protocol,
		Host:        req.Host,
		Port:        req.Port,
		Status:      "active",
		Enabled:     true,
	}

	if err := m.db.WithContext(ctx).Create(server).Error; err != nil {
		return nil, err
	}

	return server, nil
}

// UpdateMCPConfig 更新 MCP 配置
func (m *MCPManager) UpdateMCPConfig(ctx context.Context, userID uint, serverID uint, req *models.UpdateMCPServerRequest) (*models.MCPServer, error) {
	server, err := m.GetMCPServer(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if len(req.Config) > 0 {
		data, _ := json.Marshal(req.Config)
		updates["config"] = data
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if err := m.db.WithContext(ctx).Model(server).Updates(updates).Error; err != nil {
		return nil, err
	}

	return server, nil
}

// UninstallMCP 卸载 MCP 服务器
func (m *MCPManager) UninstallMCP(ctx context.Context, userID uint, serverID uint) error {
	return m.db.WithContext(ctx).Where("id = ? AND user_id = ?", serverID, userID).Delete(&models.MCPServer{}).Error
}

// EnableMCP 启用 MCP 服务器
func (m *MCPManager) EnableMCP(ctx context.Context, userID uint, serverID uint) error {
	return m.db.WithContext(ctx).
		Model(&models.MCPServer{}).
		Where("id = ? AND user_id = ?", serverID, userID).
		Update("enabled", true).Error
}

// DisableMCP 禁用 MCP 服务器
func (m *MCPManager) DisableMCP(ctx context.Context, userID uint, serverID uint) error {
	return m.db.WithContext(ctx).
		Model(&models.MCPServer{}).
		Where("id = ? AND user_id = ?", serverID, userID).
		Update("enabled", false).Error
}

// GetMCPTools 获取 MCP 工具列表
func (m *MCPManager) GetMCPTools(ctx context.Context, userID uint, serverID uint) ([]models.MCPTool, error) {
	// 首先验证用户权限
	var server models.MCPServer
	if err := m.db.WithContext(ctx).Where("id = ? AND user_id = ?", serverID, userID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("server not found or unauthorized")
	}

	// 如果是远程 HTTP 服务，从远程获取工具列表
	if server.Protocol == "http" && server.Host != "" && server.Port > 0 {
		return m.getRemoteMCPTools(ctx, &server)
	}

	// 否则从数据库获取
	var tools []models.MCPTool
	if err := m.db.WithContext(ctx).Where("server_id = ?", serverID).Find(&tools).Error; err != nil {
		return nil, err
	}

	return tools, nil
}

// getRemoteMCPTools 从远程 MCP 服务获取工具列表
func (m *MCPManager) getRemoteMCPTools(ctx context.Context, server *models.MCPServer) ([]models.MCPTool, error) {
	// 构建请求 URL
	url := fmt.Sprintf("http://%s:%d/tools", server.Host, server.Port)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP service: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP service returned status %d", resp.StatusCode)
	}

	// 解析响应
	var respData struct {
		Code int                      `json:"code"`
		Msg  string                   `json:"msg"`
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查响应代码
	if respData.Code != 200 {
		return nil, fmt.Errorf("MCP service error: %s", respData.Msg)
	}

	// 转换为 MCPTool 对象
	var tools []models.MCPTool
	for _, toolData := range respData.Data {
		tool := models.MCPTool{
			ServerID: server.ID,
		}

		if name, ok := toolData["name"].(string); ok {
			tool.Name = name
		}

		if desc, ok := toolData["description"].(string); ok {
			tool.Description = desc
		}

		if schema, ok := toolData["inputSchema"]; ok {
			schemaBytes, _ := json.Marshal(schema)
			tool.InputSchema = schemaBytes
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// CallMCPTool 调用 MCP 工具
func (m *MCPManager) CallMCPTool(ctx context.Context, userID uint, serverID uint, req *models.CallMCPToolRequest) (*models.CallMCPToolResponse, error) {
	// 验证用户权限
	var server models.MCPServer
	if err := m.db.WithContext(ctx).Where("id = ? AND user_id = ?", serverID, userID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("server not found or unauthorized")
	}

	if !server.Enabled {
		return nil, fmt.Errorf("server is disabled")
	}

	// 记录开始时间
	startTime := time.Now()

	// 调用实际的 MCP 服务
	result, err := m.callRemoteMCPTool(ctx, &server, req)
	if err != nil {
		// 记录错误日志
		log := &models.MCPCallLog{
			UserID:   userID,
			ServerID: serverID,
			ToolName: req.ToolName,
			Status:   "error",
			Error:    err.Error(),
			Duration: time.Since(startTime).Milliseconds(),
		}

		if len(req.Arguments) > 0 {
			data, _ := json.Marshal(req.Arguments)
			log.Arguments = data
		}

		m.db.WithContext(ctx).Create(log)

		// 更新服务器错误计数
		m.db.WithContext(ctx).Model(&server).Updates(map[string]interface{}{
			"error_count": gorm.Expr("error_count + ?", 1),
		})

		return nil, err
	}

	result.Duration = time.Since(startTime).Milliseconds()

	// 记录成功日志
	log := &models.MCPCallLog{
		UserID:   userID,
		ServerID: serverID,
		ToolName: req.ToolName,
		Status:   "success",
		Duration: result.Duration,
	}

	if len(req.Arguments) > 0 {
		data, _ := json.Marshal(req.Arguments)
		log.Arguments = data
	}

	if result.Result != nil {
		data, _ := json.Marshal(result.Result)
		log.Result = data
	}

	m.db.WithContext(ctx).Create(log)

	// 更新服务器统计信息
	m.db.WithContext(ctx).Model(&server).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + ?", 1),
		"last_used_at": time.Now(),
	})

	return result, nil
}

// callRemoteMCPTool 调用远程 MCP 工具
func (m *MCPManager) callRemoteMCPTool(ctx context.Context, server *models.MCPServer, req *models.CallMCPToolRequest) (*models.CallMCPToolResponse, error) {
	// 构建请求 URL
	url := fmt.Sprintf("http://%s:%d/call-tool", server.Host, server.Port)

	// 构建请求体
	reqBody := map[string]interface{}{
		"tool_name": req.ToolName,
		"arguments": req.Arguments,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 设置超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP service: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var respData struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查响应代码
	if respData.Code != 200 {
		return nil, fmt.Errorf("MCP service error: %s", respData.Msg)
	}

	return &models.CallMCPToolResponse{
		Result: respData.Data,
	}, nil
}

// GetMCPLogs 获取 MCP 调用日志
func (m *MCPManager) GetMCPLogs(ctx context.Context, userID uint, serverID uint, limit int) ([]models.MCPCallLog, error) {
	// 验证用户权限
	var server models.MCPServer
	if err := m.db.WithContext(ctx).Where("id = ? AND user_id = ?", serverID, userID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("server not found or unauthorized")
	}

	var logs []models.MCPCallLog
	if err := m.db.WithContext(ctx).
		Where("server_id = ?", serverID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// GetMCPServerStats 获取 MCP 服务器统计信息
func (m *MCPManager) GetMCPServerStats(ctx context.Context, userID uint, serverID uint) (map[string]interface{}, error) {
	server, err := m.GetMCPServer(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	// 获取工具数量
	var toolCount int64
	m.db.WithContext(ctx).Model(&models.MCPTool{}).Where("server_id = ?", serverID).Count(&toolCount)

	// 获取调用统计
	var successCount, errorCount int64
	m.db.WithContext(ctx).Model(&models.MCPCallLog{}).
		Where("server_id = ?", serverID).
		Where("status = ?", "success").
		Count(&successCount)

	m.db.WithContext(ctx).Model(&models.MCPCallLog{}).
		Where("server_id = ?", serverID).
		Where("status = ?", "error").
		Count(&errorCount)

	return map[string]interface{}{
		"id":              server.ID,
		"name":            server.Name,
		"status":          server.Status,
		"enabled":         server.Enabled,
		"tool_count":      toolCount,
		"usage_count":     server.UsageCount,
		"error_count":     server.ErrorCount,
		"success_count":   successCount,
		"log_error_count": errorCount,
		"last_used_at":    server.LastUsedAt,
	}, nil
}

// TestMCPConnection 测试 MCP 连接
func (m *MCPManager) TestMCPConnection(ctx context.Context, userID uint, serverID uint) (bool, error) {
	server, err := m.GetMCPServer(ctx, userID, serverID)
	if err != nil {
		return false, err
	}

	if !server.Enabled {
		return false, fmt.Errorf("server is disabled")
	}

	// 如果是远程 HTTP 服务，测试连接
	if server.Protocol == "http" && server.Host != "" && server.Port > 0 {
		return m.testRemoteMCPConnection(ctx, server)
	}

	return true, nil
}

// testRemoteMCPConnection 测试远程 MCP 服务连接
func (m *MCPManager) testRemoteMCPConnection(ctx context.Context, server *models.MCPServer) (bool, error) {
	// 构建请求 URL
	url := fmt.Sprintf("http://%s:%d/health", server.Host, server.Port)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置超时
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to connect to MCP service: %w", err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("MCP service returned status %d", resp.StatusCode)
	}

	return true, nil
}
