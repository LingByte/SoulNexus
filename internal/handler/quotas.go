package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateUserQuotaRequest creates user quota request
type CreateUserQuotaRequest struct {
	QuotaType   models.QuotaType   `json:"quotaType" binding:"required"`
	TotalQuota  int64              `json:"totalQuota" binding:"required,min=0"`
	Period      models.QuotaPeriod `json:"period"`
	Description string             `json:"description"`
}

// UpdateUserQuotaRequest updates user quota request
type UpdateUserQuotaRequest struct {
	TotalQuota  *int64              `json:"totalQuota"`
	Period      *models.QuotaPeriod `json:"period"`
	Description *string             `json:"description"`
}

// CreateGroupQuotaRequest creates group quota request
type CreateGroupQuotaRequest struct {
	QuotaType   models.QuotaType   `json:"quotaType" binding:"required"`
	TotalQuota  int64              `json:"totalQuota" binding:"required,min=0"`
	Period      models.QuotaPeriod `json:"period"`
	Description string             `json:"description"`
}

// UpdateGroupQuotaRequest updates group quota request
type UpdateGroupQuotaRequest struct {
	TotalQuota  *int64              `json:"totalQuota"`
	Period      *models.QuotaPeriod `json:"period"`
	Description *string             `json:"description"`
}

// ListUserQuotas gets user quota list
func (h *Handlers) ListUserQuotas(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	var quotas []models.UserQuota
	if err := h.db.Where("user_id = ?", user.ID).Find(&quotas).Error; err != nil {
		response.Fail(c, "Query failed", err.Error())
		return
	}

	// Update usage of all quotas (real-time statistics from UsageRecord table)
	for i := range quotas {
		models.UpdateUserQuotaUsage(h.db, user.ID, quotas[i].QuotaType)
		// Re-fetch updated quota
		updatedQuota, _ := models.GetUserQuota(h.db, user.ID, quotas[i].QuotaType)
		if updatedQuota != nil {
			quotas[i] = *updatedQuota
		}
	}

	response.Success(c, "Query successful", quotas)
}

// GetUserQuota gets user quota details
func (h *Handlers) GetUserQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	quotaType := models.QuotaType(c.Param("type"))
	quota, err := models.GetUserQuota(h.db, user.ID, quotaType)
	if err != nil {
		response.Fail(c, "Query failed", err.Error())
		return
	}

	// Update actual usage
	models.UpdateUserQuotaUsage(h.db, user.ID, quotaType)
	quota, _ = models.GetUserQuota(h.db, user.ID, quotaType)

	response.Success(c, "Query successful", quota)
}

// CreateUserQuota creates user quota
func (h *Handlers) CreateUserQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	var req CreateUserQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	var existing models.UserQuota
	if err := h.db.Where("user_id = ? AND quota_type = ?", user.ID, req.QuotaType).First(&existing).Error; err == nil {
		response.Fail(c, "Quota already exists", "This type of quota is already configured, please use update interface")
		return
	}

	quota := models.UserQuota{
		UserID:      user.ID,
		QuotaType:   req.QuotaType,
		TotalQuota:  req.TotalQuota,
		Period:      req.Period,
		Description: req.Description,
	}

	if req.Period == "" {
		quota.Period = models.QuotaPeriodLifetime
	}

	if err := h.db.Create(&quota).Error; err != nil {
		response.Fail(c, "Creation failed", err.Error())
		return
	}

	response.Success(c, "创建成功", quota)
}

// UpdateUserQuota updates user quota
func (h *Handlers) UpdateUserQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	quotaType := models.QuotaType(c.Param("type"))
	var quota models.UserQuota
	if err := h.db.Where("user_id = ? AND quota_type = ?", user.ID, quotaType).First(&quota).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "配额不存在", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	var req UpdateUserQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	if req.TotalQuota != nil {
		quota.TotalQuota = *req.TotalQuota
	}
	if req.Period != nil {
		quota.Period = *req.Period
	}
	if req.Description != nil {
		quota.Description = *req.Description
	}

	if err := h.db.Save(&quota).Error; err != nil {
		response.Fail(c, "Update failed", err.Error())
		return
	}

	response.Success(c, "更新成功", quota)
}

