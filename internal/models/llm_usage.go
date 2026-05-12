// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// LLMUsageChannelAttempt 多渠道路由/重试时单次走向（失败后再换渠道成功时形成数组）。
type LLMUsageChannelAttempt struct {
	Order        int    `json:"order"` // 从 1 递增
	ChannelID    int    `json:"channel_id"`
	BaseURL      string `json:"base_url,omitempty"`
	Success      bool   `json:"success"`
	StatusCode   int    `json:"status_code,omitempty"`
	LatencyMs    int64  `json:"latency_ms,omitempty"` // 该次上游往返耗时
	TTFTMs       int64  `json:"ttft_ms,omitempty"`    // 非流式时常与 LatencyMs 同量级；流式场景可填首包
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// LLMUsageChannelAttempts JSON 列类型。
type LLMUsageChannelAttempts []LLMUsageChannelAttempt

func (a LLMUsageChannelAttempts) Value() (driver.Value, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(a)
}

func (a *LLMUsageChannelAttempts) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("LLMUsageChannelAttempts: unsupported scan type %T", value)
	}
	if len(raw) == 0 || string(raw) == "null" {
		*a = nil
		return nil
	}
	var sl []LLMUsageChannelAttempt
	if err := json.Unmarshal(raw, &sl); err != nil {
		return err
	}
	*a = LLMUsageChannelAttempts(sl)
	return nil
}

// LLMUsageUserDaily 用户 + UTC 日期维度的用量汇总（listener 增量 upsert；面板按区间 SUM，避免每次扫 llm_usage 全表）。
type LLMUsageUserDaily struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	UserID       string `json:"user_id" gorm:"size:64;uniqueIndex:ux_llm_usage_user_daily;not null;index"`
	StatDate     string `json:"stat_date" gorm:"size:10;uniqueIndex:ux_llm_usage_user_daily;not null"` // YYYY-MM-DD UTC
	RequestCount int64  `json:"request_count" gorm:"default:0"`
	SuccessCount int64  `json:"success_count" gorm:"default:0"`
	TokenSum     int64  `json:"token_sum" gorm:"default:0"`
	QuotaSum     int64  `json:"quota_sum" gorm:"default:0"`
}

func (LLMUsageUserDaily) TableName() string { return "llm_usage_user_daily" }

// LLMUsageUserModelDaily 用户 + 日期 + 模型维度的汇总（用于面板模型榜等）。
type LLMUsageUserModelDaily struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	UserID       string `json:"user_id" gorm:"size:64;uniqueIndex:ux_llm_usage_um_daily;not null;index"`
	StatDate     string `json:"stat_date" gorm:"size:10;uniqueIndex:ux_llm_usage_um_daily;not null"`
	Model        string `json:"model" gorm:"size:255;uniqueIndex:ux_llm_usage_um_daily;not null"`
	RequestCount int64  `json:"request_count" gorm:"default:0"`
	SuccessCount int64  `json:"success_count" gorm:"default:0"`
	TokenSum     int64  `json:"token_sum" gorm:"default:0"`
	QuotaSum     int64  `json:"quota_sum" gorm:"default:0"`
}

func (LLMUsageUserModelDaily) TableName() string { return "llm_usage_user_model_daily" }
