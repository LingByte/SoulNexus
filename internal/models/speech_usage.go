// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// SpeechUsage：每次 ASR / TTS 调用一行；用于审计、用量统计、对账。
// 不落原始音频内容（base64），只记录字节数与文本字符数。

package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SpeechUsageKind 调用类型。
type SpeechUsageKind string

const (
	SpeechUsageKindASR SpeechUsageKind = "asr"
	SpeechUsageKindTTS SpeechUsageKind = "tts"
)

// SpeechUsage 一次语音调用的审计记录。
type SpeechUsage struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	RequestID  string `json:"request_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	Kind       string `json:"kind" gorm:"type:varchar(8);not null;index;comment:asr|tts"`
	UserID     uint   `json:"user_id" gorm:"index;comment:LLMToken.UserID 拷贝"`
	TokenID    uint   `json:"token_id" gorm:"index;comment:LLMToken.ID"`
	GroupID    uint   `json:"group_id" gorm:"index;comment:租户 group id；0 = 系统级 / 个人"`
	GroupKey   string `json:"group_key" gorm:"size:64;index;comment:LLMToken.Group / 路由 group 字符串"`
	ChannelID  uint   `json:"channel_id" gorm:"index;comment:命中的 ASR/TTS Channel.ID"`
	Provider   string `json:"provider" gorm:"size:64;index"`
	Model      string `json:"model" gorm:"size:128;index;comment:provider 侧模型/音色，可为空"`
	Status     int    `json:"status" gorm:"index;comment:HTTP status"`
	Success    bool   `json:"success" gorm:"index"`
	ErrorMsg   string `json:"error_msg,omitempty" gorm:"type:text"`
	AudioBytes int64  `json:"audio_bytes"`
	TextChars  int    `json:"text_chars"`
	DurationMs int64  `json:"duration_ms"`
	ClientIP   string `json:"client_ip" gorm:"size:64"`
	UserAgent  string `json:"user_agent,omitempty" gorm:"size:255"`
	ReqSnap    string `json:"req_snap,omitempty" gorm:"type:text;comment:请求关键字段 JSON 摘要（无音频）"`
	RespSnap   string `json:"resp_snap,omitempty" gorm:"type:text;comment:响应关键字段 JSON 摘要（无 base64）"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName GORM 表名。
func (SpeechUsage) TableName() string { return "speech_usage" }

// NewSpeechUsageID 生成 32 字符 hex（去掉 dash 的 uuid v4）。
func NewSpeechUsageID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// CreateSpeechUsage 写入一行；id/request_id 若为空自动填充。
func CreateSpeechUsage(db *gorm.DB, row *SpeechUsage) error {
	if db == nil || row == nil {
		return nil
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = NewSpeechUsageID()
	}
	if strings.TrimSpace(row.RequestID) == "" {
		row.RequestID = row.ID
	}
	return db.Create(row).Error
}
