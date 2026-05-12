// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 语音渠道（ASR / TTS）。设计参照 LingVoice：
//   - 与 LLMChannel 完全分表，避免一张超宽表。
//   - 厂商相关参数走 ConfigJSON（JSON 文本）。
//   - Group 用于路由（与 LLMToken.Group + LLMAbility 类似）。
//   - SortOrder 用于同 group 内优先级（数字越大越优先）。

package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// TTSChannel 语音合成上游渠道。
type TTSChannel struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	OrgID      uint           `json:"org_id" gorm:"index;default:0;comment:租户隔离：0=全局共享渠道；>0 仅该 Group(组织) 可见"`
	Provider   string         `json:"provider" gorm:"size:64;not null;index;comment:厂商如 aliyun_cosyvoice、azure、edge、openai"`
	Name       string         `json:"name" gorm:"size:128;not null;comment:展示名称"`
	Enabled    bool           `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Group      string         `json:"group" gorm:"size:64;default:'';index;comment:路由分组"`
	SortOrder  int            `json:"sort_order" gorm:"not null;default:0;index;comment:同组内优先级，越大越优先"`
	Models     string         `json:"models" gorm:"type:text;comment:支持模型/音色，逗号分隔"`
	ConfigJSON string         `json:"config_json,omitempty" gorm:"type:text;comment:厂商相关 JSON 配置"`
	CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName GORM 表名。
func (TTSChannel) TableName() string {
	return "tts_channels"
}

// GetID / GetProvider / GetGroup 实现 channelIDProvider 抽象（供 SpeechUsage 记录使用）。
func (t TTSChannel) GetID() uint         { return t.ID }
func (t TTSChannel) GetProvider() string { return t.Provider }
func (t TTSChannel) GetGroup() string    { return t.Group }

// ASRChannel 语音识别上游渠道。
type ASRChannel struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	OrgID      uint           `json:"org_id" gorm:"index;default:0;comment:租户隔离：0=全局共享渠道；>0 仅该 Group(组织) 可见"`
	Provider   string         `json:"provider" gorm:"size:64;not null;index;comment:厂商如 aliyun_funasr、azure、whisper"`
	Name       string         `json:"name" gorm:"size:128;not null;comment:展示名称"`
	Enabled    bool           `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Group      string         `json:"group" gorm:"size:64;default:'';index;comment:路由分组"`
	SortOrder  int            `json:"sort_order" gorm:"not null;default:0;index;comment:同组内优先级，越大越优先"`
	Models     string         `json:"models" gorm:"type:text;comment:支持模型，逗号分隔"`
	ConfigJSON string         `json:"config_json,omitempty" gorm:"type:text;comment:厂商相关 JSON 配置"`
	CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName GORM 表名。
func (ASRChannel) TableName() string {
	return "asr_channels"
}

// GetID / GetProvider / GetGroup 实现 channelIDProvider 抽象。
func (a ASRChannel) GetID() uint         { return a.ID }
func (a ASRChannel) GetProvider() string { return a.Provider }
func (a ASRChannel) GetGroup() string    { return a.Group }

// PickASRChannel 选出 group 内 enabled=true 优先级最高的渠道。
func PickASRChannel(db *gorm.DB, group string) (*ASRChannel, error) {
	if db == nil {
		return nil, errors.New("db nil")
	}
	g := strings.TrimSpace(group)
	if g == "" {
		g = "default"
	}
	var ch ASRChannel
	q := db.Where("enabled = ? AND `group` IN (?, '')", true, g).Order("sort_order DESC, id ASC")
	if err := q.First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// PickTTSChannel 选出 group 内 enabled=true 优先级最高的渠道。
func PickTTSChannel(db *gorm.DB, group string) (*TTSChannel, error) {
	if db == nil {
		return nil, errors.New("db nil")
	}
	g := strings.TrimSpace(group)
	if g == "" {
		g = "default"
	}
	var ch TTSChannel
	q := db.Where("enabled = ? AND `group` IN (?, '')", true, g).Order("sort_order DESC, id ASC")
	if err := q.First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}
