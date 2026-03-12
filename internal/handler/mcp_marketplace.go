package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/service"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// MCPMarketplaceHandler MCP 广场处理器
type MCPMarketplaceHandler struct {
	service *service.MCPMarketplaceService
}

// NewMCPMarketplaceHandler 创建 MCP 广场处理器
func NewMCPMarketplaceHandler(service *service.MCPMarketplaceService) *MCPMarketplaceHandler {
	return &MCPMarketplaceHandler{service: service}
}

// ListMarketplace 列出广场项目
// GET /api/mcp/marketplace
func (h *MCPMarketplaceHandler) ListMarketplace(c *gin.Context) {
	var req models.SearchMCPRequest

	req.Query = c.Query("query")
	req.Category = c.Query("category")
	req.SortBy = c.Query("sort_by")
	req.Order = c.Query("order")

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		req.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil {
		req.PageSize = pageSize
	}

	resp, err := h.service.ListMarketplaceItems(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", resp)
}

// GetMarketplaceItem 获取广场项目详情
// GET /api/mcp/marketplace/:id
func (h *MCPMarketplaceHandler) GetMarketplaceItem(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的项目 ID", nil)
		return
	}

	item, err := h.service.GetMarketplaceItem(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, "项目不存在", nil)
		return
	}

	response.Success(c, "获取成功", item)
}

// InstallMCP 安装 MCP
// POST /api/mcp/marketplace/:id/install
func (h *MCPMarketplaceHandler) InstallMCP(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的项目 ID", nil)
		return
	}

	var req models.InstallMCPRequest
	// Try to bind JSON, but it's optional
	_ = c.ShouldBindJSON(&req)

	req.MarketplaceID = uint(id)

	installation, err := h.service.InstallMCP(c.Request.Context(), user.ID, &req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "安装成功", installation)
}

// UninstallMCP 卸载 MCP
// DELETE /api/mcp/marketplace/installations/:id
func (h *MCPMarketplaceHandler) UninstallMCP(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的安装 ID", nil)
		return
	}

	if err := h.service.UninstallMCP(c.Request.Context(), user.ID, uint(id)); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "卸载成功", nil)
}

// GetUserInstalledMCPs 获取用户已安装的 MCP
// GET /api/mcp/marketplace/my-installations
func (h *MCPMarketplaceHandler) GetUserInstalledMCPs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	installations, err := h.service.GetUserInstalledMCPs(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", installations)
}

// UpdateInstallationConfig 更新安装配置
// PATCH /api/mcp/marketplace/installations/:id/config
func (h *MCPMarketplaceHandler) UpdateInstallationConfig(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的安装 ID", nil)
		return
	}

	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	if err := h.service.UpdateInstallationConfig(c.Request.Context(), user.ID, uint(id), config); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "配置更新成功", nil)
}

// ReviewMCP 评论 MCP
// POST /api/mcp/marketplace/:id/reviews
func (h *MCPMarketplaceHandler) ReviewMCP(c *gin.Context) {
	userID := c.GetUint("userId")
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的项目 ID", nil)
		return
	}

	var req models.ReviewMCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	review, err := h.service.ReviewMCP(c.Request.Context(), userID, uint(id), &req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "评论成功", review)
}

// GetMCPReviews 获取 MCP 评论
// GET /api/mcp/marketplace/:id/reviews
func (h *MCPMarketplaceHandler) GetMCPReviews(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的项目 ID", nil)
		return
	}

	page := 1
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		page = p
	}

	pageSize := 20
	if ps, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil {
		pageSize = ps
	}

	reviews, total, err := h.service.GetMCPReviews(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"reviews":   reviews,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetCategories 获取分类列表
// GET /api/mcp/marketplace/categories
func (h *MCPMarketplaceHandler) GetCategories(c *gin.Context) {
	categories, err := h.service.GetCategories(c.Request.Context())
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", categories)
}

// GetFeaturedMCPs 获取推荐的 MCP
// GET /api/mcp/marketplace/featured
func (h *MCPMarketplaceHandler) GetFeaturedMCPs(c *gin.Context) {
	limit := 10
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	items, err := h.service.GetFeaturedMCPs(c.Request.Context(), limit)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", items)
}

// GetTrendingMCPs 获取热门 MCP
// GET /api/mcp/marketplace/trending
func (h *MCPMarketplaceHandler) GetTrendingMCPs(c *gin.Context) {
	limit := 10
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	items, err := h.service.GetTrendingMCPs(c.Request.Context(), limit)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", items)
}

// SearchByTag 按标签搜索
// GET /api/mcp/marketplace/search/tag/:tag
func (h *MCPMarketplaceHandler) SearchByTag(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		response.Fail(c, "标签不能为空", nil)
		return
	}

	limit := 20
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	items, err := h.service.SearchByTag(c.Request.Context(), tag, limit)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	response.Success(c, "获取成功", items)
}
