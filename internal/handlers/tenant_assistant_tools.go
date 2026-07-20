package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func (h *Handlers) registerTenantAssistantToolRoutes(g *humax.Group) {
	read := g.Group("")
	read.Use(middleware.RequireTenantPermissionAny(
		"api.assistants.read",
	))
	{
		read.GET("/assistant-tools", h.listTenantAssistantTools)
		read.GET("/assistant-tools/:id", h.getTenantAssistantTool)
	}
	write := g.Group("")
	write.Use(middleware.RequireTenantPermissionAny(
		"api.assistants.write",
	))
	{
		write.POST("/assistant-tools", h.createTenantAssistantTool)
		write.PUT("/assistant-tools/:id", h.updateTenantAssistantTool)
		write.DELETE("/assistant-tools/:id", h.deleteTenantAssistantTool)
		write.POST("/assistant-tools/:id/discover", h.discoverTenantAssistantTool)
	}
}

func tenantAssistantToolResp(row models.TenantAssistantTool, published *models.McpMarketItem) gin.H {
	var headers any
	var parameters any
	var mcpArgs any
	var mcpEnvs any
	var discovered any
	_ = json.Unmarshal(row.HeadersJSON, &headers)
	_ = json.Unmarshal(row.ParametersJSON, &parameters)
	_ = json.Unmarshal(row.McpArgsJSON, &mcpArgs)
	_ = json.Unmarshal(row.McpEnvsJSON, &mcpEnvs)
	_ = json.Unmarshal(row.DiscoveredToolsJSON, &discovered)
	marketItemID := ""
	if row.MarketItemID != nil {
		marketItemID = strconv.FormatUint(uint64(*row.MarketItemID), 10)
	}
	src := row.Source
	if src == "" {
		src = models.AssistantToolSourceCustom
	}
	marketPublished := published != nil && published.Status == models.McpMarketStatusPublished
	publishedMarketItemID := ""
	if published != nil {
		publishedMarketItemID = strconv.FormatUint(uint64(published.ID), 10)
	}
	return gin.H{
		"id":                    strconv.FormatUint(uint64(row.ID), 10),
		"tenantId":              strconv.FormatUint(uint64(row.TenantID), 10),
		"name":                  row.Name,
		"displayName":           row.DisplayName,
		"description":           row.Description,
		"kind":                  row.Kind,
		"enabled":               row.Enabled,
		"source":                src,
		"marketItemId":          marketItemID,
		"marketPublished":       marketPublished,
		"publishedMarketItemId": publishedMarketItemID,
		"method":                row.Method,
		"url":                   row.URL,
		"headers":               headers,
		"bodyTemplate":          row.BodyTemplate,
		"timeoutMs":             row.TimeoutMS,
		"parameters":            parameters,
		"mcpCommand":            row.McpCommand,
		"mcpArgs":               mcpArgs,
		"mcpEnvs":               mcpEnvs,
		"mcpSseUrl":             row.McpSSEURL,
		"discoveredTools":       discovered,
		"createdAt":             row.CreatedAt,
		"updatedAt":             row.UpdatedAt,
	}
}

type tenantAssistantToolWriteReq struct {
	Name         string          `json:"name"`
	DisplayName  string          `json:"displayName"`
	Description  string          `json:"description"`
	Kind         string          `json:"kind"`
	Enabled      *bool           `json:"enabled"`
	Method       string          `json:"method"`
	URL          string          `json:"url"`
	Headers      json.RawMessage `json:"headers"`
	BodyTemplate string          `json:"bodyTemplate"`
	TimeoutMS    *int            `json:"timeoutMs"`
	Parameters   json.RawMessage `json:"parameters"`
	McpCommand   string          `json:"mcpCommand"`
	McpArgs      json.RawMessage `json:"mcpArgs"`
	McpEnvs      json.RawMessage `json:"mcpEnvs"`
	McpSSEURL    string          `json:"mcpSseUrl"`
}

