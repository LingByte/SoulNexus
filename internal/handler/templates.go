package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	_ "net/http"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/constants"
	jsPkg "github.com/LingByte/SoulNexus/pkg/js"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CreateJSTemplate creates JS template
func (h *Handlers) CreateJSTemplate(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	var template models.JSTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.Fail(c, "Invalid parameters: "+err.Error(), nil)
		return
	}

	// If it's a custom template, set the user ID
	if template.Type == "custom" {
		template.UserID = user.ID
	}

	// 如果设置了 GroupID，验证用户是否有权限共享到该组织
	if template.GroupID != nil {
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		// 检查用户是否是组织成员或创建者
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *template.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "You are not a member of this organization")
				return
			}
		}
	}

	// AST白名单检查
	if template.Content != "" {
		whitelist := jsPkg.DefaultWhitelist
		isValid, violations := jsPkg.ValidateAST(template.Content, whitelist)
		if !isValid {
			response.Fail(c, "代码不符合安全规范", gin.H{
				"violations": violations,
			})
			return
		}

		// 资源配额检查
		maxExecutionTime := template.QuotaMaxExecutionTime
		if maxExecutionTime == 0 {
			maxExecutionTime = 30000 // 默认30秒
		}
		maxMemoryMB := template.QuotaMaxMemoryMB
		if maxMemoryMB == 0 {
			maxMemoryMB = 100 // 默认100MB
		}
		maxAPICalls := template.QuotaMaxAPICalls
		if maxAPICalls == 0 {
			maxAPICalls = 1000 // 默认1000次
		}

		isQuotaValid, quotaViolations := jsPkg.CheckResourceQuota(template.Content, maxExecutionTime, maxMemoryMB, maxAPICalls)
		if !isQuotaValid {
			response.Fail(c, "代码超出资源配额限制", gin.H{
				"violations": quotaViolations,
			})
			return
		}
	}

	// 设置默认值
	if template.Version == 0 {
		template.Version = 1
	}
	if template.Status == "" {
		template.Status = "active"
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)
	if err := models.CreateJSTemplate(db, &template); err != nil {
		response.Fail(c, "Failed to create template: "+err.Error(), nil)
		return
	}

	// 创建初始版本记录
	version := models.JSTemplateVersion{
		ID:         uuid.New().String(),
		TemplateID: template.ID,
		Version:    template.Version,
		Name:       template.Name,
		Content:    template.Content,
		Status:     template.Status,
		Grayscale:  100, // 初始版本100%灰度
		ChangeNote: "初始版本",
		CreatedBy:  user.ID,
	}
	if err := models.CreateJSTemplateVersion(db, &version); err != nil {
		logger.Warn("failed to create template version", zap.Error(err))
		// 不阻止模板创建，只记录警告
	}

	response.Success(c, "Template created successfully", template)
}

// GetJSTemplate gets a single JS template
func (h *Handlers) GetJSTemplate(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	template, err := models.GetJSTemplateByID(db, id)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	// 检查权限：用户自己的模板或组织共享的模板（用户是组织成员）
	if template.UserID != user.ID {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		// 检查用户是否是组织成员
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *template.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "You are not a member of this organization")
				return
			}
		}
	}

	response.Success(c, "Template retrieved successfully", template)
}

// GetJSTemplateByName gets JS template list by name (because names may be duplicated)
func (h *Handlers) GetJSTemplateByName(c *gin.Context) {
	name := c.Param("name")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	templates, err := models.GetJSTemplatesByName(db, name)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	response.Success(c, "Templates retrieved successfully", templates)
}

