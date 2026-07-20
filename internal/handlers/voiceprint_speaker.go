package handlers

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	stagespeaker "github.com/LingByte/SoulNexus/pkg/dialog/stages/speaker"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type speakerAttrBody struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Visibility string `json:"visibility"`
}

type speakerCredBody struct {
	Provider  string `json:"provider"`
	SecretRef string `json:"secretRef"`
	Scopes    string `json:"scopes"`
	Clear     bool   `json:"clear"`
}

type upsertSpeakerBody struct {
	DisplayName string            `json:"displayName"`
	Notes       string            `json:"notes"`
	Attributes  []speakerAttrBody `json:"attributes"`
	Credentials []speakerCredBody `json:"credentials"`
}

func (h *Handlers) getVoiceprintSpeaker(c *gin.Context) {
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
	out := gin.H{
		"profileId":   row.ID,
		"featureId":   row.FeatureID,
		"name":        row.Name,
		"subjectId":   row.SubjectID,
		"subject":     nil,
		"attributes":  []any{},
		"credentials": []any{},
	}
	if row.SubjectID == nil || *row.SubjectID == 0 {
		response.SuccessI18n(c, i18n.KeySuccess, out)
		return
	}
	subject, err := models.GetSpeakerSubject(h.db, tid, *row.SubjectID)
	if err != nil {
		response.SuccessI18n(c, i18n.KeySuccess, out)
		return
	}
	attrs, _ := models.ListSpeakerAttributes(h.db, tid, subject.ID)
	creds, _ := models.ListSpeakerCredentials(h.db, tid, subject.ID)
	credViews := make([]gin.H, 0, len(creds))
	for _, cr := range creds {
		credViews = append(credViews, gin.H{
			"provider":  cr.Provider,
			"scopes":    cr.Scopes,
			"hasSecret": strings.TrimSpace(cr.SecretRef) != "",
		})
	}
	out["subject"] = subject
	out["attributes"] = attrs
	out["credentials"] = credViews
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) upsertVoiceprintSpeaker(c *gin.Context) {
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
	var body upsertSpeakerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	displayName := strings.TrimSpace(body.DisplayName)
	if displayName == "" {
		displayName = row.Name
	}
	subject, err := models.EnsureSpeakerSubject(h.db, tid, displayName, strings.TrimSpace(body.Notes))
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "ensure subject failed", err))
		return
	}
	if body.Notes != "" || body.DisplayName != "" {
		_ = models.UpdateSpeakerSubjectFields(h.db, tid, subject.ID, map[string]any{
			"display_name": displayName,
			"notes":        strings.TrimSpace(body.Notes),
		})
		subject.DisplayName = displayName
		subject.Notes = strings.TrimSpace(body.Notes)
	}
	if err := models.UpdateVoiceprintProfileSubject(h.db, tid, id, &subject.ID); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "link subject failed", err))
		return
	}
	attrRows := make([]models.SpeakerAttribute, 0, len(body.Attributes))
	for _, a := range body.Attributes {
		key := strings.TrimSpace(a.Key)
		if key == "" {
			continue
		}
		attrRows = append(attrRows, models.SpeakerAttribute{
			Key:        key,
			Value:      strings.TrimSpace(a.Value),
			Visibility: a.Visibility,
		})
	}
	if err := models.ReplaceSpeakerAttributes(h.db, tid, subject.ID, attrRows); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "save attributes failed", err))
		return
	}
	for _, cr := range body.Credentials {
		provider := strings.TrimSpace(cr.Provider)
		if provider == "" {
			continue
		}
		if cr.Clear {
			_ = models.DeleteSpeakerCredential(h.db, tid, subject.ID, provider)
			continue
		}
		if err := models.UpsertSpeakerCredential(h.db, tid, subject.ID, provider, cr.SecretRef, cr.Scopes); err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, fmt.Sprintf("save credential %s failed", provider), err))
			return
		}
	}
	row.SubjectID = &subject.ID
	h.getVoiceprintSpeaker(c)
}

// bindIdentifyToCall optionally stores SpeakerContext when callId is provided on identify.
func (h *Handlers) bindIdentifyToCall(callID string, tid uint, matched *models.VoiceprintProfile, featureID string, score, threshold float64, isMatch bool, confidence string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	fact := stagespeaker.MatchFact{
		FeatureID:  featureID,
		Score:      score,
		Threshold:  threshold,
		IsMatch:    isMatch,
		Confidence: confidence,
	}
	if matched != nil {
		fact.ProfileID = matched.ID
	}
	if _, err := stagespeaker.BindAfterIdentify(callID, tid, fact, logger.Lg); err != nil {
		logger.Warn("speaker bind after identify failed", zap.String("call_id", callID), zap.Error(err))
	}
}

func speakerPublicFromCall(callID string) gin.H {
	ctx, ok := callbinding.GetSpeakerContext(callID)
	if !ok {
		return gin.H{"bound": false}
	}
	return gin.H{"bound": true, "speakerJSON": stagespeaker.PublicJSON(ctx)}
}

func (h *Handlers) ensureVoiceprintSubjectOnCreate(tid uint, row *models.VoiceprintProfile) {
	if row == nil || tid == 0 {
		return
	}
	subject, err := models.EnsureSpeakerSubject(h.db, tid, row.Name, row.Description)
	if err != nil {
		return
	}
	if err := models.UpdateVoiceprintProfileSubject(h.db, tid, row.ID, &subject.ID); err != nil {
		return
	}
	row.SubjectID = &subject.ID
}
