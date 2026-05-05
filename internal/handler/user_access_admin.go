package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// handleAdminGetUserAccess 聚合：用户基础信息、角色（含权限）、附加权限、最终权限 key。
func (h *Handlers) handleAdminGetUserAccess(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}
	u, err := models.GetUserByID(h.db, uint(id))
	if err != nil || u == nil {
		response.Fail(c, "user not found", err)
		return
	}
	var urs []models.UserRole
	_ = h.db.Where("user_id = ?", u.ID).Find(&urs).Error
	roleIDs := make([]uint, 0, len(urs))
	for _, x := range urs {
		roleIDs = append(roleIDs, x.RoleID)
	}
	var roles []models.Role
	if len(roleIDs) > 0 {
		_ = h.db.Where("id IN ? AND is_deleted = ?", roleIDs, models.SoftDeleteStatusActive).
			Preload("Permissions", "is_deleted = ?", models.SoftDeleteStatusActive).
			Order("slug ASC").
			Find(&roles).Error
	}
	var ups []models.UserPermission
	_ = h.db.Where("user_id = ?", u.ID).Find(&ups).Error
	pids := make([]uint, 0, len(ups))
	for _, x := range ups {
		pids = append(pids, x.PermissionID)
	}
	var extraPerms []models.Permission
	if len(pids) > 0 {
		_ = h.db.Where("id IN ? AND is_deleted = ?", pids, models.SoftDeleteStatusActive).
			Order("`key` ASC").
			Find(&extraPerms).Error
	}
	effKeys, _ := models.EffectivePermissionKeys(h.db, u.ID)
	response.Success(c, "success", gin.H{
		"user": gin.H{
			"id":          u.ID,
			"email":       u.Email,
			"displayName": u.Profile.DisplayName,
			"legacyRole":  u.Role,
		},
		"roles":                   roles,
		"extraPermissions":        extraPerms,
		"effectivePermissionKeys": effKeys,
	})
}

type userAccessSetBody struct {
	RoleIDs       []uint `json:"roleIds"`
	PermissionIDs []uint `json:"permissionIds"`
}

// handleAdminSetUserAccess 一次性写入用户的角色与附加权限，并同步 users.role。
func (h *Handlers) handleAdminSetUserAccess(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}
	var body userAccessSetBody
	if err = c.ShouldBindJSON(&body); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var u models.User
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&u).Error; err != nil {
		response.Fail(c, "user not found", err)
		return
	}
	tx := h.db.Begin()

	if err = tx.Where("user_id = ?", u.ID).Delete(&models.UserRole{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "failed", err)
		return
	}
	primarySlug := ""
	for _, rid := range body.RoleIDs {
		if rid == 0 {
			continue
		}
		var cnt int64
		if err = tx.Model(&models.Role{}).Where("id = ? AND is_deleted = ?", rid, models.SoftDeleteStatusActive).Count(&cnt).Error; err != nil || cnt == 0 {
			tx.Rollback()
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid role id"))
			return
		}
		if err = tx.Create(&models.UserRole{UserID: u.ID, RoleID: rid}).Error; err != nil {
			tx.Rollback()
			response.Fail(c, "failed", err)
			return
		}
		var role models.Role
		if err = tx.First(&role, rid).Error; err == nil {
			if role.Slug == models.RoleSuperAdmin {
				primarySlug = models.RoleSuperAdmin
			} else if primarySlug != models.RoleSuperAdmin && role.Slug == models.RoleAdmin {
				primarySlug = models.RoleAdmin
			} else if primarySlug == "" {
				primarySlug = role.Slug
			}
		}
	}

	if err = tx.Where("user_id = ?", u.ID).Delete(&models.UserPermission{}).Error; err != nil {
		tx.Rollback()
		response.Fail(c, "failed", err)
		return
	}
	for _, pid := range body.PermissionIDs {
		if pid == 0 {
			continue
		}
		var cnt int64
		if err = tx.Model(&models.Permission{}).Where("id = ? AND is_deleted = ?", pid, models.SoftDeleteStatusActive).Count(&cnt).Error; err != nil || cnt == 0 {
			tx.Rollback()
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid permission id"))
			return
		}
		if err = tx.Create(&models.UserPermission{UserID: u.ID, PermissionID: pid}).Error; err != nil {
			tx.Rollback()
			response.Fail(c, "failed", err)
			return
		}
	}

	if primarySlug != "" && primarySlug != u.Role {
		if err = tx.Model(&u).Update("role", primarySlug).Error; err != nil {
			tx.Rollback()
			response.Fail(c, "failed to sync legacy role", err)
			return
		}
	}

	if err = tx.Commit().Error; err != nil {
		response.Fail(c, "commit failed", err)
		return
	}
	response.Success(c, "updated", nil)
}