// ListJSTemplates gets JS template list
func (h *Handlers) ListJSTemplates(c *gin.Context) {
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	userId := user.ID

	// Pagination parameters
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 || limitInt > 100 {
		limitInt = 10
	}

	offset := (pageInt - 1) * limitInt

	// 获取用户所属的组织ID列表
	var groupIDs []uint
	var groupMembers []models.GroupMember
	if err := h.db.Where("user_id = ?", userId).Find(&groupMembers).Error; err == nil {
		for _, member := range groupMembers {
			groupIDs = append(groupIDs, member.GroupID)
		}
	}
	// 获取用户创建的组织ID
	var userGroups []models.Group
	if err := h.db.Where("creator_id = ?", userId).Find(&userGroups).Error; err == nil {
		for _, group := range userGroups {
			groupIDs = append(groupIDs, group.ID)
		}
	}

	// 查询：用户自己的模板 + 组织共享的模板
	var templates []models.JSTemplate
	query := db.Offset(offset).Limit(limitInt).Order("created_at DESC")

	// 构建查询条件：用户自己的模板 OR 组织共享的模板（用户是成员）
	if len(groupIDs) > 0 {
		query = query.Where("user_id = ? OR (group_id IS NOT NULL AND group_id IN (?))", userId, groupIDs)
	} else {
		query = query.Where("user_id = ?", userId)
	}

	if err := query.Find(&templates).Error; err != nil {
		response.Fail(c, "Failed to get template list: "+err.Error(), nil)
		return
	}

	// 获取总数
	var total int64
	countQuery := db.Model(&models.JSTemplate{})
	if len(groupIDs) > 0 {
		countQuery = countQuery.Where("user_id = ? OR (group_id IS NOT NULL AND group_id IN (?))", userId, groupIDs)
	} else {
		countQuery = countQuery.Where("user_id = ?", userId)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		response.Fail(c, "Failed to get total template count: "+err.Error(), nil)
		return
	}

	response.Success(c, "Template list retrieved successfully", gin.H{
		"data":  templates,
		"page":  pageInt,
		"limit": limitInt,
		"total": total,
	})
}

// UpdateJSTemplate updates JS template
func (h *Handlers) UpdateJSTemplate(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 检查模板是否存在
	template, err := models.GetJSTemplateByID(db, id)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	userId := user.ID

	// 检查权限：只有创建者或组织管理员可以更新
	if template.UserID != userId {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		// 检查用户是否是组织创建者或管理员
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != userId {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *template.GroupID, userId, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creator or admin can update organization-shared templates")
				return
			}
		}
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Fail(c, "Invalid parameters: "+err.Error(), nil)
		return
	}

	// 如果更新了内容，进行AST和配额检查
	if content, ok := updates["content"].(string); ok && content != "" {
		// AST白名单检查
		whitelist := jsPkg.DefaultWhitelist
		isValid, violations := jsPkg.ValidateAST(content, whitelist)
		if !isValid {
			response.Fail(c, "代码不符合安全规范", gin.H{
				"violations": violations,
			})
			return
		}

		// 资源配额检查
		maxExecutionTime := template.QuotaMaxExecutionTime
		if maxExecutionTime == 0 {
			maxExecutionTime = 30000
		}
		maxMemoryMB := template.QuotaMaxMemoryMB
		if maxMemoryMB == 0 {
			maxMemoryMB = 100
		}
		maxAPICalls := template.QuotaMaxAPICalls
		if maxAPICalls == 0 {
			maxAPICalls = 1000
		}

		isQuotaValid, quotaViolations := jsPkg.CheckResourceQuota(content, maxExecutionTime, maxMemoryMB, maxAPICalls)
		if !isQuotaValid {
			response.Fail(c, "代码超出资源配额限制", gin.H{
				"violations": quotaViolations,
			})
			return
		}
	}

	// 如果更新了 GroupID，验证权限
	if groupIDVal, ok := updates["group_id"]; ok {
		if groupIDVal != nil {
			groupID := uint(groupIDVal.(float64))
			var group models.Group
			if err := h.db.Where("id = ?", groupID).First(&group).Error; err != nil {
				response.Fail(c, "Organization not found", nil)
				return
			}
			if group.CreatorID != userId {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", groupID, userId).First(&member).Error; err != nil {
					response.Fail(c, "Insufficient permissions", "You are not a member of this organization")
					return
				}
			}
		}
	}

	// 保存当前版本到历史（如果内容有变化）
	var changeNote string
	if note, ok := updates["change_note"].(string); ok {
		changeNote = note
		delete(updates, "change_note")
	} else {
		changeNote = "模板更新"
	}

	// 如果内容有变化，创建新版本
	if content, ok := updates["content"].(string); ok && content != template.Content {
		// 保存当前版本到历史
		currentVersion := models.JSTemplateVersion{
			ID:         uuid.New().String(),
			TemplateID: template.ID,
			Version:    template.Version,
			Name:       template.Name,
			Content:    template.Content,
			Status:     template.Status,
			ChangeNote: changeNote,
			CreatedBy:  user.ID,
		}
		if err := models.CreateJSTemplateVersion(db, &currentVersion); err != nil {
			logger.Warn("failed to save version history", zap.Error(err))
		}

		// 增加版本号
		updates["version"] = template.Version + 1
	}

	// Avoid updating certain key fields
	delete(updates, "id")
	delete(updates, "user_id")
	delete(updates, "created_at")

	if err := models.UpdateJSTemplate(db, id, updates); err != nil {
		response.Fail(c, "Failed to update template: "+err.Error(), nil)
		return
	}

	// Get updated template
	updatedTemplate, err := models.GetJSTemplateByID(db, id)
	if err != nil {
		response.Fail(c, "Failed to get updated template: "+err.Error(), nil)
		return
	}

	// 热更新：清除模板缓存，使新版本立即生效
	cacheClient := cache.GetGlobalCache()
	ctx := context.Background()
	cacheKey := fmt.Sprintf("js:template:content:%s", id)
	cacheClient.Delete(ctx, cacheKey)
	cacheKey2 := fmt.Sprintf("js:template:loader:%s", updatedTemplate.JsSourceID)
	cacheClient.Delete(ctx, cacheKey2)

	response.Success(c, "Template updated successfully", updatedTemplate)
}

