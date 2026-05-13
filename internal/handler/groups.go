package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CreateGroupRequest struct {
	Name       string                 `json:"name" binding:"required"`
	Type       string                 `json:"type"`
	Extra      string                 `json:"extra"`
	Permission models.GroupPermission `json:"permission"`
}

type UpdateGroupRequest struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Extra      string                 `json:"extra"`
	Permission models.GroupPermission `json:"permission"`
}

type InviteUserRequest struct {
	UserID uint `json:"userId" binding:"required"`
}

type SearchUsersRequest struct {
	Keyword string `json:"keyword" form:"keyword"`
	Limit   int    `json:"limit" form:"limit"`
}

type GroupResponse struct {
	ID          uint                   `json:"id"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Extra       string                 `json:"extra,omitempty"`
	Avatar      string                 `json:"avatar,omitempty"` // 组织头像URL
	Permission  models.GroupPermission `json:"permission,omitempty"`
	CreatorID   uint                   `json:"creatorId"`
	Creator     *models.User           `json:"creator,omitempty"`
	MemberCount int64                  `json:"memberCount"`
	MyRole      string                 `json:"myRole,omitempty"`
	Members     []GroupMemberResponse  `json:"members,omitempty"`
}

type GroupMemberResponse struct {
	ID        uint        `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UserID    uint        `json:"userId"`
	User      models.User `json:"user"`
	GroupID   uint        `json:"groupId"`
	Role      string      `json:"role"`
}

