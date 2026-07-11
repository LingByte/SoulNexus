package svcmodels

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PresetTemplateType 模板类型常量
const (
	PresetTypeAgent        = "agent"         // 完整 Agent 预设
	PresetTypeSystemPrompt = "system_prompt" // 系统提示词模板
	PresetTypeVoice        = "voice"         // 语音配置预设
	PresetTypeKnowledge    = "knowledge"     // 知识库配置预设
)

// ValidPresetTypes 所有合法的模板类型
var ValidPresetTypes = []string{
	PresetTypeAgent,
	PresetTypeSystemPrompt,
	PresetTypeVoice,
	PresetTypeKnowledge,
}

// PresetVisibility 可见性常量
const (
	PresetVisibilityPrivate = "private" // 仅创建者可见
	PresetVisibilityGroup   = "group"   // 组织内可见
	PresetVisibilityPublic  = "public"  // 公开（内置模板）
)

// PresetTemplate 表示一个可复用的预设模板。
// 用户可以将自己的 Agent / SystemPrompt / Voice / Knowledge 配置保存为模板，
// 也可以使用系统内置或组织共享的模板快速创建新资源。
type PresetTemplate struct {
	ID        int64  `json:"id,string" gorm:"primaryKey;autoIncrement"`
	GroupID   uint   `json:"groupId" gorm:"index;not null;default:0;comment:tenant group id (0=system)"`
	CreatedBy uint   `json:"createdBy" gorm:"index;comment:user who created (0=system built-in)"`

	Name        string `json:"name" gorm:"type:varchar(255);not null;comment:template display name"`
	Description string `json:"description" gorm:"type:text;comment:template description / usage guide"`
	Type        string `json:"type" gorm:"type:varchar(32);index;not null;comment:agent|system_prompt|voice|knowledge"`
	Category    string `json:"category" gorm:"type:varchar(64);index;comment:optional category tag"`
	Tags        string `json:"tags" gorm:"type:varchar(512);comment:comma-separated tags"`
	Visibility  string `json:"visibility" gorm:"type:varchar(16);index;not null;default:'private';comment:private|group|public"`

	// 核心：模板内容以 JSON 存储，结构随 type 变化
	// agent:        AgentPresetPayload
	// system_prompt:SystemPromptPresetPayload
	// voice:        VoicePresetPayload
	// knowledge:    KnowledgePresetPayload
	Content string `json:"content" gorm:"type:longtext;not null;comment:JSON template payload"`

	// 元数据
	UseCount  int64  `json:"useCount" gorm:"default:0;comment:how many times this template has been applied"`
	IsBuiltin bool   `json:"isBuiltin" gorm:"default:false;comment:system built-in template (immutable)"`
	Status    string `json:"status" gorm:"type:varchar(16);index;not null;default:'active';comment:active|archived"`

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (PresetTemplate) TableName() string { return "preset_templates" }

// ============================================================================
// Payload 结构体 —— 每种模板类型对应的 JSON Content 结构
// ============================================================================

// AgentPresetPayload agent 类型模板的完整内容。
type AgentPresetPayload struct {
	Name             string  `json:"name"`
	SystemPrompt     string  `json:"systemPrompt"`
	OpeningStatement string  `json:"openingStatement,omitempty"`
	PersonaTag       string  `json:"personaTag,omitempty"`
	Temperature      float32 `json:"temperature"`
	MaxTokens        int     `json:"maxTokens"`
	LLMModel         string  `json:"llmModel,omitempty"`

	// 语音配置（可选，允许只保存 Agent 核心配置）
	Speaker              string   `json:"speaker,omitempty"`
	VoiceCloneID         *int     `json:"voiceCloneId,omitempty"`
	TtsProvider          string   `json:"ttsProvider,omitempty"`
	EnableVAD            *bool    `json:"enableVAD,omitempty"`
	VADThreshold         *float64 `json:"vadThreshold,omitempty"`
	VADConsecutiveFrames *int     `json:"vadConsecutiveFrames,omitempty"`
	EnableJSONOutput     *bool    `json:"enableJSONOutput,omitempty"`
}

// SystemPromptPresetPayload system_prompt 类型模板。
type SystemPromptPresetPayload struct {
	SystemPrompt string `json:"systemPrompt"`
	PersonaTag   string `json:"personaTag,omitempty"`
	// 变量占位符说明，例如 "{{user_name}}", "{{current_date}}"
	Variables []PresetVariable `json:"variables,omitempty"`
}

// PresetVariable 提示词模板中的可替换变量定义。
type PresetVariable struct {
	Name        string `json:"name"`        // 变量名，如 "user_name"
	Label       string `json:"label"`       // 显示标签
	DefaultVal  string `json:"defaultVal"`  // 默认值
	Description string `json:"description"` // 说明
	Required    bool   `json:"required"`    // 是否必填
}

// VoicePresetPayload voice 类型模板。
type VoicePresetPayload struct {
	Speaker              string   `json:"speaker"`
	TtsProvider          string   `json:"ttsProvider"`
	VoiceCloneID         *int     `json:"voiceCloneId,omitempty"`
	EnableVAD            bool     `json:"enableVAD"`
	VADThreshold         float64  `json:"vadThreshold"`
	VADConsecutiveFrames int      `json:"vadConsecutiveFrames"`
	Speed                *float64 `json:"speed,omitempty"`  // 语速
	Volume               *float64 `json:"volume,omitempty"` // 音量
	Pitch                *float64 `json:"pitch,omitempty"`  // 音调
}

// KnowledgePresetPayload knowledge 类型模板。
type KnowledgePresetPayload struct {
	Namespace      string `json:"namespace"`   // 知识库 namespace 名称模板
	Name           string `json:"name"`        // 知识库显示名称
	Description    string `json:"description"` // 描述
	VectorProvider string `json:"vectorProvider,omitempty"`
	EmbedModel     string `json:"embedModel,omitempty"`
	VectorDim      int    `json:"vectorDim,omitempty"`
	// 预设的文档处理配置
	ChunkSize    int    `json:"chunkSize,omitempty"`
	ChunkOverlap int    `json:"chunkOverlap,omitempty"`
	FileTypes    string `json:"fileTypes,omitempty"` // 支持的文件类型，逗号分隔
}

// ============================================================================
// 请求 / 响应结构体
// ============================================================================

// CreatePresetReq 创建预设模板请求。
type CreatePresetReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type" binding:"required"`
	Category    string `json:"category"`
	Tags        string `json:"tags"`
	Visibility  string `json:"visibility"`
	Content     string `json:"content" binding:"required"` // JSON string
}