// DeleteJSTemplate deletes JS template
func (h *Handlers) DeleteJSTemplate(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 检查模板是否存在
	template, err := models.GetJSTemplateByID(db, id)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	userId := user.ID
	if template.Type == "default" {
		response.Fail(c, "Cannot delete default template", nil)
		return
	}

	// 检查权限：只有创建者或组织管理员可以删除
	if template.UserID != userId {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		// 检查用户是否是组织创建者或管理员
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != userId {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *template.GroupID, userId, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creator or admin can delete organization-shared templates")
				return
			}
		}
	}

	if err := models.DeleteJSTemplate(db, id); err != nil {
		response.Fail(c, "Failed to delete template: "+err.Error(), nil)
		return
	}

	response.Success(c, "Template deleted successfully", nil)
}

// ListDefaultJSTemplates gets default template list
func (h *Handlers) ListDefaultJSTemplates(c *gin.Context) {
	db := c.MustGet(constants.DbField).(*gorm.DB)

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 || limitInt > 100 {
		limitInt = 10
	}

	offset := (pageInt - 1) * limitInt

	templates, err := models.ListJSTemplatesByType(db, "default", 0, offset, limitInt)
	if err != nil {
		response.Fail(c, "Failed to get default templates: "+err.Error(), nil)
		return
	}

	// Get total count for pagination
	total, err := models.GetJSTemplatesCount(db, "default", 0)
	if err != nil {
		response.Fail(c, "Failed to get total default templates count: "+err.Error(), nil)
		return
	}

	response.Success(c, "Get default templates successfully", gin.H{
		"data":  templates,
		"page":  pageInt,
		"limit": limitInt,
		"total": total,
	})
}

// ListCustomJSTemplates gets user custom template list
func (h *Handlers) ListCustomJSTemplates(c *gin.Context) {
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	userId := user.ID

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 || limitInt > 100 {
		limitInt = 10
	}

	offset := (pageInt - 1) * limitInt

	templates, err := models.ListJSTemplatesByType(db, "custom", userId, offset, limitInt)
	if err != nil {
		response.Fail(c, "Failed to get custom templates: "+err.Error(), nil)
		return
	}

	// Get total count for pagination
	total, err := models.GetJSTemplatesCount(db, "custom", userId)
	if err != nil {
		response.Fail(c, "Failed to get total custom templates count: "+err.Error(), nil)
		return
	}

	response.Success(c, "Get custom templates successfully", gin.H{
		"data":  templates,
		"page":  pageInt,
		"limit": limitInt,
		"total": total,
	})
}

// SearchJSTemplates search JS templates
func (h *Handlers) SearchJSTemplates(c *gin.Context) {
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	userId := user.ID

	keyword := c.Query("keyword")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 || limitInt > 100 {
		limitInt = 10
	}

	offset := (pageInt - 1) * limitInt

	templates, err := models.SearchJSTemplates(db, keyword, userId, offset, limitInt)
	if err != nil {
		response.Fail(c, "Failed to search templates: "+err.Error(), nil)
		return
	}

	// Get total count for pagination
	total, err := models.GetJSTemplatesCount(db, "", userId)
	if err != nil {
		response.Fail(c, "Failed to get search results total count: "+err.Error(), nil)
		return
	}

	response.Success(c, "Search templates successfully", gin.H{
		"data":  templates,
		"page":  pageInt,
		"limit": limitInt,
		"total": total,
	})
}

