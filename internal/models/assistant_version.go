package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AssistantVersion is an immutable published snapshot of an assistant config.
type AssistantVersion struct {
	common.BaseModel
	TenantID           uint           `json:"tenantId,string" gorm:"index;not null;default:0"`
	AssistantID        uint           `json:"assistantId,string" gorm:"index;not null"`
	Scene              string         `json:"scene" gorm:"size:64;index"`
	Version            string         `json:"version" gorm:"size:64;not null;index"`
	VersionNo          int            `json:"versionNo" gorm:"not null;index"`
	Description        string         `json:"description" gorm:"type:text"`
	AsrConfig          datatypes.JSON `json:"asrConfig,omitempty" gorm:"column:asr_config;type:json"`
	TtsConfig          datatypes.JSON `json:"ttsConfig,omitempty" gorm:"column:tts_config;type:json"`
	LlmConfig          datatypes.JSON `json:"llmConfig,omitempty" gorm:"column:llm_config;type:json"`
	RealtimeConfig     datatypes.JSON `json:"realtimeConfig,omitempty" gorm:"column:realtime_config;type:json"`
	Welcome            string         `json:"welcome,omitempty" gorm:"type:text"`
	Prompt             string         `json:"prompt,omitempty" gorm:"type:text"`
	KnowledgeNamespace string         `json:"knowledgeNamespace,omitempty" gorm:"column:knowledge_namespace;size:128"`
	VadConfig          datatypes.JSON `json:"vadConfig,omitempty" gorm:"column:vad_config;type:json"`
	AgentConfig        datatypes.JSON `json:"agentConfig,omitempty" gorm:"column:agent_config;type:json"`
	HotWords           datatypes.JSON `json:"hotWords,omitempty" gorm:"column:hot_words;type:json"`
	InterruptionConfig datatypes.JSON `json:"interruptionConfig,omitempty" gorm:"column:interruption_config;type:json"`
	AudioTrackConfig   datatypes.JSON `json:"audioTrackConfig,omitempty" gorm:"column:audio_track_config;type:json"`
	AudioProcessConfig datatypes.JSON `json:"audioProcessConfig,omitempty" gorm:"column:audio_process_config;type:json"`
	QueryRewriter      datatypes.JSON `json:"queryRewriter,omitempty" gorm:"column:query_rewriter;type:json"`
	McpServers         datatypes.JSON `json:"mcpServers,omitempty" gorm:"column:mcp_servers;type:json"`
	TtsVoice           string         `json:"ttsVoice,omitempty" gorm:"column:tts_voice;size:128"`
	RealtimeVoice      string         `json:"realtimeVoice,omitempty" gorm:"column:realtime_voice;size:128"`
	VoiceDialogWsURL   string         `json:"voiceDialogWsUrl,omitempty" gorm:"column:voice_dialog_ws_url;size:512"`
	Collect            datatypes.JSON `json:"collect,omitempty" gorm:"type:json"`
	PublishedAt        time.Time      `json:"publishedAt" gorm:"index;not null"`
	PublishedBy        string         `json:"publishedBy" gorm:"size:128"`
}

func (AssistantVersion) TableName() string {
	return constants.ASSISTANT_VERSION_TABLE_NAME
}

func NextAssistantVersionNo(db *gorm.DB, assistantID uint) (int, error) {
	var maxNo int
	err := db.Model(&AssistantVersion{}).
		Where("assistant_id = ?", assistantID).
		Select("COALESCE(MAX(version_no), 0)").
		Scan(&maxNo).Error
	return maxNo + 1, err
}

func ListAssistantVersions(db *gorm.DB, assistantID uint) ([]AssistantVersion, error) {
	var list []AssistantVersion
	err := db.Where("assistant_id = ?", assistantID).Order("version_no DESC").Find(&list).Error
	return list, err
}

func GetAssistantVersion(db *gorm.DB, assistantID, versionID uint) (AssistantVersion, error) {
	var row AssistantVersion
	err := db.Where("id = ? AND assistant_id = ?", versionID, assistantID).First(&row).Error
	return row, err
}

// PublishAssistantVersion snapshots the current assistant draft as a new version.
func PublishAssistantVersion(db *gorm.DB, ast Assistant, operator string) (AssistantVersion, error) {
	var out AssistantVersion
	err := db.Transaction(func(tx *gorm.DB) error {
		no, err := NextAssistantVersionNo(tx, ast.ID)
		if err != nil {
			return err
		}
		versionLabel := strings.TrimSpace(ast.Version)
		if versionLabel == "" {
			versionLabel = fmt.Sprintf("v%d", no)
		}
		now := time.Now()
		row := AssistantVersion{
			TenantID:           ast.TenantID,
			AssistantID:        ast.ID,
			Scene:              ast.Scene,
			Version:            versionLabel,
			VersionNo:          no,
			Description:        ast.Description,
			AsrConfig:          ast.AsrConfig,
			TtsConfig:          ast.TtsConfig,
			LlmConfig:          ast.LlmConfig,
			RealtimeConfig:     ast.RealtimeConfig,
			Welcome:            ast.Welcome,
			Prompt:             ast.Prompt,
			KnowledgeNamespace: ast.KnowledgeNamespace,
			VadConfig:          ast.VadConfig,
			AgentConfig:        ast.AgentConfig,
			HotWords:           ast.HotWords,
			InterruptionConfig: ast.InterruptionConfig,
			AudioTrackConfig:   ast.AudioTrackConfig,
			AudioProcessConfig: ast.AudioProcessConfig,
			QueryRewriter:      ast.QueryRewriter,
			McpServers:         ast.McpServers,
			TtsVoice:           ast.TtsVoice,
			RealtimeVoice:      ast.RealtimeVoice,
			VoiceDialogWsURL:   ast.VoiceDialogWsURL,
			Collect:            ast.Collect,
			PublishedAt:        now,
			PublishedBy:        strings.TrimSpace(operator),
		}
		row.SetCreateInfo(operator)
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&Assistant{}).Where("id = ?", ast.ID).Updates(map[string]any{
			"published_version_id": row.ID,
			"version":              versionLabel,
			"updated_at":           now,
			"update_by":            operator,
		}).Error; err != nil {
			return err
		}
		out = row
		return nil
	})
	return out, err
}