// UpdatePresetReq 更新预设模板请求。
type UpdatePresetReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Tags        string `json:"tags"`
	Visibility  string `json:"visibility"`
	Content     string `json:"content"`
}

// ListPresetReq 列表查询参数。
type ListPresetReq struct {
	Type       string `form:"type"`
	Category   string `form:"category"`
	Visibility string `form:"visibility"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

// PresetListResult 分页列表响应。
type PresetListResult struct {
	List      []PresetTemplate `json:"list"`
	Total     int64            `json:"total"`
	Page      int              `json:"page"`
	PageSize  int              `json:"pageSize"`
	TotalPage int              `json:"totalPage"`
}

// ApplyPresetReq 应用预设模板的请求。
// 根据 Type 不同，Content 会被解析为对应的 Payload 结构体。
type ApplyPresetReq struct {
	PresetID int64  `json:"presetId" binding:"required"`
	// 对于 system_prompt 类型，可选填入变量值
	Variables map[string]string `json:"variables,omitempty"`
	// 目标 Agent ID（agent 类型模板应用时必须提供）
	AgentID *int64 `json:"agentId,omitempty"`
}

// ============================================================================
// 数据库查询方法
// ============================================================================

// ListPresetTemplates 查询预设模板列表（分页）。
// groupIDs: 用户所属的组织 ID 列表，用于可见性过滤。
func ListPresetTemplates(db *gorm.DB, groupIDs []uint, req *ListPresetReq) (*PresetListResult, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	q := db.Model(&PresetTemplate{}).Where("status = ?", "active")

	// 可见性过滤：
	// - public: 所有人可见
	// - group: 同组织可见（group_id IN groupIDs）
	// - private: 仅创建者可见（在 handler 层额外过滤）
	if len(groupIDs) > 0 {
		q = q.Where("(visibility = 'public') OR (visibility = 'group' AND group_id IN ?)", groupIDs)
	} else {
		q = q.Where("visibility = 'public'")
	}

	if t := strings.TrimSpace(req.Type); t != "" {
		q = q.Where("type = ?", t)
	}
	if cat := strings.TrimSpace(req.Category); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(name LIKE ? OR description LIKE ? OR tags LIKE ?)", like, like, like)
	}
	if v := strings.TrimSpace(req.Visibility); v != "" {
		q = q.Where("visibility = ?", v)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var list []PresetTemplate
	if err := q.Order("is_builtin DESC, use_count DESC, id DESC").
		Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	totalPage := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPage++
	}

	return &PresetListResult{
		List:      list,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}, nil
}

// GetPresetTemplate 按 ID 获取单个预设模板。
func GetPresetTemplate(db *gorm.DB, id int64) (*PresetTemplate, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var row PresetTemplate
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// IncrementPresetUseCount 增加模板使用次数。
func IncrementPresetUseCount(db *gorm.DB, id int64) error {
	if db == nil {
		return errors.New("nil db")
	}
	return db.Model(&PresetTemplate{}).Where("id = ?", id).
		UpdateColumn("use_count", gorm.Expr("use_count + 1")).Error
}
