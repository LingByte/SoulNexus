// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"strings"

	"gorm.io/gorm"
)

// LLMModelMeta 模型目录元数据（展示/说明；路由仍以 llm_channels + llm_abilities 为准）。
type LLMModelMeta struct {
	Id                   uint    `json:"id" gorm:"primaryKey"`
	ModelName            string  `json:"model_name" gorm:"size:255;uniqueIndex;not null"`
	Description          string  `json:"description,omitempty" gorm:"type:text"`
	Tags                 string  `json:"tags,omitempty" gorm:"size:255"`
	Status               int     `json:"status" gorm:"default:1"`               // 1 启用展示 0 停用
	IconURL              string  `json:"icon_url,omitempty" gorm:"size:512"`    // 可选覆盖；空则前端按 vendor / 模型名推断
	Vendor               string  `json:"vendor,omitempty" gorm:"size:64;index"` // 如 openai、anthropic、deepseek，便于图标与筛选
	SortOrder            int     `json:"sort_order" gorm:"default:0;index"`
	ContextLength        *int    `json:"context_length,omitempty"`
	MaxOutputTokens      *int    `json:"max_output_tokens,omitempty"`
	QuotaBillingMode     string  `json:"quota_billing_mode,omitempty" gorm:"size:16;default:''"`
	QuotaModelRatio      float64 `json:"quota_model_ratio" gorm:"default:1"`         // 全局倍率
	QuotaPromptRatio     float64 `json:"quota_prompt_ratio" gorm:"default:1"`        // 非缓存输入 token 权重
	QuotaCompletionRatio float64 `json:"quota_completion_ratio" gorm:"default:1"`    // 输出 token 权重
	QuotaCacheReadRatio  float64 `json:"quota_cache_read_ratio" gorm:"default:0.25"` // 缓存命中 prompt 相对非缓存的折算（对齐常见「缓存计费」思路）
	CreatedTime          int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime          int64   `json:"updated_time" gorm:"bigint"`
}

func (LLMModelMeta) TableName() string {
	return "llm_model_metas"
}

// LLMAbility 分组 + 模型名 → 可承载的 LLM 渠道（对齐 new-api abilities 思路）。
type LLMAbility struct {
	OrgID       uint    `json:"org_id" gorm:"index;default:0;comment:租户隔离：0=全局；>0 仅该 Group(组织) 可见"`
	Group       string  `json:"group" gorm:"primaryKey;size:64;index:idx_llm_ability_group_model,priority:1"`
	Model       string  `json:"model" gorm:"primaryKey;size:255;index:idx_llm_ability_group_model,priority:2"`
	ChannelId   int     `json:"channel_id" gorm:"primaryKey;index"`
	ModelMetaID *uint   `json:"model_meta_id,omitempty" gorm:"index"` // 可选：与 llm_model_metas 关联（展示/校验）
	Enabled     bool    `json:"enabled" gorm:"default:true;index"`
	Priority    int64   `json:"priority" gorm:"default:0;index"`
	Weight      uint    `json:"weight" gorm:"default:1"`
	Tag         *string `json:"tag,omitempty" gorm:"size:64"`
}

func (LLMAbility) TableName() string {
	return "llm_abilities"
}

// SplitLLMModelNamesCSV 解析渠道「模型」配置字段（逗号/分号/换行分隔）。
func SplitLLMModelNamesCSV(s string) []string {
	return splitLLMModelNamesCSV(s)
}

func splitLLMModelNamesCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	repl := strings.NewReplacer("\n", ",", "\r", "", ";", ",", "，", ",")
	s = repl.Replace(s)
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitLLMChannelGroups(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{"default"}
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"default"}
	}
	return out
}

// SyncLLMAbilitiesFromChannel 按渠道 models、group 重建该 channel_id 下全部能力行。
func SyncLLMAbilitiesFromChannel(db *gorm.DB, ch *LLMChannel) error {
	if db == nil || ch == nil {
		return nil
	}
	if err := db.Where("channel_id = ?", ch.Id).Delete(&LLMAbility{}).Error; err != nil {
		return err
	}
	modelNames := SplitLLMModelNamesCSV(ch.Models)
	if len(modelNames) == 0 {
		return nil
	}
	metaIDByLower := map[string]uint{}
	var metas []LLMModelMeta
	if err := db.Select("id", "model_name").Where("model_name IN ?", modelNames).Find(&metas).Error; err == nil {
		for i := range metas {
			metaIDByLower[strings.ToLower(metas[i].ModelName)] = metas[i].Id
		}
	}
	groups := splitLLMChannelGroups(ch.Group)
	enabled := ch.Status == 1
	pr := int64(0)
	if ch.Priority != nil {
		pr = *ch.Priority
	}
	w := uint(1)
	if ch.Weight != nil {
		w = *ch.Weight
	}
	var rows []LLMAbility
	for _, g := range groups {
		for _, m := range modelNames {
			var metaID *uint
			if id, ok := metaIDByLower[strings.ToLower(m)]; ok {
				metaID = new(uint)
				*metaID = id
			}
			rows = append(rows, LLMAbility{
				Group:       g,
				Model:       m,
				ChannelId:   ch.Id,
				ModelMetaID: metaID,
				Enabled:     enabled,
				Priority:    pr,
				Weight:      w,
				Tag:         ch.Tag,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return db.CreateInBatches(&rows, 80).Error
}
