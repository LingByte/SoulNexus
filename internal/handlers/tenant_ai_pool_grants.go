package handlers

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type tenantPoolGrantWriteReq struct {
	PoolID     uint  `json:"poolId"`
	QuotaLimit int64 `json:"quotaLimit"`
	Enabled    *bool `json:"enabled"`
}

func (h *Handlers) registerTenantAIPoolGrantRoutes(r *humax.Group) {
	g := r.Group("tenants")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("/:id/ai-pool-grants", h.listTenantAIPoolGrants)
		g.POST("/:id/ai-pool-grants", h.upsertTenantAIPoolGrant)
		g.DELETE("/:id/ai-pool-grants/:grantId", h.deleteTenantAIPoolGrant)
	}
}

func (h *Handlers) listTenantAIPoolGrants(c *gin.Context) {
	tenantID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetActiveTenantByID(h.db, tenantID); ginutil.WriteGORMError(c, err, "tenant not found") {
		return
	}
	var grants []models.TenantAIPoolGrant
	if ginutil.WriteInternalError(c, h.db.Where("tenant_id = ?", tenantID).Order("id DESC").Find(&grants).Error) {
		return
	}
	type row struct {
		models.TenantAIPoolGrant
		Pool *models.AIProviderPool `json:"pool,omitempty"`
	}
	out := make([]row, 0, len(grants))
	for _, g := range grants {
		item := row{TenantAIPoolGrant: g}
		var p models.AIProviderPool
		if err := h.db.First(&p, g.PoolID).Error; err == nil {
			item.Pool = &p
		}
		out = append(out, item)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": out})
}

func (h *Handlers) upsertTenantAIPoolGrant(c *gin.Context) {
	tenantID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := models.GetActiveTenantByID(h.db, tenantID); ginutil.WriteGORMError(c, err, "tenant not found") {
		return
	}
	var req tenantPoolGrantWriteReq
	if !ginutil.BindJSON(c, &req) || req.PoolID == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "poolId required", nil))
		return
	}
	var pool models.AIProviderPool
	if ginutil.WriteGORMError(c, h.db.First(&pool, req.PoolID).Error, "pool not found") {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var existing models.TenantAIPoolGrant
	err := h.db.Where("tenant_id = ? AND pool_id = ?", tenantID, req.PoolID).First(&existing).Error
	if err != nil {
		row := models.TenantAIPoolGrant{
			TenantID:   tenantID,
			PoolID:     req.PoolID,
			QuotaLimit: req.QuotaLimit,
			Enabled:    enabled,
		}
		row.SetCreateInfo(middleware.AuthEmail(c))
		if ginutil.WriteInternalError(c, h.db.Create(&row).Error) {
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, row)
		return
	}
	updates := map[string]any{
		"quota_limit": req.QuotaLimit,
		"enabled":     enabled,
	}
	if ginutil.WriteInternalError(c, h.db.Model(&existing).Updates(updates).Error) {
		return
	}
	_ = h.db.First(&existing, existing.ID)
	response.SuccessI18n(c, i18n.KeySuccess, existing)
}

func (h *Handlers) deleteTenantAIPoolGrant(c *gin.Context) {
	tenantID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	grantID, ok := ginutil.ParamID(c, "grantId")
	if !ok {
		return
	}
	res := h.db.Where("id = ? AND tenant_id = ?", grantID, tenantID).Delete(&models.TenantAIPoolGrant{})
	if ginutil.WriteInternalError(c, res.Error) {
		return
	}
	if res.RowsAffected == 0 {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": grantID})
}
