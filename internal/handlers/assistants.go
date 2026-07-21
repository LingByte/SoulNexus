package handlers

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

type assistantWriteReq struct {
	Name               string `json:"name"`
	Scene              string `json:"scene"`
	Version            string `json:"version"`
	Description        string `json:"description"`
	Enabled            *bool  `json:"enabled"`
	Welcome            string `json:"welcome"`
	Prompt             string `json:"prompt"`
	KnowledgeNamespace string `json:"knowledgeNamespace"`
	VoiceDialogWsURL          string `json:"voiceDialogWsUrl"`
	BoundJsTemplateSourceID   string `json:"boundJsTemplateSourceId"`
	NluModelID                string `json:"nluModelId"`
	AsrConfig                 string `json:"asrConfig"`
	TtsConfig          string `json:"ttsConfig"`
	LlmConfig          string `json:"llmConfig"`
	RealtimeConfig     string `json:"realtimeConfig"`
	VadConfig          string `json:"vadConfig"`
	AgentConfig        string `json:"agentConfig"`
	HotWords           string `json:"hotWords"`
	InterruptionConfig string `json:"interruptionConfig"`
	AudioTrackConfig   string `json:"audioTrackConfig"`
	AudioProcessConfig string `json:"audioProcessConfig"`
	QueryRewriter      string `json:"queryRewriter"`
	McpServers         string `json:"mcpServers"`
	TtsVoice           string `json:"ttsVoice"`
	RealtimeVoice      string `json:"realtimeVoice"`
	Collect            string `json:"collect"`
	CredentialID       string `json:"credentialId"`
	TenantID           string `json:"tenantId"`
}

