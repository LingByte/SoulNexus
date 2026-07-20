package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/voice/cloneconfig"
	voicepreview "github.com/LingByte/SoulNexus/pkg/voice/preview"
	"github.com/LingByte/lingllm/voiceclone"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type voiceCloneCreateReq struct {
	Name      string `json:"name"`
	SpeakerID string `json:"speakerId"`
	Sex       int    `json:"sex"`
	AgeGroup  int    `json:"ageGroup"`
	Language  string `json:"language"`
	TrainText string `json:"trainText"`
	TextID    int64  `json:"textId"`
	TextSegID int64  `json:"textSegId"`
}

type voiceClonePreviewReq struct {
	Text string `json:"text"`
}

func (h *Handlers) getVoiceCloneConfig(c *gin.Context) {
	slug, prov, ok := cloneconfig.ResolveEnabled()
	if !ok {
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"provider": "",
			"enabled":  false,
		})
		return
	}
	out := cloneconfig.ConfigSummary(prov)
	out["provider"] = slug
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) listVoiceClones(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	status := strings.TrimSpace(c.Query("status"))
	rows, err := models.ListVoiceCloneProfiles(h.db, tid, status)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, rows)
}

func (h *Handlers) getVoiceCloneTrainingTexts(c *gin.Context) {
	provider := cloneconfig.DefaultProvider()
	if provider == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone not enabled", fmt.Errorf("VOICE_CLONE_PROVIDER not configured")))
		return
	}
	if provider != voiceclone.ProviderXunfei {
		response.Render(c, response.Wrap(response.CodeBadRequest, "training texts unsupported", fmt.Errorf("provider=%s", provider)))
		return
	}
	textID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("textId")), 10, 64)
	if textID <= 0 {
		textID = 5001 // 讯飞通用训练文本 ID
	}
	svc, _, err := cloneconfig.NewService()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone service unavailable", err))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	out, err := svc.GetTrainingTexts(ctx, textID)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}
	if out == nil || len(out.Segments) == 0 {
		if fallback := voiceclone.DefaultTrainingText(textID); fallback != nil && len(fallback.Segments) > 0 {
			out = fallback
		} else {
			response.Render(c, response.Wrap(response.CodeBadRequest, fmt.Sprintf("training texts empty for textId=%d", textID), fmt.Errorf("no segments")))
			return
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) createVoiceClone(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req voiceCloneCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid request", fmt.Errorf("name required")))
		return
	}
	provider := cloneconfig.DefaultProvider()
	if provider == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone not enabled", fmt.Errorf("VOICE_CLONE_PROVIDER not configured")))
		return
	}
	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = "zh"
	}
	row := &models.VoiceCloneProfile{
		TenantID:  tid,
		Name:      name,
		Provider:  string(provider),
		Status:    models.VoiceCloneStatusPending,
		Language:  lang,
		TrainText: strings.TrimSpace(req.TrainText),
		TextID:    req.TextID,
		TextSegID: req.TextSegID,
		Sex:       req.Sex,
	}
	if row.TextID <= 0 {
		row.TextID = 5001
	}
	if row.TextSegID <= 0 {
		row.TextSegID = 1
	}
	if row.TrainText == "" && provider == voiceclone.ProviderXunfei {
		row.TrainText = "您好，欢迎致电，我是您的智能客服助手。"
	}

	svc, _, err := cloneconfig.NewService()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone service unavailable", err))
		return
	}

	switch provider {
	case voiceclone.ProviderXunfei:
		sex := req.Sex
		if sex == 0 {
			sex = 2
		}
		age := req.AgeGroup
		if age == 0 {
			age = 2
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		task, err := svc.CreateTask(ctx, &voiceclone.CreateTaskRequest{
			TaskName:      name,
			Sex:           sex,
			AgeGroup:      age,
			Language:      lang,
			ResourceType:  12,
			EngineVersion: "omni_v1",
			Denoise:       1,
			MosRatio:      0.5,
		})
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "create clone task failed", err))
			return
		}
		row.TaskID = task.TaskID
		row.Status = models.VoiceCloneStatusPending
	case voiceclone.ProviderVolcengine:
		speakerID := strings.TrimSpace(req.SpeakerID)
		if speakerID == "" {
			response.Render(c, response.Wrap(response.CodeBadRequest, "speakerId required", fmt.Errorf("volcengine speaker id (S_xxx) required")))
			return
		}
		row.SpeakerID = speakerID
		row.TaskID = speakerID
	default:
		response.Render(c, response.Wrap(response.CodeBadRequest, "unsupported provider", fmt.Errorf("%s", provider)))
		return
	}
	if err := models.CreateVoiceCloneProfile(h.db, row); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceVoiceClone, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created voice clone %s", row.Name),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) getVoiceClone(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceCloneID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceCloneProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func parseVoiceCloneID(c *gin.Context) (uint, bool) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil || id == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid id", err))
		return 0, false
	}
	return id, true
}

