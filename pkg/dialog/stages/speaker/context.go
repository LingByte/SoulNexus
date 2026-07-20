// Package speaker resolves voiceprint matches into call-scoped SpeakerContext
// and provides LLM tools for identity inspect / on-demand identify.
package speaker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/voice/voiceprintconfig"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	dbMu sync.RWMutex
	db   *gorm.DB
)

// SetDB installs the process-wide DB handle used by Resolve / tools.
func SetDB(handle *gorm.DB) {
	dbMu.Lock()
	db = handle
	dbMu.Unlock()
}

func getDB() *gorm.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return db
}

// MatchFact is the identity fact from 1:N identify (no business attrs yet).
type MatchFact struct {
	FeatureID  string
	ProfileID  uint
	Score      float64
	Threshold  float64
	IsMatch    bool
	Confidence string
}

// Resolve builds SpeakerContext from a voiceprint feature id / profile.
func Resolve(tenantID uint, fact MatchFact) (callbinding.SpeakerContext, error) {
	ctx := callbinding.SpeakerContext{
		FeatureID:  strings.TrimSpace(fact.FeatureID),
		ProfileID:  fact.ProfileID,
		Score:      fact.Score,
		Threshold:  fact.Threshold,
		Verified:   fact.IsMatch,
		Confidence: strings.TrimSpace(fact.Confidence),
	}
	handle := getDB()
	if handle == nil || tenantID == 0 {
		return ctx, nil
	}
	var profile models.VoiceprintProfile
	var err error
	if fact.ProfileID > 0 {
		profile, err = models.GetVoiceprintProfile(handle, tenantID, fact.ProfileID)
	} else if ctx.FeatureID != "" {
		profile, err = models.FindVoiceprintProfileByFeatureID(handle, tenantID, ctx.FeatureID)
	} else {
		return ctx, nil
	}
	if err != nil {
		return ctx, err
	}
	ctx.ProfileID = profile.ID
	ctx.FeatureID = profile.FeatureID
	ctx.DisplayName = profile.Name

	if profile.SubjectID == nil || *profile.SubjectID == 0 {
		return ctx, nil
	}
	subject, err := models.GetSpeakerSubject(handle, tenantID, *profile.SubjectID)
	if err != nil {
		return ctx, nil
	}
	ctx.SubjectID = subject.ID
	if n := strings.TrimSpace(subject.DisplayName); n != "" {
		ctx.DisplayName = n
	}
	attrs, _ := models.ListSpeakerAttributes(handle, tenantID, subject.ID)
	for _, a := range attrs {
		ctx.Attributes = append(ctx.Attributes, callbinding.SpeakerAttribute{
			Key:        a.Key,
			Value:      a.Value,
			Visibility: a.Visibility,
		})
	}
	creds, _ := models.ListSpeakerCredentials(handle, tenantID, subject.ID)
	for _, c := range creds {
		scopes := splitCSV(c.Scopes)
		ctx.Credentials = append(ctx.Credentials, callbinding.SpeakerCredentialRef{
			Provider:  c.Provider,
			SecretRef: c.SecretRef,
			Scopes:    scopes,
			HasSecret: strings.TrimSpace(c.SecretRef) != "",
		})
	}
	return ctx, nil
}

// BindAfterIdentify resolves and stores SpeakerContext on the call.
func BindAfterIdentify(callID string, tenantID uint, fact MatchFact, lg *zap.Logger) (callbinding.SpeakerContext, error) {
	ctx, err := Resolve(tenantID, fact)
	if err != nil {
		return ctx, err
	}
	if strings.TrimSpace(callID) != "" {
		callbinding.SetSpeakerContext(callID, ctx)
		if lg != nil {
			lg.Info("speaker: context bound",
				zap.String("call_id", callID),
				zap.Uint("subject_id", ctx.SubjectID),
				zap.Uint("profile_id", ctx.ProfileID),
				zap.String("feature_id", ctx.FeatureID),
				zap.Bool("verified", ctx.Verified),
				zap.Float64("score", ctx.Score),
			)
		}
	}
	return ctx, nil
}