// ========== JS模板Webhook相关 ==========

// TriggerJSTemplateWebhook 触发JS模板Webhook
func (h *Handlers) TriggerJSTemplateWebhook(c *gin.Context) {
	jsSourceID := c.Param("jsSourceId")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 查找模板
	template, err := models.GetJSTemplateByJsSourceID(db, jsSourceID)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	// 检查Webhook是否启用
	if !template.WebhookEnabled {
		response.Fail(c, "Webhook is not enabled for this template", nil)
		return
	}

	// 初始化Webhook管理器
	cacheClient := cache.GetGlobalCache()
	webhookManager := jsPkg.NewWebhookManager(cacheClient)
	ctx := context.Background()

	// 验证签名
	if err := webhookManager.VerifyWebhookSignature(c, template.WebhookSecret); err != nil {
		response.Fail(c, "Webhook signature verification failed", err.Error())
		return
	}

	// 检查nonce防重复
	nonce := c.GetHeader("X-Nonce")
	if nonce == "" {
		nonce = c.DefaultQuery("nonce", "")
	}
	isDuplicate, err := webhookManager.CheckNonceDuplicate(ctx, nonce, template.ID)
	if err != nil {
		response.Fail(c, "Failed to check nonce", err.Error())
		return
	}
	if isDuplicate {
		response.Fail(c, "Duplicate request", "This request has already been processed")
		return
	}

	// 检查幂等性
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.DefaultQuery("request_id", "")
	}
	cachedResult, isIdempotent, err := webhookManager.CheckIdempotency(ctx, requestID, template.ID)
	if err != nil {
		response.Fail(c, "Failed to check idempotency", err.Error())
		return
	}
	if isIdempotent {
		response.Success(c, "Request processed (idempotent)", cachedResult)
		return
	}

	// 解析请求参数
	var parameters map[string]interface{}
	if err := c.ShouldBindJSON(&parameters); err != nil && err.Error() != "EOF" {
		parameters = make(map[string]interface{})
		c.Request.ParseForm()
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				parameters[k] = v[0]
			}
		}
	}

	// 添加请求元数据
	parameters["_webhook_headers"] = c.Request.Header
	parameters["_webhook_method"] = c.Request.Method
	parameters["_webhook_path"] = c.Request.URL.Path
	parameters["_webhook_timestamp"] = time.Now().Unix()

	// 执行模板逻辑（这里可以扩展为实际执行JS模板）
	result := gin.H{
		"template_id":  template.ID,
		"js_source_id": template.JsSourceID,
		"parameters":   parameters,
		"executed_at":  time.Now().Unix(),
		"status":       "success",
	}

	// 存储幂等性结果
	if requestID != "" {
		if err := webhookManager.StoreIdempotencyResult(ctx, requestID, template.ID, result); err != nil {
			logger.Warn("failed to store idempotency result", zap.Error(err))
		}
	}

	response.Success(c, "Webhook triggered successfully", result)
}

// ========== JS模板版本管理相关 ==========

// ListJSTemplateVersions 获取模板版本列表
func (h *Handlers) ListJSTemplateVersions(c *gin.Context) {
	templateID := c.Param("id")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// 检查模板权限
	template, err := models.GetJSTemplateByID(db, templateID)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	if template.UserID != user.ID {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *template.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", nil)
				return
			}
		}
	}

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	pageInt, _ := strconv.Atoi(page)
	if pageInt < 1 {
		pageInt = 1
	}

	limitInt, _ := strconv.Atoi(limit)
	if limitInt < 1 || limitInt > 100 {
		limitInt = 20
	}

	offset := (pageInt - 1) * limitInt

	versions, err := models.GetJSTemplateVersions(db, templateID, offset, limitInt)
	if err != nil {
		response.Fail(c, "Failed to get versions: "+err.Error(), nil)
		return
	}

	// 获取总数
	var total int64
	if err := db.Model(&models.JSTemplateVersion{}).Where("template_id = ?", templateID).Count(&total).Error; err != nil {
		response.Fail(c, "Failed to get total count: "+err.Error(), nil)
		return
	}

	response.Success(c, "Versions retrieved successfully", gin.H{
		"data":  versions,
		"page":  pageInt,
		"limit": limitInt,
		"total": total,
	})
}

