package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

type tenantUserCreateReq struct {
	Email       string `json:"email" binding:"required,email"`
	Phone       string `json:"phone"`
	Username    string `json:"username"`
	Password    string `json:"password"` // plain text; always hashed server-side
	DisplayName string `json:"displayName"`
	Status      string `json:"status"` // active | disabled | pending
	Source      string `json:"source"`
}

type tenantUserUpdateReq struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

type tenantUserStatusReq struct {
	Status string `json:"status" binding:"required"` // active | disabled | pending
}

// listTenantUsers paginates members of the authenticated tenant.
//
//   - GET /tenant-users
//   - Query: page (int, default 1), size (int, default 100),
//     status (string, optional), search (string, optional, name/email substring).
//
// Response is a paginated envelope with the user list and total count.
func (h *Handlers) listTenantUsers(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	page, size := ginutil.QueryPage(c, 100)

	list, total, err := models.ListTenantUsersPage(h.db, tenantID, page, size, c.Query("status"), c.Query("search"))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	pub := make([]apiresponse.TenantUserResponse, 0, len(list))
	for _, row := range list {
		pub = append(pub, apiresponse.NewTenantUserResponse(h.db, row))
	}
	ginutil.PageSuccess(c, pub, total, page, size)
}

// getTenantUser returns the public view of a single tenant user by id.
//
//   - GET /tenant-users/:id
//   - Path: id (uint) — the user id. The user must belong to the caller's tenant.
//
// Response is the public user record; 404 when not found or cross-tenant.
func (h *Handlers) getTenantUser(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetActiveTenantUserByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if row.TenantID != tenantID {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, row))
}

// createTenantUser adds a new member to the caller's tenant.
//
//   - POST /tenant-users — body: { email, phone, username, password, displayName, status, source }
//
// Request body:
//   - email       (string, required) — unique across the system.
//   - phone       (string, optional) — unique if provided.
//   - username    (string, optional) — unique if provided.
//   - password    (string, required, >= 8 chars) — plain text; hashed server-side.
//   - displayName (string, optional)
//   - status      (string, optional) — active | disabled | pending (default active).
//   - source      (string, optional) — default manual.
//
// The tenant user limit is enforced; returns 400 when the tenant is out of
// seats. Response is the public view of the newly created user.
func (h *Handlers) createTenantUser(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	var req tenantUserCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}

	email := utils.TrimLower(req.Email)
	if email == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyEmailRequired))
		return
	}

	// Check for duplicates
	exists, _ := models.CheckTenantUserEmailExists(h.db, email, 0)
	if exists {
		response.Render(c, response.NewI18n(response.CodeDuplicate, i18n.KeyTenantEmailExists))
		return
	}
	if req.Phone != "" {
		exists, _ = models.CheckTenantUserPhoneExists(h.db, strings.TrimSpace(req.Phone), 0)
		if exists {
			response.Render(c, response.NewI18n(response.CodeDuplicate, i18n.KeyPhoneExists))
			return
		}
	}
	if req.Username != "" {
		exists, _ = models.CheckTenantUserUsernameExists(h.db, strings.TrimSpace(req.Username), 0)
		if exists {
			response.Render(c, response.NewI18n(response.CodeDuplicate, i18n.KeyUsernameExists))
			return
		}
	}
	if err := models.EnforceTenantUserLimit(h.db, tenantID); err != nil {
		if errors.Is(err, models.ErrTenantUserLimit) {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyTenantUserLimitReached))
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}

	pw := strings.TrimSpace(req.Password)
	if len(pw) < 8 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyPasswordRequired))
		return
	}
	hash, err := access.HashPassword(pw)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = constants.TenantUserStatusActive
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = constants.TenantUserSourceManual
	}

	user := &models.TenantUser{
		TenantID:     tenantID,
		Email:        email,
		Phone:        strings.TrimSpace(req.Phone),
		Username:     strings.TrimSpace(req.Username),
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(req.DisplayName),
		Status:       status,
		Source:       source,
	}
	if op := middleware.AuditOperator(c); op != "" {
		user.SetCreateInfo(op)
	}

	if err := models.CreateTenantUser(h.db, user); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionCreate,
		Resource: constants.OpResourceTenantUser, ResourceID: user.ID, ResourceName: user.Email,
		Summary: fmt.Sprintf("Created member %s", user.Email), Detail: audit.Redact(req),
	}, nil, user)
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, *user))
}

