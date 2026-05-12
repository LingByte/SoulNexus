package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// MultiKeyMode 多 Key 调度方式（仅 LLM 渠道使用）。
type MultiKeyMode string

const (
	MultiKeyModeRandom  MultiKeyMode = "random"  // 随机
	MultiKeyModePolling MultiKeyMode = "polling" // 轮询
)

// 上游 LLM 渠道商 / 协议（与路由、鉴权头、Base URL 约定对齐）。
const (
	LLMChannelProtocolOpenAI    = "openai"
	LLMChannelProtocolAnthropic = "anthropic"
	LLMChannelProtocolCoze      = "coze"
	LLMChannelProtocolOllama    = "ollama"
	LLMChannelProtocolLMStudio  = "lmstudio"
)

// LLMChannelProtocols 管理端可选协议列表。
var LLMChannelProtocols = []string{
	LLMChannelProtocolOpenAI,
	LLMChannelProtocolAnthropic,
	LLMChannelProtocolCoze,
	LLMChannelProtocolOllama,
	LLMChannelProtocolLMStudio,
}

// IsLLMChannelProtocolKnown 校验 API 入参。
func IsLLMChannelProtocolKnown(p string) bool {
	p = strings.ToLower(strings.TrimSpace(p))
	for _, x := range LLMChannelProtocols {
		if p == x {
			return true
		}
	}
	return false
}

// LLMChannel 大模型上游渠道（OpenAI 兼容、自建网关等），与 ASR/TTS 分表。
type LLMChannel struct {
	Id                 int            `json:"id"`
	OrgID              uint           `json:"org_id" gorm:"index;default:0;comment:租户隔离：0=全局共享渠道；>0 仅该 Group(组织) 可见"`
	Protocol           string         `json:"protocol" gorm:"size:32;not null;default:'openai';index;comment:渠道协议 openai|anthropic|coze|ollama|lmstudio"`
	Type               int            `json:"type" gorm:"default:0"`
	Key                string         `json:"key" gorm:"not null"`
	OpenAIOrganization *string        `json:"openai_organization"`
	TestModel          *string        `json:"test_model"`
	Status             int            `json:"status" gorm:"default:1"`
	Name               string         `json:"name" gorm:"index"`
	Weight             *uint          `json:"weight" gorm:"default:0"`
	CreatedTime        int64          `json:"created_time" gorm:"bigint"`
	TestTime           int64          `json:"test_time" gorm:"bigint"`
	ResponseTime       int            `json:"response_time"` // in milliseconds
	BaseURL            *string        `json:"base_url" gorm:"column:base_url;default:''"`
	Balance            float64        `json:"balance"` // in USD
	BalanceUpdatedTime int64          `json:"balance_updated_time" gorm:"bigint"`
	Models             string         `json:"models"`
	Group              string         `json:"group" gorm:"type:varchar(64);default:'default'"`
	UsedQuota          int64          `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string        `json:"model_mapping" gorm:"type:text"`
	StatusCodeMapping  *string        `json:"status_code_mapping" gorm:"type:varchar(1024);default:''"`
	Priority           *int64         `json:"priority" gorm:"bigint;default:0"`
	AutoBan            *int           `json:"auto_ban" gorm:"default:1"`
	Tag                *string        `json:"tag" gorm:"index"`
	ChannelInfo        LLMChannelInfo `json:"channel_info" gorm:"column:channel_info;type:json"`
	Keys               []string       `json:"-" gorm:"-"`
}

// LLMChannelInfo 多 Key 与轮询状态（仅 LLM）。
type LLMChannelInfo struct {
	IsMultiKey             bool           `json:"is_multi_key"`
	MultiKeySize           int            `json:"multi_key_size"`
	MultiKeyStatusList     map[int]int    `json:"multi_key_status_list"`
	MultiKeyDisabledReason map[int]string `json:"multi_key_disabled_reason,omitempty"`
	MultiKeyDisabledTime   map[int]int64  `json:"multi_key_disabled_time,omitempty"`
	MultiKeyPollingIndex   int            `json:"multi_key_polling_index"`
	MultiKeyMode           MultiKeyMode   `json:"multi_key_mode"`
}

// Value 实现 driver.Valuer，供 GORM 写入 JSON 列（sqlite/mysql 等）。
func (i LLMChannelInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan 实现 sql.Scanner，供 GORM 从 JSON 列扫描到结构体。
func (i *LLMChannelInfo) Scan(value interface{}) error {
	if value == nil {
		*i = LLMChannelInfo{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("LLMChannelInfo: unsupported scan type %T", value)
	}
	if len(raw) == 0 || string(raw) == "null" {
		*i = LLMChannelInfo{}
		return nil
	}
	return json.Unmarshal(raw, i)
}

// TableName GORM 表名（与 ASR/TTS 分表）。
func (LLMChannel) TableName() string {
	return "llm_channels"
}
