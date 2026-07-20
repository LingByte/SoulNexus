package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/nlu"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func (h *Handlers) registerPlatformNLURoutes(r *humax.Group) {
	g := r.Group("platform/nlu-models")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("/config", h.getTenantNluConfig)
		g.GET("", h.listPlatformNluModels)
		g.GET("/:id", h.getPlatformNluModel)
		g.PUT("/:id", h.updatePlatformNluModel)
		g.DELETE("/:id", h.deletePlatformNluModel)
		g.POST("/:id/train", h.trainPlatformNluModel)
		g.POST("/:id/parse", h.parsePlatformNluModel)
	}
}

func (h *Handlers) listPlatformNluModels(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 100)
	tenantID := uint(0)
	if tid, ok, invalid := ginutil.QueryOptionalID(c, "tenantId"); invalid {
		response.Render(c, response.New(response.CodeBadRequest, i18n.TGin(c, i18n.KeyInvalidParams)))
		return
	} else if ok {
		tenantID = tid
	}
	rows, total, err := models.ListTenantNluModelsAdmin(h.db, tenantID, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	names := h.tenantNamesForNlu(rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		bound, _ := models.CountAssistantsByNluModel(h.db, row.TenantID, row.ID)
		item := tenantNluModelResp(row, bound)
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

func (h *Handlers) tenantNamesForNlu(rows []models.TenantNluModel) map[uint]string {
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

func (h *Handlers) getPlatformNluModel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModelByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	bound, _ := models.CountAssistantsByNluModel(h.db, row.TenantID, row.ID)
	names := h.tenantNamesForNlu([]models.TenantNluModel{row})
	item := tenantNluModelResp(row, bound)
	item["tenantName"] = names[row.TenantID]
	response.SuccessI18n(c, i18n.KeySuccess, item)
}

func (h *Handlers) updatePlatformNluModel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModelByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req tenantNluWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.MinConfidence != nil {
		updates["min_confidence"] = *req.MinConfidence
	}
	if req.Spec != nil {
		raw, err := json.Marshal(req.Spec)
		if ginutil.WriteInternalError(c, err) {
			return
		}
		updates["spec"] = datatypes.JSON(raw)
		if row.Status == models.TenantNluStatusReady {
			updates["status"] = models.TenantNluStatusDraft
		}
	}
	if len(updates) == 0 {
		bound, _ := models.CountAssistantsByNluModel(h.db, row.TenantID, row.ID)
		response.SuccessI18n(c, i18n.KeySuccess, tenantNluModelResp(row, bound))
		return
	}
	if ginutil.WriteInternalError(c, models.UpdateTenantNluModel(h.db, row.TenantID, id, updates)) {
		return
	}
	row, _ = models.GetTenantNluModelByID(h.db, id)
	invalidateTenantNluProfile(row)
	bound, _ := models.CountAssistantsByNluModel(h.db, row.TenantID, row.ID)
	response.SuccessI18n(c, i18n.KeySuccess, tenantNluModelResp(row, bound))
}

func (h *Handlers) deletePlatformNluModel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModelByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	_ = h.db.Model(&models.Assistant{}).Where("tenant_id = ? AND nlu_model_id = ?", row.TenantID, id).Update("nlu_model_id", 0).Error
	if ginutil.WriteInternalError(c, models.DeleteTenantNluModel(h.db, row.TenantID, id)) {
		return
	}
	invalidateTenantNluProfile(row)
	models.RemoveDirBestEffort(row.StorageDir)
	if row.StorageDir == "" {
		models.RemoveDirBestEffort(models.TenantNluDir(row.TenantID, id))
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": strconv.FormatUint(uint64(id), 10)})
}

func (h *Handlers) trainPlatformNluModel(c *gin.Context) {
	if !nlu.DeployEnabled() {
		response.FailWithCode(c, http.StatusBadRequest, "NLU 未在 .env 中启用（NLU_ENABLED）", nil)
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModelByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	spec, err := models.ParseTenantNluSpec(row.Spec)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	_ = models.UpdateTenantNluModel(h.db, row.TenantID, id, map[string]any{
		"status":      models.TenantNluStatusTraining,
		"train_error": "",
	})
	row.Status = models.TenantNluStatusTraining
	if err := trainTenantNluModelMaterialize(&row, spec); err != nil {
		_ = models.UpdateTenantNluModel(h.db, row.TenantID, id, map[string]any{
			"status":      models.TenantNluStatusFailed,
			"train_error": err.Error(),
		})
		response.FailWithCode(c, http.StatusBadGateway, "训练失败", gin.H{"error": err.Error()})
		return
	}
	rawSpec, _ := json.Marshal(spec)
	_ = models.UpdateTenantNluModel(h.db, row.TenantID, id, map[string]any{
		"status":      models.TenantNluStatusReady,
		"num_classes": row.NumClasses,
		"storage_dir": row.StorageDir,
		"train_error": "",
		"spec":        datatypes.JSON(rawSpec),
	})
	row, _ = models.GetTenantNluModelByID(h.db, id)
	bound, _ := models.CountAssistantsByNluModel(h.db, row.TenantID, row.ID)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"message": "训练完成",
		"model":   tenantNluModelResp(row, bound),
	})
}

func (h *Handlers) parsePlatformNluModel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModelByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if row.Status != models.TenantNluStatusReady {
		response.FailWithCode(c, http.StatusBadRequest, "模型尚未训练完成", nil)
		return
	}
	var req tenantNluParseReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	start := time.Now()
	out, err := nlu.ParseProfile(tenantNluProfilePaths(row), req.Text)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		response.FailWithCode(c, http.StatusBadGateway, "解析失败", gin.H{"error": err.Error()})
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nluParseResp(out, latency))
}
