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

type permissionListItem struct {
	models.Permission
	RoleCount            int64 `json:"roleCount"`
	DirectUserGrantCount int64 `json:"directUserGrantCount"`
}

func (h *Handlers) handleAdminListPermissions(c *gin.Context) {
	base := h.db.Model(&models.Permission{}).Where("is_deleted = ?", models.SoftDeleteStatusActive)
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		base = base.Where("LOWER(`key`) LIKE ? OR LOWER(name) LIKE ?", like, like)
	}
	var total int64
	_ = base.Count(&total).Error
	withRoles := c.Query("withRoles") == "1" || strings.EqualFold(c.Query("withRoles"), "true")
	page, pageSize := h.parsePagination(c)
	q := base.Session(&gorm.Session{})
	if withRoles {
		q = q.Preload("Roles", "is_deleted = ?", models.SoftDeleteStatusActive)
	}
	var rows []models.Permission
	err := q.Order("`key` ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	if err != nil {
		response.Fail(c, "list permissions failed", err)
		return
	}
	ids := make([]uint, 0, len(rows))
	for _, p := range rows {
		ids = append(ids, p.ID)
	}
	roleCnt := map[uint]int64{}
	userCnt := map[uint]int64{}
	if len(ids) > 0 {
		type cntRow struct {
			PermissionID uint  `gorm:"column:permission_id"`
			N            int64 `gorm:"column:n"`
		}
		var rc []cntRow
		_ = h.db.Raw(`
			SELECT permission_id, COUNT(*) AS n FROM role_permissions WHERE permission_id IN ? GROUP BY permission_id
		`, ids).Scan(&rc).Error
		for _, r := range rc {
			roleCnt[r.PermissionID] = r.N
		}
		var uc []cntRow
		_ = h.db.Raw(`
			SELECT permission_id, COUNT(*) AS n FROM user_permissions WHERE permission_id IN ? GROUP BY permission_id
		`, ids).Scan(&uc).Error
		for _, r := range uc {
			userCnt[r.PermissionID] = r.N
		}
	}
	items := make([]permissionListItem, 0, len(rows))
	for _, p := range rows {
		items = append(items, permissionListItem{
			Permission:           p,
			RoleCount:            roleCnt[p.ID],
			DirectUserGrantCount: userCnt[p.ID],
		})
	}
	response.Success(c, "success", gin.H{
		"items": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

type permissionUpsertReq struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
}

func (h *Handlers) handleAdminCreatePermission(c *gin.Context) {
	var req permissionUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	req.Name = strings.TrimSpace(req.Name)
	if req.Key == "" || req.Name == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("key and name required"))
		return
	}
	p := models.Permission{
		Key: req.Key, Name: req.Name, Description: req.Description, Resource: strings.TrimSpace(req.Resource),
	}
	p.SetCreateInfo(operatorEmail(c))
	if err := h.db.Create(&p).Error; err != nil {
		response.Fail(c, "create permission failed", err)
		return
	}
	response.Success(c, "created", p)
}

func (h *Handlers) handleAdminUpdatePermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var req permissionUpsertReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var p models.Permission
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&p).Error; err != nil {
		response.Fail(c, "permission not found", err)
		return
	}
	vals := map[string]any{}
	if req.Name != "" {
		vals["name"] = req.Name
	}
	if req.Description != "" {
		vals["description"] = req.Description
	}
	if req.Resource != "" {
		vals["resource"] = strings.TrimSpace(req.Resource)
	}
	if nk := strings.TrimSpace(req.Key); nk != "" && nk != p.Key {
		vals["key"] = nk
	}
	if len(vals) == 0 {
		response.Success(c, "noop", p)
		return
	}
	vals["update_by"] = operatorEmail(c)
	if err = h.db.Model(&p).Updates(vals).Error; err != nil {
		response.Fail(c, "update failed", err)
		return
	}
	_ = h.db.First(&p, id).Error
	response.Success(c, "updated", p)
}

func (h *Handlers) handleAdminDeletePermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var p models.Permission
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&p).Error; err != nil {
		response.Fail(c, "permission not found", err)
		return
	}
	if p.Key == models.PermWildcard || p.Key == models.PermAdminAccess || p.Key == models.PermManageRoles {
		response.AbortWithJSONError(c, http.StatusForbidden, errors.New("cannot delete core permission"))
		return
	}
	op := operatorEmail(c)
	if err = h.db.Model(&p).Updates(map[string]any{
		"is_deleted": models.SoftDeleteStatusDeleted,
		"update_by":  op,
	}).Error; err != nil {
		response.Fail(c, "delete failed", err)
		return
	}
	_ = h.db.Where("permission_id = ?", id).Delete(&models.RolePermission{}).Error
	_ = h.db.Where("permission_id = ?", id).Delete(&models.UserPermission{}).Error
	response.Success(c, "deleted", nil)
}
