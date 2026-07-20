package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/nlu"
	"github.com/LingByte/SoulNexus/pkg/voice/cloneconfig"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"gorm.io/gorm"
)

// LoadCallVoiceConfigBundle resolves voice JSON for one call leg.
// Resolution order for assistant:
//  1. call-scoped assistant set by dialog agent binding
//  2. tenant-level asr/tts/llm/realtime JSON columns (legacy)
func LoadCallVoiceConfigBundle(ctx context.Context, db *gorm.DB, tenantID uint, callID string) (tenantcfg.VoiceConfigBundle, bool) {
	var empty tenantcfg.VoiceConfigBundle
	if db == nil || tenantID == 0 {
		return empty, false
	}
	var tenant Tenant
	if err := db.WithContext(ctx).Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		return empty, false
	}

	fallback := tenantcfg.VoiceConfigBundle{
		Asr:       []byte(tenant.AsrConfig),
		Tts:       []byte(tenant.TtsConfig),
		Llm:       []byte(tenant.LlmConfig),
		Realtime:  []byte(tenant.RealtimeConfig),
		VoiceMode: tenant.VoiceMode,
	}

	assistantID := callbinding.GetAssistantID(callID)
	if assistantID == 0 {
		return fallback, true
	}

	ast, err := GetActiveAssistantForTenant(db.WithContext(ctx), assistantID, tenantID)
	if err != nil || !ast.Enabled {
		return fallback, true
	}

	bundle, ok := bundleFromAssistant(db.WithContext(ctx), tenant, ast, fallback)
	if !ok {
		return bundle, ok
	}
	// Debug sessions load draft agent/mcp overlay so unpublished tool bindings work.
	if callbinding.IsDebugCall(callID) {
		if len(ast.AgentConfig) > 0 {
			bundle.AgentConfig = append([]byte(nil), ast.AgentConfig...)
		}
		if len(ast.McpServers) > 0 {
			bundle.McpServers = append([]byte(nil), ast.McpServers...)
		}
	}
	return bundle, true
}

func bundleFromAssistant(db *gorm.DB, tenant Tenant, ast Assistant, fallback tenantcfg.VoiceConfigBundle) (tenantcfg.VoiceConfigBundle, bool) {
	spec, err := ResolveAssistantSpec(db, ast)
	if err != nil {
		return fallback, true
	}

	llmBytes := []byte(spec.LlmConfig)
	// Credentials live on tenant AI config; assistant llm_config may only
	// override model/baseUrl. Always merge tenant secrets into assistant JSON.
	llmBytes = mergeLLMCredentials(fallback.Llm, llmBytes)

	welcome := strings.TrimSpace(spec.Welcome)
	if welcome == "" {
		welcome = welcomeFromAssistantConfigJSON([]byte(spec.RealtimeConfig))
	}
	if welcome == "" {
		welcome = welcomeFromAssistantConfigJSON([]byte(spec.LlmConfig))
	}
	if welcome == "" {
		welcome = welcomeFromAssistantConfigJSON([]byte(spec.AgentConfig))
	}

	bundle := tenantcfg.VoiceConfigBundle{
		Asr:                []byte(spec.AsrConfig),
		Tts:                []byte(spec.TtsConfig),
		Llm:                llmBytes,
		Realtime:           []byte(spec.RealtimeConfig),
		Welcome:            welcome,
		Prompt:             strings.TrimSpace(spec.Prompt),
		KnowledgeNamespace: strings.TrimSpace(spec.KnowledgeNamespace),
		VadConfig:          []byte(spec.VadConfig),
		AgentConfig:        []byte(spec.AgentConfig),
		HotWords:           []byte(spec.HotWords),
		InterruptionConfig: []byte(spec.InterruptionConfig),
		AudioTrackConfig:   []byte(spec.AudioTrackConfig),
		AudioProcessConfig: []byte(spec.AudioProcessConfig),
		QueryRewriter:      []byte(spec.QueryRewriter),
		McpServers:         []byte(spec.McpServers),
		VoiceDialogWsURL:   strings.TrimSpace(spec.VoiceDialogWsURL),
		AssistantID:        ast.ID,
		AssistantVersionID: ast.PublishedVersionID,
	}
	// Voice mode is tenant-scoped only.
	if vm := strings.TrimSpace(fallback.VoiceMode); vm != "" {
		bundle.VoiceMode = vm
	} else {
		bundle.VoiceMode = "pipeline"
	}

	if ns := bundle.KnowledgeNamespace; ns != "" {
		if row, err := knmodels.GetKnowledgeNamespaceByNamespaceAndGroup(db, ns, tenant.ID); err == nil {
			bundle.KnowledgeNamespaceID = row.ID
			bundle.KnowledgeCollection = strings.TrimSpace(row.Namespace)
		}
	}

	if ast.NluModelID > 0 {
		if nluRow, err := GetTenantNluModel(db, tenant.ID, ast.NluModelID); err == nil &&
			nluRow.ID > 0 && strings.EqualFold(strings.TrimSpace(nluRow.Status), TenantNluStatusReady) {
			bundle.NluModelID = nluRow.ID
			bundle.NluIntentsPath = nluRow.IntentsPath()
			bundle.NluPrototypesPath = nluRow.PrototypesPath()
			bundle.NluMinConfidence = nluRow.MinConfidence
			global := nlu.Get()
			if global.IsEmbeddingMode() {
				bundle.NluModelPath = global.ModelPath
				bundle.NluTokenizerPath = global.TokenizerPath
			} else {
				bundle.NluModelPath = nluRow.ModelPath()
				bundle.NluTokenizerPath = nluRow.TokenizerPath()
			}
		}
	}

	if len(bytesTrim(bundle.Asr)) == 0 {
		bundle.Asr = fallback.Asr
	}
	ttsBase := fallback.Tts
	if len(bytesTrim(spec.TtsConfig)) > 0 {
		ttsBase = []byte(spec.TtsConfig)
	}
	bundle.Tts = tenantcfg.ApplyTTSVoice(ttsBase, spec.TtsVoice)
	rtBase := fallback.Realtime
	if len(bytesTrim(spec.RealtimeConfig)) > 0 {
		rtBase = []byte(spec.RealtimeConfig)
	}
	bundle.Realtime = tenantcfg.ApplyRealtimeVoice(rtBase, spec.RealtimeVoice)
	if cloneRaw, pipeline := applyAssistantCloneVoice(db, tenant.ID, spec.TtsVoice); pipeline {
		bundle.VoiceMode = "pipeline"
		if len(cloneRaw) > 0 {
			bundle.Tts = cloneRaw
		}
	}
	if len(bytesTrim(bundle.Llm)) == 0 {
		bundle.Llm = fallback.Llm
	}
	return bundle, true
}

