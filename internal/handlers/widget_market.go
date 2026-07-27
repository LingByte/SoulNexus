package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func widgetMarketCORS(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type")
}

func widgetMarketPublicDTO(row models.WidgetMarketItem) gin.H {
	embedPath := fmt.Sprintf("/lingecho/embed/v1/t/%s/embed.js", row.JsSourceID)
	return gin.H{
		"id":                 strconv.FormatUint(uint64(row.ID), 10),
		"slug":               row.Slug,
		"name":               row.Name,
		"displayName":        row.DisplayName,
		"description":        row.Description,
		"category":           row.Category,
		"avatarUrl":          row.AvatarURL,
		"tags":               row.Tags,
		"version":            row.Version,
		"author":             row.Author,
		"jsSourceId":         row.JsSourceID,
		"embedPath":          embedPath,
		"installCount":       row.InstallCount,
		"sourceJsTemplateId": strconv.FormatUint(uint64(row.SourceJSTemplateID), 10),
	}
}

func (h *Handlers) registerWidgetMarketPublicRoutes(r *humax.Group) {
	g := r.Group("public/widget-market")
	{
		g.GET("/items", h.publicListWidgetMarketItems)
		g.GET("/items/:id", h.publicGetWidgetMarketItem)
		g.GET("/items/:id/download", h.publicTrackWidgetMarketDownload)
		g.POST("/items/:id/download", h.publicTrackWidgetMarketDownload)
	}
}

func (h *Handlers) registerWidgetMarketRoutes(r *humax.Group) {
	read := r.Group("")
	read.Use(middleware.RequireTenantPermissionAny("api.assistants.read", "menu.res.assistant"))
	{
		read.GET("/widget-market/items", h.tenantListWidgetMarketItems)
		read.GET("/widget-market/items/:id", h.tenantGetWidgetMarketItem)
	}
	write := r.Group("")
	write.Use(middleware.RequireTenantPermissionAny("api.assistants.write", "menu.res.assistant"))
	write.Use(middleware.RequireHumanJWTUser())
	{
		write.POST("/widget-market/publish", h.publishJSTemplateToWidgetMarket)
		write.POST("/widget-market/delist", h.delistWidgetMarketItem)
	}
}

