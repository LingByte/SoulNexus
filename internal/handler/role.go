package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) handleAdminListRoles(c *gin.Context) {
	base := h.db.Model(&models.Role{}).Where("is_deleted = ?", models.SoftDeleteStatusActive)
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		base = base.Where("LOWER(slug) LIKE ? OR LOWER(name) LIKE ?", like, like)
	}
	var total int64
	_ = base.Count(&total).Error
	page, pageSize := h.parsePagination(c)
	q := base.Session(&gorm.Session{}).Preload("Permissions", "is_deleted = ?", models.SoftDeleteStatusActive)
	var rows []models.Role
	err := q.Order("slug ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	if err != nil {
		response.Fail(c, "list roles failed", err)
		return
	}
	response.Success(c, "success", gin.H{
		"items": rows, "total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var role models.Role
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).
		Preload("Permissions", "is_deleted = ?", models.SoftDeleteStatusActive).
		First(&role).Error; err != nil {
		response.Fail(c, "role not found", err)
		return
	}
	response.Success(c, "success", role)
}

type roleUpsertReq struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (h *Handlers) handleAdminCreateRole(c *gin.Context) {
	var req roleUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.TrimSpace(strings.ToLower(req.Slug))
	if req.Name == "" || req.Slug == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("name and slug required"))
		return
	}
	role := models.Role{Name: req.Name, Slug: req.Slug, Description: req.Description, IsSystem: false}
	role.SetCreateInfo(operatorEmail(c))
	if err := h.db.Create(&role).Error; err != nil {
		response.Fail(c, "create role failed", err)
		return
	}
	response.Success(c, "created", role)
}

func (h *Handlers) handleAdminUpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var req roleUpsertReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var role models.Role
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&role).Error; err != nil {
		response.Fail(c, "role not found", err)
		return
	}
	if role.IsSystem && strings.TrimSpace(req.Slug) != "" && !strings.EqualFold(req.Slug, role.Slug) {
		response.AbortWithJSONError(c, http.StatusForbidden, errors.New("cannot change system role slug"))
		return
	}
	vals := map[string]any{}
	if req.Name != "" {
		vals["name"] = req.Name
	}
	if req.Description != "" {
		vals["description"] = req.Description
	}
	if ns := strings.TrimSpace(strings.ToLower(req.Slug)); ns != "" && !role.IsSystem {
		vals["slug"] = ns
	}
	if len(vals) == 0 {
		response.Success(c, "noop", role)
		return
	}
	vals["update_by"] = operatorEmail(c)
	if err = h.db.Model(&role).Updates(vals).Error; err != nil {
		response.Fail(c, "update failed", err)
		return
	}
	_ = h.db.Preload("Permissions", "is_deleted = ?", models.SoftDeleteStatusActive).First(&role, id).Error
	response.Success(c, "updated", role)
}

func (h *Handlers) handleAdminDeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var role models.Role
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&role).Error; err != nil {
		response.Fail(c, "role not found", err)
		return
	}
	if role.IsSystem {
		response.AbortWithJSONError(c, http.StatusForbidden, errors.New("cannot delete system role"))
		return
	}
	op := operatorEmail(c)
	if err = h.db.Model(&role).Updates(map[string]any{
		"is_deleted": models.SoftDeleteStatusDeleted,
		"update_by":  op,
	}).Error; err != nil {
		response.Fail(c, "delete failed", err)
		return
	}
	_ = h.db.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error
	_ = h.db.Where("role_id = ?", id).Delete(&models.UserRole{}).Error
	response.Success(c, "deleted", nil)
}

type rolePermissionIDsBody struct {
	PermissionIDs []uint `json:"permissionIds"`
}

func (h *Handlers) handleAdminSetRolePermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var body rolePermissionIDsBody
	if err = c.ShouldBindJSON(&body); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var role models.Role
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&role).Error; err != nil {
		response.Fail(c, "role not found", err)
		return
	}
	tx := h.db.Begin()
	if err = tx.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; err != nil {
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
		if err = tx.Create(&models.RolePermission{RoleID: role.ID, PermissionID: pid}).Error; err != nil {
			tx.Rollback()
			response.Fail(c, "failed", err)
			return
		}
	}
	if err = tx.Commit().Error; err != nil {
		response.Fail(c, "commit failed", err)
		return
	}
	_ = h.db.Preload("Permissions", "is_deleted = ?", models.SoftDeleteStatusActive).First(&role, role.ID).Error
	response.Success(c, "updated", role)
}
