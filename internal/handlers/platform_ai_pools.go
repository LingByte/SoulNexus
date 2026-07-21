package handlers

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type aiPoolWriteReq struct {
	Name        string          `json:"name"`
	Modality    string          `json:"modality"`
	Provider    string          `json:"provider"`
	Config      json.RawMessage `json:"config"`
	VoiceIDs    []string        `json:"voiceIds"`
	Priority    int             `json:"priority"`
	Enabled     *bool           `json:"enabled"`
	QuotaLimit  int64           `json:"quotaLimit"`
	Description string          `json:"description"`
}

func (h *Handlers) registerPlatformAIPoolRoutes(r *humax.Group) {
	g := r.Group("platform/ai-pools")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("", h.listAIProviderPools)
		g.POST("", h.createAIProviderPool)
		g.PUT("/:id", h.updateAIProviderPool)
		g.DELETE("/:id", h.deleteAIProviderPool)
	}
}

func (h *Handlers) listAIProviderPools(c *gin.Context) {
	modality := strings.TrimSpace(c.Query("modality"))
	q := h.db.Model(&models.AIProviderPool{})
	if modality != "" {
		q = q.Where("modality = ?", models.NormalizeAIPoolModality(modality))
	}
	var rows []models.AIProviderPool
	if ginutil.WriteInternalError(c, q.Order("priority DESC, id ASC").Find(&rows).Error) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": rows})
}

func (h *Handlers) createAIProviderPool(c *gin.Context) {
	var req aiPoolWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	modality := models.NormalizeAIPoolModality(req.Modality)
	if modality == "" || strings.TrimSpace(req.Provider) == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid pool", nil))
		return
	}
	cfg, err := models.ParseOptionalJSONColumn(string(req.Config))
	if err != nil || len(cfg) == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid pool", nil))
		return
	}
	voiceJSON, _ := json.Marshal(req.VoiceIDs)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := models.AIProviderPool{
		Name:        strings.TrimSpace(req.Name),
		Modality:    modality,
		Provider:    strings.TrimSpace(req.Provider),
		Config:      cfg,
		VoiceIDs:    datatypes.JSON(voiceJSON),
		Priority:    req.Priority,
		Enabled:     enabled,
		QuotaLimit:  req.QuotaLimit,
		Description: strings.TrimSpace(req.Description),
	}
	if row.Name == "" {
		row.Name = row.Provider + " " + modality
	}
	if ginutil.WriteInternalError(c, h.db.Create(&row).Error) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) updateAIProviderPool(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.AIProviderPool
	if ginutil.WriteGORMError(c, h.db.First(&row, id).Error, "not found") {
		return
	}
	var req aiPoolWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if m := models.NormalizeAIPoolModality(req.Modality); m != "" {
		updates["modality"] = m
	}
	if p := strings.TrimSpace(req.Provider); p != "" {
		updates["provider"] = p
	}
	if len(req.Config) > 0 {
		cfg, err := models.ParseOptionalJSONColumn(string(req.Config))
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "invalid pool", nil))
			return
		}
		updates["config"] = cfg
	}
	if req.VoiceIDs != nil {
		voiceJSON, _ := json.Marshal(req.VoiceIDs)
		updates["voice_ids"] = datatypes.JSON(voiceJSON)
	}
	if req.Priority != 0 || c.Request.ContentLength > 0 {
		updates["priority"] = req.Priority
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.QuotaLimit >= 0 {
		updates["quota_limit"] = req.QuotaLimit
	}
	if req.Description != "" {
		updates["description"] = strings.TrimSpace(req.Description)
	}
	if ginutil.WriteInternalError(c, h.db.Model(&row).Updates(updates).Error) {
		return
	}
	_ = h.db.First(&row, id)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) deleteAIProviderPool(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if ginutil.WriteInternalError(c, h.db.Delete(&models.AIProviderPool{}, id).Error) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}
