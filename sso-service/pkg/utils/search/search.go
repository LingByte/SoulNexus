package search

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchHandlers 封装搜索相关的API处理
type SearchHandlers struct {
	engine Engine
	db     *gorm.DB
}

// SetDB 设置数据库连接（用于检查配置）
func (h *SearchHandlers) SetDB(db *gorm.DB) {
	h.db = db
}

// GetEngine 获取搜索引擎实例
func (h *SearchHandlers) GetEngine() Engine {
	return h.engine
}

// isSearchEnabled 检查搜索是否启用
func (h *SearchHandlers) isSearchEnabled() bool {
	if h.db == nil {
		return false
	}
	enabled := utils.GetBoolValue(h.db, constants.KEY_SEARCH_ENABLED)
	return enabled
}

// NewSearchHandlers 创建一个新的SearchHandlers实例
func NewSearchHandlers(engine Engine) *SearchHandlers {
	return &SearchHandlers{
		engine: engine,
	}
}

// RegisterSearchRoutes 注册与搜索相关的路由
// 注意：调用此函数前，调用方应该已经检查了搜索是否启用
func (h *SearchHandlers) RegisterSearchRoutes(r *gin.RouterGroup) {
	// Search API 路由
	// 注意：搜索接口不需要强制认证，但如果用户已登录，会自动过滤用户数据
	// 直接注册路由，不使用 Group，避免路径匹配问题
	r.POST("/search", h.handleSearch)
	r.POST("/search/", h.handleSearch) // 同时注册带斜杠的版本
	// 索引文档接口（需要认证）
	r.POST("/search/index", h.handleIndex)
	// 删除文档接口（需要认证）
	r.POST("/search/delete", h.handleDelete)
	// 自动补全接口
	r.POST("/search/auto-complete", h.handleAutoComplete)
	// 搜索建议接口
	r.POST("/search/suggest", h.handleSuggest)
}

// handleSearch 处理搜索请求
func (h *SearchHandlers) handleSearch(c *gin.Context) {
	// 检查搜索是否启用
	if !h.isSearchEnabled() {
		response.Fail(c, "Search is disabled", gin.H{"error": "搜索功能未启用，请在系统设置中启用搜索功能"})
		return
	}

	if h.engine == nil {
		response.Fail(c, "Search engine not initialized", gin.H{"error": "搜索引擎未初始化"})
		return
	}

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid search request", gin.H{"error": err.Error()})
		return
	}

	// 验证请求参数
	if req.Size < 0 {
		req.Size = 10
	}
	if req.From < 0 {
		req.From = 0
	}

	// 从上下文获取用户ID（如果已登录）
	// 使用 CurrentUser 函数获取当前用户
	user := models.CurrentUser(c)
	if user != nil && user.ID > 0 {
		// 添加用户ID过滤，确保用户只能搜索到自己的数据
		userID := fmt.Sprintf("%d", user.ID)
		if req.MustTerms == nil {
			req.MustTerms = make(map[string][]string)
		}
		req.MustTerms["userId"] = []string{userID}
	}

	// 执行搜索
	result, err := h.engine.Search(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, "Search failed", gin.H{"error": err.Error()})
		return
	}

	response.Success(c, "Get Search Result", result)
}

// handleIndex 处理文档索引请求
func (h *SearchHandlers) handleIndex(c *gin.Context) {
	// 检查搜索是否启用
	if !h.isSearchEnabled() {
		response.Fail(c, "Search is disabled", gin.H{"error": "搜索功能未启用"})
		return
	}

	if h.engine == nil {
		response.Fail(c, "Search engine not initialized", gin.H{"error": "搜索引擎未初始化"})
		return
	}

	var doc Doc
	if err := c.ShouldBindJSON(&doc); err != nil {
		response.Fail(c, "Invalid document", gin.H{"error": err.Error()})
		return
	}

	// 验证文档
	if doc.ID == "" {
		response.Fail(c, "Document ID is required", nil)
		return
	}

	// 索引文档
	err := h.engine.Index(c.Request.Context(), doc)
	if err != nil {
		response.Fail(c, "Failed to index document", gin.H{"error": err.Error()})
		return
	}
	response.Success(c, "Document indexed successfully", gin.H{"doc": doc})
}

// handleDelete 处理文档删除请求
func (h *SearchHandlers) handleDelete(c *gin.Context) {
	// 检查搜索是否启用
	if !h.isSearchEnabled() {
		response.Fail(c, "Search is disabled", gin.H{"error": "搜索功能未启用"})
		return
	}

	if h.engine == nil {
		response.Fail(c, "Search engine not initialized", gin.H{"error": "搜索引擎未初始化"})
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid document", gin.H{"error": err.Error()})
		return
	}

	// 验证ID
	if req.ID == "" {
		response.Fail(c, "Document ID is required", nil)
		return
	}

	// 删除文档
	err := h.engine.Delete(c.Request.Context(), req.ID)
	if err != nil {
		response.Fail(c, "Failed to delete document", gin.H{"error": err.Error()})
		return
	}
	response.Success(c, "Document deleted successfully", nil)
}

// handleAutoComplete 处理自动补全请求
func (h *SearchHandlers) handleAutoComplete(c *gin.Context) {
	// 检查搜索是否启用
	if !h.isSearchEnabled() {
		response.Fail(c, "Search is disabled", gin.H{"error": "搜索功能未启用"})
		return
	}

	if h.engine == nil {
		response.Fail(c, "Search engine not initialized", gin.H{"error": "搜索引擎未初始化"})
		return
	}

	var req struct {
		Keyword string `json:"keyword"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid keyword", gin.H{"error": err.Error()})
		return
	}

	// 获取自动补全建议
	suggestions, err := h.engine.GetAutoCompleteSuggestions(c.Request.Context(), req.Keyword)
	if err != nil {
		response.Fail(c, "Failed to get suggestions", gin.H{"error": err.Error()})
		return
	}
	response.Success(c, "Get Suggestion successfully", suggestions)
}

// handleSuggest 处理搜索建议请求
func (h *SearchHandlers) handleSuggest(c *gin.Context) {
	// 检查搜索是否启用
	if !h.isSearchEnabled() {
		response.Fail(c, "Search is disabled", gin.H{"error": "搜索功能未启用"})
		return
	}

	if h.engine == nil {
		response.Fail(c, "Search engine not initialized", gin.H{"error": "搜索引擎未初始化"})
		return
	}

	var req struct {
		Keyword string `json:"keyword"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid keyword", gin.H{"error": err.Error()})
		return
	}

	// 获取基于关键词的搜索建议
	suggestions, err := h.engine.GetSearchSuggestions(c.Request.Context(), req.Keyword)
	if err != nil {
		response.Fail(c, "Failed to get suggestions", gin.H{"error": err.Error()})
		return
	}

	response.Success(c, "Get Suggestion successfully", suggestions)
}