type GroupInvitationResponse struct {
	ID        uint         `json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	GroupID   uint         `json:"groupId"`
	Group     models.Group `json:"group"`
	InviterID uint         `json:"inviterId"`
	Inviter   models.User  `json:"inviter"`
	InviteeID uint         `json:"inviteeId"`
	Invitee   models.User  `json:"invitee"`
	Status    string       `json:"status"`
	ExpiresAt *time.Time   `json:"expiresAt,omitempty"`
}

// CreateGroup 创建组织
func (h *Handlers) CreateGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	if strings.TrimSpace(req.Type) == models.GroupTypePersonal {
		response.Fail(c, "参数错误", "个人组织由系统自动创建，不能手动创建")
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = models.GroupTypeTeam
	}

	group := models.Group{
		Name:       req.Name,
		Type:       req.Type,
		Extra:      req.Extra,
		Permission: req.Permission,
		CreatorID:  user.ID,
	}

	if err := h.db.Create(&group).Error; err != nil {
		response.Fail(c, "创建组织失败", err.Error())
		return
	}

	// 自动将创建者添加为管理员
	member := models.GroupMember{
		UserID:  user.ID,
		GroupID: group.ID,
		Role:    models.GroupRoleAdmin,
	}
	if err := h.db.Create(&member).Error; err != nil {
		// 如果创建成员失败，删除组织
		h.db.Delete(&group)
		response.Fail(c, "创建组织失败", "无法添加创建者为成员")
		return
	}

	// 加载创建者信息
	h.db.Preload("Creator").First(&group, group.ID)

	// 记录活动日志
	h.logGroupActivity(group.ID, user.ID, models.ActionGroupCreated, models.ResourceTypeGroup, &group.ID, group.Name, nil, c.ClientIP())

	response.Success(c, "创建组织成功", group)
}

// ListGroups 获取用户创建或加入的组织列表
func (h *Handlers) ListGroups(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	var groups []models.Group
	// 查询用户创建或加入的组织
	if err := h.db.Model(&models.Group{}).
		Joins("LEFT JOIN group_members ON groups.id = group_members.group_id").
		Where("groups.creator_id = ? OR group_members.user_id = ?", user.ID, user.ID).
		Group("groups.id").
		Preload("Creator").
		Find(&groups).Error; err != nil {
		response.Fail(c, "查询组织列表失败", err.Error())
		return
	}

	// 构建响应，包含成员数量和当前用户角色
	var groupResponses []GroupResponse
	for _, group := range groups {
		// 获取成员数量
		var memberCount int64
		h.db.Model(&models.GroupMember{}).Where("group_id = ?", group.ID).Count(&memberCount)

		// 获取当前用户在组织中的角色
		var myRole string
		if group.CreatorID == user.ID {
			// 创建者默认是管理员
			myRole = models.GroupRoleAdmin
		} else {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, user.ID).First(&member).Error; err == nil {
				myRole = member.Role
			}
		}

		groupResponses = append(groupResponses, GroupResponse{
			ID:          group.ID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
			Name:        group.Name,
			Type:        group.Type,
			Extra:       group.Extra,
			Avatar:      group.Avatar,
			Permission:  group.Permission,
			CreatorID:   group.CreatorID,
			Creator:     &group.Creator,
			MemberCount: memberCount,
			MyRole:      myRole,
		})
	}

	response.Success(c, "查询成功", groupResponses)
}

// GetGroup 获取组织详情
func (h *Handlers) GetGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var group models.Group
	if err := h.db.Preload("Creator").First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查用户是否有权限查看（必须是创建者或成员）
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, user.ID).First(&member).Error; err != nil {
		if group.CreatorID != user.ID {
			response.Fail(c, "权限不足", "您不是该组织的成员")
			return
		}
	}

	// 获取成员列表（重新查询以确保获取最新数据）
	var members []models.GroupMember
	if err := h.db.Preload("User").Where("group_id = ?", group.ID).Find(&members).Error; err != nil {
		response.Fail(c, "查询成员列表失败", err.Error())
		return
	}

	// 获取成员数量
	var memberCount int64
	h.db.Model(&models.GroupMember{}).Where("group_id = ?", group.ID).Count(&memberCount)

	// 获取当前用户角色
	var myRole string
	if group.CreatorID == user.ID {
		myRole = models.GroupRoleAdmin
	} else if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, user.ID).First(&member).Error; err == nil {
		myRole = member.Role
	}

	// 构建成员响应
	var memberResponses []GroupMemberResponse
	for _, m := range members {
		memberResponses = append(memberResponses, GroupMemberResponse{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UserID:    m.UserID,
			User:      m.User,
			GroupID:   m.GroupID,
			Role:      m.Role,
		})
	}

	groupResponse := GroupResponse{
		ID:          group.ID,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
		Name:        group.Name,
		Type:        group.Type,
		Extra:       group.Extra,
		Avatar:      group.Avatar,
		Permission:  group.Permission,
		CreatorID:   group.CreatorID,
		Creator:     &group.Creator,
		MemberCount: memberCount,
		MyRole:      myRole,
		Members:     memberResponses,
	}

	response.Success(c, "查询成功", groupResponse)
}

// UpdateGroup 更新组织信息
func (h *Handlers) UpdateGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查权限：只有创建者或管理员可以更新
	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以更新组织信息")
			return
		}
	}

	// 更新字段
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Type != "" {
		group.Type = req.Type
	}
	if req.Extra != "" {
		group.Extra = req.Extra
	}
	if len(req.Permission.Permissions) > 0 {
		group.Permission = req.Permission
	}

	if err := h.db.Save(&group).Error; err != nil {
		response.Fail(c, "更新失败", err.Error())
		return
	}

	h.db.Preload("Creator").First(&group, group.ID)
	response.Success(c, "更新成功", group)
}

// DeleteGroup 删除组织
func (h *Handlers) DeleteGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 只有创建者可以删除组织
	if group.CreatorID != user.ID {
		response.Fail(c, "权限不足", "只有创建者可以删除组织")
		return
	}

	// 删除组织成员
	h.db.Where("group_id = ?", group.ID).Delete(&models.GroupMember{})
	// 删除组织邀请
	h.db.Where("group_id = ?", group.ID).Delete(&models.GroupInvitation{})
	// 删除组织
	if err := h.db.Delete(&group).Error; err != nil {
		response.Fail(c, "删除失败", err.Error())
		return
	}

	response.Success(c, "删除成功", nil)
}

// SearchUsers 搜索用户（用于邀请）
func (h *Handlers) SearchUsers(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	keyword := c.Query("keyword")
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	if keyword == "" {
		response.Fail(c, "参数错误", "搜索关键词不能为空")
		return
	}

	// 模糊搜索用户：通过名字、邮箱、显示名称搜索（资料在 user_profiles）
	type userSearchRow struct {
		ID          uint      `gorm:"column:id"`
		Email       string    `gorm:"column:email"`
		DisplayName string    `gorm:"column:display_name"`
		FirstName   string    `gorm:"column:first_name"`
		LastName    string    `gorm:"column:last_name"`
		Avatar      string    `gorm:"column:avatar"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}
	var rows []userSearchRow
	q := h.db.Table("users").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.is_deleted = ?", models.SoftDeleteStatusActive).
		Where("users.status = ?", models.UserStatusActive).
		Where("users.id != ?", user.ID).
		Where(`user_profiles.display_name LIKE ? OR users.email LIKE ? OR user_profiles.first_name LIKE ? OR user_profiles.last_name LIKE ?`,
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).
		Select("users.id, users.email, user_profiles.display_name AS display_name, user_profiles.first_name AS first_name, user_profiles.last_name AS last_name, user_profiles.avatar AS avatar, users.created_at")

	if err := q.Scan(&rows).Error; err != nil {
		response.Fail(c, "搜索失败", err.Error())
		return
	}

	type UserSearchResult struct {
		ID          uint      `json:"id"`
		Email       string    `json:"email"`
		DisplayName string    `json:"displayName"`
		FirstName   string    `json:"firstName"`
		LastName    string    `json:"lastName"`
		Avatar      string    `json:"avatar"`
		CreatedAt   time.Time `json:"createdAt"`
	}

	results := make([]UserSearchResult, 0, len(rows))
	for _, r := range rows {
		results = append(results, UserSearchResult{
			ID:          r.ID,
			Email:       r.Email,
			DisplayName: r.DisplayName,
			FirstName:   r.FirstName,
			LastName:    r.LastName,
			Avatar:      r.Avatar,
			CreatedAt:   r.CreatedAt,
		})
	}

	response.Success(c, "搜索成功", results)
}

