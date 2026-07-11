package svcmodels
// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"gorm.io/gorm"
)

// Agent 表示一个自定义 AI Agent
type Agent struct {
	ID                   int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID              uint      `json:"groupId" gorm:"index"`
	CreatedBy            uint      `json:"createdBy" gorm:"index"`
	Name                 string    `json:"name" gorm:"index"`
	SystemPrompt         string    `json:"systemPrompt"`
	OpeningStatement     string    `json:"openingStatement" gorm:"column:opening_statement;type:text"`
	PersonaTag           string    `json:"personaTag"`
	Temperature          float32   `json:"temperature"`
	JsSourceID             string    `json:"jsSourceId" gorm:"index:idx_agent_js_source"`
	BoundJsTemplateSourceID string   `json:"boundJsTemplateSourceId" gorm:"column:bound_js_template_source_id;size:64"`
	MaxTokens            int       `json:"maxTokens"`
	Speaker              string    `json:"speaker" gorm:"column:speaker"`
	VoiceCloneID         *int      `json:"voiceCloneId" gorm:"column:voice_clone_id"`
	TtsProvider          string    `json:"ttsProvider" gorm:"column:tts_provider"`
	ApiKey               string    `json:"apiKey" gorm:"column:api_key"`
	ApiSecret            string    `json:"apiSecret" gorm:"column:api_secret"`
	LLMModel             string    `json:"llmModel" gorm:"column:llm_model"`
	EnableVAD            bool      `json:"enableVAD" gorm:"column:enable_vad;default:true"`
	VADThreshold         float64   `json:"vadThreshold" gorm:"column:vad_threshold;default:500"`
	VADConsecutiveFrames int       `json:"vadConsecutiveFrames" gorm:"column:vad_consecutive_frames;default:2"`
	EnableJSONOutput     bool      `json:"enableJSONOutput" gorm:"column:enable_json_output;default:false"`

	// === 角色卡核心字段 ===
	AvatarURL        string  `json:"avatarUrl" gorm:"column:avatar_url;size:512"`       // 头像 URL
	Description      string  `json:"description" gorm:"column:description;type:text"` // 简短描述
	Personality      string  `json:"personality" gorm:"column:personality;type:text"`    // 人格详情
	Scenario         string  `json:"scenario" gorm:"column:scenario;type:text"`          // 场景/世界观
	ExampleDialogues string  `json:"exampleDialogues" gorm:"column:example_dialogues;type:text"` // 示例对话 (JSON)
	Tags             string  `json:"tags" gorm:"column:tags;size:500"`                   // 标签（逗号分隔）
	CreatorNote      string  `json:"creatorNote" gorm:"column:creator_note;size:1000"`   // 创作者备注
	SpecVersion      string  `json:"specVersion" gorm:"column:spec_version;size:10"`     // 角色卡规范版本 (v2/v3)
	Visibility       string  `json:"visibility" gorm:"column:visibility;size:20;default:'group'"` // 可见性: private/group/public
	DownloadCount    int     `json:"downloadCount" gorm:"column:download_count;default:0"`       // 下载次数
	Rating           float64 `json:"rating" gorm:"column:rating;default:0"`                      // 评分
	RatingCount      int     `json:"ratingCount" gorm:"column:rating_count;default:0"`           // 评分人数
	ForkedFrom       *int64  `json:"forkedFrom" gorm:"column:forked_from;index"`                 // Fork 来源 Agent ID

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (Agent) TableName() string { return "agents" }

// GetAgentByJSTemplateID 根据 JS 模板 ID 获取关联的 Agent
func GetAgentByJSTemplateID(db *gorm.DB, jsTemplateID string) (*Agent, error) {
	var agent Agent
	err := db.Where("bound_js_template_source_id = ?", jsTemplateID).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}
