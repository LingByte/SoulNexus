package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
)

// credentialOperationAudit records write-like HTTP ops performed with an API key.
func (h *Handlers) credentialOperationAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if h.db == nil || middleware.AuthCredentialID(c) == 0 {
			return
		}
		if middleware.OperationAlreadyLogged(c) {
			return
		}
		method := strings.ToUpper(c.Request.Method)
		switch method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return
		}
		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		h.recordOp(c, OpLogEntry{
			Action:   constants.OpActionAPICall,
			Resource: constants.OpResourceAPI,
			Summary:  fmt.Sprintf("API Key %s %s", method, path),
			Success:  true,
		})
	}
}

func credentialIDString(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func credentialToJSON(row models.Credential) gin.H {
	var pc []string
	if strings.TrimSpace(row.PermissionCodes) != "" {
		_ = json.Unmarshal([]byte(row.PermissionCodes), &pc)
	}
	routeIDs := models.ParseCredentialAllowedRouteIDs(row.AllowedRouteIDs)
	out := gin.H{
		"id":              credentialIDString(row.ID),
		"tenantId":        credentialIDString(row.TenantID),
		"name":            row.Name,
		"kind":            models.NormalizeCredentialKind(row.Kind),
		"usesTenantAi":    models.CredentialUsesTenantAIConfig(row),
		"apiKeyPrefix":    row.AccessKey,
		"accessKey":       row.AccessKey, // backward-compatible alias
		"legacyHmac":      models.IsLegacyHMACCredential(row),
		"status":          row.Status,
		"allowIp":         row.AllowIP,
		"permissionCodes": pc,
		"allowedRouteIds": routeIDs,
		"voiceMode":       strings.TrimSpace(row.VoiceMode),
		"asrConfig":       utils.JSONValueFromBytes(row.AsrConfig),
		"ttsConfig":       utils.JSONValueFromBytes(row.TtsConfig),
		"llmConfig":       utils.JSONValueFromBytes(row.LlmConfig),
		"realtimeConfig":  utils.JSONValueFromBytes(row.RealtimeConfig),
		"hasAiConfig":     models.CredentialHasAIConfig(row),
		"requestCount":    row.RequestCount,
		"createdAt":       row.CreatedAt,
		"updatedAt":       row.UpdatedAt,
		"createBy":        row.CreateBy,
	}
	if row.ExpiresAt != nil {
		out["expiresAt"] = row.ExpiresAt
	}
	if row.LastUsedAt != nil {
		out["lastUsedAt"] = row.LastUsedAt
	}
	return out
}

// createCredential issues a bearer API key for the tenant (human JWT only).
func (h *Handlers) createCredential(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	var req dto.CredentialCreateReq
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	if strings.TrimSpace(req.Kind) == "" {
		req.Kind = constants.CredentialKindUserBundle
	}
	h.issueCredential(c, tenantID, req, middleware.AuthEmail(c))
}

func (h *Handlers) listCredentials(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	page, size := ginutil.QueryPage(c, 100)
	statusFilter := strings.TrimSpace(c.Query("status"))
	nameFilter := strings.TrimSpace(c.Query("name"))

	q := h.db.Model(&models.Credential{}).Where("tenant_id = ?", tenantID)
	if statusFilter == constants.CredentialStatusActive || statusFilter == constants.CredentialStatusDisabled {
		q = q.Where("status = ?", statusFilter)
	}
	if nameFilter != "" {
		q = q.Where("name LIKE ?", "%"+nameFilter+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}

	var rows []models.Credential
	if err := q.
		Select("id", "tenant_id", "name", "kind", "access_key", "status", "allow_ip", "permission_codes", "allowed_route_ids",
			"voice_mode", "asr_config", "tts_config", "llm_config", "realtime_config",
			"expires_at", "last_used_at", "request_count", "created_at", "updated_at", "create_by").
		Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&rows).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	list := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		list = append(list, credentialToJSON(row))
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) updateCredential(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}

	var req dto.CredentialUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	meta := common.BaseModel{}
	meta.SetUpdateInfo(middleware.AuthEmail(c))
	updates := map[string]any{}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			response.FailI18n(c, i18n.KeyCredNameEmpty, nil)
			return
		}
		updates["name"] = name
	}
	_ = refreshCredentialFixedRoutes(updates)
	if aiPatch, err := credentialAIUpdatesFromPatch(req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid ai config", err))
		return
	} else if !models.CredentialUsesTenantAIConfig(row) {
		for k, v := range aiPatch {
			updates[k] = v
		}
	}
	if req.ExpiresAt != nil {
		expiresAt, err := timeutil.ParseOptionalRFC3339(req.ExpiresAt)
		if err != nil {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidExpiresAt))
			return
		}
		updates["expires_at"] = expiresAt
	}
	before := row
	if ginutil.WriteInternalError(c, h.db.Model(&models.Credential{}).Where("id = ?", row.ID).Updates(updates).Error) {
		return
	}
	after, _ := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceCredential, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Updated API key %s", row.Name), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": credentialIDString(row.ID)})
}