// listAssistants is the API handler for GET /assistant — list assistants
// (tenant-scoped; platform admins can view across tenants via ?tenantId=).
//
// Query parameters:
//   - page / pageSize (int): pagination (default page=1, size=100)
//   - scene (string): filter by assistant scene
//   - name (string): fuzzy name filter
//
// Response: { list: [Assistant], total: int, page: int, pageSize: int }
func (h *Handlers) listAssistants(c *gin.Context) {
	tid, err := h.assistantScopeTenantID(c, c.Query("tenantId"))
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	page, size := ginutil.QueryPage(c, 100)
	list, total, err := models.ListAssistantsPage(h.db, tid, page, size, c.Query("scene"), c.Query("name"))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

// getAssistant returns a single assistant row by :id (tenant-scoped).
//
// Path parameters:
//   - id (uint): assistant id
//
// Response: { code: 0, msg: "success", data: Assistant }
func (h *Handlers) getAssistant(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// createAssistant creates a new assistant. For tenant users, tenantId is
// derived from the JWT; platform admins must pass tenantId via query or body.
//
// Request body (assistantWriteReq):
//   - name (string, required): assistant display name
//   - scene (string): assistant_scene constant
//   - version (string): semantic version label
//   - description (string): long description
//   - enabled (bool): whether the assistant can be used for calls
//   - welcome, prompt, knowledgeNamespace (string): dialog config
//   - asrConfig, ttsConfig, llmConfig, realtimeConfig, vadConfig,
//     agentConfig, collect (string, optional JSON): raw JSON
//     blobs that are validated when non-empty
//
// Response: { code: 0, msg: "success", data: Assistant }
func (h *Handlers) createAssistant(c *gin.Context) {
	var req assistantWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	tid, err := h.assistantScopeTenantID(c, utils.FirstNonEmpty(c.Query("tenantId"), req.TenantID))
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameRequired))
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	asr, _ := models.ParseOptionalJSONColumnNullable(req.AsrConfig)
	tts, _ := models.ParseOptionalJSONColumnNullable(req.TtsConfig)
	llm, _ := models.ParseOptionalJSONColumnNullable(req.LlmConfig)
	realtime, _ := models.ParseOptionalJSONColumnNullable(req.RealtimeConfig)
	vad, _ := models.ParseOptionalJSONColumnNullable(req.VadConfig)
	agent, _ := models.ParseOptionalJSONColumnNullable(req.AgentConfig)
	hotWords, _ := models.ParseOptionalJSONColumnNullable(req.HotWords)
	interruption, _ := models.ParseOptionalJSONColumnNullable(req.InterruptionConfig)
	audioTrack, _ := models.ParseOptionalJSONColumnNullable(req.AudioTrackConfig)
	audioProcess, _ := models.ParseOptionalJSONColumnNullable(req.AudioProcessConfig)
	queryRewriter, _ := models.ParseOptionalJSONColumnNullable(req.QueryRewriter)
	mcpServers, _ := models.ParseOptionalJSONColumnNullable(req.McpServers)
	collect, _ := models.ParseOptionalJSONColumnNullable(req.Collect)

	row := models.NewAssistantForCreate(
		tid, name, req.Scene, req.Version, req.Description,
		req.Welcome, req.Prompt, req.KnowledgeNamespace, req.VoiceDialogWsURL, req.BoundJsTemplateSourceID,
		req.TtsVoice, req.RealtimeVoice, enabled,
		asr, tts, llm, realtime, vad, agent, hotWords, interruption, audioTrack, audioProcess, queryRewriter, mcpServers, collect,
	)
	nluID, err := h.resolveAssistantNluModelID(c, tid, req.NluModelID)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	row.NluModelID = nluID
	if cid := utils.ParseOptionalID(req.CredentialID); cid > 0 {
		if _, err := models.GetCredentialByIDForTenant(h.db, cid, tid); err != nil {
			ginutil.WriteGORMError(c, err, "credential not found")
			return
		}
		row.CredentialID = cid
	}
	row.SetCreateInfo(middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, h.db.Create(&row).Error) {
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceAssistant, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created assistant %s", row.Name), Detail: audit.Redact(req),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// updateAssistant applies a PATCH-style update to an existing assistant.
// Only the fields present in the request body are modified; missing fields
// keep their previous value.
//
// Path parameters:
//   - id (uint): assistant id
//
// Request body: see assistantWriteReq (same shape as createAssistant)
//
// Response: { code: 0, msg: "success", data: Assistant }
func (h *Handlers) updateAssistant(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req assistantWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	before, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	updates, err := models.BuildAssistantUpdates(
		before, req.Name, req.Scene, req.Version, req.Description, req.Enabled,
		req.Welcome, req.Prompt, req.KnowledgeNamespace, req.VoiceDialogWsURL, req.BoundJsTemplateSourceID,
		req.TtsVoice, req.RealtimeVoice,
		req.AsrConfig, req.TtsConfig, req.LlmConfig, req.RealtimeConfig,
		req.VadConfig, req.AgentConfig, req.HotWords, req.InterruptionConfig, req.AudioTrackConfig,
		req.AudioProcessConfig, req.QueryRewriter, req.McpServers, req.Collect,
		middleware.AuditOperator(c),
	)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	nluID, err := h.resolveAssistantNluModelID(c, before.TenantID, req.NluModelID)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	// Always persist binding (including clear to 0 when client sends empty/"0").
	updates["nlu_model_id"] = nluID
	if req.CredentialID != "" {
		cid := utils.ParseOptionalID(req.CredentialID)
		if cid > 0 {
			if _, err := models.GetCredentialByIDForTenant(h.db, cid, before.TenantID); err != nil {
				ginutil.WriteGORMError(c, err, "credential not found")
				return
			}
		}
		updates["credential_id"] = cid
	}
	if ginutil.WriteInternalError(c, h.db.Model(&before).Updates(updates).Error) {
		return
	}
	after, _ := models.ReloadAssistantByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		TenantID: before.TenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceAssistant, ResourceID: id, ResourceName: after.Name,
		Summary: fmt.Sprintf("Updated assistant %s", after.Name), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, after)
}

// deleteAssistant soft-deletes an assistant (tenant-scoped). The row remains
// in the database with deleted_at set, so call history can still reference it.
//
// Path parameters:
//   - id (uint): assistant id
//
// Response: { code: 0, msg: "success", data: { id: string } }
func (h *Handlers) deleteAssistant(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	tid := before.TenantID
	if middleware.AuthPlatformAdminID(c) != 0 {
		n, err := models.SoftDeleteAssistantByID(h.db, id, middleware.AuditOperator(c))
		if ginutil.WriteInternalError(c, err) {
			return
		}
		if n == 0 {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
	} else {
		n, err := models.SoftDeleteAssistantByIDForTenant(h.db, id, tid, middleware.AuditOperator(c))
		if ginutil.WriteInternalError(c, err) {
			return
		}
		if n == 0 {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionDelete,
		Resource: constants.OpResourceAssistant, ResourceID: id, ResourceName: before.Name,
		Summary: fmt.Sprintf("Deleted assistant %s", before.Name),
	}, before, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": fmt.Sprintf("%d", id)})
}

// importAssistantFromTenant clones the tenant-level AI configuration into a
// new assistant. Intended to ease migration from the legacy tenant-scoped
// AI configuration to per-assistant configuration.
//
// Request body:
//   - name (string): new assistant display name
//   - tenantId (string, required for platform admin): target tenant
//
// Response: { code: 0, msg: "success", data: Assistant }
func (h *Handlers) importAssistantFromTenant(c *gin.Context) {
	var req importAssistantReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	tid, err := h.assistantScopeTenantID(c, utils.FirstNonEmpty(c.Query("tenantId"), req.TenantID))
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	tenant, err := models.GetActiveTenantByID(h.db, tid)
	if ginutil.WriteGORMError(c, err, "tenant not found") {
		return
	}
	ast, err := models.ImportTenantAIConfigToAssistant(h.db, tenant, req.Name, middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceAssistant, ResourceID: ast.ID, ResourceName: ast.Name,
		Summary: fmt.Sprintf("Imported tenant AI config to assistant %s", ast.Name),
	}, nil, ast)
	response.SuccessI18n(c, i18n.KeySuccess, ast)
}

// getAssistantRow returns an assistant row with appropriate access control —
// tenant users can only see their own tenant's assistants; platform admins can
// access any assistant by id.
func (h *Handlers) getAssistantRow(c *gin.Context, id uint) (models.Assistant, error) {
	if middleware.AuthPlatformAdminID(c) != 0 {
		return models.GetActiveAssistantByID(h.db, id)
	}
	tid := middleware.CurrentTenantID(c)
	return models.GetActiveAssistantForTenant(h.db, id, tid)
}

// assistantScopeTenantID resolves the effective tenant for CRUD operations on
// assistants. Regular tenant JWTs always resolve to CurrentTenantID;
// platform-admin JWTs must pass tenantId explicitly (query or body).
func (h *Handlers) assistantScopeTenantID(c *gin.Context, queryTenantID string) (uint, error) {
	if middleware.AuthPlatformAdminID(c) == 0 {
		tid := middleware.CurrentTenantID(c)
		if tid == 0 {
			return 0, fmt.Errorf("tenant context required")
		}
		return tid, nil
	}
	raw := strings.TrimSpace(queryTenantID)
	if raw == "" || raw == "0" {
		return 0, fmt.Errorf("tenantId query required for platform admin")
	}
	return utils.RequireScopeID(raw, "tenantId")
}

// resolveAssistantNluModelID validates optional nluModelId for the tenant (0 = unbound).
func (h *Handlers) resolveAssistantNluModelID(c *gin.Context, tenantID uint, raw string) (uint, error) {
	_ = c
	s := strings.TrimSpace(raw)
	if s == "" || s == "0" {
		return 0, nil
	}
	id, err := utils.RequireScopeID(s, "nluModelId")
	if err != nil {
		return 0, err
	}
	row, err := models.GetTenantNluModel(h.db, tenantID, id)
	if err != nil || row.ID == 0 {
		return 0, fmt.Errorf("nlu model not found for tenant")
	}
	return row.ID, nil
}

type importAssistantReq struct {
	Name     string `json:"name"`
	TenantID string `json:"tenantId"`
}
