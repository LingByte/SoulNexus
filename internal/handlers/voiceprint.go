package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/voice/voiceprintconfig"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func parseFeatureIDList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (h *Handlers) getVoiceprintConfig(c *gin.Context) {
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

func (h *Handlers) voiceprintSelfTest(c *gin.Context) {
	probe := strings.EqualFold(strings.TrimSpace(c.Query("probe")), "true") ||
		strings.TrimSpace(c.Query("probe")) == "1"
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	report := voiceprintconfig.RunSelfTest(ctx, probe)
	response.SuccessI18n(c, i18n.KeySuccess, report)
}

func (h *Handlers) listVoiceprints(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	rows, err := models.ListVoiceprintProfiles(h.db, tid)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, rows)
}

func (h *Handlers) getVoiceprint(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceprintID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceprintProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func parseVoiceprintID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid id", err))
		return 0, false
	}
	return uint(id), true
}

// parseOptionalSnowflakeID accepts null / "" / number / numeric string without JS precision loss on the wire.
func parseOptionalSnowflakeID(v any) (*uint, error) {
	if v == nil {
		return nil, nil
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "0" || strings.EqualFold(s, "null") {
			return nil, nil
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil || n == 0 {
			return nil, fmt.Errorf("invalid id %q", s)
		}
		u := uint(n)
		return &u, nil
	case float64:
		// JSON numbers may already be rounded; reject non-integers / zero.
		if t <= 0 || t != float64(uint64(t)) {
			return nil, fmt.Errorf("invalid numeric id")
		}
		u := uint(uint64(t))
		return &u, nil
	case json.Number:
		n, err := t.Int64()
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid id")
		}
		u := uint(n)
		return &u, nil
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		return parseOptionalSnowflakeID(s)
	}
}

func (h *Handlers) createVoiceprint(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid request", fmt.Errorf("name required")))
		return
	}
	description := strings.TrimSpace(c.PostForm("description"))
	featureID := strings.TrimSpace(c.PostForm("featureId"))
	if featureID == "" {
		featureID = uuid.NewString()
	}

	slug, _, ok := voiceprintconfig.ResolveEnabled()
	if !ok {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint not enabled", fmt.Errorf("VOICEPRINT_PROVIDER not configured")))
		return
	}

	file, err := c.FormFile("audio")
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "audio file required", err))
		return
	}
	fh, err := file.Open()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "open audio failed", err))
		return
	}
	defer fh.Close()
	audio, err := io.ReadAll(io.LimitReader(fh, 8<<20))
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "read audio failed", err))
		return
	}

	bridge, err := voiceprintconfig.NewBridge()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint service unavailable", err))
		return
	}
	defer bridge.Close()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	row := &models.VoiceprintProfile{
		TenantID:    tid,
		Scene:       models.VoiceprintSceneBusiness,
		Name:        name,
		Provider:    slug,
		FeatureID:   featureID,
		Status:      models.VoiceprintStatusActive,
		Description: description,
	}
	if err := models.CreateVoiceprintProfile(h.db, row); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "save profile failed", err))
		return
	}

	enrolledID, err := bridge.Enroll(ctx, tid, nil, row.ID, featureID, name, description, audio)
	if err != nil {
		_ = models.DeleteVoiceprintProfile(h.db, tid, row.ID)
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}
	row.FeatureID = enrolledID
	h.ensureVoiceprintSubjectOnCreate(tid, row)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceVoiceprint, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created voiceprint %s", row.Name),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) identifyVoiceprint(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	slug, _, ok := voiceprintconfig.ResolveEnabled()
	if !ok {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint not enabled", fmt.Errorf("VOICEPRINT_PROVIDER not configured")))
		return
	}

	file, err := c.FormFile("audio")
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "audio file required", err))
		return
	}
	fh, err := file.Open()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "open audio failed", err))
		return
	}
	defer fh.Close()
	audio, err := io.ReadAll(io.LimitReader(fh, 8<<20))
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "read audio failed", err))
		return
	}

	threshold := 0.0
	if v := strings.TrimSpace(c.PostForm("threshold")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = f
		}
	}

	selected := parseFeatureIDList(c.PostForm("featureIds"))
	candidates, err := models.VoiceprintCandidateFeatureIDs(h.db, tid, selected, nil)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "load candidates failed", err))
		return
	}
	if len(candidates) == 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "no enrolled voiceprints", fmt.Errorf("empty candidates")))
		return
	}

	bridge, err := voiceprintconfig.NewBridge()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint service unavailable", err))
		return
	}
	defer bridge.Close()
	if !bridge.SupportsIdentify() {
		response.Render(c, response.Wrap(response.CodeBadRequest, "identify unsupported", fmt.Errorf("provider=%s", slug)))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	out, err := bridge.Identify(ctx, tid, nil, candidates, audio, threshold)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}

	var matched *models.VoiceprintProfile
	if out.FeatureID != "" {
		if row, err := models.FindVoiceprintProfileByFeatureID(h.db, tid, out.FeatureID); err == nil {
			matched = &row
		}
	}
	callID := strings.TrimSpace(c.PostForm("callId"))
	h.bindIdentifyToCall(callID, tid, matched, out.FeatureID, out.Score, out.Threshold, out.IsMatch, out.Confidence)
	resp := gin.H{
		"result":  out,
		"profile": matched,
	}
	if callID != "" {
		resp["callSpeaker"] = speakerPublicFromCall(callID)
	}
	response.SuccessI18n(c, i18n.KeySuccess, resp)
}

