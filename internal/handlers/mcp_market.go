package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

const maxMcpMarketLogoBytes = 5 << 20

func (h *Handlers) registerTenantMcpMarketRoutes(g *humax.Group) {
	read := g.Group("")
	read.Use(middleware.RequireTenantPermissionAny("api.assistants.read"))
	{
		read.GET("/mcp-market", h.listTenantMcpMarket)
		read.GET("/mcp-market/:id", h.getTenantMcpMarketItem)
	}
	write := g.Group("")
	write.Use(middleware.RequireTenantPermissionAny("api.assistants.write"))
	{
		write.POST("/mcp-market/:id/activate", h.activateTenantMcpMarketItem)
		write.POST("/mcp-market/publish", h.publishCustomMcpToMarket)
		write.POST("/mcp-market/delist", h.delistTenantMcpMarketItem)
		write.POST("/mcp-market/logo", h.uploadMcpMarketLogo)
	}
}

func mcpMarketItemResp(row models.McpMarketItem, activated bool) gin.H {
	var headers any
	_ = json.Unmarshal(row.HeadersJSON, &headers)
	return gin.H{
		"id":             strconv.FormatUint(uint64(row.ID), 10),
		"slug":           row.Slug,
		"name":           row.Name,
		"displayName":    row.DisplayName,
		"description":    row.Description,
		"category":       row.Category,
		"icon":           row.Icon,
		"logoUrl":        row.LogoURL,
		"tags":           row.Tags,
		"version":        row.Version,
		"status":         row.Status,
		"author":         row.Author,
		"authorTenantId": strconv.FormatUint(uint64(row.AuthorTenantID), 10),
		"sourceToolId": func() string {
			if row.SourceToolID == nil {
				return ""
			}
			return strconv.FormatUint(uint64(*row.SourceToolID), 10)
		}(),
		"kind":         row.Kind,
		"mcpSseUrl":    row.McpSSEURL,
		"headers":      headers,
		"timeoutMs":    row.TimeoutMS,
		"installCount": row.InstallCount,
		"activated":    activated,
		"createdAt":    row.CreatedAt,
		"updatedAt":    row.UpdatedAt,
	}
}