// InviteUser 邀请用户加入组织
func (h *Handlers) InviteUser(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var req InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查权限：只有创建者或管理员可以邀请
	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以邀请用户")
			return
		}
	}

	// 查找被邀请用户（含 profile，用于通知与邮件偏好）
	invitee, err := models.GetUserByID(h.db, req.UserID)
	if err != nil || invitee == nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "用户不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查用户是否已经是成员
	var existingMember models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, invitee.ID).First(&existingMember).Error; err == nil {
		response.Fail(c, "用户已是成员", nil)
		return
	}

	// 检查是否已有待处理的邀请
	var existingInvitation models.GroupInvitation
	if err := h.db.Where("group_id = ? AND invitee_id = ? AND status = ?", group.ID, invitee.ID, "pending").First(&existingInvitation).Error; err == nil {
		response.Fail(c, "邀请已存在", "该用户已有待处理的邀请")
		return
	}

	// 创建邀请
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7天后过期
	invitation := models.GroupInvitation{
		GroupID:   group.ID,
		InviterID: user.ID,
		InviteeID: invitee.ID,
		Status:    "pending",
		ExpiresAt: &expiresAt,
	}

	if err := h.db.Create(&invitation).Error; err != nil {
		response.Fail(c, "创建邀请失败", err.Error())
		return
	}

	// 加载关联信息
	h.db.Preload("Group").Preload("Inviter.Profile").Preload("Invitee.Profile").First(&invitation, invitation.ID)

	// 发送站内通知
	go func() {
		notificationService := models.NewInternalNotificationService(h.db)
		title := "组织邀请"
		content := fmt.Sprintf("%s 邀请您加入组织「%s」",
			user.EffectiveDisplayName(),
			group.Name)
		if err := notificationService.Send(invitee.ID, title, content); err != nil {
			logger.Warn("发送站内通知失败", zap.Error(err), zap.Uint("userId", invitee.ID))
		}
	}()

	// 发送邮件通知（如果用户启用了邮件通知）
	go func() {
		if invitee.Profile.EmailNotifications {
			mailer := notification.NewMailer(h.db, 0, invitee.ID, "")

			// 构建接受邀请的URL
			siteURL := utils.GetValue(h.db, constants.KEY_SITE_URL)
			if siteURL == "" {
				siteURL = "http://localhost:3000"
			}
			acceptURL := fmt.Sprintf("%s/groups?invitation=%d", siteURL, invitation.ID)

			// 获取组织描述（截取前50个字符）
			groupDesc := group.Extra
			if len(groupDesc) > 50 {
				groupDesc = groupDesc[:50] + "..."
			}

			err := mailer.SendGroupInvitationEmail(
				invitee.Email,
				invitee.EffectiveDisplayName(),
				user.EffectiveDisplayName(),
				group.Name,
				group.Type,
				groupDesc,
				acceptURL,
			)

			if err != nil {
				logger.Error("发送组织邀请邮件失败", zap.Error(err), zap.String("email", invitee.Email))
			} else {
				logger.Info("组织邀请邮件发送成功", zap.String("email", invitee.Email))
			}
		}
	}()

	response.Success(c, "邀请已发送", invitation)
}

// ListInvitations 获取用户的邀请列表
func (h *Handlers) ListInvitations(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 获取用户收到的邀请
	var invitations []models.GroupInvitation
	if err := h.db.Where("invitee_id = ? AND status = ?", user.ID, "pending").
		Preload("Group").
		Preload("Inviter").
		Order("created_at desc").
		Find(&invitations).Error; err != nil {
		response.Fail(c, "查询失败", err.Error())
		return
	}

	// 过滤已过期的邀请
	var validInvitations []GroupInvitationResponse
	now := time.Now()
	for _, inv := range invitations {
		if inv.ExpiresAt != nil && inv.ExpiresAt.Before(now) {
			// 标记为过期
			inv.Status = "expired"
			h.db.Save(&inv)
			continue
		}
		validInvitations = append(validInvitations, GroupInvitationResponse{
			ID:        inv.ID,
			CreatedAt: inv.CreatedAt,
			UpdatedAt: inv.UpdatedAt,
			GroupID:   inv.GroupID,
			Group:     inv.Group,
			InviterID: inv.InviterID,
			Inviter:   inv.Inviter,
			InviteeID: inv.InviteeID,
			Invitee:   inv.Invitee,
			Status:    inv.Status,
			ExpiresAt: inv.ExpiresAt,
		})
	}

	response.Success(c, "查询成功", validInvitations)
}

// AcceptInvitation 接受邀请
func (h *Handlers) AcceptInvitation(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的邀请ID")
		return
	}

	var invitation models.GroupInvitation
	if err := h.db.Preload("Group").First(&invitation, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "邀请不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查是否是当前用户的邀请
	if invitation.InviteeID != user.ID {
		response.Fail(c, "权限不足", "这不是您的邀请")
		return
	}

	// 检查邀请状态
	if invitation.Status != "pending" {
		response.Fail(c, "邀请已处理", "该邀请已被处理")
		return
	}

	// 检查是否过期
	if invitation.ExpiresAt != nil && invitation.ExpiresAt.Before(time.Now()) {
		response.Fail(c, "邀请已过期", nil)
		return
	}

	// 检查用户是否已经是成员
	var existingMember models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", invitation.GroupID, user.ID).First(&existingMember).Error; err == nil {
		// 用户已是成员，更新邀请状态
		invitation.Status = "accepted"
		h.db.Save(&invitation)
		response.Success(c, "您已是该组织的成员", nil)
		return
	}

	// 创建成员记录
	member := models.GroupMember{
		UserID:  user.ID,
		GroupID: invitation.GroupID,
		Role:    models.GroupRoleMember,
	}

	if err := h.db.Create(&member).Error; err != nil {
		response.Fail(c, "加入组织失败", err.Error())
		return
	}

	// 更新邀请状态
	invitation.Status = "accepted"
	h.db.Save(&invitation)

	response.Success(c, "成功加入组织", nil)
}

