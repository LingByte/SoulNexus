package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/intentonnx"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/nlu"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func (h *Handlers) registerTenantNLURoutes(g *humax.Group) {
	read := g.Group("")
	read.Use(middleware.RequireTenantPermissionAny(
		"api.nlu.read", "api.assistants.read",
	))
	{
		read.GET("/nlu-models/config", h.getTenantNluConfig)
		read.GET("/nlu-models", h.listTenantNluModels)
		read.GET("/nlu-models/:id", h.getTenantNluModel)
		read.GET("/nlu-models/:id/status", h.getTenantNluModelStatus)
	}
	write := g.Group("")
	write.Use(middleware.RequireTenantPermissionAny(
		"api.nlu.write", "api.assistants.write",
	))
	{
		write.POST("/nlu-models", h.createTenantNluModel)
		write.PUT("/nlu-models/:id", h.updateTenantNluModel)
		write.DELETE("/nlu-models/:id", h.deleteTenantNluModel)
		write.POST("/nlu-models/:id/train", h.trainTenantNluModel)
		write.POST("/nlu-models/:id/parse", h.parseTenantNluModel)
	}
}

func (h *Handlers) getTenantNluConfig(c *gin.Context) {
	global := nlu.Get()
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"deployEnabled": nlu.DeployEnabled(),
		"platformReady": global.Ready(),
		"mode":          global.Mode,
		"maxIntents":    intentonnx.MaxEmbedIntents,
		"baseModelPath": global.ModelPath,
		"baseTokenizer": global.TokenizerPath,
	})
}

func tenantNluProfilePaths(row models.TenantNluModel) nlu.ProfilePaths {
	global := nlu.Get()
	paths := nlu.ProfilePaths{
		IntentsPath:   row.IntentsPath(),
		MinConfidence: row.MinConfidence,
	}
	if global.IsEmbeddingMode() {
		paths.ModelPath = global.ModelPath
		paths.TokenizerPath = global.TokenizerPath
		paths.PrototypesPath = row.PrototypesPath()
		return paths
	}
	paths.ModelPath = row.ModelPath()
	paths.TokenizerPath = row.TokenizerPath()
	return paths
}

func invalidateTenantNluProfile(row models.TenantNluModel) {
	p := tenantNluProfilePaths(row)
	nlu.InvalidateProfileFull(p.ModelPath, p.TokenizerPath, p.IntentsPath, p.PrototypesPath)
}

func (h *Handlers) listTenantNluModels(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	if tid == 0 {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	rows, err := models.ListTenantNluModels(h.db, tid)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		bound, _ := models.CountAssistantsByNluModel(h.db, tid, row.ID)
		out = append(out, tenantNluModelResp(row, bound))
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) getTenantNluModel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModel(h.db, tid, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	bound, _ := models.CountAssistantsByNluModel(h.db, tid, row.ID)
	response.SuccessI18n(c, i18n.KeySuccess, tenantNluModelResp(row, bound))
}

func tenantNluModelResp(row models.TenantNluModel, boundAssistants int64) gin.H {
	spec, _ := models.ParseTenantNluSpec(row.Spec)
	return gin.H{
		"id":              strconv.FormatUint(uint64(row.ID), 10),
		"tenantId":        strconv.FormatUint(uint64(row.TenantID), 10),
		"name":            row.Name,
		"description":     row.Description,
		"status":          row.Status,
		"numClasses":      row.NumClasses,
		"minConfidence":   row.MinConfidence,
		"spec":            spec,
		"storageDir":      row.StorageDir,
		"trainError":      row.TrainError,
		"modelPath":       row.ModelPath(),
		"tokenizerPath":   row.TokenizerPath(),
		"intentsPath":     row.IntentsPath(),
		"boundAssistants": boundAssistants,
		"createdAt":       row.CreatedAt,
		"updatedAt":       row.UpdatedAt,
	}
}

type tenantNluWriteReq struct {
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	MinConfidence *float64              `json:"minConfidence"`
	Spec          *models.TenantNluSpec `json:"spec"`
}

func (h *Handlers) createTenantNluModel(c *gin.Context) {
	if !nlu.DeployEnabled() {
		response.FailWithCode(c, http.StatusBadRequest, "NLU 未在 .env 中启用（NLU_ENABLED）", nil)
		return
	}
	tid := middleware.CurrentTenantID(c)
	var req tenantNluWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if req.Name == "" {
		response.FailWithCode(c, http.StatusBadRequest, "名称不能为空", nil)
		return
	}
	spec := models.DefaultTenantNluSpec()
	if req.Spec != nil {
		spec = *req.Spec
	}
	raw, err := json.Marshal(spec)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	minConf := 0.85
	if req.MinConfidence != nil {
		minConf = *req.MinConfidence
	}
	row := &models.TenantNluModel{
		TenantID:      tid,
		Name:          req.Name,
		Description:   req.Description,
		Status:        models.TenantNluStatusDraft,
		MinConfidence: minConf,
		Spec:          datatypes.JSON(raw),
	}
	if ginutil.WriteInternalError(c, models.CreateTenantNluModel(h.db, row)) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, tenantNluModelResp(*row, 0))
}

func (h *Handlers) updateTenantNluModel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModel(h.db, tid, id)
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
		response.SuccessI18n(c, i18n.KeySuccess, tenantNluModelResp(row, 0))
		return
	}
	if ginutil.WriteInternalError(c, models.UpdateTenantNluModel(h.db, tid, id, updates)) {
		return
	}
	row, _ = models.GetTenantNluModel(h.db, tid, id)
	invalidateTenantNluProfile(row)
	bound, _ := models.CountAssistantsByNluModel(h.db, tid, row.ID)
	response.SuccessI18n(c, i18n.KeySuccess, tenantNluModelResp(row, bound))
}