func (h *Handlers) publicListWidgetMarketItems(c *gin.Context) {
	widgetMarketCORS(c)
	page, size := ginutil.QueryPage(c, 24)
	list, total, err := models.ListPublishedWidgetMarketItemsPage(
		h.db, c.Query("category"), c.Query("keyword"), page, size,
	)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, row := range list {
		out = append(out, widgetMarketPublicDTO(row))
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

func (h *Handlers) publicGetWidgetMarketItem(c *gin.Context) {
	widgetMarketCORS(c)
	param := strings.TrimSpace(c.Param("id"))
	var row models.WidgetMarketItem
	var err error
	if id, parseErr := strconv.ParseUint(param, 10, 64); parseErr == nil && id > 0 {
		row, err = models.GetWidgetMarketItemPublished(h.db, uint(id))
	} else {
		row, err = models.GetWidgetMarketItemByJsSourceIDPublished(h.db, param)
	}
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.Success(c, "", widgetMarketPublicDTO(row))
}

func (h *Handlers) publicTrackWidgetMarketDownload(c *gin.Context) {
	widgetMarketCORS(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := models.IncrementWidgetMarketInstallCount(h.db, id); ginutil.WriteInternalError(c, err) {
		return
	}
	response.Success(c, "", gin.H{"ok": true})
}

func (h *Handlers) tenantListWidgetMarketItems(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 24)
	list, total, err := models.ListPublishedWidgetMarketItemsPage(
		h.db, c.Query("category"), c.Query("keyword"), page, size,
	)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, row := range list {
		out = append(out, widgetMarketPublicDTO(row))
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

func (h *Handlers) tenantGetWidgetMarketItem(c *gin.Context) {
	h.publicGetWidgetMarketItem(c)
}

type publishWidgetMarketReq struct {
	JSTemplateID string `json:"jsTemplateId" binding:"required"`
	DisplayName  string `json:"displayName"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Tags         string `json:"tags"`
	Author       string `json:"author"`
	Publish      bool   `json:"publish"`
}

func (h *Handlers) publishJSTemplateToWidgetMarket(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req publishWidgetMarketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误: "+err.Error(), nil)
		return
	}
	tplID, err := strconv.ParseUint(strings.TrimSpace(req.JSTemplateID), 10, 64)
	if err != nil || tplID == 0 {
		response.Fail(c, "无效的网页挂件 ID", nil)
		return
	}
	tpl, err := models.GetJSTemplateByIDForTenant(h.db, uint(tplID), tid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "网页挂件不存在", nil)
		} else {
			response.Fail(c, "查询失败", nil)
		}
		return
	}
	if tpl.Status != models.JSTemplateStatusActive {
		response.Fail(c, "请先启用该网页挂件（status=active）再发布到市场", nil)
		return
	}
	if _, err := models.GetActiveJSTemplateByJsSourceID(h.db, tpl.JsSourceID); err != nil {
		response.Fail(c, "挂件 embed 不可用，请确认已启用", nil)
		return
	}

	status := models.WidgetMarketStatusDraft
	if req.Publish {
		status = models.WidgetMarketStatusPublished
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = tpl.Name
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = strings.TrimSpace(tpl.Usage)
	}
	slugBase := models.NormalizeWidgetMarketSlug(displayName)
	if slugBase == "" {
		slugBase = "widget-" + tpl.JsSourceID
	}

	existing, err := models.GetWidgetMarketItemForTenantTemplate(h.db, tid, tpl.ID)
	if err != nil && err != gorm.ErrRecordNotFound {
		ginutil.WriteInternalError(c, err)
		return
	}
	if err == gorm.ErrRecordNotFound {
		item := &models.WidgetMarketItem{
			Slug:               slugBase,
			Name:               tpl.Name,
			DisplayName:        displayName,
			Description:        description,
			Category:           strings.TrimSpace(req.Category),
			Tags:               strings.TrimSpace(req.Tags),
			Status:             status,
			Author:             strings.TrimSpace(req.Author),
			AuthorTenantID:     tid,
			AvatarURL:          strings.TrimSpace(tpl.AvatarURL),
			JsSourceID:         tpl.JsSourceID,
			SourceJSTemplateID: tpl.ID,
		}
		if item.Category == "" {
			item.Category = models.WidgetMarketCategoryUtility
		}
		if err := models.CreateWidgetMarketItem(h.db, item); err != nil {
			response.Fail(c, "发布失败: "+err.Error(), nil)
			return
		}
		response.Success(c, "", widgetMarketPublicDTO(*item))
		return
	}

	updates := map[string]any{
		"display_name": displayName,
		"description":  description,
		"category":     strings.TrimSpace(req.Category),
		"tags":         strings.TrimSpace(req.Tags),
		"status":       status,
		"avatar_url":   strings.TrimSpace(tpl.AvatarURL),
		"js_source_id": tpl.JsSourceID,
		"name":         tpl.Name,
	}
	if a := strings.TrimSpace(req.Author); a != "" {
		updates["author"] = a
	}
	if err := models.UpdateWidgetMarketItem(h.db, existing.ID, updates); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	var row models.WidgetMarketItem
	_ = h.db.Where("id = ?", existing.ID).First(&row).Error
	response.Success(c, "", widgetMarketPublicDTO(row))
}

func (h *Handlers) delistWidgetMarketItem(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req struct {
		JSTemplateID string `json:"jsTemplateId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", nil)
		return
	}
	tplID, err := strconv.ParseUint(strings.TrimSpace(req.JSTemplateID), 10, 64)
	if err != nil || tplID == 0 {
		response.Fail(c, "无效的网页挂件 ID", nil)
		return
	}
	row, err := models.GetWidgetMarketItemForTenantTemplate(h.db, tid, uint(tplID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "未发布到市场", nil)
		} else {
			ginutil.WriteInternalError(c, err)
		}
		return
	}
	if err := models.UpdateWidgetMarketItem(h.db, row.ID, map[string]any{
		"status": models.WidgetMarketStatusArchived,
	}); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.Success(c, "", gin.H{"ok": true})
}