// IdentifyAndBind runs 1:N identify then binds call context.
func IdentifyAndBind(
	ctx context.Context,
	callID string,
	tenantID uint,
	assistantID *uint,
	audio []byte,
	featureIDs []string,
	threshold float64,
	lg *zap.Logger,
) (callbinding.SpeakerContext, *voiceprintconfig.IdentifyOutcome, error) {
	handle := getDB()
	if handle == nil {
		return callbinding.SpeakerContext{}, nil, fmt.Errorf("speaker: db not configured")
	}
	candidates, err := models.VoiceprintCandidateFeatureIDs(handle, tenantID, featureIDs, assistantID)
	if err != nil {
		return callbinding.SpeakerContext{}, nil, err
	}
	if len(candidates) == 0 {
		return callbinding.SpeakerContext{}, nil, fmt.Errorf("speaker: no enrolled voiceprints")
	}
	bridge, err := voiceprintconfig.NewBridge()
	if err != nil {
		return callbinding.SpeakerContext{}, nil, err
	}
	defer bridge.Close()
	if !bridge.SupportsIdentify() {
		return callbinding.SpeakerContext{}, nil, fmt.Errorf("speaker: identify unsupported")
	}
	if threshold <= 0 {
		threshold = bridge.SimilarityThreshold()
	}
	out, err := bridge.Identify(ctx, tenantID, assistantID, candidates, audio, threshold)
	if err != nil {
		return callbinding.SpeakerContext{}, nil, err
	}
	fact := MatchFact{
		FeatureID:  out.FeatureID,
		Score:      out.Score,
		Threshold:  out.Threshold,
		IsMatch:    out.IsMatch,
		Confidence: out.Confidence,
	}
	if out.FeatureID != "" {
		if row, err := models.FindVoiceprintProfileByFeatureID(handle, tenantID, out.FeatureID); err == nil {
			fact.ProfileID = row.ID
		}
	}
	bound, err := BindAfterIdentify(callID, tenantID, fact, lg)
	return bound, out, err
}

// PublicJSON returns a safe JSON view (secrets masked) for tools / debug.
func PublicJSON(ctx callbinding.SpeakerContext) string {
	type credView struct {
		Provider  string   `json:"provider"`
		Scopes    []string `json:"scopes,omitempty"`
		HasSecret bool     `json:"hasSecret"`
	}
	type view struct {
		SubjectID   uint                           `json:"subjectId,omitempty"`
		ProfileID   uint                           `json:"profileId,omitempty"`
		FeatureID   string                         `json:"featureId,omitempty"`
		DisplayName string                         `json:"displayName,omitempty"`
		Score       float64                        `json:"score,omitempty"`
		Threshold   float64                        `json:"threshold,omitempty"`
		Verified    bool                           `json:"verified"`
		Confidence  string                         `json:"confidence,omitempty"`
		Attributes  []callbinding.SpeakerAttribute `json:"attributes,omitempty"`
		Credentials []credView                     `json:"credentials,omitempty"`
	}
	v := view{
		SubjectID:   ctx.SubjectID,
		ProfileID:   ctx.ProfileID,
		FeatureID:   ctx.FeatureID,
		DisplayName: ctx.DisplayName,
		Score:       ctx.Score,
		Threshold:   ctx.Threshold,
		Verified:    ctx.Verified,
		Confidence:  ctx.Confidence,
		Attributes:  ctx.LLMAttributes(),
	}
	for _, c := range ctx.Credentials {
		v.Credentials = append(v.Credentials, credView{
			Provider:  c.Provider,
			Scopes:    c.Scopes,
			HasSecret: c.HasSecret || strings.TrimSpace(c.SecretRef) != "",
		})
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