func (h *Handlers) submitVoiceCloneAudio(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceCloneID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceCloneProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "audio file required", err))
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "open audio failed", err))
		return
	}
	defer f.Close()

	svc, provider, err := cloneconfig.NewService()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone service unavailable", err))
		return
	}

	taskID := strings.TrimSpace(row.TaskID)
	if provider == voiceclone.ProviderVolcengine {
		speakerID := strings.TrimSpace(row.SpeakerID)
		if speakerID == "" {
			speakerID = taskID
		}
		taskID = fmt.Sprintf("speaker_id:%s:wav", speakerID)
	}
	if taskID == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "task id missing", fmt.Errorf("empty task id")))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	submit := &voiceclone.SubmitAudioRequest{
		TaskID:    taskID,
		TextID:    row.TextID,
		TextSegID: row.TextSegID,
		AudioFile: f,
		Language:  row.Language,
		TrainText: row.TrainText,
		MosRatio:  0.5,
	}
	if err := svc.SubmitAudio(ctx, submit); err != nil {
		_ = models.UpdateVoiceCloneProfile(h.db, row.ID, map[string]any{
			"status":        models.VoiceCloneStatusFailed,
			"failed_reason": err.Error(),
		})
		response.Render(c, response.Wrap(response.CodeBadRequest, "submit audio failed", err))
		return
	}
	_ = models.UpdateVoiceCloneProfile(h.db, row.ID, map[string]any{
		"status": models.VoiceCloneStatusTraining,
	})
	row.Status = models.VoiceCloneStatusTraining
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) syncVoiceCloneStatus(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceCloneID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceCloneProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	updated, err := h.pollVoiceCloneStatus(c.Request.Context(), row)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "query status failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, updated)
}

func (h *Handlers) pollVoiceCloneStatus(ctx context.Context, row models.VoiceCloneProfile) (models.VoiceCloneProfile, error) {
	svc, provider, err := cloneconfig.NewService()
	if err != nil {
		return row, err
	}
	queryID := strings.TrimSpace(row.TaskID)
	if provider == voiceclone.ProviderVolcengine {
		if s := strings.TrimSpace(row.SpeakerID); s != "" {
			queryID = s
		}
	}
	if queryID == "" {
		return row, fmt.Errorf("empty task id")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	st, err := svc.QueryTaskStatus(ctx, queryID)
	if err != nil {
		return row, err
	}
	updates := map[string]any{
		"progress": st.Progress,
	}
	switch st.Status {
	case voiceclone.TrainingStatusSuccess:
		updates["status"] = models.VoiceCloneStatusSuccess
		asset := strings.TrimSpace(st.AssetID)
		if asset == "" {
			asset = strings.TrimSpace(row.SpeakerID)
		}
		updates["asset_id"] = asset
		row.AssetID = asset
		row.Status = models.VoiceCloneStatusSuccess
	case voiceclone.TrainingStatusFailed:
		updates["status"] = models.VoiceCloneStatusFailed
		updates["failed_reason"] = st.FailedDesc
		row.Status = models.VoiceCloneStatusFailed
		row.FailedReason = st.FailedDesc
	case voiceclone.TrainingStatusQueued, voiceclone.TrainingStatusInProgress:
		updates["status"] = models.VoiceCloneStatusTraining
		row.Status = models.VoiceCloneStatusTraining
	}
	row.Progress = st.Progress
	_ = models.UpdateVoiceCloneProfile(h.db, row.ID, updates)
	return row, nil
}

func (h *Handlers) previewVoiceClone(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceCloneID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceCloneProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	if row.Status != models.VoiceCloneStatusSuccess {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice not ready", fmt.Errorf("status=%s", row.Status)))
		return
	}
	assetID := strings.TrimSpace(row.AssetID)
	if assetID == "" {
		assetID = strings.TrimSpace(row.SpeakerID)
	}
	if assetID == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "asset id missing", fmt.Errorf("no asset id")))
		return
	}
	var req voiceClonePreviewReq
	_ = c.ShouldBindJSON(&req)
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = defaultVoicePreviewText
	}
	providerSlug := clonePreviewProviderSlug(row)
	cacheVoiceID := clonePreviewCacheVoiceID(row.ID)
	useCache := text == defaultVoicePreviewText
	if useCache {
		if key, ok, err := voicepreview.ResolveObjectKey(providerSlug, "clone", cacheVoiceID); err == nil && ok && key != "" {
			audioURL := resolvePreviewPublicURL(c, key)
			response.SuccessI18n(c, i18n.KeySuccess, gin.H{
				"audioUrl":   audioURL,
				"sampleRate": 24000,
				"format":     "wav",
				"assetId":    assetID,
				"cached":     true,
			})
			return
		}
	}
	svc, _, err := cloneconfig.NewService()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone service unavailable", err))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	out, err := svc.Synthesize(ctx, &voiceclone.SynthesizeRequest{
		AssetID:  assetID,
		Text:     text,
		Language: row.Language,
	})
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "synthesize failed", err))
		return
	}
	outPayload := gin.H{
		"sampleRate": out.SampleRate,
		"format":     out.Format,
		"assetId":    assetID,
		"cached":     false,
	}
	if useCache {
		if key, upErr := voicepreview.UploadPCM(providerSlug, "clone", cacheVoiceID, out.AudioData, out.SampleRate); upErr == nil && key != "" {
			_ = voicepreview.SetCachedObjectKey(providerSlug, "clone", cacheVoiceID, key)
			outPayload["audioUrl"] = resolvePreviewPublicURL(c, key)
			outPayload["format"] = "wav"
		} else {
			outPayload["pcmBase64"] = base64.StdEncoding.EncodeToString(out.AudioData)
		}
	} else {
		outPayload["pcmBase64"] = base64.StdEncoding.EncodeToString(out.AudioData)
	}
	response.SuccessI18n(c, i18n.KeySuccess, outPayload)
}

