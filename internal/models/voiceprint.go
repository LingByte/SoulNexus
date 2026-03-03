package models

import (
	"time"
)

// Voiceprint 声纹记录模型
type Voiceprint struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	SpeakerID     string    `json:"speaker_id" gorm:"type:varchar(255);not null"`
	AssistantID   string    `json:"assistant_id" gorm:"type:varchar(255);not null"`
	SpeakerName   string    `json:"speaker_name" gorm:"type:varchar(255)"`
	Description   string    `json:"description" gorm:"type:text;comment:用户描述：身份、职业、成就等"`
	FeatureVector []byte    `json:"-" gorm:"type:longblob;not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Voiceprint) TableName() string {
	return "voiceprints"
}

// VoiceprintCreateRequest 创建声纹请求
type VoiceprintCreateRequest struct {
	SpeakerID   string `json:"speaker_id" binding:"required"`
	AssistantID string `json:"assistant_id" binding:"required"`
	SpeakerName string `json:"speaker_name" binding:"required"`
	Description string `json:"description"`
}

// VoiceprintUpdateRequest 更新声纹请求
type VoiceprintUpdateRequest struct {
	SpeakerName string `json:"speaker_name"`
	Description string `json:"description"`
}

// VoiceprintResponse 声纹响应
type VoiceprintResponse struct {
	ID          int        `json:"id"`
	SpeakerID   string     `json:"speaker_id"`
	AssistantID string     `json:"assistant_id"`
	SpeakerName string     `json:"speaker_name"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	Confidence  float64    `json:"confidence,omitempty"`
}

// VoiceprintListResponse 声纹列表响应
type VoiceprintListResponse struct {
	Total       int                  `json:"total"`
	Voiceprints []VoiceprintResponse `json:"voiceprints"`
}

// VoiceprintRegisterRequest 声纹注册请求（用于调用voiceprint-api）
type VoiceprintRegisterRequest struct {
	SpeakerID   string `json:"speaker_id"`
	AssistantID string `json:"assistant_id"`
	AudioFile   []byte `json:"-"`
}

// VoiceprintIdentifyRequest 声纹识别请求
type VoiceprintIdentifyRequest struct {
	AssistantID  string   `json:"assistant_id" binding:"required"`
	CandidateIDs []string `json:"candidate_ids"`
	AudioFile    []byte   `json:"-"`
}

// VoiceprintIdentifyResponse 声纹识别响应
type VoiceprintIdentifyResponse struct {
	SpeakerID  string  `json:"speaker_id"`
	Score      float64 `json:"score"`
	Confidence string  `json:"confidence"`
	IsMatch    bool    `json:"is_match"`
}

// VoiceprintVerifyResponse 声纹验证响应
type VoiceprintVerifyResponse struct {
	TargetSpeakerID     string  `json:"target_speaker_id"`     // 目标说话人ID
	IdentifiedSpeakerID string  `json:"identified_speaker_id"` // 识别出的说话人ID
	Score               float64 `json:"score"`                 // 相似度分数
	Confidence          string  `json:"confidence"`            // 置信度等级
	IsMatch             bool    `json:"is_match"`              // 是否匹配（基于置信度阈值）
	IsTargetSpeaker     bool    `json:"is_target_speaker"`     // 是否为目标说话人
	VerificationPassed  bool    `json:"verification_passed"`   // 验证是否通过
}
