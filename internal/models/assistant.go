package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HotWord biases ASR recognition and optional post-correction.
type HotWord struct {
	Word             string   `json:"word"`
	Weight           int      `json:"weight,omitempty"`
	ReplacedWords    []string `json:"replacedWords,omitempty"`
	EnableFuzzyMatch bool     `json:"enableFuzzyMatch,omitempty"`
}

// InterruptionConfig controls barge-in behaviour during assistant playback.
type InterruptionConfig struct {
	Method                        string `json:"method,omitempty"`
	UnInterruptableAfterPlayStart int    `json:"unInterruptableAfterPlayStart,omitempty"`
	TalkOverThreshold             int    `json:"talkOverThreshold,omitempty"`
	ResumePlay                    bool   `json:"resumePlay,omitempty"`
}

// VadConfig is voice-activity detection tuning (assistant-level).
type VadConfig struct {
	PaddingDuration         int     `json:"paddingDuration,omitempty"`
	Ratio                   float64 `json:"ratio,omitempty"`
	VadMode                 int     `json:"vadMode,omitempty"`
	PositiveSpeechThreshold float64 `json:"positiveSpeechThreshold,omitempty"`
	NegativeSpeechThreshold float64 `json:"negativeSpeechThreshold,omitempty"`
	RedemptionFrames        int     `json:"redemptionFrames,omitempty"`
	MinSpeechFrames         int     `json:"minSpeechFrames,omitempty"`
	EnergyThreshold         int     `json:"energyThreshold,omitempty"`
	SpeechStartMs           int     `json:"speechStartMs,omitempty"`
	SpeechEndMs             int     `json:"speechEndMs,omitempty"`
}