func mergeLLMCredentials(tenantRaw, assistantRaw []byte) []byte {
	tenant := parseJSONMap(tenantRaw)
	assistant := parseJSONMap(assistantRaw)
	if len(assistant) == 0 {
		if len(tenant) == 0 {
			return assistantRaw
		}
		out, err := json.Marshal(tenant)
		if err != nil {
			return tenantRaw
		}
		return out
	}
	for _, k := range []string{"apiKey", "api_key", "baseUrl", "base_url", "model", "appId", "app_id"} {
		if !jsonFieldNonempty(assistant, k) && jsonFieldNonempty(tenant, k) {
			assistant[k] = tenant[k]
		}
	}
	if !jsonFieldNonempty(assistant, "provider") && jsonFieldNonempty(tenant, "provider") {
		assistant["provider"] = tenant["provider"]
	}
	out, err := json.Marshal(assistant)
	if err != nil {
		return assistantRaw
	}
	return out
}

func parseJSONMap(raw []byte) map[string]any {
	m := map[string]any{}
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return m
	}
	_ = json.Unmarshal([]byte(s), &m)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func jsonFieldNonempty(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) != ""
	default:
		return fmt.Sprint(t) != ""
	}
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func applyAssistantCloneVoice(db *gorm.DB, tenantID uint, ttsVoice string) ([]byte, bool) {
	voiceID := strings.TrimSpace(ttsVoice)
	if voiceID == "" || db == nil || tenantID == 0 {
		return nil, false
	}
	row, err := FindVoiceCloneProfileByVoiceID(db, tenantID, voiceID)
	if err != nil {
		return nil, false
	}
	raw, ok := cloneconfig.BuildTTSJSONForCloneProfile(cloneconfig.ProviderCloneRow{
		Provider:  row.Provider,
		AssetID:   row.AssetID,
		SpeakerID: row.SpeakerID,
	})
	return raw, ok
}

func welcomeFromAssistantConfigJSON(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil || len(m) == 0 {
		return ""
	}
	for _, k := range []string{"welcome", "greeting"} {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				if w := strings.TrimSpace(s); w != "" {
					return w
				}
			}
		}
	}
	return ""
}

// LoadTenantVoiceConfigBundle is kept for callers that only have tenant scope.
func LoadTenantVoiceConfigBundle(ctx context.Context, db *gorm.DB, tenantID uint) (tenantcfg.VoiceConfigBundle, bool) {
	return LoadCallVoiceConfigBundle(ctx, db, tenantID, "")
}