// GetJSTemplateVersion 获取指定版本
func (h *Handlers) GetJSTemplateVersion(c *gin.Context) {
	templateID := c.Param("id")
	versionID := c.Param("versionId")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// 检查权限
	template, err := models.GetJSTemplateByID(db, templateID)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	if template.UserID != user.ID {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *template.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", nil)
				return
			}
		}
	}

	version, err := models.GetJSTemplateVersion(db, templateID, versionID)
	if err != nil {
		response.Fail(c, "Version not found", nil)
		return
	}

	response.Success(c, "Version retrieved successfully", version)
}

// RollbackJSTemplateVersion 回滚到指定版本
func (h *Handlers) RollbackJSTemplateVersion(c *gin.Context) {
	templateID := c.Param("id")
	versionID := c.Param("versionId")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// 检查权限
	template, err := models.GetJSTemplateByID(db, templateID)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	if template.UserID != user.ID {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *template.GroupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creator or admin can rollback")
				return
			}
		}
	}

	if err := models.RollbackJSTemplateVersion(db, templateID, versionID); err != nil {
		response.Fail(c, "Failed to rollback: "+err.Error(), nil)
		return
	}

	// 热更新：清除缓存
	cacheClient := cache.GetGlobalCache()
	ctx := context.Background()
	cacheKey := fmt.Sprintf("js:template:content:%s", templateID)
	cacheClient.Delete(ctx, cacheKey)
	cacheKey2 := fmt.Sprintf("js:template:loader:%s", template.JsSourceID)
	cacheClient.Delete(ctx, cacheKey2)

	// 获取更新后的模板
	updatedTemplate, err := models.GetJSTemplateByID(db, templateID)
	if err != nil {
		response.Fail(c, "Failed to get updated template: "+err.Error(), nil)
		return
	}

	response.Success(c, "Template rolled back successfully", updatedTemplate)
}

// PublishJSTemplateVersion 发布版本（支持灰度）
func (h *Handlers) PublishJSTemplateVersion(c *gin.Context) {
	templateID := c.Param("id")
	versionID := c.Param("versionId")
	db := c.MustGet(constants.DbField).(*gorm.DB)

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	var input struct {
		Grayscale int `json:"grayscale"` // 灰度百分比 0-100
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		input.Grayscale = 100 // 默认100%发布
	}

	if input.Grayscale < 0 || input.Grayscale > 100 {
		response.Fail(c, "Invalid grayscale value", "Grayscale must be between 0 and 100")
		return
	}

	// 检查权限
	template, err := models.GetJSTemplateByID(db, templateID)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}

	if template.UserID != user.ID {
		if template.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		var group models.Group
		if err := h.db.Where("id = ?", *template.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *template.GroupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creator or admin can publish")
				return
			}
		}
	}

	// 获取版本
	version, err := models.GetJSTemplateVersion(db, templateID, versionID)
	if err != nil {
		response.Fail(c, "Version not found", nil)
		return
	}

	// 更新版本状态和灰度
	updates := map[string]interface{}{
		"status":    "active",
		"grayscale": input.Grayscale,
	}
	if err := models.UpdateJSTemplateVersion(db, versionID, updates); err != nil {
		response.Fail(c, "Failed to publish version: "+err.Error(), nil)
		return
	}

	// 如果灰度100%，直接更新模板内容
	if input.Grayscale == 100 {
		templateUpdates := map[string]interface{}{
			"content": version.Content,
			"version": template.Version + 1,
		}
		if err := models.UpdateJSTemplate(db, templateID, templateUpdates); err != nil {
			response.Fail(c, "Failed to update template: "+err.Error(), nil)
			return
		}
	}

	// 热更新：清除缓存
	cacheClient := cache.GetGlobalCache()
	ctx := context.Background()
	cacheKey := fmt.Sprintf("js:template:content:%s", templateID)
	cacheClient.Delete(ctx, cacheKey)
	cacheKey2 := fmt.Sprintf("js:template:loader:%s", template.JsSourceID)
	cacheClient.Delete(ctx, cacheKey2)

	response.Success(c, "Version published successfully", gin.H{
		"version_id": versionID,
		"grayscale":  input.Grayscale,
	})
}