// DeleteUserQuota deletes user quota
func (h *Handlers) DeleteUserQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	quotaType := models.QuotaType(c.Param("type"))
	if err := h.db.Where("user_id = ? AND quota_type = ?", user.ID, quotaType).Delete(&models.UserQuota{}).Error; err != nil {
		response.Fail(c, "Deletion failed", err.Error())
		return
	}

	response.Success(c, "删除成功", nil)
}

// ListGroupQuotas gets group quota list
func (h *Handlers) ListGroupQuotas(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	// Check permissions: only creator or admin can view
	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "组织不存在", nil)
		return
	}

	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以查看配额")
			return
		}
	}

	var quotas []models.GroupQuota
	if err := h.db.Where("group_id = ?", id).Find(&quotas).Error; err != nil {
		response.Fail(c, "Query failed", err.Error())
		return
	}

	for i := range quotas {
		models.UpdateGroupQuotaUsage(h.db, uint(id), quotas[i].QuotaType)
		updatedQuota, _ := models.GetGroupQuota(h.db, uint(id), quotas[i].QuotaType)
		if updatedQuota != nil {
			quotas[i] = *updatedQuota
		}
	}

	response.Success(c, "查询成功", quotas)
}

// GetGroupQuota gets organization quota details
func (h *Handlers) GetGroupQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	quotaType := models.QuotaType(c.Param("type"))

	// Check permissions
	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "组织不存在", nil)
		return
	}

	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以查看配额")
			return
		}
	}

	quota, err := models.GetGroupQuota(h.db, uint(id), quotaType)
	if err != nil {
		response.Fail(c, "Query failed", err.Error())
		return
	}

	// Update actual usage
	models.UpdateGroupQuotaUsage(h.db, uint(id), quotaType)
	quota, _ = models.GetGroupQuota(h.db, uint(id), quotaType)

	response.Success(c, "查询成功", quota)
}

// CreateGroupQuota creates organization quota
func (h *Handlers) CreateGroupQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	// Check permissions: only creator can create quota
	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "组织不存在", nil)
		return
	}

	if group.CreatorID != user.ID {
		response.Fail(c, "权限不足", "只有创建者可以创建配额")
		return
	}

	var req CreateGroupQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	// Check if already exists
	var existing models.GroupQuota
	if err := h.db.Where("group_id = ? AND quota_type = ?", id, req.QuotaType).First(&existing).Error; err == nil {
		response.Fail(c, "配额已存在", "该类型的配额已配置，请使用更新接口")
		return
	}

	quota := models.GroupQuota{
		GroupID:     uint(id),
		QuotaType:   req.QuotaType,
		TotalQuota:  req.TotalQuota,
		Period:      req.Period,
		Description: req.Description,
	}

	if req.Period == "" {
		quota.Period = models.QuotaPeriodLifetime
	}

	if err := h.db.Create(&quota).Error; err != nil {
		response.Fail(c, "Creation failed", err.Error())
		return
	}

	response.Success(c, "创建成功", quota)
}

// UpdateGroupQuota updates organization quota
func (h *Handlers) UpdateGroupQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	quotaType := models.QuotaType(c.Param("type"))

	// 检查权限
	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "组织不存在", nil)
		return
	}

	if group.CreatorID != user.ID {
		response.Fail(c, "权限不足", "只有创建者可以更新配额")
		return
	}

	var quota models.GroupQuota
	if err := h.db.Where("group_id = ? AND quota_type = ?", id, quotaType).First(&quota).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "配额不存在", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	var req UpdateGroupQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	if req.TotalQuota != nil {
		quota.TotalQuota = *req.TotalQuota
	}
	if req.Period != nil {
		quota.Period = *req.Period
	}
	if req.Description != nil {
		quota.Description = *req.Description
	}

	if err := h.db.Save(&quota).Error; err != nil {
		response.Fail(c, "Update failed", err.Error())
		return
	}

	response.Success(c, "更新成功", quota)
}

// DeleteGroupQuota deletes organization quota
func (h *Handlers) DeleteGroupQuota(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}
	quotaType := models.QuotaType(c.Param("type"))
	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "组织不存在", nil)
		return
	}

	if group.CreatorID != user.ID {
		response.Fail(c, "权限不足", "只有创建者可以删除配额")
		return
	}

	if err := h.db.Where("group_id = ? AND quota_type = ?", id, quotaType).Delete(&models.GroupQuota{}).Error; err != nil {
		response.Fail(c, "Deletion failed", err.Error())
		return
	}

	response.Success(c, "删除成功", nil)
}
