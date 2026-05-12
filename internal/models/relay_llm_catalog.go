// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 解析 LLMToken.OpenAPIModelCatalogJSON 与从 abilities/channels 推导模型清单。
//
// 优先级（与 LingVoice 对齐）：
//  1) Token.ModelWhitelist 强制裁剪（任何来源都要过这一层）；
//  2) Token.OpenAPIModelCatalogJSON 显式列表（数组对象 / 数组字符串）；
//  3) llm_abilities WHERE group = token.group AND enabled；
//  4) 退化：llm_channels WHERE group = token.group AND status = 1，按 Models CSV 聚合。
//
// 仅 OpenAI 协议会落 ability 行；Anthropic 协议的 channel 也会通过 channel.Models 兜底，让 GET /v1/models 至少能列出非 OpenAI 渠道。

package models

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

// OpenAPIRelayModelItem GET /v1/models data[] 的形状（OpenAI 兼容）。
type OpenAPIRelayModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// SplitChannelModelList 把"a,b\nc"风格切成 trim 后的字符串列表。
func SplitChannelModelList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := make([]string, 0, 4)
	seen := make(map[string]struct{})
	for _, raw := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	}) {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// ParseTokenOpenAPIModelCatalog 把 token.OpenAPIModelCatalogJSON 解析为模型 ID 列表。
// 支持两种 JSON 形态：
//   - ["gpt-4o","claude-3-5-sonnet"]
//   - [{"id":"gpt-4o","owned_by":"openai"}, ...]
//
// 返回 nil 表示未配置（应走推导逻辑），返回空切片表示"显式空列表"（前端会看到 0 模型）。
func ParseTokenOpenAPIModelCatalog(jsonStr string) []OpenAPIRelayModelItem {
	s := strings.TrimSpace(jsonStr)
	if s == "" {
		return nil
	}
	var asObj []OpenAPIRelayModelItem
	if err := json.Unmarshal([]byte(s), &asObj); err == nil && len(asObj) > 0 {
		out := make([]OpenAPIRelayModelItem, 0, len(asObj))
		for _, it := range asObj {
			id := strings.TrimSpace(it.ID)
			if id == "" {
				continue
			}
			out = append(out, OpenAPIRelayModelItem{ID: id, OwnedBy: strings.TrimSpace(it.OwnedBy)})
		}
		return out
	}
	var asArr []string
	if err := json.Unmarshal([]byte(s), &asArr); err == nil {
		out := make([]OpenAPIRelayModelItem, 0, len(asArr))
		for _, x := range asArr {
			id := strings.TrimSpace(x)
			if id == "" {
				continue
			}
			out = append(out, OpenAPIRelayModelItem{ID: id})
		}
		return out
	}
	return nil
}

// CollectAbilityModelIDsForGroup 取 group 下 enabled 的 ability.model 去重列表。
// orgID > 0 时只返回 OrgID 为 0（全局）或 = orgID（该租户专享）的 ability。
func CollectAbilityModelIDsForGroup(db *gorm.DB, group string, orgID uint) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	g := strings.TrimSpace(group)
	if g == "" {
		g = "default"
	}
	type row struct {
		Model string
	}
	var rows []row
	q := db.Model(&LLMAbility{}).Distinct("model").Where("`group` = ? AND enabled = ?", g, true)
	q = ScopeOrgVisible(q, orgID)
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{})
	for _, r := range rows {
		v := strings.TrimSpace(r.Model)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out, nil
}

// CollectChannelModelIDsForGroup 退化路径：从启用的 LLMChannel.Models 聚合；OrgID 可见性同 ScopeOrgVisible。
func CollectChannelModelIDsForGroup(db *gorm.DB, group string, orgID uint) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	g := strings.TrimSpace(group)
	if g == "" {
		g = "default"
	}
	var chans []LLMChannel
	q := db.Model(&LLMChannel{}).Where("`group` = ? AND status = ?", g, 1)
	q = ScopeOrgVisible(q, orgID)
	if err := q.Find(&chans).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, ch := range chans {
		for _, m := range SplitChannelModelList(ch.Models) {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			out = append(out, m)
		}
	}
	return out, nil
}

// BuildOpenAPIModelListForToken 按优先级组装最终 GET /v1/models 列表。
func BuildOpenAPIModelListForToken(db *gorm.DB, tok *LLMToken) ([]OpenAPIRelayModelItem, error) {
	if tok == nil {
		return nil, nil
	}
	// 1) catalog override
	items := ParseTokenOpenAPIModelCatalog(tok.OpenAPIModelCatalogJSON)
	if items == nil {
		orgID := TokenOrgID(tok)
		// 2) abilities
		ids, err := CollectAbilityModelIDsForGroup(db, tok.Group, orgID)
		if err != nil {
			return nil, err
		}
		if len(ids) > 0 {
			for _, id := range ids {
				items = append(items, OpenAPIRelayModelItem{ID: id})
			}
		} else {
			// 3) channel models fallback
			ids, err = CollectChannelModelIDsForGroup(db, tok.Group, orgID)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				items = append(items, OpenAPIRelayModelItem{ID: id})
			}
		}
	}
	// 4) ModelWhitelist 裁剪
	out := make([]OpenAPIRelayModelItem, 0, len(items))
	for _, it := range items {
		if !tok.AllowsModel(it.ID) {
			continue
		}
		if it.OwnedBy == "" {
			it.OwnedBy = "soulnexus"
		}
		it.Object = "model"
		out = append(out, it)
	}
	return out, nil
}