// RejectInvitation 拒绝邀请
func (h *Handlers) RejectInvitation(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的邀请ID")
		return
	}

	var invitation models.GroupInvitation
	if err := h.db.First(&invitation, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "邀请不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查是否是当前用户的邀请
	if invitation.InviteeID != user.ID {
		response.Fail(c, "权限不足", "这不是您的邀请")
		return
	}

	// 检查邀请状态
	if invitation.Status != "pending" {
		response.Fail(c, "邀请已处理", "该邀请已被处理")
		return
	}

	// 更新邀请状态
	invitation.Status = "rejected"
	h.db.Save(&invitation)

	response.Success(c, "已拒绝邀请", nil)
}

// LeaveGroup 离开组织
func (h *Handlers) LeaveGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 创建者不能离开组织，只能删除组织
	if group.CreatorID == user.ID {
		response.Fail(c, "无法离开", "创建者不能离开组织，请删除组织")
		return
	}

	// 删除成员记录
	if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, user.ID).Delete(&models.GroupMember{}).Error; err != nil {
		response.Fail(c, "离开组织失败", err.Error())
		return
	}

	response.Success(c, "已离开组织", nil)
}

// RemoveMember 移除成员（仅管理员）
func (h *Handlers) RemoveMember(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的成员ID")
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查权限：只有创建者或管理员可以移除成员
	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以移除成员")
			return
		}
	}

	// 不能移除创建者
	if group.CreatorID == uint(memberID) {
		response.Fail(c, "无法移除", "不能移除组织创建者")
		return
	}

	// 删除成员记录
	if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, memberID).Delete(&models.GroupMember{}).Error; err != nil {
		response.Fail(c, "移除成员失败", err.Error())
		return
	}

	response.Success(c, "已移除成员", nil)
}

// UpdateMemberRole 更新成员角色（仅创建者或管理员）
func (h *Handlers) UpdateMemberRole(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的成员ID")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 验证角色
	if req.Role != models.GroupRoleAdmin && req.Role != models.GroupRoleMember {
		response.Fail(c, "参数错误", "无效的角色")
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查权限：只有创建者或管理员可以更新成员角色
	if group.CreatorID != user.ID {
		var adminMember models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&adminMember).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以更新成员角色")
			return
		}
	}

	// 不能修改创建者的角色
	if group.CreatorID == uint(memberID) {
		response.Fail(c, "无法修改", "不能修改组织创建者的角色")
		return
	}

	// 查找成员
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND id = ?", group.ID, memberID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "成员不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 更新角色
	member.Role = req.Role
	if err := h.db.Save(&member).Error; err != nil {
		response.Fail(c, "更新角色失败", err.Error())
		return
	}

	response.Success(c, "角色更新成功", nil)
}

// GetGroupSharedResources 获取组织共享的资源（助手和知识库）
func (h *Handlers) GetGroupSharedResources(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查权限：只有创建者或管理员可以查看资源
	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以查看组织资源")
			return
		}
	}

	// 查询组织共享的助手
	var assistants []models.Agent
	if err := h.db.Where("group_id = ?", groupID).Order("created_at DESC").Find(&assistants).Error; err != nil {
		response.Fail(c, "查询助手失败", err.Error())
		return
	}

	// 构建响应
	result := map[string]interface{}{
		"agents": assistants,
	}

	response.Success(c, "获取成功", result)
}

// UploadGroupAvatar 上传组织头像
func (h *Handlers) UploadGroupAvatar(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查权限：只有创建者或管理员可以上传头像
	if group.CreatorID != user.ID {
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", group.ID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", "只有创建者或管理员可以上传组织头像")
			return
		}
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Fail(c, "获取上传文件失败", err.Error())
		return
	}
	defer file.Close()

	// 验证文件类型
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		response.Fail(c, "无效的文件类型", "只允许上传 jpeg, jpg, png, gif, webp 格式的图片")
		return
	}

	// 验证文件大小 (最大5MB)
	maxSize := int64(5 * 1024 * 1024)
	if header.Size > maxSize {
		response.Fail(c, "文件过大", "文件大小不能超过5MB")
		return
	}

	// 生成文件名
	fileExt := filepath.Ext(header.Filename)
	if fileExt == "" {
		fileExt = ".jpg"
	}
	fileName := fmt.Sprintf("group_avatars/%d_%d%s", group.ID, time.Now().Unix(), fileExt)

	// 使用本地存储
	//store := stores.Default()

	// 如果组织已有头像，删除旧头像
	//if group.Avatar != "" {
	//	oldKey := extractKeyFromURL(group.Avatar)
	//	if oldKey != "" {
	//		store.Delete(oldKey)
	//	}
	//}

	// 上传新头像
	//err = store.Write(fileName, file)
	//if err != nil {
	//	response.Fail(c, "上传头像失败", err.Error())
	//	return
	//}
	st := stores.Default()
	if err := st.Write(fileName, file); err != nil {
		response.Fail(c, "上传头像失败", err.Error())
		return
	}

	// 获取头像URL
	avatarURL := strings.TrimSpace(st.PublicURL(fileName))

	// 保存相对路径用于返回
	avatarRelativePath := avatarURL

	// 如果是相对路径，转换为完整URL用于数据库存储
	if strings.HasPrefix(avatarURL, "/") {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		if host == "" {
			host = "localhost:7072"
		}
		avatarURL = fmt.Sprintf("%s://%s%s", scheme, host, avatarURL)
	}

	// 更新组织头像
	if err := h.db.Model(&group).Update("avatar", avatarURL).Error; err != nil {
		//store.Delete(fileName)
		response.Fail(c, "更新组织头像失败", err.Error())
		return
	}

	// 返回相对路径，方便反向代理
	response.Success(c, "头像上传成功", gin.H{
		"avatar": avatarRelativePath,
	})
}

