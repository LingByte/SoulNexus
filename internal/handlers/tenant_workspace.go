package handlers

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type tenantWorkspaceAIReq struct {
	VoiceMode      *string          `json:"voiceMode"`
	AsrConfig      *json.RawMessage `json:"asrConfig"`
	TtsConfig      *json.RawMessage `json:"ttsConfig"`
	LlmConfig      *json.RawMessage `json:"llmConfig"`
	RealtimeConfig *json.RawMessage `json:"realtimeConfig"`
}

func (h *Handlers) registerTenantWorkspaceRoutes(r *humax.Group) {
	g := r.Group("tenant/workspace")
	g.GET("/ai", h.getTenantWorkspaceAI)
	g.PUT("/ai", h.updateTenantWorkspaceAI)
}

func (h *Handlers) getTenantWorkspaceAI(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	if middleware.AuthCredentialID(c) != 0 {
		response.FailI18n(c, i18n.KeyForbidden, nil)
		return
	}
	var tenant models.Tenant
	if ginutil.WriteGORMError(c, h.db.Where("id = ?", tenantID).First(&tenant).Error, "tenant not found") {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"voiceMode":      strings.TrimSpace(tenant.VoiceMode),
		"asrConfig":      utils.JSONValueFromBytes(tenant.AsrConfig),
		"ttsConfig":      utils.JSONValueFromBytes(tenant.TtsConfig),
		"llmConfig":      utils.JSONValueFromBytes(tenant.LlmConfig),
		"realtimeConfig": utils.JSONValueFromBytes(tenant.RealtimeConfig),
	})
}

func (h *Handlers) updateTenantWorkspaceAI(c *gin.Context) {
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	if middleware.AuthCredentialID(c) != 0 {
		response.FailI18n(c, i18n.KeyForbidden, nil)
		return
	}
	var req tenantWorkspaceAIReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	patch := map[string]any{}
	if req.VoiceMode != nil {
		patch["voice_mode"] = strings.TrimSpace(*req.VoiceMode)
	}
	for _, pair := range []struct {
		raw *json.RawMessage
		col string
	}{
		{req.AsrConfig, "asr_config"},
		{req.TtsConfig, "tts_config"},
		{req.LlmConfig, "llm_config"},
		{req.RealtimeConfig, "realtime_config"},
	} {
		if pair.raw == nil {
			continue
		}
		j, err := models.ParseOptionalJSONColumn(string(*pair.raw))
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "invalid json", err))
			return
		}
		patch[pair.col] = j
	}
	if len(patch) == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "empty body", nil))
		return
	}
	patch["update_by"] = middleware.AuditOperator(c)
	if ginutil.WriteInternalError(c, h.db.Model(&models.Tenant{}).Where("id = ?", tenantID).Updates(patch).Error) {
		return
	}
	h.getTenantWorkspaceAI(c)
}
