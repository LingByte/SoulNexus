package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerPlatformAdminRoutes mounts the CRUD endpoints used by the
// superuser console. Every sub-route here is gated by
// middleware.RequirePlatformAdmin — tenant users are rejected before any
// handler body runs.
//
// Sub-routes registered:
//   - GET    /platform-admins              → listPlatformAdmins
//   - GET    /platform-admins/:id          → getPlatformAdmin
//   - POST   /platform-admins              → createPlatformAdmin
//   - PUT    /platform-admins/:id          → updatePlatformAdmin
//   - PUT    /platform-admins/:id/status   → updatePlatformAdminStatus
//   - PUT    /platform-admins/:id/password → resetPlatformAdminPassword
//   - DELETE /platform-admins/:id          → deletePlatformAdmin
func (h *Handlers) registerPlatformAdminRoutes(r *humax.Group) {
	g := r.Group("platform-admins")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("", h.listPlatformAdmins)
		g.GET("/:id", h.getPlatformAdmin)
		g.POST("", h.createPlatformAdmin)
		g.PUT("/:id", h.updatePlatformAdmin)
		g.PUT("/:id/status", h.updatePlatformAdminStatus)
		g.PUT("/:id/password", h.resetPlatformAdminPassword)
		g.DELETE("/:id", h.deletePlatformAdmin)
	}
}

// listPlatformAdmins returns a paginated list of platform admin accounts.
//
//	Endpoint: GET /platform/admins (platform-admin scoped).
//
// Path parameters: none.
//
// Query parameters:
//
//	page   (int, default 1)   - pagination page number.
//	size   (int, default 100) - page size.
//	search (string)           - optional free-text search over displayName/email.
//
// Request body: none.
//
// Response (paginated):
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": [ apiresponse.NewPlatformAdminResponse(row) ... ],
//	  "total": int, "page": int, "size": int
//	}
func (h *Handlers) listPlatformAdmins(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 100)
	list, total, err := models.ListPlatformAdminsPage(h.db, page, size, c.Query("search"))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]apiresponse.PlatformAdminResponse, 0, len(list))
	for _, row := range list {
		out = append(out, apiresponse.NewPlatformAdminResponse(row))
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

// getPlatformAdmin loads a single platform admin row by id.
//
//	Endpoint: GET /platform/admins/:id
//
// Path parameters:
//
//	id (uint) - platform admin row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": apiresponse.NewPlatformAdminResponse(row) }
func (h *Handlers) getPlatformAdmin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetPlatformAdminByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminResponse(row))
}

// createPlatformAdmin creates a new platform admin account. Rejects duplicate
// email addresses and normalizes status to "active" when not provided. The
// creation is also recorded as an audit log entry.
//
//	Endpoint: POST /platform/admins
//
// Path parameters: none.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{
//	  "email":        "admin@example.com (required, valid email)",
//	  "password":     "8..128 chars (required)",
//	  "displayName":  "optional display name",
//	  "status":       "active | disabled (optional; default active)"
//	}
//
// Response: { "code": 200, "msg": "success", "data": apiresponse.NewPlatformAdminResponse(row) }
func (h *Handlers) createPlatformAdmin(c *gin.Context) {
	var req dto.PlatformAdminCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	email := utils.TrimLower(req.Email)
	var existing models.PlatformAdmin
	if err := h.db.Where("email = ?", email).First(&existing).Error; err == nil {
		response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyEmailExistsConflict))
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	hash, err := access.HashPassword(req.Password)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "hash password failed", err))
		return
	}
	row := &models.PlatformAdmin{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(req.DisplayName),
		Status:       req.Status,
	}
	if op := middleware.AuditOperator(c); op != "" {
		row.SetCreateInfo(op)
	}
	if err := h.db.Create(row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action:   constants.OpActionCreate,
		Resource: constants.OpResourcePlatformAdmin, ResourceID: row.ID, ResourceName: row.Email,
		Summary: fmt.Sprintf("Created platform admin %s", row.Email), Detail: audit.Redact(req),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminResponse(*row))
}

