package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) listTenantCredentialsPlatform(c *gin.Context) {
	tenantID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetActiveTenantByID(h.db, tenantID); ginutil.WriteGORMError(c, err, "tenant not found") {
		return
	}
	page, size := ginutil.QueryPage(c, 100)
	q := h.db.Model(&models.Credential{}).Where("tenant_id = ?", tenantID)
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

func (h *Handlers) createTenantCredentialPlatform(c *gin.Context) {
	tenantID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetActiveTenantByID(h.db, tenantID); ginutil.WriteGORMError(c, err, "tenant not found") {
		return
	}
	if !models.TenantPlatformKeyAllowed(h.db, tenantID) {
		response.FailI18n(c, i18n.KeyCredPlatformNoCapacity, nil)
		return
	}
	var req dto.CredentialCreateReq
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	req.Kind = constants.CredentialKindPlatformBundle
	h.issueCredential(c, tenantID, req, middleware.AuthEmail(c))
}

func (h *Handlers) issueCredential(c *gin.Context, tenantID uint, req dto.CredentialCreateReq, actor string) {
	kind := models.NormalizeCredentialKind(req.Kind)

	if kind == constants.CredentialKindPlatformBundle {
		if _, err := models.GetActiveTenantByID(h.db, tenantID); ginutil.WriteGORMError(c, err, "tenant not found") {
			return
		}
		if !models.TenantPlatformKeyAllowed(h.db, tenantID) {
			response.FailI18n(c, i18n.KeyCredPlatformNoCapacity, nil)
			return
		}
	}

	fullKey, lookupPrefix, keyHash, err := models.IssueAPIKeyForKind(kind)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	voiceMode, asrCfg, ttsCfg, llmCfg, rtCfg, cfgErr := credentialAIFieldsFromCreate(req)
	if cfgErr != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid ai config", cfgErr))
		return
	}
	if kind == constants.CredentialKindPlatformBundle {
		asrCfg, ttsCfg, llmCfg, rtCfg = nil, nil, nil, nil
	} else if !models.CredentialHasAIConfig(models.Credential{
		AsrConfig: asrCfg, TtsConfig: ttsCfg, LlmConfig: llmCfg, RealtimeConfig: rtCfg,
	}) {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyCredAIConfigRequired))
		return
	}

	permJSON, routeJSON, authErr := fixedCredentialAuth()
	if authErr != nil {
		ginutil.WriteAppError(c, response.CredentialRouteAppError(authErr))
		return
	}
	expiresAt, err := timeutil.ParseOptionalRFC3339(req.ExpiresAt)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidExpiresAt))
		return
	}
	row := &models.Credential{
		TenantID:        tenantID,
		Name:            strings.TrimSpace(req.Name),
		Kind:            kind,
		AccessKey:       lookupPrefix,
		SecretKey:       keyHash,
		Status:          constants.CredentialStatusActive,
		AllowIP:         "",
		PermissionCodes: permJSON,
		AllowedRouteIDs: routeJSON,
		VoiceMode:       voiceMode,
		AsrConfig:       asrCfg,
		TtsConfig:       ttsCfg,
		LlmConfig:       llmCfg,
		RealtimeConfig:  rtCfg,
		ExpiresAt:       expiresAt,
	}
	row.SetCreateInfo(actor)
	if row.Name == "" {
		if kind == constants.CredentialKindPlatformBundle {
			row.Name = "Platform API Key"
		} else {
			row.Name = "Integration"
		}
	}
	if creErr := h.db.Create(row).Error; creErr != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, creErr)
		return
	}

	h.recordOpChange(c, OpLogEntry{
		TenantID: tenantID, Action: constants.OpActionCreate,
		Resource: constants.OpResourceCredential, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created API key %s", row.Name),
		Detail:  audit.Redact(req),
	}, nil, row)
	out := credentialToJSON(*row)
	out["apiKey"] = fullKey
	out["notice"] = i18n.TGin(c, i18n.KeyCredNoticeSecretOnce)
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

// getCredentialExternalAPI lists fixed HTTP capabilities for tenant API keys.
func (h *Handlers) getCredentialExternalAPI(c *gin.Context) {
	ids := models.ExternalAPIKeyRouteIDs()
	routes := models.CatalogEntriesForIDs(ids)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"routeIds": ids,
		"routes":   routes,
	})
}
