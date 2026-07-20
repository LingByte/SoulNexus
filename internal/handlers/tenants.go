package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type tenantRegisterReq struct {
	CompanyName       string `json:"companyName" binding:"required,min=2,max=128"`
	AdminEmail        string `json:"adminEmail" binding:"required,email"`
	AdminPassword     string `json:"adminPassword" binding:"required,min=8,max=128"`
	AdminDisplayName  string `json:"adminDisplayName"`
	TenantDescription string `json:"tenantDescription"`
	MaxUserCount      int    `json:"maxUserCount"`
	DeviceID          string `json:"deviceId"`
	captcha.CaptchaFields
}

func signTenantAccessToken(db *gorm.DB, user models.TenantUser, tenant models.Tenant, sessionID string, deviceRecordID uint) (string, time.Duration, error) {
	if bootstrap.GlobalKeyManager == nil {
		return "", 0, apperror.ErrJWTKeyManagerNotInitialized
	}
	role := constants.JWTRoleTenantMember
	if ok, _ := models.TenantUserHasRoleName(db, user.ID, constants.TenantAdminRoleName); ok {
		role = constants.JWTRoleTenantAdmin
	}
	ttl := time.Duration(utils.SessionMaxLifetime(user.SessionIdleTimeoutHours, user.SessionMaxLifetimeHours)) * time.Hour
	p := access.AccessPayload{
		UserID:         user.ID,
		TenantID:       tenant.ID,
		TenantSlug:     tenant.Slug,
		Email:          user.Email,
		Role:           role,
		SessionID:      sessionID,
		DeviceRecordID: deviceRecordID,
	}
	token, err := access.SignAccessTokenWithKey(p, bootstrap.GlobalKeyManager, ttl)
	return token, ttl, err
}

type tenantPlatformUpdateReq struct {
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Status       string           `json:"status"`
	ContactEmail string           `json:"contactEmail"`
	MaxUserCount int              `json:"maxUserCount"`
	AsrConfig    *json.RawMessage `json:"asrConfig"`
	TtsConfig    *json.RawMessage `json:"ttsConfig"`
	LlmConfig    *json.RawMessage `json:"llmConfig"`
	// VoiceMode flips the voice attach kind: "pipeline" (3-layer) or
	// "realtime" (single full-duplex WS). Empty string = no change.
	VoiceMode string `json:"voiceMode"`
	// RealtimeConfig is the credential blob (provider + per-vendor
	// fields) consulted only when voiceMode == "realtime".
	RealtimeConfig *json.RawMessage `json:"realtimeConfig"`
	// AutomationConfig patches tenant-scoped automation toggles (platform admin only).
	AutomationConfig *models.TenantAutomationConfig `json:"automationConfig"`
}

// getTenant returns the full platform-detail view of a tenant organization.
//
// Platform admin only (enforced by route middleware).
//
// Path parameters:
//   - id: tenant snowflake ID (parsed with utils.ParseID). Non-numeric /
//     negative values return 400. Unknown IDs return 404.
//
// Response on success:
//
//	{
//	  "tenant": {
//	    "id":               <uint>,
//	    "name":             "...",
//	    "description":      "...",
//	    "status":           "active|suspended|...",
//	    "contactEmail":     "...",
//	    "maxUserCount":     <int>,
//	    "asrConfig":        { ... } | null,
//	    "ttsConfig":        { ... } | null,
//	    "llmConfig":        { ... } | null,
//	    "voiceMode":        "pipeline|realtime",
//	    "realtimeConfig":   { ... } | null,
//	    "createdAt":        "<RFC3339>",
//	    "updatedAt":        "<RFC3339>"
//	  }
//	}
func (h *Handlers) getTenant(c *gin.Context) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidID))
		return
	}
	t, err := models.GetActiveTenantByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"tenant": apiresponse.NewTenantPlatformDetailResponse(t)})
}

// createTenantPlatform provisions a new tenant, default admin role, and
// first admin user from the platform-console UI.
//
// Platform admin only (enforced by route middleware). Unlike registerTenant
// the caller is already authed, so no JWT is returned — the call only emits
// the database rows and an operation-log entry.
//
// Request body (JSON, bound to tenantRegisterReq):
//
//	{
//	  "companyName":       "Acme Co.",           // required, 2..128 chars
//	  "adminEmail":        "admin@acme.co",      // required, valid email
//	  "adminPassword":     "supersecret",        // required, 8..128 chars
//	  "adminDisplayName":  "Alice",              // optional
//	  "tenantDescription": "Acme tenant",        // optional
//	  "maxUserCount":      5                     // optional
//	}
//
// Response on success:
//
//	{
//	  "tenant":    { "id":..., "name":..., ... },
//	  "adminUser": { "id":..., "email":..., ... },
//	  "roleId":    <uint>
//	}
func (h *Handlers) createTenantPlatform(c *gin.Context) {
	var req tenantRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid params", err))
		return
	}
	hash, err := access.HashPassword(req.AdminPassword)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	takenMail, mailErr := models.CheckTenantUserEmailExists(h.db, req.AdminEmail, 0)
	if mailErr != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, mailErr)
		return
	}
	if takenMail {
		response.FailI18n(c, i18n.KeyTenantEmailExists, nil)
		return
	}
	tenant, user, role, err := models.ProvisionTenantWithAdmin(h.db, models.TenantProvisionInput{
		CompanyName: req.CompanyName, AdminEmail: req.AdminEmail, AdminDisplayName: req.AdminDisplayName,
		TenantDescription: req.TenantDescription, MaxUserCount: req.MaxUserCount,
	}, hash, "platform")
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenant.ID, Action: constants.OpActionCreate,
		Resource: constants.OpResourceTenant, ResourceID: tenant.ID, ResourceName: tenant.Name,
		Summary: fmt.Sprintf("Created tenant %s", tenant.Name), Detail: audit.Redact(req),
	}, nil, tenant)
	utils.Sig().Emit(constants.SigNotifyTenantProvisioned, nil, constants.NotifyTenantProvisionedPayload{
		TenantID:         tenant.ID,
		TenantName:       tenant.Name,
		AdminUserID:      user.ID,
		AdminEmail:       user.Email,
		AdminDisplayName: user.DisplayName,
		Source:           "platform",
		ClientIP:         c.ClientIP(),
	}, h.db)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"tenant":    apiresponse.NewTenantPlatformDetailResponse(tenant),
		"adminUser": apiresponse.NewTenantUserResponse(h.db, user),
		"roleId":    role.ID,
	})
}