func (h *Handlers) bindVoiceprintAssistant(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceprintID(c)
	if !ok {
		return
	}
	var body struct {
		AssistantID any `json:"assistantId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	aid, err := parseOptionalSnowflakeID(body.AssistantID)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid assistantId", err))
		return
	}
	row, err := models.GetVoiceprintProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	if err := models.UpdateVoiceprintProfileAssistant(h.db, tid, id, aid); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "update profile failed", err))
		return
	}
	row.AssistantID = aid
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) deleteVoiceprint(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := parseVoiceprintID(c)
	if !ok {
		return
	}
	row, err := models.GetVoiceprintProfile(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}

	bridge, err := voiceprintconfig.NewBridge()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint service unavailable", err))
		return
	}
	defer bridge.Close()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := bridge.Delete(ctx, tid, nil, row.FeatureID); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "delete provider feature failed", err))
		return
	}
	var subjectID uint
	if row.SubjectID != nil {
		subjectID = *row.SubjectID
	}
	if err := models.DeleteVoiceprintProfile(h.db, tid, id); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "delete profile failed", err))
		return
	}
	// Drop orphan speaker subject (attrs + credentials) when no other profile references it.
	if subjectID > 0 {
		var remaining int64
		_ = h.db.Model(&models.VoiceprintProfile{}).
			Where("tenant_id = ? AND subject_id = ?", tid, subjectID).
			Count(&remaining).Error
		if remaining == 0 {
			if err := models.DeleteSpeakerSubjectCascade(h.db, tid, subjectID); err != nil && logger.Lg != nil {
				logger.Lg.Warn("voiceprint delete: speaker subject cascade failed",
					zap.Uint("tenant_id", tid),
					zap.Uint("subject_id", subjectID),
					zap.Error(err),
				)
			}
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"deleted": true})
}

func (h *Handlers) verifyUserVoiceprintAudio(tenantID, userID uint, audio []byte, threshold float64) (bool, error) {
	row, err := models.GetAccountVoiceprintForUser(h.db, tenantID, userID)
	if err != nil {
		return false, err
	}
	if row.Status != models.VoiceprintStatusActive {
		return false, fmt.Errorf("user voiceprint inactive")
	}
	featureID := strings.TrimSpace(row.FeatureID)
	if featureID == "" {
		return false, fmt.Errorf("user voiceprint missing feature")
	}
	bridge, err := voiceprintconfig.NewBridge()
	if err != nil {
		return false, err
	}
	defer bridge.Close()
	if !bridge.SupportsIdentify() {
		return false, fmt.Errorf("voiceprint identify unsupported")
	}
	if threshold <= 0 {
		threshold = bridge.SimilarityThreshold()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	out, err := bridge.Identify(ctx, tenantID, nil, []string{featureID}, audio, threshold)
	if err != nil {
		return false, err
	}
	return out != nil && out.IsMatch && strings.TrimSpace(out.FeatureID) == featureID, nil
}

func decodeLoginVoiceprintAudio(b64 string) ([]byte, error) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return nil, nil
	}
	if i := strings.Index(b64, ","); i >= 0 && strings.Contains(strings.ToLower(b64[:i]), "base64") {
		b64 = b64[i+1:]
	}
	return base64.StdEncoding.DecodeString(b64)
}

func fetchVoiceprintAudioURL(rawURL string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("audio url required")
	}
	if !strings.HasPrefix(strings.ToLower(rawURL), "http://") && !strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return nil, fmt.Errorf("invalid audio url")
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("audio url http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}