func clonePreviewProviderSlug(row models.VoiceCloneProfile) string {
	switch strings.ToLower(strings.TrimSpace(row.Provider)) {
	case string(voiceclone.ProviderVolcengine), "volcengine_clone":
		return "volcengine_clone"
	case string(voiceclone.ProviderXunfei), "xunfei_clone":
		return "xunfei_clone"
	default:
		return strings.ToLower(strings.TrimSpace(row.Provider))
	}
}

func clonePreviewCacheVoiceID(profileID uint) string {
	return strconv.FormatUint(uint64(profileID), 10)
}

func (h *Handlers) synthesizeVoiceClone(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	uid := middleware.AuthUserID(c)
	id, ok := parseVoiceCloneID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceCloneProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	if row.Status != models.VoiceCloneStatusSuccess {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice not ready", fmt.Errorf("status=%s", row.Status)))
		return
	}
	var req voiceClonePreviewReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = row.TrainText
	}
	if text == "" {
		text = "您好，这是音色克隆试听。"
	}
	assetID := strings.TrimSpace(row.AssetID)
	if assetID == "" {
		assetID = strings.TrimSpace(row.SpeakerID)
	}
	hist := &models.VoiceSynthesisHistory{
		TenantID:  tid,
		ProfileID: row.ID,
		Provider:  row.Provider,
		AssetID:   assetID,
		VoiceName: row.Name,
		Text:      text,
		Status:    models.VoiceSynthesisStatusSuccess,
		CreatedBy: uid,
	}
	svc, _, err := cloneconfig.NewService()
	if err != nil {
		hist.Status = models.VoiceSynthesisStatusFailed
		hist.ErrorMessage = err.Error()
		_ = models.CreateVoiceSynthesisHistory(h.db, hist)
		response.Render(c, response.Wrap(response.CodeBadRequest, "voice clone service unavailable", err))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	out, err := svc.Synthesize(ctx, &voiceclone.SynthesizeRequest{
		AssetID:  assetID,
		Text:     text,
		Language: row.Language,
	})
	if err != nil {
		hist.Status = models.VoiceSynthesisStatusFailed
		hist.ErrorMessage = err.Error()
		_ = models.CreateVoiceSynthesisHistory(h.db, hist)
		response.Render(c, response.Wrap(response.CodeBadRequest, "synthesize failed", err))
		return
	}
	hist.SampleRate = out.SampleRate
	providerSlug := row.Provider
	if providerSlug == string(voiceclone.ProviderVolcengine) {
		providerSlug = "volcengine_clone"
	} else if providerSlug == string(voiceclone.ProviderXunfei) {
		providerSlug = "xunfei_clone"
	}
	key := ""
	if uploaded, upErr := voicepreview.UploadPCM(providerSlug, "clone", assetID+"-"+uuid.NewString()[:8], out.AudioData, out.SampleRate); upErr == nil {
		key = uploaded
	}
	hist.AudioKey = key
	if err := models.CreateVoiceSynthesisHistory(h.db, hist); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	hist.AudioURL = ginutil.UploadURL(c, key)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"id":         hist.ID,
		"pcmBase64":  base64.StdEncoding.EncodeToString(out.AudioData),
		"sampleRate": out.SampleRate,
		"format":     out.Format,
		"assetId":    assetID,
		"audioUrl":   hist.AudioURL,
		"text":       text,
	})
}

func (h *Handlers) listVoiceSynthesisHistory(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	profileID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("profileId")), 10, 64)
	rows, err := models.ListVoiceSynthesisHistory(h.db, tid, uint(profileID), 100)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	for i := range rows {
		if rows[i].AudioKey != "" {
			rows[i].AudioURL = ginutil.UploadURL(c, rows[i].AudioKey)
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, rows)
}

func (h *Handlers) deleteVoiceSynthesisHistory(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := models.DeleteVoiceSynthesisHistoryForTenant(h.db, tid, id); err != nil {
		ginutil.WriteGORMError(c, err, "not found")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) deleteVoiceClone(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceCloneID(c)
	if !ok {
		return
	}
	if err := models.DeleteVoiceCloneProfile(h.db, tid, id); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}