// extractKeyFromURL 从URL中提取文件路径
func extractKeyFromURL(url string) string {
	if url == "" {
		return ""
	}

	if strings.Contains(url, "/avatars/") {
		parts := strings.Split(url, "/avatars/")
		if len(parts) > 1 {
			return "avatars/" + parts[1]
		}
	}

	// 简单实现：如果URL包含路径，提取路径部分
	if strings.Contains(url, "/") {
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return strings.Join(parts[len(parts)-2:], "/") // 返回最后两部分（目录+文件名）
		}
	}
	return ""
}

// GetGroupStatistics 获取组织统计数据
func (h *Handlers) GetGroupStatistics(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "参数错误", "无效的组织ID")
		return
	}

	// 检查用户是否有权限查看（必须是成员）
	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
		} else {
			response.Fail(c, "查询失败", err.Error())
		}
		return
	}

	// 检查用户是否是成员
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", group.ID, user.ID).First(&member).Error; err != nil {
		if group.CreatorID != user.ID {
			response.Fail(c, "权限不足", "您不是该组织的成员")
			return
		}
	}

	// 使用 goroutine 并行查询所有 COUNT，大幅提升性能
	type countResult struct {
		memberCount     int64
		assistantCount  int64
		deviceCount     int64
		jsTemplateCount int64
		voiceCloneCount int64
		workflowCount   int64
		assistantIDs    []uint
		callCount       int64
	}

	countChan := make(chan countResult, 1)
	go func() {
		var result countResult
		var wg sync.WaitGroup

		// 并行执行所有 COUNT 查询
		wg.Add(7)

		go func() {
			defer wg.Done()
			h.db.Model(&models.GroupMember{}).Where("group_id = ?", group.ID).Count(&result.memberCount)
		}()

		go func() {
			defer wg.Done()
			h.db.Model(&models.Agent{}).Where("group_id = ?", group.ID).Count(&result.assistantCount)
		}()

		go func() {
			defer wg.Done()
			h.db.Model(&models.Device{}).Where("group_id = ?", group.ID).Count(&result.deviceCount)
		}()

		go func() {
			defer wg.Done()
			h.db.Model(&models.JSTemplate{}).Where("group_id = ?", group.ID).Count(&result.jsTemplateCount)
		}()

		go func() {
			defer wg.Done()
			h.db.Model(&models.VoiceClone{}).Where("group_id = ?", group.ID).Count(&result.voiceCloneCount)
		}()

		go func() {
			defer wg.Done()
			h.db.Model(&models.WorkflowDefinition{}).Where("group_id = ?", group.ID).Count(&result.workflowCount)
		}()

		go func() {
			defer wg.Done()
			var ids []uint
			h.db.Model(&models.Agent{}).Where("group_id = ?", group.ID).Pluck("id", &ids)
			result.assistantIDs = ids

			// 统计通话记录数量（通过助手关联）
			if len(ids) > 0 {
				var assistantIDsInt64 []int64
				for _, id := range ids {
					assistantIDsInt64 = append(assistantIDsInt64, int64(id))
				}
				h.db.Model(&models.ChatSessionLog{}).
					Where("agent_id IN (?) AND chat_type = ?", assistantIDsInt64, "realtime").
					Count(&result.callCount)
			}
		}()

		wg.Wait()
		countChan <- result
	}()

	counts := <-countChan
	memberCount := counts.memberCount
	assistantCount := counts.assistantCount
	deviceCount := counts.deviceCount
	jsTemplateCount := counts.jsTemplateCount
	voiceCloneCount := counts.voiceCloneCount
	workflowCount := counts.workflowCount
	assistantIDs := counts.assistantIDs
	callCount := counts.callCount

	// 转换 assistantIDs 为 int64 数组（用于后续查询）
	var assistantIDsInt64 []int64
	for _, id := range assistantIDs {
		assistantIDsInt64 = append(assistantIDsInt64, int64(id))
	}

	// 获取账单统计（使用量数据，不是账单数量）
	billStats := map[string]interface{}{
		"totalLLMCalls":     int64(0),
		"totalLLMTokens":    int64(0),
		"totalCallDuration": int64(0),
		"totalCallCount":    int64(0),
		"totalASRDuration":  int64(0),
		"totalASRCount":     int64(0),
		"totalTTSDuration":  int64(0),
		"totalTTSCount":     int64(0),
		"totalAPICalls":     int64(0),
	}

	// 获取组织下助手的使用量统计（最近30天）- 使用并行查询优化性能
	if len(assistantIDsInt64) > 0 {
		startTime := time.Now().AddDate(0, 0, -30)
		endTime := time.Now()

		var billStatsWg sync.WaitGroup
		billStatsWg.Add(6)

		// LLM统计
		go func() {
			defer billStatsWg.Done()
			var llmStats struct {
				Count       int64
				TotalTokens int64
			}
			h.db.Model(&models.UsageRecord{}).
				Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
					assistantIDsInt64, models.UsageTypeLLM, startTime, endTime).
				Select("COUNT(*) as count, COALESCE(SUM(total_tokens), 0) as total_tokens").
				Scan(&llmStats)
			billStats["totalLLMCalls"] = llmStats.Count
			billStats["totalLLMTokens"] = llmStats.TotalTokens
		}()

		// 通话统计
		go func() {
			defer billStatsWg.Done()
			var callStats struct {
				Count    int64
				Duration int64
			}
			h.db.Model(&models.UsageRecord{}).
				Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
					assistantIDsInt64, models.UsageTypeCall, startTime, endTime).
				Select("COUNT(*) as count, COALESCE(SUM(call_duration), 0) as duration").
				Scan(&callStats)
			billStats["totalCallCount"] = callStats.Count
			billStats["totalCallDuration"] = callStats.Duration
		}()

		// ASR统计
		go func() {
			defer billStatsWg.Done()
			var asrStats struct {
				Count    int64
				Duration int64
			}
			h.db.Model(&models.UsageRecord{}).
				Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
					assistantIDsInt64, models.UsageTypeASR, startTime, endTime).
				Select("COUNT(*) as count, COALESCE(SUM(audio_duration), 0) as duration").
				Scan(&asrStats)
			billStats["totalASRCount"] = asrStats.Count
			billStats["totalASRDuration"] = asrStats.Duration
		}()

		// TTS统计
		go func() {
			defer billStatsWg.Done()
			var ttsStats struct {
				Count    int64
				Duration int64
			}
			h.db.Model(&models.UsageRecord{}).
				Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
					assistantIDsInt64, models.UsageTypeTTS, startTime, endTime).
				Select("COUNT(*) as count, COALESCE(SUM(audio_duration), 0) as duration").
				Scan(&ttsStats)
			billStats["totalTTSCount"] = ttsStats.Count
			billStats["totalTTSDuration"] = ttsStats.Duration
		}()

		// API统计
		go func() {
			defer billStatsWg.Done()
			var apiStats struct {
				Count int64
			}
			h.db.Model(&models.UsageRecord{}).
				Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
					assistantIDsInt64, models.UsageTypeAPI, startTime, endTime).
				Select("COUNT(*) as count").
				Scan(&apiStats)
			billStats["totalAPICalls"] = apiStats.Count
		}()

		billStatsWg.Wait()
	}

	// 生成真实的图表数据（最近7天的每日使用量趋势）- 使用单个聚合查询优化
	chartData := []map[string]interface{}{}
	weekDays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	now := time.Now()

	if len(assistantIDsInt64) > 0 {
		// 使用单个查询获取7天的数据，大幅减少查询次数
		type dailyStats struct {
			Date      time.Time
			LLMCalls  int64
			Tokens    int64
			CallCount int64
		}

		startDate := now.AddDate(0, 0, -6)
		dayStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
		dayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

		// 使用 DATE() 函数按天分组，一次性获取所有数据
		var llmDailyData []struct {
			Date   string
			Count  int64
			Tokens int64
		}
		h.db.Model(&models.UsageRecord{}).
			Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
				assistantIDsInt64, models.UsageTypeLLM, dayStart, dayEnd).
			Select("DATE(usage_time) as date, COUNT(*) as count, COALESCE(SUM(total_tokens), 0) as tokens").
			Group("DATE(usage_time)").
			Scan(&llmDailyData)

		var callDailyData []struct {
			Date  string
			Count int64
		}
		h.db.Model(&models.UsageRecord{}).
			Where("agent_id IN (?) AND usage_type = ? AND usage_time >= ? AND usage_time <= ?",
				assistantIDsInt64, models.UsageTypeCall, dayStart, dayEnd).
			Select("DATE(usage_time) as date, COUNT(*) as count").
			Group("DATE(usage_time)").
			Scan(&callDailyData)

		// 构建日期到统计数据的映射
		llmMap := make(map[string]struct{ Count, Tokens int64 })
		for _, d := range llmDailyData {
			llmMap[d.Date] = struct{ Count, Tokens int64 }{Count: d.Count, Tokens: d.Tokens}
		}

		callMap := make(map[string]int64)
		for _, d := range callDailyData {
			callMap[d.Date] = d.Count
		}

		// 生成7天的数据
		for i := 6; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			dateStr := date.Format("2006-01-02")

			llmData, hasLLM := llmMap[dateStr]
			callCount, hasCall := callMap[dateStr]

			chartData = append(chartData, map[string]interface{}{
				"name": weekDays[date.Weekday()],
				"value": func() int64 {
					if hasLLM {
						return llmData.Tokens
					}
					return 0
				}(),
				"count": func() int64 {
					if hasLLM {
						return llmData.Count
					}
					return 0
				}(),
				"calls": func() int64 {
					if hasCall {
						return callCount
					}
					return 0
				}(),
			})
		}
	} else {
		// 如果没有助手，返回空数据
		for i := 6; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			chartData = append(chartData, map[string]interface{}{
				"name":  weekDays[date.Weekday()],
				"value": int64(0),
				"count": int64(0),
				"calls": int64(0),
			})
		}
	}

	// 生成真实的表格数据（组织资源列表）
	tableRows := [][]interface{}{}

	// 助手数据
	var assistants []models.Agent
	h.db.Where("group_id = ?", group.ID).Order("created_at DESC").Limit(5).Find(&assistants)
	for _, a := range assistants {
		tableRows = append(tableRows, []interface{}{
			a.Name,
			"AI助手",
			"运行中",
			"-",
			a.CreatedAt.Format("2006-01-02"),
		})
	}

	// 工作流数据
	var workflows []models.WorkflowDefinition
	h.db.Where("group_id = ?", group.ID).Order("created_at DESC").Limit(3).Find(&workflows)
	for _, w := range workflows {
		tableRows = append(tableRows, []interface{}{
			w.Name,
			"工作流",
			w.Status,
			"-",
			w.CreatedAt.Format("2006-01-02"),
		})
	}

	tableData := map[string]interface{}{
		"columns": []string{"名称", "类型", "状态", "数量", "日期"},
		"rows":    tableRows,
	}

	// 生成真实的活动流数据（最近的活动记录）
	recentActivity := []map[string]interface{}{}

	// 从助手表获取最近创建的活动
	var recentAssistants []models.Agent
	h.db.Where("group_id = ?", group.ID).Order("created_at DESC").Limit(3).Find(&recentAssistants)
	for _, a := range recentAssistants {
		recentActivity = append(recentActivity, map[string]interface{}{
			"type":        "create",
			"description": fmt.Sprintf("创建了助手: %s", a.Name),
			"time":        a.CreatedAt.Format("2006-01-02 15:04:05"),
			"user":        user.EffectiveDisplayName(),
		})
	}

	// 按时间排序
	sort.Slice(recentActivity, func(i, j int) bool {
		timeI, _ := time.Parse("2006-01-02 15:04:05", recentActivity[i]["time"].(string))
		timeJ, _ := time.Parse("2006-01-02 15:04:05", recentActivity[j]["time"].(string))
		return timeI.After(timeJ)
	})

	// 只取最近5条
	if len(recentActivity) > 5 {
		recentActivity = recentActivity[:5]
	}

	stats := map[string]interface{}{
		"totalMembers":    memberCount,
		"totalAssistants": assistantCount,
		"totalDevices":    deviceCount,
		"totalScripts":    jsTemplateCount, // JS模板
		"totalVoices":     voiceCloneCount, // 音色
		"totalWorkflows":  workflowCount,
		"totalCalls":      callCount,
		"billStatistics":  billStats, // 账单统计（使用量数据）
		"recentActivity":  recentActivity,
		"chartData":       chartData,
		"usageTrend":      chartData,
		"activityData":    chartData,
		"table":           tableData,
	}

	response.Success(c, "查询成功", stats)
}