func (h *Handlers) deleteTenantNluModel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModel(h.db, tid, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	_ = h.db.Model(&models.Assistant{}).Where("tenant_id = ? AND nlu_model_id = ?", tid, id).Update("nlu_model_id", 0).Error
	if ginutil.WriteInternalError(c, models.DeleteTenantNluModel(h.db, tid, id)) {
		return
	}
	invalidateTenantNluProfile(row)
	models.RemoveDirBestEffort(row.StorageDir)
	if row.StorageDir == "" {
		models.RemoveDirBestEffort(models.TenantNluDir(tid, id))
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

func (h *Handlers) trainTenantNluModel(c *gin.Context) {
	if !nlu.DeployEnabled() {
		response.FailWithCode(c, http.StatusBadRequest, "NLU 未在 .env 中启用（NLU_ENABLED）", nil)
		return
	}
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModel(h.db, tid, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	spec, err := models.ParseTenantNluSpec(row.Spec)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	_ = models.UpdateTenantNluModel(h.db, tid, id, map[string]any{
		"status":      models.TenantNluStatusTraining,
		"train_error": "",
	})
	row.Status = models.TenantNluStatusTraining
	if err := trainTenantNluModelMaterialize(&row, spec); err != nil {
		_ = models.UpdateTenantNluModel(h.db, tid, id, map[string]any{
			"status":      models.TenantNluStatusFailed,
			"train_error": err.Error(),
		})
		response.FailWithCode(c, http.StatusBadGateway, "训练失败", gin.H{"error": err.Error()})
		return
	}
	rawSpec, _ := json.Marshal(spec)
	_ = models.UpdateTenantNluModel(h.db, tid, id, map[string]any{
		"status":      models.TenantNluStatusReady,
		"num_classes": row.NumClasses,
		"storage_dir": row.StorageDir,
		"train_error": "",
		"spec":        datatypes.JSON(rawSpec),
	})
	row, _ = models.GetTenantNluModel(h.db, tid, id)
	bound, _ := models.CountAssistantsByNluModel(h.db, tid, row.ID)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"message": "训练完成",
		"model":   tenantNluModelResp(row, bound),
	})
}

type tenantNluParseReq struct {
	Text string `json:"text" binding:"required"`
}

func (h *Handlers) parseTenantNluModel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModel(h.db, tid, id)
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
		models.RecordAIInvocation(models.AIInvocationRecord{
			TenantID:     tid,
			UserID:       middleware.AuthUserID(c),
			Component:    models.AIComponentNLU,
			Provider:     nlu.Get().Mode,
			Model:        row.Name,
			Status:       models.AIStatusError,
			Source:       "nlu_parse",
			LatencyMs:    latency,
			InputChars:   utf8.RuneCountInString(req.Text),
			RequestText:  req.Text,
			ResponseText: err.Error(),
			ErrorMsg:     err.Error(),
			Meta:         nluParseErrorMeta(row.ID, middleware.AuthCredentialID(c)),
		})
		response.FailWithCode(c, http.StatusBadGateway, "解析失败", gin.H{"error": err.Error()})
		return
	}
	entry := models.AIInvocationRecord{
		TenantID:    tid,
		UserID:      middleware.AuthUserID(c),
		Component:   models.AIComponentNLU,
		Provider:    nlu.Get().Mode,
		Model:       row.Name,
		Status:      models.AIStatusOK,
		Source:      "nlu_parse",
		LatencyMs:   latency,
		InputChars:  utf8.RuneCountInString(req.Text),
		RequestText: req.Text,
		Meta:        map[string]any{"model_id": strconv.FormatUint(uint64(row.ID), 10)},
	}
	if credID := middleware.AuthCredentialID(c); credID > 0 {
		entry.Meta["credential_id"] = strconv.FormatUint(uint64(credID), 10)
	}
	if out != nil {
		entry.IntentName = out.Prediction.IntentName
		entry.Confidence = out.Prediction.Confidence
		entry.OutputChars = utf8.RuneCountInString(out.Reply)
		entry.Meta["channel"] = out.Channel
		if b, err := json.Marshal(out); err == nil {
			entry.ResponseText = string(b)
		}
	}
	models.RecordAIInvocation(entry)
	response.SuccessI18n(c, i18n.KeySuccess, nluParseResp(out, latency))
}

// nluParseResp augments the RouteOutput with server-measured inference latency
// so the Lab UI can surface 耗时 (how long the parse took) per request.
func nluParseResp(out *intentonnx.RouteOutput, latencyMs int64) gin.H {
	resp := gin.H{"latencyMs": latencyMs}
	if out != nil {
		resp["channel"] = out.Channel
		resp["reply"] = out.Reply
		resp["prediction"] = out.Prediction
	}
	return resp
}

func nluParseErrorMeta(modelID uint, credentialID uint) map[string]any {
	meta := map[string]any{"model_id": strconv.FormatUint(uint64(modelID), 10)}
	if credentialID > 0 {
		meta["credential_id"] = strconv.FormatUint(uint64(credentialID), 10)
	}
	return meta
}

func (h *Handlers) getTenantNluModelStatus(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantNluModel(h.db, tid, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	out := gin.H{
		"deployEnabled": nlu.DeployEnabled(),
		"status":        row.Status,
		"numClasses":    row.NumClasses,
		"engineOnline":  false,
	}
	if row.Status == models.TenantNluStatusReady {
		st, err := nlu.ProfileStatus(tenantNluProfilePaths(row))
		if err == nil && st.Ready {
			out["engineOnline"] = true
			out["intents"] = st.Intents
		} else if err != nil {
			out["engineError"] = err.Error()
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}