func (h *Handlers) listTenantAssistantTools(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	if tid == 0 {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	source := c.Query("source")
	enabledOnly := c.Query("enabled") == "1" || c.Query("enabled") == "true"
	rows, err := models.ListTenantAssistantToolsFiltered(h.db, tid, source, enabledOnly)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	var publishedMap map[uint]models.McpMarketItem
	if source == "" || source == models.AssistantToolSourceCustom {
		toolIDs := make([]uint, 0, len(rows))
		for _, row := range rows {
			if models.NormalizeAssistantToolKind(row.Kind) == models.AssistantToolKindMCPSSE &&
				row.Source != models.AssistantToolSourceMarket {
				toolIDs = append(toolIDs, row.ID)
			}
		}
		publishedMap, err = models.MapPublishedMcpMarketItemsBySourceTool(h.db, tid, toolIDs)
		if ginutil.WriteInternalError(c, err) {
			return
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		var published *models.McpMarketItem
		if publishedMap != nil {
			if item, ok := publishedMap[row.ID]; ok {
				cp := item
				published = &cp
			}
		}
		out = append(out, tenantAssistantToolResp(row, published))
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) getTenantAssistantTool(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantAssistantTool(h.db, tid, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var published *models.McpMarketItem
	if row.Source != models.AssistantToolSourceMarket {
		published, _ = models.GetPublishedMcpMarketItemForSourceTool(h.db, tid, id)
	}
	response.SuccessI18n(c, i18n.KeySuccess, tenantAssistantToolResp(row, published))
}

func (h *Handlers) createTenantAssistantTool(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req tenantAssistantToolWriteReq
	if !ginutil.BindJSON(c, &req) {
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
		Source:       models.AssistantToolSourceCustom,
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
	response.SuccessI18n(c, i18n.KeySuccess, tenantAssistantToolResp(*row, nil))
}

func (h *Handlers) updateTenantAssistantTool(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetTenantAssistantTool(h.db, tid, id); err != nil {
		ginutil.WriteGORMError(c, err, "not found")
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
		row, _ := models.GetTenantAssistantTool(h.db, tid, id)
		response.SuccessI18n(c, i18n.KeySuccess, tenantAssistantToolResp(row, nil))
		return
	}
	if ginutil.WriteInternalError(c, models.UpdateTenantAssistantTool(h.db, tid, id, updates)) {
		return
	}
	row, _ := models.GetTenantAssistantTool(h.db, tid, id)
	response.SuccessI18n(c, i18n.KeySuccess, tenantAssistantToolResp(row, nil))
}

func (h *Handlers) deleteTenantAssistantTool(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetTenantAssistantTool(h.db, tid, id); err != nil {
		ginutil.WriteGORMError(c, err, "not found")
		return
	}
	if err := models.AssertTenantAssistantToolDeletable(h.db, tid, id); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if ginutil.WriteInternalError(c, models.DeleteTenantAssistantTool(h.db, tid, id)) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": strconv.FormatUint(uint64(id), 10)})
}

func (h *Handlers) discoverTenantAssistantTool(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantAssistantTool(h.db, tid, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if models.NormalizeAssistantToolKind(row.Kind) != models.AssistantToolKindMCPSSE {
		response.FailWithCode(c, http.StatusBadRequest, "discover is only supported for mcp_sse tools", nil)
		return
	}
	headers, _ := models.ParseToolHeadersJSON(row.HeadersJSON)
	tools, err := providers.DiscoverMCPSSETools(c.Request.Context(), row.McpSSEURL, headers, row.TimeoutMS)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	raw, err := json.Marshal(tools)
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if ginutil.WriteInternalError(c, models.UpdateTenantAssistantTool(h.db, tid, id, map[string]any{
		"discovered_tools_json": datatypes.JSON(raw),
	})) {
		return
	}
	row, _ = models.GetTenantAssistantTool(h.db, tid, id)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"tool":            tenantAssistantToolResp(row, nil),
		"discoveredTools": tools,
	})
}