// ArchiveGroup 归档组织
func (h *Handlers) ArchiveGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", nil)
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, "无效的组织ID", nil)
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
			return
		}
		response.Fail(c, "查询失败", err.Error())
		return
	}

	// 检查权限（只有创建者可以归档）
	if group.CreatorID != user.ID {
		response.Fail(c, "权限不足", "只有创建者可以归档组织")
		return
	}

	// 归档组织
	now := time.Now()
	group.IsArchived = true
	group.ArchivedAt = &now
	group.ArchivedBy = &user.ID

	if err := h.db.Save(&group).Error; err != nil {
		response.Fail(c, "归档失败", err.Error())
		return
	}

	// 记录日志
	h.logGroupActivity(group.ID, user.ID, models.ActionGroupArchived, models.ResourceTypeGroup, &group.ID, group.Name, nil, c.ClientIP())

	response.Success(c, "组织已归档", group)
}

// RestoreGroup 恢复归档的组织
func (h *Handlers) RestoreGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", nil)
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, "无效的组织ID", nil)
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
			return
		}
		response.Fail(c, "查询失败", err.Error())
		return
	}

	// 检查权限
	if group.CreatorID != user.ID {
		response.Fail(c, "权限不足", "只有创建者可以恢复组织")
		return
	}

	// 恢复组织
	group.IsArchived = false
	group.ArchivedAt = nil
	group.ArchivedBy = nil

	if err := h.db.Save(&group).Error; err != nil {
		response.Fail(c, "恢复失败", err.Error())
		return
	}

	// 记录日志
	h.logGroupActivity(group.ID, user.ID, models.ActionGroupRestored, models.ResourceTypeGroup, &group.ID, group.Name, nil, c.ClientIP())

	response.Success(c, "组织已恢复", group)
}