// RollbackAssistantToVersion restores draft fields from a historical version.
func RollbackAssistantToVersion(db *gorm.DB, ast Assistant, ver AssistantVersion, operator string) (Assistant, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&Assistant{}).Where("id = ?", ast.ID).Updates(map[string]any{
			"scene":                ver.Scene,
			"version":              ver.Version,
			"description":          ver.Description,
			"asr_config":           ver.AsrConfig,
			"tts_config":           ver.TtsConfig,
			"llm_config":           ver.LlmConfig,
			"realtime_config":      ver.RealtimeConfig,
			"welcome":              ver.Welcome,
			"prompt":               ver.Prompt,
			"knowledge_namespace":  ver.KnowledgeNamespace,
			"vad_config":           ver.VadConfig,
			"agent_config":         ver.AgentConfig,
			"hot_words":            ver.HotWords,
			"interruption_config":  ver.InterruptionConfig,
			"audio_track_config":   ver.AudioTrackConfig,
			"audio_process_config": ver.AudioProcessConfig,
			"query_rewriter":       ver.QueryRewriter,
			"mcp_servers":          ver.McpServers,
			"tts_voice":            ver.TtsVoice,
			"realtime_voice":       ver.RealtimeVoice,
			"voice_dialog_ws_url":  ver.VoiceDialogWsURL,
			"collect":              ver.Collect,
			"published_version_id": ver.ID,
			"updated_at":           time.Now(),
			"update_by":            operator,
		}).Error
	})
	if err != nil {
		return ast, err
	}
	return ReloadAssistantByID(db, ast.ID)
}

// DiffAssistantConfigs compares two JSON config blobs by top-level keys
// whose values differ (or appear in only one side).
func DiffAssistantConfigs(a, b datatypes.JSON) ([]string, error) {
	var ma, mb map[string]json.RawMessage
	if len(a) > 0 {
		if err := json.Unmarshal(a, &ma); err != nil {
			return nil, err
		}
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &mb); err != nil {
			return nil, err
		}
	}
	if ma == nil {
		ma = map[string]json.RawMessage{}
	}
	if mb == nil {
		mb = map[string]json.RawMessage{}
	}
	seen := map[string]struct{}{}
	var keys []string
	for k, va := range ma {
		seen[k] = struct{}{}
		vb, ok := mb[k]
		if !ok || string(va) != string(vb) {
			keys = append(keys, k)
		}
	}
	for k := range mb {
		if _, ok := seen[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

// ParseOptionalJSONColumn validates optional JSON for assistant PATCH fields.
func ParseOptionalJSONColumn(raw string) (datatypes.JSON, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty")
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("invalid JSON")
	}
	return datatypes.JSON(raw), nil
}

// ParseOptionalJSONColumnNullable parses an optional JSON column where empty
// string is a valid input (returning nil, nil). Used by handlers that treat
// "no change" and "set to null" identically for optional JSON fields.
func ParseOptionalJSONColumnNullable(raw string) (datatypes.JSON, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	return ParseOptionalJSONColumn(raw)
}

// AssistantVersionSnapshot serializes an AssistantVersion into a JSON blob
// that mirrors the same top-level shape used by AssistantConfigSnapshot and
// AssistantResolvedSnapshot. Used to diff two published versions.
func AssistantVersionSnapshot(ver AssistantVersion) (datatypes.JSON, error) {
	spec := AssistantResolvedSpec{
		Scene:              ver.Scene,
		AsrConfig:          ver.AsrConfig,
		TtsConfig:          ver.TtsConfig,
		LlmConfig:          ver.LlmConfig,
		RealtimeConfig:     ver.RealtimeConfig,
		Welcome:            ver.Welcome,
		Prompt:             ver.Prompt,
		KnowledgeNamespace: ver.KnowledgeNamespace,
		VadConfig:          ver.VadConfig,
		AgentConfig:        ver.AgentConfig,
		TtsVoice:           ver.TtsVoice,
		RealtimeVoice:      ver.RealtimeVoice,
		VoiceDialogWsURL:   ver.VoiceDialogWsURL,
		Collect:            ver.Collect,
	}
	return AssistantResolvedSnapshot(spec)
}