// updateTenantPlatform mutates a tenant's base metadata and, optionally, its
// AI config blobs (asr / tts / llm / realtime / voiceMode) from the
// platform console.
//
// Platform admin only (enforced by route middleware).
//
// Path parameters:
//   - id: tenant snowflake ID. Unknown or invalid IDs short-circuit with
//     404/400 respectively.
//
// Request body (JSON, bound to tenantPlatformUpdateReq):
//
//	{
//	  "name":             "Acme",                 // optional display name
//	  "description":      "...",                  // optional
//	  "status":           "active|suspended",     // optional, validated
//	  "contactEmail":     "ops@acme.co",          // optional, validated as email
//	  "maxUserCount":     10,                     // optional
//	  "asrConfig":        { ... } | null,         // optional
//	  "ttsConfig":        { ... } | null,         // optional
//	  "llmConfig":        { ... } | null,         // optional
//	  "voiceMode":        "pipeline|realtime",    // optional
//	  "realtimeConfig":   { ... } | null          // optional
//	}
//
// Response on success:
//
//	{ "tenant": { "id":..., "name":..., ... (same shape as getTenant) } }
func (h *Handlers) updateTenantPlatform(c *gin.Context) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidID))
		return
	}
	before, err := models.GetActiveTenantByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req tenantPlatformUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	st := strings.TrimSpace(req.Status)
	if st != "" && st != constants.TenantStatusActive && st != constants.TenantStatusSuspended {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidStatus))
		return
	}
	if req.ContactEmail != "" && !utils.IsEmail(req.ContactEmail) {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidContactEmail))
		return
	}
	op := "platform"
	if err := models.UpdateActiveTenant(
		h.db,
		id,
		strings.TrimSpace(req.Name),
		req.Description,
		st,
		req.ContactEmail,
		req.MaxUserCount,
		op,
	); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if req.AsrConfig != nil || req.TtsConfig != nil || req.LlmConfig != nil ||
		req.RealtimeConfig != nil || strings.TrimSpace(req.VoiceMode) != "" {
		if err := models.PatchTenantAIConfigJSON(h.db, id,
			req.AsrConfig, req.TtsConfig, req.LlmConfig, req.RealtimeConfig,
			req.VoiceMode, op); err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
	}
	autoCfg := req.AutomationConfig
	if autoCfg != nil {
		if err := models.PatchTenantAutomationConfig(h.db, id, *autoCfg, op); err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
	}
	after, err := models.GetActiveTenantByID(h.db, id)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: id, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenant, ResourceID: id, ResourceName: before.Name,
		Summary: fmt.Sprintf("Updated tenant %s", after.Name), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"tenant": apiresponse.NewTenantPlatformDetailResponse(after)})
}

// deleteTenantPlatform soft-deletes a tenant and its cascade resources.
//
// Platform admin only (enforced by route middleware). Soft deletion in the
// current schema means models.SoftDeleteTenant — the DB row is preserved
// (deleted_at set) so historical call records retain referential integrity.
//
// Path parameters:
//   - id: tenant snowflake ID.
//
// Response on success:
//
//	{ "id": <uint> }
func (h *Handlers) deleteTenantPlatform(c *gin.Context) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidID))
		return
	}
	before, err := models.GetActiveTenantByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if err := models.SoftDeleteTenant(h.db, id, "platform"); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: id, Action: constants.OpActionDelete,
		Resource: constants.OpResourceTenant, ResourceID: id, ResourceName: before.Name,
		Summary: fmt.Sprintf("Deleted tenant %s", before.Name),
	}, before, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

// listTenants returns a paginated, searchable list of tenant organizations
// for the platform admin.
//
// Platform admin only (enforced by route middleware). Typical caller is the
// "assign trunk numbers" page in the platform console.
//
// Query parameters:
//   - page:     1-based page number (defaults to 1 via ginutil.QueryPage).
//   - pageSize: rows per page (defaults to / capped at 500).
//   - search:   substring filter matched against tenant names.
//
// Response on success (paginated envelope):
//
//	{
//	  "list":     [ { "id":..., "name":..., ... }, ... ],
//	  "total":    <int64>,
//	  "page":     <int>,
//	  "pageSize": <int>
//	}
func (h *Handlers) listTenants(c *gin.Context) {
	// Platform admin only (enforced by route middleware).
	page, size := ginutil.QueryPage(c, 500)
	list, total, err := models.ListTenantsPage(h.db, page, size, strings.TrimSpace(c.Query("search")))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	rows := make([]apiresponse.TenantResponse, len(list))
	for i := range list {
		rows[i] = apiresponse.NewTenantResponse(list[i])
	}
	ginutil.PageSuccess(c, rows, total, page, size)
}
