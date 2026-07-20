package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerAIInvocationLogRoutes(r *humax.Group) {
	tenant := r.Group("ai-invocations")
	tenant.Use(middleware.RequireTenantPermissionAll(constants.PermAPIAIInvocationsRead))
	{
		tenant.GET("/tenant", h.listTenantAIInvocationLogs)
		tenant.GET("/tenant/:id", h.getTenantAIInvocationLog)
	}

	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	adminAI := admin.Group("admin/ai-invocations")
	{
		adminAI.GET("", h.listPlatformAIInvocationLogs)
		adminAI.GET("/:id", h.getPlatformAIInvocationLog)
	}
}

func aiInvocationFilterFromQuery(c *gin.Context) models.AIInvocationLogFilter {
	return models.AIInvocationLogFilter{
		Component: strings.TrimSpace(c.Query("component")),
		Status:    strings.TrimSpace(c.Query("status")),
		Provider:  strings.TrimSpace(c.Query("provider")),
		CallID:    strings.TrimSpace(c.Query("call_id")),
		Source:    strings.TrimSpace(c.Query("source")),
	}
}

func (h *Handlers) listTenantAIInvocationLogs(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	if tid == 0 {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	page, pageSize := ginutil.QueryPage(c, 200)
	f := aiInvocationFilterFromQuery(c)
	f.TenantID = tid
	list, total, err := models.ListAIInvocationLogsPage(h.db, page, pageSize, f)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]models.AIInvocationLogTenantView, 0, len(list))
	for _, row := range list {
		out = append(out, models.ToTenantView(row))
	}
	ginutil.PageSuccess(c, out, total, page, pageSize)
}

func (h *Handlers) getTenantAIInvocationLog(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	if tid == 0 {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.AIInvocationLog
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tid).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "AI invocation log not found", nil)
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, models.ToTenantView(row))
}

func (h *Handlers) listPlatformAIInvocationLogs(c *gin.Context) {
	page, pageSize := ginutil.QueryPage(c, 200)
	f := aiInvocationFilterFromQuery(c)
	if tidStr := strings.TrimSpace(c.Query("tenant_id")); tidStr != "" {
		if parsed, err := strconv.ParseUint(tidStr, 10, 64); err == nil && parsed > 0 {
			f.TenantID = uint(parsed)
		}
	}
	list, total, err := models.ListAIInvocationLogsPage(h.db, page, pageSize, f)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	// List omits bodies; use GET /:id for full request/response text.
	out := make([]models.AIInvocationLogTenantView, 0, len(list))
	for _, row := range list {
		out = append(out, models.ToTenantView(row))
	}
	ginutil.PageSuccess(c, out, total, page, pageSize)
}

func (h *Handlers) getPlatformAIInvocationLog(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.AIInvocationLog
	if err := h.db.Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "AI invocation log not found", nil)
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}