// updatePlatformAdmin changes mutable fields (email / displayName) of an
// existing platform admin and records the change in the audit log.
//
//	Endpoint: PUT /platform/admins/:id
//
// Path parameters:
//
//	id (uint) - platform admin row id.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{ "email": "optional", "displayName": "optional" }
//
// Response: { "code": 200, "msg": "success", "data": apiresponse.NewPlatformAdminResponse(after) }
func (h *Handlers) updatePlatformAdmin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := models.GetPlatformAdminByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req dto.PlatformAdminUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	n, err := models.UpdatePlatformAdminProfile(h.db, id, req.Email, req.DisplayName, middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	if n == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNoFieldsToUpdate))
		return
	}
	after, _ := models.GetPlatformAdminByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		Action:   constants.OpActionUpdate,
		Resource: constants.OpResourcePlatformAdmin, ResourceID: id, ResourceName: before.Email,
		Summary: fmt.Sprintf("Updated platform admin %s", after.Email), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminResponse(after))
}

// updatePlatformAdminStatus enables or disables a platform admin account.
// Self-disable is forbidden, and disabling the last active platform admin is
// rejected to prevent lockout. The status change is recorded in the audit log.
//
//	Endpoint: PUT /platform/admins/:id/status
//
// Path parameters:
//
//	id (uint) - platform admin row id.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{ "status": "active | disabled (required)" }
//
// Response: { "code": 200, "msg": "success", "data": { "id": uint, "status": string } }
func (h *Handlers) updatePlatformAdminStatus(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	selfID := middleware.AuthPlatformAdminID(c)
	before, err := models.GetPlatformAdminByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req dto.PlatformAdminStatusReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if req.Status == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyStatusActiveOrDisabled))
		return
	}
	if req.Status == constants.PlatformAdminStatusDisabled {
		if selfID > 0 && selfID == id {
			response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyCannotDisableSelf))
			return
		}
		if err := models.EnsureNotLastActivePlatformAdmin(h.db, id); err != nil {
			if errors.Is(err, apperror.ErrLastActivePlatformAdmin) {
				response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyCannotDisableLastAdmin))
				return
			}
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
	}
	n, err := models.UpdatePlatformAdminStatus(h.db, id, req.Status, middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	if n == 0 {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	after, _ := models.GetPlatformAdminByID(h.db, id)
	action := constants.OpActionDisable
	if req.Status == constants.PlatformAdminStatusActive {
		action = constants.OpActionEnable
	}
	h.recordOpChange(c, OpLogEntry{
		Action:   action,
		Resource: constants.OpResourcePlatformAdmin, ResourceID: id, ResourceName: before.Email,
		Summary: fmt.Sprintf("%s platform admin %s", action, before.Email), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id, "status": req.Status})
}

// resetPlatformAdminPassword replaces an admin's password hash. The password
// itself is not returned, only the id. The action is recorded in the audit log
// (sensitive fields are redacted).
//
//	Endpoint: PUT /platform/admins/:id/password
//
// Path parameters:
//
//	id (uint) - platform admin row id.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{ "password": "8..128 chars (required)" }
//
// Response: { "code": 200, "msg": "success", "data": { "id": uint } }
func (h *Handlers) resetPlatformAdminPassword(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := models.GetPlatformAdminByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req dto.PlatformAdminPasswordReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	hash, err := access.HashPassword(req.Password)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "hash password failed", err))
		return
	}
	if err := models.UpdatePlatformAdminPassword(h.db, id, hash); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	after, _ := models.GetPlatformAdminByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		Action:   constants.OpActionUpdate,
		Resource: constants.OpResourcePlatformAdmin, ResourceID: id, ResourceName: before.Email,
		Summary: fmt.Sprintf("Reset password for platform admin %s", before.Email), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

// deletePlatformAdmin soft-deletes a platform admin account. Self-delete and
// deleting the last active platform admin are rejected to prevent lockout.
// The deletion is recorded as an audit log entry.
//
//	Endpoint: DELETE /platform/admins/:id
//
// Path parameters:
//
//	id (uint) - platform admin row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": { "id": uint } }
func (h *Handlers) deletePlatformAdmin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if selfID := middleware.AuthPlatformAdminID(c); selfID > 0 && selfID == id {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyCannotDeleteSelf))
		return
	}
	before, err := models.GetPlatformAdminByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if err := models.EnsureNotLastActivePlatformAdmin(h.db, id); err != nil {
		if errors.Is(err, apperror.ErrLastActivePlatformAdmin) {
			response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyCannotDeleteLastAdmin))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	n, err := models.SoftDeletePlatformAdmin(h.db, id, middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	if n == 0 {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action:   constants.OpActionDelete,
		Resource: constants.OpResourcePlatformAdmin, ResourceID: id, ResourceName: before.Email,
		Summary: fmt.Sprintf("Deleted platform admin %s", before.Email),
	}, before, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}
