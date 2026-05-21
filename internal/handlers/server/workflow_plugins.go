package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/LingByte/SoulNexus/pkg/response"
)

// WorkflowPluginHandler 工作流插件处理器
type WorkflowPluginHandler struct {
	db *gorm.DB
}

// NewWorkflowPluginHandler 创建工作流插件处理器
func NewWorkflowPluginHandler(db *gorm.DB) *WorkflowPluginHandler {
	return &WorkflowPluginHandler{db: db}
}

// PublishWorkflowAsPlugin 将工作流发布为插件
func (h *WorkflowPluginHandler) PublishWorkflowAsPlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	workflowID, err := strconv.ParseUint(c.Param("workflowId"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的工作流ID", nil)
		return
	}

	var req struct {
		Name         string                        `json:"name" binding:"required"`
		DisplayName  string                        `json:"displayName" binding:"required"`
		Description  string                        `json:"description"`
		Category     svcmodels.WorkflowPluginCategory `json:"category" binding:"required"`
		Icon         string                        `json:"icon"`
		Color        string                        `json:"color"`
		Tags         []string                      `json:"tags"`
		InputSchema  svcmodels.WorkflowPluginIOSchema `json:"inputSchema"`
		OutputSchema svcmodels.WorkflowPluginIOSchema `json:"outputSchema"`
		Author       string                        `json:"author"`
		Homepage     string                        `json:"homepage"`
		Repository   string                        `json:"repository"`
		License      string                        `json:"license"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}

	groupIDs, gerr := svcmodels.MemberGroupIDs(h.db, userID)
	if gerr != nil {
		response.Fail(c, "查询失败: "+gerr.Error(), nil)
		return
	}
	var workflow svcmodels.WorkflowDefinition
	if err := h.db.Where("id = ? AND group_id IN ?", workflowID, groupIDs).First(&workflow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "工作流不存在或无权限", nil)
		} else {
			response.Fail(c, "查询工作流失败: "+err.Error(), nil)
		}
		return
	}
	if !svcmodels.CanManageTenantResource(h.db, userID, workflow.GroupID, workflow.CreatorUID) {
		response.Fail(c, "工作流不存在或无权限", nil)
		return
	}

	// 生成唯一的slug
	slug := generateSlug(req.Name)

	// 检查slug是否已存在
	var existingPlugin svcmodels.WorkflowPlugin
	if err := h.db.Where("slug = ?", slug).First(&existingPlugin).Error; err == nil {
		response.Fail(c, "插件名称已存在", nil)
		return
	}

	plugin := svcmodels.WorkflowPlugin{
		CreatedBy:        userID,
		GroupID:          workflow.GroupID,
		WorkflowID:       uint(workflowID),
		Name:             req.Name,
		Slug:             slug,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		Category:         req.Category,
		Version:          "1.0.0",
		Status:           svcmodels.WorkflowPluginStatusDraft,
		Icon:             req.Icon,
		Color:            req.Color,
		Tags:             svcmodels.StringArray(req.Tags),
		InputSchema:      ensureIOSchema(req.InputSchema),
		OutputSchema:     ensureIOSchema(req.OutputSchema),
		WorkflowSnapshot: workflow.Definition,
		Author:           req.Author,
		Homepage:         req.Homepage,
		Repository:       req.Repository,
		License:          req.License,
	}

	if err := h.db.Create(&plugin).Error; err != nil {
		response.Fail(c, "创建插件失败: "+err.Error(), nil)
		return
	}

	// 创建初始版本
	version := svcmodels.WorkflowPluginVersion{
		PluginID:         plugin.ID,
		Version:          plugin.Version,
		WorkflowSnapshot: plugin.WorkflowSnapshot,
		InputSchema:      plugin.InputSchema,
		OutputSchema:     plugin.OutputSchema,
		ChangeLog:        "初始版本",
	}

	if err := h.db.Create(&version).Error; err != nil {
		response.Fail(c, "创建版本失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "发布成功", plugin)
}

// ListWorkflowPlugins 获取工作流插件列表
func (h *WorkflowPluginHandler) ListWorkflowPlugins(c *gin.Context) {
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

	query := h.db.Model(&svcmodels.WorkflowPlugin{})

	// 过滤条件
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	// 注意：当没有指定status时，显示所有状态的插件
	if req.UserID != 0 {
		query = query.Where("created_by = ?", req.UserID)
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
	var plugins []svcmodels.WorkflowPlugin
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

// GetWorkflowPlugin 获取工作流插件详情
func (h *WorkflowPluginHandler) GetWorkflowPlugin(c *gin.Context) {
	pluginID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的插件ID", nil)
		return
	}

	var plugin svcmodels.WorkflowPlugin
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

// UpdateWorkflowPlugin 更新工作流插件
func (h *WorkflowPluginHandler) UpdateWorkflowPlugin(c *gin.Context) {
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

	var plugin svcmodels.WorkflowPlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	if !svcmodels.CanManageTenantResource(h.db, userID, plugin.GroupID, plugin.CreatedBy) {
		response.Fail(c, "无权限操作", nil)
		return
	}

	var req struct {
		DisplayName  string                        `json:"displayName"`
		Description  string                        `json:"description"`
		Category     svcmodels.WorkflowPluginCategory `json:"category"`
		Icon         string                        `json:"icon"`
		Color        string                        `json:"color"`
		Tags         []string                      `json:"tags"`
		InputSchema  svcmodels.WorkflowPluginIOSchema `json:"inputSchema"`
		OutputSchema svcmodels.WorkflowPluginIOSchema `json:"outputSchema"`
		Version      string                        `json:"version"`
		ChangeLog    string                        `json:"changeLog"`
		Author       string                        `json:"author"`
		Homepage     string                        `json:"homepage"`
		Repository   string                        `json:"repository"`
		License      string                        `json:"license"`
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
		updates["tags"] = svcmodels.StringArray(req.Tags)
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
		updates["input_schema"] = ensureIOSchema(req.InputSchema)
		updates["output_schema"] = ensureIOSchema(req.OutputSchema)

		// 创建新版本记录
		version := svcmodels.WorkflowPluginVersion{
			PluginID:         plugin.ID,
			Version:          req.Version,
			WorkflowSnapshot: plugin.WorkflowSnapshot,
			InputSchema:      ensureIOSchema(req.InputSchema),
			OutputSchema:     ensureIOSchema(req.OutputSchema),
			ChangeLog:        req.ChangeLog,
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

// PublishWorkflowPlugin 发布工作流插件
func (h *WorkflowPluginHandler) PublishWorkflowPlugin(c *gin.Context) {
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

	var plugin svcmodels.WorkflowPlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	if !svcmodels.CanManageTenantResource(h.db, userID, plugin.GroupID, plugin.CreatedBy) {
		response.Fail(c, "无权限操作", nil)
		return
	}

	// 更新状态为已发布
	if err := h.db.Model(&plugin).Update("status", svcmodels.WorkflowPluginStatusPublished).Error; err != nil {
		response.Fail(c, "发布失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "发布成功", gin.H{"message": "发布成功"})
}

// InstallWorkflowPlugin 安装工作流插件
func (h *WorkflowPluginHandler) InstallWorkflowPlugin(c *gin.Context) {
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
		GroupID *uint                  `json:"groupId"`
		Version string                 `json:"version"`
		Config  map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}

	targetGID, err := svcmodels.ResolveWriteGroupID(h.db, userID, req.GroupID)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	// 检查插件是否存在
	var plugin svcmodels.WorkflowPlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	// 检查是否已安装
	var existing svcmodels.WorkflowPluginInstallation
	if err := h.db.Where("group_id = ? AND plugin_id = ?", targetGID, pluginID).
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
	installation := svcmodels.WorkflowPluginInstallation{
		GroupID:   targetGID,
		CreatedBy: userID,
		PluginID:  uint(pluginID),
		Version:   version,
		Status:    "active",
		Config:    svcmodels.JSONMap(req.Config),
	}

	if err := h.db.Create(&installation).Error; err != nil {
		response.Fail(c, "安装失败: "+err.Error(), nil)
		return
	}

	// 增加下载计数
	h.db.Model(&plugin).UpdateColumn("download_count", gorm.Expr("download_count + 1"))

	response.Success(c, "安装成功", gin.H{"message": "安装成功"})
}

// ListInstalledWorkflowPlugins 获取已安装的工作流插件列表
func (h *WorkflowPluginHandler) ListInstalledWorkflowPlugins(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	groupIDs, gerr := svcmodels.MemberGroupIDs(h.db, userID)
	if gerr != nil {
		response.Fail(c, "查询失败: "+gerr.Error(), nil)
		return
	}
	var installations []svcmodels.WorkflowPluginInstallation
	if err := h.db.Preload("Plugin").Where("group_id IN ? AND status = ?", groupIDs, "active").
		Find(&installations).Error; err != nil {
		response.Fail(c, "查询失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "查询成功", installations)
}

// DeleteWorkflowPlugin 删除工作流插件
func (h *WorkflowPluginHandler) DeleteWorkflowPlugin(c *gin.Context) {
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

	var plugin svcmodels.WorkflowPlugin
	if err := h.db.First(&plugin, pluginID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "插件不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	if !svcmodels.CanManageTenantResource(h.db, userID, plugin.GroupID, plugin.CreatedBy) {
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
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&svcmodels.WorkflowPluginVersion{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "删除版本记录失败: "+err.Error(), nil)
		return
	}

	// 删除相关的评价记录
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&svcmodels.WorkflowPluginReview{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "删除评价记录失败: "+err.Error(), nil)
		return
	}

	// 删除相关的安装记录
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&svcmodels.WorkflowPluginInstallation{}).Error; err != nil {
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

// GetUserWorkflowPlugins 获取用户创建的工作流插件
func (h *WorkflowPluginHandler) GetUserWorkflowPlugins(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	var plugins []svcmodels.WorkflowPlugin
	if err := h.db.Where("created_by = ?", userID).
		Order("created_at DESC").
		Find(&plugins).Error; err != nil {
		response.Fail(c, "查询失败: "+err.Error(), nil)
		return
	}

	response.Success(c, "查询成功", plugins)
}

// GetWorkflowPublishedPlugin 获取工作流已发布的插件信息
func (h *WorkflowPluginHandler) GetWorkflowPublishedPlugin(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	workflowID, err := strconv.ParseUint(c.Param("workflowId"), 10, 32)
	if err != nil {
		response.Fail(c, "无效的工作流ID", nil)
		return
	}

	var wf svcmodels.WorkflowDefinition
	if err := h.db.First(&wf, workflowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "工作流不存在", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}
	if !svcmodels.UserIsGroupMember(h.db, userID, wf.GroupID) {
		response.Fail(c, "无权限", nil)
		return
	}

	var plugin svcmodels.WorkflowPlugin
	if err := h.db.Where("workflow_id = ? AND group_id = ?", workflowID, wf.GroupID).
		First(&plugin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 工作流未发布过插件
			response.Success(c, "工作流未发布过插件", nil)
		} else {
			response.Fail(c, "查询失败: "+err.Error(), nil)
		}
		return
	}

	response.Success(c, "查询成功", plugin)
}

// ensureIOSchema 确保 IOSchema 不为 nil，如果为 nil 则返回空的 IOSchema
func ensureIOSchema(schema svcmodels.WorkflowPluginIOSchema) svcmodels.WorkflowPluginIOSchema {
	if schema.Parameters == nil {
		schema.Parameters = []svcmodels.WorkflowPluginParameter{}
	}
	return schema
}