func (h *Handlers) disableCredential(c *gin.Context) {
	patchCredentialStatus(h, c, constants.CredentialStatusDisabled)
}

func (h *Handlers) enableCredential(c *gin.Context) {
	patchCredentialStatus(h, c, constants.CredentialStatusActive)
}

func patchCredentialStatus(h *Handlers, c *gin.Context, target string) {
	tenantID := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if row.Status == target {
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": credentialIDString(row.ID), "status": row.Status})
		return
	}
	if ginutil.WriteInternalError(c, models.UpdateCredentialStatus(h.db, &row, target, middleware.AuthEmail(c))) {
		return
	}
	before := row
	action := constants.OpActionDisable
	if target == constants.CredentialStatusActive {
		action = constants.OpActionEnable
	}
	after, _ := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: action,
		Resource: constants.OpResourceCredential, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("%s API key %s", action, row.Name),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": credentialIDString(row.ID), "status": target})
}

// regenerateCredential rotates the raw API key (human JWT only). Old key stops working immediately.
func (h *Handlers) regenerateCredential(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	fullKey, lookupPrefix, keyHash, err := models.RegenerateAPIKeyForCredential(row)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	meta := common.BaseModel{}
	meta.SetUpdateInfo(middleware.AuthEmail(c))
	updates := map[string]any{
		"access_key": lookupPrefix,
		"secret_key": keyHash,
		"status":     constants.CredentialStatusActive,
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	before := row
	if ginutil.WriteInternalError(c, h.db.Model(&models.Credential{}).Where("id = ?", row.ID).Updates(updates).Error) {
		return
	}
	after, _ := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionRegenerate,
		Resource: constants.OpResourceCredential, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Regenerated API key %s", row.Name),
	}, before, after)
	out := credentialToJSON(after)
	out["apiKey"] = fullKey
	out["notice"] = i18n.TGin(c, i18n.KeyCredNoticeSecretOnce)
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) deleteCredential(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetCredentialByIDForTenant(h.db, id, tenantID)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	meta := common.BaseModel{}
	meta.SoftDelete(middleware.AuthEmail(c))
	if ginutil.WriteInternalError(c, h.db.Model(&models.Credential{}).
		Where("id = ?", row.ID).
		Updates(map[string]any{
			"status":     constants.CredentialStatusDisabled,
			"update_by":  meta.UpdateBy,
			"updated_at": meta.UpdatedAt,
			"deleted_at": meta.DeletedAt,
		}).Error) {
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionDelete,
		Resource: constants.OpResourceCredential, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Deleted API key %s", row.Name),
	}, row, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": credentialIDString(row.ID)})
}

// getCredentialAKSKRouteCatalog returns platform-open routes for tenant API key UI.
func (h *Handlers) getCredentialAKSKRouteCatalog(c *gin.Context) {
	enabled := models.SystemEnabledAKSKRouteIDs()
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"groups": models.FilterCatalogGroupsByRouteIDs(enabled),
		"total":  len(enabled),
	})
}
