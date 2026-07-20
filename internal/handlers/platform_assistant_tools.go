package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func (h *Handlers) registerPlatformAssistantToolRoutes(r *humax.Group) {
	g := r.Group("platform/assistant-tools")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("", h.listPlatformAssistantTools)
		g.GET("/:id", h.getPlatformAssistantTool)
		g.POST("", h.createPlatformAssistantTool)
		g.PUT("/:id", h.updatePlatformAssistantTool)
		g.DELETE("/:id", h.deletePlatformAssistantTool)
	}
}

func (h *Handlers) listPlatformAssistantTools(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 100)
	tenantID := uint(0)
	if tid, ok, invalid := ginutil.QueryOptionalID(c, "tenantId"); invalid {
		response.Render(c, response.New(response.CodeBadRequest, i18n.TGin(c, i18n.KeyInvalidParams)))
		return
	} else if ok {
		tenantID = tid
	}
	rows, total, err := models.ListTenantAssistantToolsAdmin(h.db, tenantID, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	names := h.tenantNamesForAssistantTools(rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		item := tenantAssistantToolResp(row, nil)
		item["tenantName"] = names[row.TenantID]
		out = append(out, item)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"list":  out,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func (h *Handlers) tenantNamesForAssistantTools(rows []models.TenantAssistantTool) map[uint]string {
	ids := make([]uint, 0, len(rows))
	seen := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.TenantID]; ok {
			continue
		}
		seen[row.TenantID] = struct{}{}
		ids = append(ids, row.TenantID)
	}
	out := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return out
	}
	var tenants []models.Tenant
	_ = h.db.Where("id IN ?", ids).Find(&tenants).Error
	for _, t := range tenants {
		out[t.ID] = t.Name
	}
	return out
}

func (h *Handlers) getPlatformAssistantTool(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantAssistantToolByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	item := tenantAssistantToolResp(row, nil)
	names := h.tenantNamesForAssistantTools([]models.TenantAssistantTool{row})
	item["tenantName"] = names[row.TenantID]
	response.SuccessI18n(c, i18n.KeySuccess, item)
}

type platformAssistantToolCreateReq struct {
	tenantAssistantToolWriteReq
	TenantID string `json:"tenantId"`
}

func (h *Handlers) createPlatformAssistantTool(c *gin.Context) {
	var req platformAssistantToolCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	tid, err := assistantToolScopeTenantID(c, req.TenantID)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if req.Name == "" {
		response.FailWithCode(c, http.StatusBadRequest, "名称不能为空", nil)
		return
	}
	row := &models.TenantAssistantTool{
		TenantID:     tid,
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Kind:         models.NormalizeAssistantToolKind(req.Kind),
		Enabled:      true,
		Method:       req.Method,
		URL:          req.URL,
		BodyTemplate: req.BodyTemplate,
		McpCommand:   req.McpCommand,
		McpSSEURL:    req.McpSSEURL,
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.TimeoutMS != nil {
		row.TimeoutMS = *req.TimeoutMS
	}
	if len(req.Headers) > 0 {
		row.HeadersJSON = datatypes.JSON(req.Headers)
	}
	if len(req.Parameters) > 0 {
		row.ParametersJSON = datatypes.JSON(req.Parameters)
	}
	if len(req.McpArgs) > 0 {
		row.McpArgsJSON = datatypes.JSON(req.McpArgs)
	}
	if len(req.McpEnvs) > 0 {
		row.McpEnvsJSON = datatypes.JSON(req.McpEnvs)
	}
	if err := models.CreateTenantAssistantTool(h.db, row); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	item := tenantAssistantToolResp(*row, nil)
	names := h.tenantNamesForAssistantTools([]models.TenantAssistantTool{*row})
	item["tenantName"] = names[row.TenantID]
	response.SuccessI18n(c, i18n.KeySuccess, item)
}

func (h *Handlers) updatePlatformAssistantTool(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantAssistantToolByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req tenantAssistantToolWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Kind != "" {
		updates["kind"] = models.NormalizeAssistantToolKind(req.Kind)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Method != "" {
		updates["method"] = req.Method
	}
	if req.URL != "" {
		updates["url"] = req.URL
	}
	if len(req.Headers) > 0 {
		updates["headers_json"] = datatypes.JSON(req.Headers)
	}
	if req.BodyTemplate != "" {
		updates["body_template"] = req.BodyTemplate
	}
	if req.TimeoutMS != nil {
		updates["timeout_ms"] = *req.TimeoutMS
	}
	if len(req.Parameters) > 0 {
		updates["parameters_json"] = datatypes.JSON(req.Parameters)
	}
	if req.McpCommand != "" {
		updates["mcp_command"] = req.McpCommand
	}
	if len(req.McpArgs) > 0 {
		updates["mcp_args_json"] = datatypes.JSON(req.McpArgs)
	}
	if len(req.McpEnvs) > 0 {
		updates["mcp_envs_json"] = datatypes.JSON(req.McpEnvs)
	}
	if req.McpSSEURL != "" {
		updates["mcp_sse_url"] = req.McpSSEURL
	}
	if len(updates) == 0 {
		response.SuccessI18n(c, i18n.KeySuccess, tenantAssistantToolResp(row, nil))
		return
	}
	if ginutil.WriteInternalError(c, models.UpdateTenantAssistantTool(h.db, row.TenantID, id, updates)) {
		return
	}
	row, _ = models.GetTenantAssistantToolByID(h.db, id)
	item := tenantAssistantToolResp(row, nil)
	names := h.tenantNamesForAssistantTools([]models.TenantAssistantTool{row})
	item["tenantName"] = names[row.TenantID]
	response.SuccessI18n(c, i18n.KeySuccess, item)
}

func (h *Handlers) deletePlatformAssistantTool(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantAssistantToolByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if ginutil.WriteInternalError(c, models.DeleteTenantAssistantTool(h.db, row.TenantID, id)) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": strconv.FormatUint(uint64(id), 10)})
}

func assistantToolScopeTenantID(c *gin.Context, bodyTenantID string) (uint, error) {
	if middleware.AuthPlatformAdminID(c) == 0 {
		tid := middleware.CurrentTenantID(c)
		if tid == 0 {
			return 0, fmt.Errorf("tenant context required")
		}
		return tid, nil
	}
	raw := strings.TrimSpace(bodyTenantID)
	if raw == "" {
		if q := strings.TrimSpace(c.Query("tenantId")); q != "" {
			raw = q
		}
	}
	if raw == "" || raw == "0" {
		return 0, fmt.Errorf("tenantId is required for platform admin")
	}
	return utils.RequireScopeID(raw, "tenantId")
}