// CloneGroup 克隆组织
func (h *Handlers) CloneGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", nil)
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, "无效的组织ID", nil)
		return
	}

	var sourceGroup models.Group
	if err := h.db.First(&sourceGroup, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
			return
		}
		response.Fail(c, "查询失败", err.Error())
		return
	}

	// 检查权限（只有创建者或管理员可以克隆）
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", groupID, user.ID).First(&member).Error; err != nil {
		response.Fail(c, "权限不足", "您不是该组织的成员")
		return
	}

	if sourceGroup.CreatorID != user.ID && member.Role != models.GroupRoleAdmin {
		response.Fail(c, "权限不足", "只有创建者或管理员可以克隆组织")
		return
	}

	// 创建新组织
	newGroup := models.Group{
		Name:       sourceGroup.Name + " (副本)",
		Type:       sourceGroup.Type,
		Extra:      sourceGroup.Extra,
		Permission: sourceGroup.Permission,
		CreatorID:  user.ID,
		ClonedFrom: &sourceGroup.ID,
	}

	if err := h.db.Create(&newGroup).Error; err != nil {
		response.Fail(c, "克隆失败", err.Error())
		return
	}

	// 添加创建者为成员
	newMember := models.GroupMember{
		UserID:  user.ID,
		GroupID: newGroup.ID,
		Role:    models.GroupRoleAdmin,
	}
	h.db.Create(&newMember)

	// 记录日志
	details := map[string]interface{}{
		"source_group_id":   sourceGroup.ID,
		"source_group_name": sourceGroup.Name,
	}
	h.logGroupActivity(newGroup.ID, user.ID, models.ActionGroupCloned, models.ResourceTypeGroup, &newGroup.ID, newGroup.Name, details, c.ClientIP())

	response.Success(c, "组织克隆成功", newGroup)
}

