package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
)

// NodePluginHandler 节点插件处理器
type NodePluginHandler struct {
	db *gorm.DB
}

// NewNodePluginHandler 创建节点插件处理器
func NewNodePluginHandler(db *gorm.DB) *NodePluginHandler {
	return &NodePluginHandler{db: db}
}

// CreatePlugin 创建插件
func (h *NodePluginHandler) CreatePlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	var req struct {
		Name        string                      `json:"name" binding:"required"`
		DisplayName string                      `json:"displayName" binding:"required"`
		Description string                      `json:"description"`
		Category    models.NodePluginCategory   `json:"category" binding:"required"`
		Icon        string                      `json:"icon"`
		Color       string                      `json:"color"`
		Tags        []string                    `json:"tags"`
		Definition  models.NodePluginDefinition `json:"definition" binding:"required"`
		Schema      models.NodePluginSchema     `json:"schema"`
		Author      string                      `json:"author"`
		Homepage    string                      `json:"homepage"`
		Repository  string                      `json:"repository"`
		License     string                      `json:"license"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}

	// 生成唯一的slug
	slug := generateSlug(req.Name)

	// 检查slug是否已存在
	var existingPlugin models.NodePlugin
	if err := h.db.Where("slug = ?", slug).First(&existingPlugin).Error; err == nil {
		response.Fail(c, "插件名称已存在", nil)
		return
	}

	plugin := models.NodePlugin{
		UserID:      userID,
		Name:        req.Name,
		Slug:        slug,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Category:    req.Category,
		Version:     "1.0.0",
		Status:      models.NodePluginStatusDraft,
		Icon:        req.Icon,
		Color:       req.Color,
		Tags:        models.StringArray(req.Tags),
		Definition:  req.Definition,
		Schema:      req.Schema,
		Author:      req.Author,
		Homepage:    req.Homepage,
		Repository:  req.Repository,
		License:     req.License,
	}

	if err := h.db.Create(&plugin).Error; err != nil {
		response.Fail(c, "创建插件失败: "+err.Error(), nil)
		return
	}

	// 创建初始版本
	version := models.NodePluginVersion{
		PluginID:   plugin.ID,
		Version:    plugin.Version,
		Definition: plugin.Definition,
		Schema:     plugin.Schema,
		ChangeLog:  "初始版本",
	}

	if err := h.db.Create(&version).Error; err != nil {
		response.Fail(c, "创建版本失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "创建成功", plugin)
}

// ListPlugins 获取插件列表
func (h *NodePluginHandler) ListPlugins(c *gin.Context) {
	var req struct {
		Category string `form:"category"`
		Status   string `form:"status"`
		Keyword  string `form:"keyword"`
		UserID   uint   `form:"userId"`
		Page     int    `form:"page"`
		PageSize int    `form:"pageSize"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := h.db.Model(&models.NodePlugin{})

	// 过滤条件
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	} else {
		// 默认只显示已发布的插件
		query = query.Where("status = ?", models.NodePluginStatusPublished)
	}
	if req.UserID != 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "查询失败: "+err.Error(), nil)
		return
	}

	// 分页查询
	var plugins []models.NodePlugin
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).
		Order("download_count DESC, star_count DESC, created_at DESC").
		Find(&plugins).Error; err != nil {
		response.Fail(c, "查询失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "查询成功", gin.H{
		"plugins":  plugins,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

// GetPlugin 获取插件详情
func (h *NodePluginHandler) GetPlugin(c *gin.Context) {
	pluginID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的插件ID", nil)
		return
	}

	var plugin models.NodePlugin
	if err := h.db.Preload("Versions").Preload("Reviews.User").
		First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	response.Success(c, "查询成功", plugin)
}

// UpdatePlugin 更新插件
func (h *NodePluginHandler) UpdatePlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	pluginID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的插件ID", nil)
		return
	}

	var plugin models.NodePlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	// 检查权限
	if plugin.UserID != userID {
		response.Fail(c, "无权限操作", nil)
		return
	}

	var req struct {
		DisplayName string                      `json:"displayName"`
		Description string                      `json:"description"`
		Category    models.NodePluginCategory   `json:"category"`
		Icon        string                      `json:"icon"`
		Color       string                      `json:"color"`
		Tags        []string                    `json:"tags"`
		Definition  models.NodePluginDefinition `json:"definition"`
		Schema      models.NodePluginSchema     `json:"schema"`
		Version     string                      `json:"version"`
		ChangeLog   string                      `json:"changeLog"`
		Author      string                      `json:"author"`
		Homepage    string                      `json:"homepage"`
		Repository  string                      `json:"repository"`
		License     string                      `json:"license"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}

	// 更新插件信息
	updates := map[string]interface{}{}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if len(req.Tags) > 0 {
		updates["tags"] = models.StringArray(req.Tags)
	}
	if req.Author != "" {
		updates["author"] = req.Author
	}
	if req.Homepage != "" {
		updates["homepage"] = req.Homepage
	}
	if req.Repository != "" {
		updates["repository"] = req.Repository
	}
	if req.License != "" {
		updates["license"] = req.License
	}

	// 如果有新版本，创建版本记录
	if req.Version != "" && req.Version != plugin.Version {
		updates["version"] = req.Version
		updates["definition"] = req.Definition
		updates["schema"] = req.Schema

		// 创建新版本记录
		version := models.NodePluginVersion{
			PluginID:   plugin.ID,
			Version:    req.Version,
			Definition: req.Definition,
			Schema:     req.Schema,
			ChangeLog:  req.ChangeLog,
		}

		if err := h.db.Create(&version).Error; err != nil {
			response.Fail(c, "创建版本失败: "+err.Error(), nil)
			return
		}
	}

	if err := h.db.Model(&plugin).Updates(updates).Error; err != nil {
		response.Fail(c, "更新失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "更新成功", gin.H{"message": "更新成功"})
}

// PublishPlugin 发布插件
func (h *NodePluginHandler) PublishPlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	pluginID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的插件ID", nil)
		return
	}

	var plugin models.NodePlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	// 检查权限
	if plugin.UserID != userID {
		response.Fail(c, "无权限操作", nil)
		return
	}

	// 更新状态为已发布
	if err := h.db.Model(&plugin).Update("status", models.NodePluginStatusPublished).Error; err != nil {
		response.Fail(c, "发布失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "发布成功", gin.H{"message": "发布成功"})
}

// InstallPlugin 安装插件
func (h *NodePluginHandler) InstallPlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	pluginID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的插件ID", nil)
		return
	}

	var req struct {
		Version string                 `json:"version"`
		Config  map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}

	// 检查插件是否存在
	var plugin models.NodePlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	// 检查是否已安装
	var existing models.NodePluginInstallation
	if err := h.db.Where("user_id = ? AND plugin_id = ?", userID, pluginID).
		First(&existing).Error; err == nil {
		response.Fail(c, "插件已安装", nil)
		return
	}

	// 使用指定版本或最新版本
	version := req.Version
	if version == "" {
		version = plugin.Version
	}

	// 创建安装记录
	installation := models.NodePluginInstallation{
		UserID:   userID,
		PluginID: uint(pluginID),
		Version:  version,
		Status:   "active",
		Config:   models.JSONMap(req.Config),
	}

	if err := h.db.Create(&installation).Error; err != nil {
		response.Fail(c, "安装失败: "+err.Error(), nil)
		return
	}

	// 增加下载计数
	h.db.Model(&plugin).UpdateColumn("download_count", gorm.Expr("download_count + 1"))

	response.Success(c, "安装成功", gin.H{"message": "安装成功"})
}

// DeletePlugin 删除插件
func (h *NodePluginHandler) DeletePlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	pluginID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的插件ID", nil)
		return
	}

	var plugin models.NodePlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	// 检查权限
	if plugin.UserID != userID {
		response.Fail(c, "无权限操作", nil)
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除相关的版本记录
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&models.NodePluginVersion{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "删除版本记录失败: "+err.Error(), nil)
		return
	}

	// 删除相关的评价记录
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&models.NodePluginReview{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "删除评价记录失败: "+err.Error(), nil)
		return
	}

	// 删除相关的安装记录
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&models.NodePluginInstallation{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "删除安装记录失败: "+err.Error(), nil)
		return
	}

	// 删除插件本身
	if err := tx.Delete(&plugin).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "删除插件失败: "+err.Error(), nil)
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		response.Fail(c, "删除失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "删除成功", gin.H{"message": "删除成功"})
}

// ListInstalledPlugins 获取已安装插件列表
func (h *NodePluginHandler) ListInstalledPlugins(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	var installations []models.NodePluginInstallation
	if err := h.db.Preload("Plugin").Where("user_id = ? AND status = ?", userID, "active").
		Find(&installations).Error; err != nil {
		response.Fail(c, "查询失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "查询成功", installations)
}