func (h *Handlers) listTenantMcpMarket(c *gin.Context) {
	_ = models.EnsureDefaultMcpMarketItems(h.db)
	tid := middleware.CurrentTenantID(c)
	category := c.Query("category")
	keyword := c.Query("keyword")
	rows, err := models.ListPublishedMcpMarketItems(h.db, category, keyword)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	activated := map[uint]struct{}{}
	if tid > 0 {
		mine, _ := models.ListTenantAssistantToolsFiltered(h.db, tid, models.AssistantToolSourceMarket, false)
		for _, r := range mine {
			if r.MarketItemID != nil {
				activated[*r.MarketItemID] = struct{}{}
			}
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		_, ok := activated[row.ID]
		out = append(out, mcpMarketItemResp(row, ok))
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) getTenantMcpMarketItem(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetMcpMarketItem(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if row.Status != models.McpMarketStatusPublished {
		response.FailWithCode(c, http.StatusNotFound, "not found", nil)
		return
	}
	tid := middleware.CurrentTenantID(c)
	activated := false
	if tid > 0 {
		var n int64
		_ = h.db.Model(&models.TenantAssistantTool{}).
			Where("tenant_id = ? AND market_item_id = ?", tid, id).
			Count(&n).Error
		activated = n > 0
	}
	response.SuccessI18n(c, i18n.KeySuccess, mcpMarketItemResp(row, activated))
}

func (h *Handlers) activateTenantMcpMarketItem(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.ActivateMcpMarketItemForTenant(h.db, tid, id)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, tenantAssistantToolResp(*row, nil))
}

type publishCustomMcpReq struct {
	ToolID      string `json:"toolId"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Version     string `json:"version"`
	LogoURL     string `json:"logoUrl"`
	Tags        string `json:"tags"`
	Publish     *bool  `json:"publish"` // true → published immediately
}

func (h *Handlers) publishCustomMcpToMarket(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req publishCustomMcpReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	toolID, err := strconv.ParseUint(strings.TrimSpace(req.ToolID), 10, 64)
	if err != nil || toolID == 0 {
		response.FailWithCode(c, http.StatusBadRequest, "toolId is required", nil)
		return
	}
	tool, err := models.GetTenantAssistantTool(h.db, tid, uint(toolID))
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if models.NormalizeAssistantToolKind(tool.Kind) != models.AssistantToolKindMCPSSE {
		response.FailWithCode(c, http.StatusBadRequest, "only mcp_sse custom tools can be published", nil)
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = tool.Name
	}
	display := req.DisplayName
	if display == "" {
		display = tool.DisplayName
	}
	desc := req.Description
	if desc == "" {
		desc = tool.Description
	}
	cat := req.Category
	if cat == "" {
		cat = models.McpMarketCategoryCustom
	}
	publish := req.Publish != nil && *req.Publish
	item, err := models.PublishOrUpdateTenantMcpMarketItem(h.db, models.PublishTenantMcpMarketInput{
		TenantID:    tid,
		ToolID:      uint(toolID),
		Slug:        slug,
		Name:        tool.Name,
		DisplayName: display,
		Description: desc,
		Category:    cat,
		Version:     strings.TrimSpace(req.Version),
		LogoURL:     strings.TrimSpace(req.LogoURL),
		Tags:        strings.TrimSpace(req.Tags),
		Publish:     publish,
		McpSSEURL:   tool.McpSSEURL,
		HeadersJSON: tool.HeadersJSON,
		TimeoutMS:   tool.TimeoutMS,
	})
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if publish {
		h.recordOpChange(c, OpLogEntry{
			TenantID: tid, Action: constants.OpActionPublish,
			Resource: constants.OpResourceMCPMarket, ResourceID: item.ID, ResourceName: item.DisplayName,
			Summary: fmt.Sprintf("Published MCP tool %s", item.DisplayName),
		}, nil, item)
	}
	response.SuccessI18n(c, i18n.KeySuccess, mcpMarketItemResp(*item, false))
}

type delistTenantMcpReq struct {
	ToolID string `json:"toolId"`
}

func (h *Handlers) delistTenantMcpMarketItem(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req delistTenantMcpReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	toolID, err := strconv.ParseUint(strings.TrimSpace(req.ToolID), 10, 64)
	if err != nil || toolID == 0 {
		response.FailWithCode(c, http.StatusBadRequest, "toolId is required", nil)
		return
	}
	item, err := models.DelistTenantMcpMarketItem(h.db, tid, uint(toolID))
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, mcpMarketItemResp(*item, false))
}

func (h *Handlers) uploadMcpMarketLogo(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeySelectImageFile))
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyCannotReadFile))
		return
	}
	defer src.Close()
	body, err := io.ReadAll(io.LimitReader(src, maxMcpMarketLogoBytes+1))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if len(body) > maxMcpMarketLogoBytes {
		response.FailWithCode(c, http.StatusBadRequest, "logo must be 5MB or smaller", nil)
		return
	}
	ct := http.DetectContentType(body)
	ext := utils.PickImageExtFromContentType(ct)
	if ext != ".jpg" && ext != ".png" {
		response.FailWithCode(c, http.StatusBadRequest, "logo must be JPG or PNG", nil)
		return
	}
	key := path.Join(
		"mcp-market-logos",
		"t"+strconv.FormatUint(uint64(tid), 10),
		fmt.Sprintf("logo_%d%s", time.Now().UnixMilli(), ext),
	)
	st := stores.Default()
	if err := st.Write(key, bytes.NewReader(body)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	url := ginutil.UploadURL(c, key)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"logoUrl": url})
}

func (h *Handlers) registerPlatformMcpMarketRoutes(r *humax.Group) {
	g := r.Group("platform/mcp-market")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("", h.listPlatformMcpMarket)
		g.POST("", h.createPlatformMcpMarketItem)
		g.PUT("/:id", h.updatePlatformMcpMarketItem)
		g.DELETE("/:id", h.deletePlatformMcpMarketItem)
		g.POST("/logo", h.uploadPlatformMcpMarketLogo)
	}
}

func (h *Handlers) listPlatformMcpMarket(c *gin.Context) {
	_ = models.EnsureDefaultMcpMarketItems(h.db)
	page, size := ginutil.QueryPage(c, 50)
	rows, total, err := models.ListMcpMarketItemsAdmin(h.db, c.Query("status"), page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, mcpMarketItemResp(row, false))
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": out, "total": total, "page": page, "size": size})
}

type platformMcpMarketWriteReq struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Icon        string          `json:"icon"`
	LogoURL     string          `json:"logoUrl"`
	Tags        string          `json:"tags"`
	Version     string          `json:"version"`
	Status      string          `json:"status"`
	Author      string          `json:"author"`
	McpSSEURL   string          `json:"mcpSseUrl"`
	Headers     json.RawMessage `json:"headers"`
	TimeoutMS   *int            `json:"timeoutMs"`
}

func (h *Handlers) createPlatformMcpMarketItem(c *gin.Context) {
	var req platformMcpMarketWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	row := &models.McpMarketItem{
		Slug:        req.Slug,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Category:    req.Category,
		Icon:        req.Icon,
		LogoURL:     strings.TrimSpace(req.LogoURL),
		Tags:        strings.TrimSpace(req.Tags),
		Version:     req.Version,
		Status:      req.Status,
		Author:      req.Author,
		Kind:        models.AssistantToolKindMCPSSE,
		McpSSEURL:   req.McpSSEURL,
	}
	if req.TimeoutMS != nil {
		row.TimeoutMS = *req.TimeoutMS
	}
	if len(req.Headers) > 0 {
		row.HeadersJSON = datatypes.JSON(req.Headers)
	}
	if row.Author == "" {
		row.Author = "platform"
	}
	if err := models.CreateMcpMarketItem(h.db, row); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, mcpMarketItemResp(*row, false))
}

func (h *Handlers) updatePlatformMcpMarketItem(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetMcpMarketItem(h.db, id); err != nil {
		ginutil.WriteGORMError(c, err, "not found")
		return
	}
	var req platformMcpMarketWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Slug != "" {
		updates["slug"] = req.Slug
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.LogoURL != "" {
		updates["logo_url"] = strings.TrimSpace(req.LogoURL)
	}
	if req.Tags != "" {
		updates["tags"] = strings.TrimSpace(req.Tags)
	}
	if req.Version != "" {
		updates["version"] = req.Version
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Author != "" {
		updates["author"] = req.Author
	}
	if req.McpSSEURL != "" {
		updates["mcp_sse_url"] = req.McpSSEURL
	}
	if req.TimeoutMS != nil {
		updates["timeout_ms"] = *req.TimeoutMS
	}
	if len(req.Headers) > 0 {
		updates["headers_json"] = datatypes.JSON(req.Headers)
	}
	if ginutil.WriteInternalError(c, models.UpdateMcpMarketItem(h.db, id, updates)) {
		return
	}
	row, _ := models.GetMcpMarketItem(h.db, id)
	response.SuccessI18n(c, i18n.KeySuccess, mcpMarketItemResp(row, false))
}

func (h *Handlers) deletePlatformMcpMarketItem(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if ginutil.WriteInternalError(c, models.DeleteMcpMarketItem(h.db, id)) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": strconv.FormatUint(uint64(id), 10)})
}

func (h *Handlers) uploadPlatformMcpMarketLogo(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeySelectImageFile))
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyCannotReadFile))
		return
	}
	defer src.Close()
	body, err := io.ReadAll(io.LimitReader(src, maxMcpMarketLogoBytes+1))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if len(body) > maxMcpMarketLogoBytes {
		response.FailWithCode(c, http.StatusBadRequest, "logo must be 5MB or smaller", nil)
		return
	}
	ct := http.DetectContentType(body)
	ext := utils.PickImageExtFromContentType(ct)
	if ext != ".jpg" && ext != ".png" {
		response.FailWithCode(c, http.StatusBadRequest, "logo must be JPG or PNG", nil)
		return
	}
	key := path.Join(
		"mcp-market-logos",
		"platform",
		fmt.Sprintf("logo_%d%s", time.Now().UnixMilli(), ext),
	)
	st := stores.Default()
	if err := st.Write(key, bytes.NewReader(body)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	url := ginutil.UploadURL(c, key)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"logoUrl": url})
}