// EffectAudioTrack is one background/effect audio layer.
type EffectAudioTrack struct {
	Filename string  `json:"filename,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
	IsActive bool    `json:"isActive,omitempty"`
	Mode     string  `json:"mode,omitempty"`
}

// AudioTrackConfig controls master/main track volumes and effect tracks.
type AudioTrackConfig struct {
	MasterVolume      float64            `json:"masterVolume,omitempty"`
	MainTrackVolume   float64            `json:"mainTrackVolume,omitempty"`
	EffectAudioTracks []EffectAudioTrack `json:"effectAudioTracks,omitempty"`
}

// AudioProcessConfig controls uplink VAD backend and noise suppression.
type AudioProcessConfig struct {
	VadType                 string `json:"vadType,omitempty"`
	NoiseSuppressionEnabled bool   `json:"noiseSuppressionEnabled,omitempty"`
	NoiseSuppressionType    string `json:"noiseSuppressionType,omitempty"`
	NoiseSuppressionLevel   string `json:"noiseSuppressionLevel,omitempty"`
}

// QueryRewriter optionally rewrites ASR text before LLM/RAG.
type QueryRewriter struct {
	UseRewriter   bool   `json:"useRewriter,omitempty"`
	RewritePrompt string `json:"rewritePrompt,omitempty"`
}

// McpServer describes one stdio MCP tool server for the assistant.
type McpServer struct {
	Name    string            `json:"name,omitempty"`
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Envs    map[string]string `json:"envs,omitempty"`
}

// Assistant is a tenant-scoped AI voice bot (name/scene + draft config).
type Assistant struct {
	common.BaseModel
	TenantID           uint           `json:"tenantId,string" gorm:"index;not null;default:0"`
	Name               string         `json:"name" gorm:"size:128;not null;index"`
	AvatarURL          string         `json:"avatarUrl,omitempty" gorm:"column:avatar_url;size:512;comment:头像 URL"`
	Scene              string         `json:"scene" gorm:"size:64;index;comment:场景 inbound_knowledge|outbound_collect|outbound_notify|general"`
	Version            string         `json:"version" gorm:"size:64;index"`
	Description        string         `json:"description" gorm:"type:text"`
	Enabled            bool           `json:"enabled" gorm:"default:true;index"`
	AsrConfig          datatypes.JSON `json:"asrConfig,omitempty" gorm:"column:asr_config;type:json"`
	TtsConfig          datatypes.JSON `json:"ttsConfig,omitempty" gorm:"column:tts_config;type:json"`
	LlmConfig          datatypes.JSON `json:"llmConfig,omitempty" gorm:"column:llm_config;type:json"`
	RealtimeConfig     datatypes.JSON `json:"realtimeConfig,omitempty" gorm:"column:realtime_config;type:json"`
	Welcome            string         `json:"welcome,omitempty" gorm:"type:text;comment:欢迎语"`
	Prompt             string         `json:"prompt,omitempty" gorm:"type:text;comment:系统提示词"`
	KnowledgeNamespace string         `json:"knowledgeNamespace,omitempty" gorm:"column:knowledge_namespace;size:128;index;comment:知识库 namespace slug"`
	VadConfig          datatypes.JSON `json:"vadConfig,omitempty" gorm:"column:vad_config;type:json"`
	AgentConfig        datatypes.JSON `json:"agentConfig,omitempty" gorm:"column:agent_config;type:json"`
	HotWords           datatypes.JSON `json:"hotWords,omitempty" gorm:"column:hot_words;type:json"`
	InterruptionConfig datatypes.JSON `json:"interruptionConfig,omitempty" gorm:"column:interruption_config;type:json"`
	AudioTrackConfig   datatypes.JSON `json:"audioTrackConfig,omitempty" gorm:"column:audio_track_config;type:json"`
	AudioProcessConfig datatypes.JSON `json:"audioProcessConfig,omitempty" gorm:"column:audio_process_config;type:json"`
	QueryRewriter      datatypes.JSON `json:"queryRewriter,omitempty" gorm:"column:query_rewriter;type:json"`
	McpServers         datatypes.JSON `json:"mcpServers,omitempty" gorm:"column:mcp_servers;type:json"`
	TtsVoice           string         `json:"ttsVoice,omitempty" gorm:"column:tts_voice;size:128;comment:Pipeline TTS timbre id"`
	RealtimeVoice      string         `json:"realtimeVoice,omitempty" gorm:"column:realtime_voice;size:128;comment:Realtime multimodal timbre id"`
	VoiceDialogWsURL   string         `json:"voiceDialogWsUrl,omitempty" gorm:"column:voice_dialog_ws_url;size:512;comment:自定义呼入/调试语音对话 WebSocket"`
	BoundJsTemplateSourceID string         `json:"boundJsTemplateSourceId,omitempty" gorm:"column:bound_js_template_source_id;size:64;index;comment:网页嵌入绑定的 JS 模板 jsSourceId"`
	NluModelID              uint           `json:"nluModelId,string,omitempty" gorm:"column:nlu_model_id;index;default:0;comment:绑定的租户 NLU 模型"`
	Collect                 datatypes.JSON `json:"collect,omitempty" gorm:"type:json;comment:会话收集字段 JSON"`
	PublishedVersionID uint           `json:"publishedVersionId,string,omitempty" gorm:"column:published_version_id;index;default:0"`
}

func (Assistant) TableName() string {
	return constants2.ASSISTANT_TABLE_NAME
}

func ListAssistantsPage(db *gorm.DB, tenantID uint, page, size int, scene, nameContains string) ([]Assistant, int64, error) {
	q := db.Model(&Assistant{}).Where("tenant_id = ?", tenantID)
	if s := strings.TrimSpace(scene); s != "" {
		q = q.Where("scene = ?", s)
	}
	if name := strings.TrimSpace(nameContains); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	return utils.FindPage[Assistant](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

func GetActiveAssistantByID(db *gorm.DB, id uint) (Assistant, error) {
	var row Assistant
	err := db.Model(&Assistant{}).Where("id = ?", id).First(&row).Error
	return row, err
}

func GetActiveAssistantForTenant(db *gorm.DB, id, tenantID uint) (Assistant, error) {
	var row Assistant
	err := db.Model(&Assistant{}).Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	return row, err
}

func SoftDeleteAssistantByIDForTenant(db *gorm.DB, id, tenantID uint, updateBy string) (int64, error) {
	meta := common.BaseModel{}
	meta.SoftDelete(updateBy)
	updates := map[string]any{
		"updated_at": meta.UpdatedAt,
		"deleted_at": meta.DeletedAt,
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	res := db.Model(&Assistant{}).Where("id = ? AND tenant_id = ?", id, tenantID).Updates(updates)
	return res.RowsAffected, res.Error
}

func SoftDeleteAssistantByID(db *gorm.DB, id uint, updateBy string) (int64, error) {
	meta := common.BaseModel{}
	meta.SoftDelete(updateBy)
	updates := map[string]any{
		"updated_at": meta.UpdatedAt,
		"deleted_at": meta.DeletedAt,
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	res := db.Model(&Assistant{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected, res.Error
}

func ReloadAssistantByID(db *gorm.DB, id uint) (Assistant, error) {
	var row Assistant
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

// AssistantResolvedSpec is the effective assistant config used at runtime.
type AssistantResolvedSpec struct {
	Scene              string
	AsrConfig          datatypes.JSON
	TtsConfig          datatypes.JSON
	LlmConfig          datatypes.JSON
	RealtimeConfig     datatypes.JSON
	Welcome            string
	Prompt             string
	KnowledgeNamespace string
	VadConfig          datatypes.JSON
	AgentConfig        datatypes.JSON
	HotWords           datatypes.JSON
	InterruptionConfig datatypes.JSON
	AudioTrackConfig   datatypes.JSON
	AudioProcessConfig datatypes.JSON
	QueryRewriter      datatypes.JSON
	McpServers         datatypes.JSON
	TtsVoice           string
	RealtimeVoice      string
	VoiceDialogWsURL   string
	Collect            datatypes.JSON
}

// ResolveAssistantSpec returns published version snapshot or draft on the assistant row.
func ResolveAssistantSpec(db *gorm.DB, ast Assistant) (AssistantResolvedSpec, error) {
	out := AssistantResolvedSpec{
		Scene:              ast.Scene,
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
	}
	if ast.PublishedVersionID == 0 {
		return out, nil
	}
	var ver AssistantVersion
	if err := db.Where("id = ? AND assistant_id = ?", ast.PublishedVersionID, ast.ID).First(&ver).Error; err != nil {
		return out, nil
	}
	out.Scene = ver.Scene
	out.AsrConfig = ver.AsrConfig
	out.TtsConfig = ver.TtsConfig
	out.LlmConfig = ver.LlmConfig
	out.RealtimeConfig = ver.RealtimeConfig
	out.Welcome = coalesceNonEmpty(ver.Welcome, ast.Welcome)
	out.Prompt = coalesceNonEmpty(ver.Prompt, ast.Prompt)
	out.KnowledgeNamespace = ver.KnowledgeNamespace
	out.VadConfig = ver.VadConfig
	out.AgentConfig = ver.AgentConfig
	out.HotWords = ver.HotWords
	out.InterruptionConfig = ver.InterruptionConfig
	out.AudioTrackConfig = ver.AudioTrackConfig
	out.AudioProcessConfig = ver.AudioProcessConfig
	out.QueryRewriter = ver.QueryRewriter
	out.McpServers = ver.McpServers
	out.TtsVoice = coalesceNonEmpty(ver.TtsVoice, ast.TtsVoice)
	out.RealtimeVoice = coalesceNonEmpty(ver.RealtimeVoice, ast.RealtimeVoice)
	out.VoiceDialogWsURL = coalesceNonEmpty(ver.VoiceDialogWsURL, ast.VoiceDialogWsURL)
	out.Collect = ver.Collect
	return out, nil
}

// BuildAssistantUpdates builds a GORM Updates map for PATCH semantics.
func BuildAssistantUpdates(existing Assistant, name, scene, version, description string, enabled *bool,
	welcome, prompt, knowledgeNamespace, voiceDialogWsURL, boundJsTemplateSourceID, ttsVoice, realtimeVoice string,
	asrRaw, ttsRaw, llmRaw, realtimeRaw, vadRaw, agentRaw, hotWordsRaw, interruptionRaw, audioTrackRaw, audioProcessRaw, queryRewriterRaw, mcpServersRaw, collectRaw, updateBy string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name required")
	}
	scene = strings.TrimSpace(scene)
	if scene == "" {
		scene = existing.Scene
	}
	updates := map[string]any{
		"name":                          name,
		"scene":                         scene,
		"version":                       strings.TrimSpace(version),
		"description":                   strings.TrimSpace(description),
		"welcome":                       strings.TrimSpace(welcome),
		"prompt":                        strings.TrimSpace(prompt),
		"knowledge_namespace":           strings.TrimSpace(knowledgeNamespace),
		"voice_dialog_ws_url":           strings.TrimSpace(voiceDialogWsURL),
		"bound_js_template_source_id":   strings.TrimSpace(boundJsTemplateSourceID),
		"tts_voice":                     strings.TrimSpace(ttsVoice),
		"realtime_voice":                strings.TrimSpace(realtimeVoice),
	}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	for _, pair := range []struct {
		raw string
		key string
	}{
		{asrRaw, "asr_config"},
		{ttsRaw, "tts_config"},
		{llmRaw, "llm_config"},
		{realtimeRaw, "realtime_config"},
		{vadRaw, "vad_config"},
		{agentRaw, "agent_config"},
		{hotWordsRaw, "hot_words"},
		{interruptionRaw, "interruption_config"},
		{audioTrackRaw, "audio_track_config"},
		{audioProcessRaw, "audio_process_config"},
		{queryRewriterRaw, "query_rewriter"},
		{mcpServersRaw, "mcp_servers"},
		{collectRaw, "collect"},
	} {
		if strings.TrimSpace(pair.raw) == "" {
			continue
		}
		j, err := ParseOptionalJSONColumn(pair.raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pair.key, err)
		}
		updates[pair.key] = j
	}
	meta := common.BaseModel{}
	meta.SetUpdateInfo(updateBy)
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	return updates, nil
}

// NewAssistantForCreate builds a row after fields are normalized.
func NewAssistantForCreate(tenantID uint, name, scene, version, description, welcome, prompt, knowledgeNamespace, voiceDialogWsURL, boundJsTemplateSourceID, ttsVoice, realtimeVoice string,
	enabled bool, asr, tts, llm, realtime, vad, agent, hotWords, interruption, audioTrack, audioProcess, queryRewriter, mcpServers, collect datatypes.JSON) Assistant {
	if strings.TrimSpace(scene) == "" {
		scene = constants.AssistantSceneGeneral
	}
	return Assistant{
		TenantID:                 tenantID,
		Name:                     name,
		Scene:                    scene,
		Version:                  version,
		Description:              description,
		Enabled:                  enabled,
		AsrConfig:                asr,
		TtsConfig:                tts,
		LlmConfig:                llm,
		RealtimeConfig:           realtime,
		Welcome:                  welcome,
		Prompt:                   prompt,
		KnowledgeNamespace:       knowledgeNamespace,
		VoiceDialogWsURL:         voiceDialogWsURL,
		BoundJsTemplateSourceID: strings.TrimSpace(boundJsTemplateSourceID),
		TtsVoice:                 strings.TrimSpace(ttsVoice),
		RealtimeVoice:            strings.TrimSpace(realtimeVoice),
		VadConfig:                vad,
		AgentConfig:              agent,
		HotWords:                 hotWords,
		InterruptionConfig:       interruption,
		AudioTrackConfig:         audioTrack,
		AudioProcessConfig:       audioProcess,
		QueryRewriter:            queryRewriter,
		McpServers:               mcpServers,
		Collect:                  collect,
	}
}

// ImportTenantAIConfigToAssistant clones tenant AI JSON into a new default assistant.
func ImportTenantAIConfigToAssistant(db *gorm.DB, tenant Tenant, name, operator string) (Assistant, error) {
	if strings.TrimSpace(name) == "" {
		name = "默认智能体"
	}
	ast := NewAssistantForCreate(
		tenant.ID, name, constants.AssistantSceneGeneral, "v1", "从租户 AI 配置迁移",
		"", "", "", "", "", "", "",
		true,
		tenant.AsrConfig, tenant.TtsConfig, tenant.LlmConfig, tenant.RealtimeConfig,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	ast.SetCreateInfo(operator)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ast).Error; err != nil {
			return err
		}
		ver, err := PublishAssistantVersion(tx, ast, operator)
		if err != nil {
			return err
		}
		ast.PublishedVersionID = ver.ID
		return tx.Model(&Tenant{}).Where("id = ?", tenant.ID).Updates(map[string]any{
			"updated_at": ast.UpdatedAt,
			"update_by":  operator,
		}).Error
	})
	return ast, err
}

func coalesceNonEmpty(primary, fallback string) string {
	if s := strings.TrimSpace(primary); s != "" {
		return s
	}
	return strings.TrimSpace(fallback)
}

// AssistantConfigSnapshot serializes the draft (current row) config into
func AssistantConfigSnapshot(ast Assistant) (datatypes.JSON, error) {
	payload := map[string]any{
		"scene":              ast.Scene,
		"asrConfig":          json.RawMessage(ast.AsrConfig),
		"ttsConfig":          json.RawMessage(ast.TtsConfig),
		"llmConfig":          json.RawMessage(ast.LlmConfig),
		"realtimeConfig":     json.RawMessage(ast.RealtimeConfig),
		"welcome":            ast.Welcome,
		"prompt":             ast.Prompt,
		"knowledgeNamespace": ast.KnowledgeNamespace,
		"vadConfig":          json.RawMessage(ast.VadConfig),
		"agentConfig":        json.RawMessage(ast.AgentConfig),
		"hotWords":           json.RawMessage(ast.HotWords),
		"interruptionConfig": json.RawMessage(ast.InterruptionConfig),
		"audioTrackConfig":   json.RawMessage(ast.AudioTrackConfig),
		"audioProcessConfig": json.RawMessage(ast.AudioProcessConfig),
		"queryRewriter":      json.RawMessage(ast.QueryRewriter),
		"mcpServers":         json.RawMessage(ast.McpServers),
		"ttsVoice":           ast.TtsVoice,
		"realtimeVoice":      ast.RealtimeVoice,
		"voiceDialogWsUrl":   ast.VoiceDialogWsURL,
		"collect":            json.RawMessage(ast.Collect),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// AssistantResolvedSnapshot serializes a resolved spec (post-publish-merge)
func AssistantResolvedSnapshot(spec AssistantResolvedSpec) (datatypes.JSON, error) {
	payload := map[string]any{
		"scene":              spec.Scene,
		"asrConfig":          json.RawMessage(spec.AsrConfig),
		"ttsConfig":          json.RawMessage(spec.TtsConfig),
		"llmConfig":          json.RawMessage(spec.LlmConfig),
		"realtimeConfig":     json.RawMessage(spec.RealtimeConfig),
		"welcome":            spec.Welcome,
		"prompt":             spec.Prompt,
		"knowledgeNamespace": spec.KnowledgeNamespace,
		"vadConfig":          json.RawMessage(spec.VadConfig),
		"agentConfig":        json.RawMessage(spec.AgentConfig),
		"hotWords":           json.RawMessage(spec.HotWords),
		"interruptionConfig": json.RawMessage(spec.InterruptionConfig),
		"audioTrackConfig":   json.RawMessage(spec.AudioTrackConfig),
		"audioProcessConfig": json.RawMessage(spec.AudioProcessConfig),
		"queryRewriter":      json.RawMessage(spec.QueryRewriter),
		"mcpServers":         json.RawMessage(spec.McpServers),
		"ttsVoice":           spec.TtsVoice,
		"realtimeVoice":      spec.RealtimeVoice,
		"voiceDialogWsUrl":   spec.VoiceDialogWsURL,
		"collect":            json.RawMessage(spec.Collect),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return b, nil
}