// updateTenantUser edits select profile fields of an existing tenant user.
//
//   - PUT /tenant-users/:id — body: { email, phone, username, displayName, status }
//
// At least one field must be provided. email/phone/username collisions
// return 409. Response on success: { id }.
func (h *Handlers) updateTenantUser(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}

	var req tenantUserUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}

	// Get existing user to check tenant
	existing, err := models.GetActiveTenantUserByID(h.db, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyUserNotFound))
		return
	}
	if existing.TenantID != tenantID {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyUserNotFound))
		return
	}

	updates := make(map[string]any)
	if req.Email != "" {
		email := utils.TrimLower(req.Email)
		exists, _ := models.CheckTenantUserEmailExists(h.db, email, uint(id))
		if exists {
			response.Render(c, response.NewI18n(response.CodeDuplicate, i18n.KeyTenantEmailExists))
			return
		}
		updates["email"] = email
	}
	if req.Phone != "" {
		phone := strings.TrimSpace(req.Phone)
		exists, _ := models.CheckTenantUserPhoneExists(h.db, phone, uint(id))
		if exists {
			response.Render(c, response.NewI18n(response.CodeDuplicate, i18n.KeyPhoneExists))
			return
		}
		updates["phone"] = phone
	}
	if req.Username != "" {
		username := strings.TrimSpace(req.Username)
		exists, _ := models.CheckTenantUserUsernameExists(h.db, username, uint(id))
		if exists {
			response.Render(c, response.NewI18n(response.CodeDuplicate, i18n.KeyUsernameExists))
			return
		}
		updates["username"] = username
	}
	if req.DisplayName != "" {
		updates["display_name"] = strings.TrimSpace(req.DisplayName)
	}
	if req.Status != "" {
		updates["status"] = strings.TrimSpace(req.Status)
	}

	if len(updates) == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNoFieldsToUpdate))
		return
	}

	n, err := models.UpdateTenantUser(h.db, uint(id), updates, middleware.AuditOperator(c))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if n == 0 {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyUserNotFound))
		return
	}
	after, _ := models.GetActiveTenantUserByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: id, ResourceName: existing.Email,
		Summary: fmt.Sprintf("Updated member %s", existing.Email), Detail: audit.Redact(req),
	}, existing, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

// updateTenantUserStatus flips a member's status (active/disabled/pending).
//
//   - PUT /tenant-users/:id/status — body: { status }
//
// Request body:
//   - status (string, required) — one of active | disabled | pending.
//
// Response on success: { id, status }.
func (h *Handlers) updateTenantUserStatus(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}

	existing, err := models.GetActiveTenantUserByID(h.db, id)
	if err != nil || existing.TenantID != tenantID {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyUserNotFound))
		return
	}

	var req tenantUserStatusReq
	if !ginutil.BindJSON(c, &req) {
		return
	}

	status := strings.TrimSpace(req.Status)
	if status != constants.TenantUserStatusActive && status != constants.TenantUserStatusDisabled && status != constants.TenantUserStatusPending {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidStatus))
		return
	}

	n, err := models.UpdateTenantUserStatus(h.db, id, status, middleware.AuditOperator(c))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if n == 0 {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyUserNotFound))
		return
	}
	after, _ := models.GetActiveTenantUserByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: id, ResourceName: existing.Email,
		Summary: fmt.Sprintf("Updated member status %s -> %s", existing.Email, status),
		Detail:  audit.Redact(req),
	}, existing, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id, "status": status})
}

// deleteTenantUser soft-deletes a member of the caller's tenant.
//
//   - DELETE /tenant-users/:id
//   - Path: id (uint) — the user id; must belong to the caller's tenant.
//
// Response on success: { id }. The user can be restored via /tenant-users/:id/restore.
func (h *Handlers) deleteTenantUser(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}

	existing, getErr := models.GetActiveTenantUserByID(h.db, id)
	if getErr != nil || existing.TenantID != tenantID {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}

	rows, err := models.SoftDeleteTenantUserByID(h.db, id, middleware.AuditOperator(c))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if rows == 0 {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionDelete,
		Resource: constants.OpResourceTenantUser, ResourceID: id, ResourceName: existing.Email,
		Summary: fmt.Sprintf("Deleted member %s", existing.Email),
	}, existing, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

// restoreTenantUser brings back a soft-deleted tenant user.
//
//   - POST /tenant-users/:id/restore — no body.
//   - Path: id (uint) — must have been soft-deleted within the caller's tenant.
//
// Response on success: { id }.
func (h *Handlers) restoreTenantUser(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}

	existing, getErr := models.GetTenantUserByID(h.db, id)
	if getErr != nil || existing.TenantID != tenantID {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}

	rows, err := models.RestoreTenantUser(h.db, id, middleware.AuditOperator(c))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if rows == 0 {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	after, _ := models.GetActiveTenantUserByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionRestore,
		Resource: constants.OpResourceTenantUser, ResourceID: id, ResourceName: existing.Email,
		Summary: fmt.Sprintf("Restored member %s", existing.Email),
	}, existing, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

// getTenantUserStats returns count breakdowns for the caller's tenant.
//
//   - GET /tenant-users/stats — no params.
//
// Response: { total, active, disabled, pending }.
func (h *Handlers) getTenantUserStats(c *gin.Context) {
	tenantID := middleware.AuthTenantID(c)
	if tenantID == 0 {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	total, _ := models.CountTenantUsers(h.db, tenantID)
	active, _ := models.CountTenantUsersByStatus(h.db, tenantID, constants.TenantUserStatusActive)
	disabled, _ := models.CountTenantUsersByStatus(h.db, tenantID, constants.TenantUserStatusDisabled)
	pending, _ := models.CountTenantUsersByStatus(h.db, tenantID, constants.TenantUserStatusPending)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"total":    total,
		"active":   active,
		"disabled": disabled,
		"pending":  pending,
	})
}
