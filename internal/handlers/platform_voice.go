package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/voice/cloneconfig"
	"github.com/LingByte/SoulNexus/pkg/voice/voiceprintconfig"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registerPlatformVoiceRoutes(r *humax.Group) {
	g := r.Group("platform/voices")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("/catalog", h.listPlatformVoiceCatalog)
		g.GET("/clone-profiles", h.listPlatformVoiceCloneProfiles)
		g.GET("/synthesis-history", h.listPlatformVoiceSynthesisHistory)
		g.DELETE("/synthesis-history/:id", h.deletePlatformVoiceSynthesisHistory)
		g.GET("/voiceprint/config", h.getPlatformVoiceprintConfig)
		g.GET("/voiceprint/self-test", h.platformVoiceprintSelfTest)
		g.GET("/voiceprint-profiles", h.listPlatformVoiceprintProfiles)
		g.DELETE("/voiceprint-profiles/:id", h.deletePlatformVoiceprintProfile)
	}
}

func (h *Handlers) listPlatformVoiceCatalog(c *gin.Context) {
	root, err := voiceCatalogDir()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice catalog unavailable", err))
		return
	}
	_ = root
	provider := strings.TrimSpace(c.Query("provider"))
	mode := strings.TrimSpace(c.Query("mode"))
	if provider == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyProviderRequired))
		return
	}
	if mode == "" {
		mode = "tts"
	}
	out, err := listVoiceCatalog(provider, mode)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeNotFound, err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) listPlatformVoiceCloneProfiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	tenantID := uint(0)
	if tid := strings.TrimSpace(c.Query("tenantId")); tid != "" {
		parsed, err := utils.ParseID(tid)
		if err != nil {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidTenantID))
			return
		}
		tenantID = parsed
	}
	offset := (page - 1) * pageSize
	rows, total, err := models.ListAllVoiceCloneProfiles(h.db, tenantID, pageSize, offset)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	slug, prov, enabled := cloneconfig.ResolveEnabled()
	label := ""
	if enabled {
		label = cloneconfig.ProviderLabel(prov)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"list":               rows,
		"total":              total,
		"page":               page,
		"pageSize":           pageSize,
		"cloneEnabled":       enabled,
		"cloneProvider":      slug,
		"cloneProviderLabel": label,
	})
}

func (h *Handlers) listPlatformVoiceSynthesisHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	tenantID := uint(0)
	if tid := strings.TrimSpace(c.Query("tenantId")); tid != "" {
		parsed, err := utils.ParseID(tid)
		if err != nil {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidTenantID))
			return
		}
		tenantID = parsed
	}
	offset := (page - 1) * pageSize
	rows, total, err := models.ListAllVoiceSynthesisHistory(h.db, tenantID, pageSize, offset)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	for i := range rows {
		if rows[i].AudioKey != "" {
			rows[i].AudioURL = ginutil.UploadURL(c, rows[i].AudioKey)
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"list":     rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) deletePlatformVoiceSynthesisHistory(c *gin.Context) {
	rawID := c.Param("id")
	id, err := utils.ParseID(rawID)
	if err != nil || id == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid id", err))
		return
	}
	res := h.db.Where("id = ?", id).Delete(&models.VoiceSynthesisHistory{})
	if res.Error != nil {
		ginutil.WriteInternalError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) getPlatformVoiceprintConfig(c *gin.Context) {
	slug, prov, ok := voiceprintconfig.ResolveEnabled()
	if !ok {
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"provider": "",
			"enabled":  false,
		})
		return
	}
	out := voiceprintconfig.ConfigSummary(prov)
	out["provider"] = slug
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) platformVoiceprintSelfTest(c *gin.Context) {
	probe := strings.EqualFold(strings.TrimSpace(c.Query("probe")), "true") ||
		strings.TrimSpace(c.Query("probe")) == "1"
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	report := voiceprintconfig.RunSelfTest(ctx, probe)
	response.SuccessI18n(c, i18n.KeySuccess, report)
}

func (h *Handlers) listPlatformVoiceprintProfiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	tenantID := uint(0)
	if tid := strings.TrimSpace(c.Query("tenantId")); tid != "" {
		parsed, err := utils.ParseID(tid)
		if err != nil {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidTenantID))
			return
		}
		tenantID = parsed
	}
	offset := (page - 1) * pageSize
	rows, total, err := models.ListAllVoiceprintProfiles(h.db, tenantID, pageSize, offset)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	slug, prov, enabled := voiceprintconfig.ResolveEnabled()
	label := ""
	if enabled {
		label = voiceprintconfig.ProviderLabel(prov)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"list":                    rows,
		"total":                   total,
		"page":                    page,
		"pageSize":                pageSize,
		"voiceprintEnabled":       enabled,
		"voiceprintProvider":      slug,
		"voiceprintProviderLabel": label,
	})
}

func (h *Handlers) deletePlatformVoiceprintProfile(c *gin.Context) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil || id == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid id", err))
		return
	}
	row, err := models.GetVoiceprintProfileByID(h.db, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}

	if _, _, ok := voiceprintconfig.ResolveEnabled(); ok {
		bridge, err := voiceprintconfig.NewBridge()
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint service unavailable", err))
			return
		}
		defer bridge.Close()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := bridge.Delete(ctx, row.TenantID, nil, row.FeatureID); err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "delete provider feature failed", err))
			return
		}
	}

	if err := models.DeleteVoiceprintProfile(h.db, row.TenantID, id); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"deleted": true})
}