// ExportGroup 导出组织数据
func (h *Handlers) ExportGroup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", nil)
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, "无效的组织ID", nil)
		return
	}

	var group models.Group
	if err := h.db.Preload("Creator").First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "组织不存在", nil)
			return
		}
		response.Fail(c, "查询失败", err.Error())
		return
	}

	// 检查权限
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", groupID, user.ID).First(&member).Error; err != nil {
		response.Fail(c, "权限不足", "您不是该组织的成员")
		return
	}

	// 获取成员列表
	var members []models.GroupMember
	h.db.Where("group_id = ?", groupID).Preload("User.Profile").Find(&members)

	// 获取活动日志（最近100条）
	var logs []models.GroupActivityLog
	h.db.Where("group_id = ?", groupID).
		Preload("User.Profile").
		Order("created_at DESC").
		Limit(100).
		Find(&logs)

	// 构建导出数据
	exportData := map[string]interface{}{
		"group": map[string]interface{}{
			"id":         group.ID,
			"name":       group.Name,
			"type":       group.Type,
			"extra":      group.Extra,
			"createdAt":  group.CreatedAt,
			"creator":    group.Creator.Email,
			"isArchived": group.IsArchived,
		},
		"members": func() []map[string]interface{} {
			result := make([]map[string]interface{}, len(members))
			for i, m := range members {
				result[i] = map[string]interface{}{
					"email":       m.User.Email,
					"displayName": m.User.EffectiveDisplayName(),
					"role":        m.Role,
					"joinedAt":    m.CreatedAt,
				}
			}
			return result
		}(),
		"activityLogs": func() []map[string]interface{} {
			result := make([]map[string]interface{}, len(logs))
			for i, l := range logs {
				result[i] = map[string]interface{}{
					"action":       l.Action,
					"resourceType": l.ResourceType,
					"resourceName": l.ResourceName,
					"user":         l.User.Email,
					"createdAt":    l.CreatedAt,
					"details":      l.Details,
				}
			}
			return result
		}(),
		"exportedAt": time.Now(),
		"exportedBy": user.Email,
	}

	// 记录日志
	h.logGroupActivity(group.ID, user.ID, models.ActionGroupExported, models.ResourceTypeGroup, &group.ID, group.Name, nil, c.ClientIP())

	// 返回JSON数据
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=group_%d_export_%s.json", groupID, time.Now().Format("20060102_150405")))
	response.Success(c, "导出成功", exportData)
}

// GetGroupActivityLogs 获取组织活动日志
func (h *Handlers) GetGroupActivityLogs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", nil)
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, "无效的组织ID", nil)
		return
	}

	// 检查权限
	var member models.GroupMember
	if err := h.db.Where("group_id = ? AND user_id = ?", groupID, user.ID).First(&member).Error; err != nil {
		response.Fail(c, "权限不足", "您不是该组织的成员")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	action := c.Query("action")
	resourceType := c.Query("resourceType")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询
	query := h.db.Where("group_id = ?", groupID)
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	// 获取总数
	var total int64
	query.Model(&models.GroupActivityLog{}).Count(&total)

	// 获取日志列表
	var logs []models.GroupActivityLog
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		response.Fail(c, "查询失败", err.Error())
		return
	}

	response.Success(c, "查询成功", map[string]interface{}{
		"logs":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// logGroupActivity 记录组织活动日志（辅助函数）
func (h *Handlers) logGroupActivity(groupID uint, userID uint, action string, resourceType string, resourceID *uint, resourceName string, details interface{}, ipAddress string) {
	detailsJSON := ""
	if details != nil {
		if jsonBytes, err := json.Marshal(details); err == nil {
			detailsJSON = string(jsonBytes)
		}
	}

	log := models.GroupActivityLog{
		GroupID:      groupID,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Details:      detailsJSON,
		IPAddress:    ipAddress,
	}

	h.db.Create(&log)
}
