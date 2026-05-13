package models

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
	Description          string    `json:"description"`
	SystemPrompt         string    `json:"systemPrompt"`
	PersonaTag           string    `json:"personaTag"`
	Temperature          float32   `json:"temperature"`
	JsSourceID           string    `json:"jsSourceId" gorm:"index:idx_agent_js_source"`
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
	CreatedAt            time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt            time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (Agent) TableName() string { return "agents" }

// GetAgentByJSTemplateID 根据 JS 模板 ID 获取关联的 Agent
func GetAgentByJSTemplateID(db *gorm.DB, jsTemplateID string) (*Agent, error) {
	var agent Agent
	err := db.Where("js_source_id = ?", jsTemplateID).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}
