// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Relay 多租户隔离：所有 relay 路由（LLM / ASR / TTS）按 OrgID 过滤上游资源。
//
// 规则：
//   - 资源 OrgID = 0  → 全局共享（所有 token 可见）
//   - 资源 OrgID = N  → 仅 token.OrgID = N 时可见
//
// 通用查询条件可由 ScopeOrgVisible(db, orgID) 拼到任意带 org_id 列的表。

package models

import "gorm.io/gorm"

// ScopeOrgVisible 给查询追加  WHERE org_id IN (0, ?)（orgID=0 时退化为只看全局）。
//
// 用法：
//   q := db.Model(&LLMChannel{}).Where("status = ?", 1)
//   q = ScopeOrgVisible(q, tok.OrgID)
//   q.Find(&out)
func ScopeOrgVisible(db *gorm.DB, orgID uint) *gorm.DB {
	if db == nil {
		return db
	}
	if orgID == 0 {
		return db.Where("org_id = ?", 0)
	}
	return db.Where("org_id IN ?", []uint{0, orgID})
}

// TokenOrgID 取 token 的有效 OrgID（nil 安全）。
func TokenOrgID(tok *LLMToken) uint {
	if tok == nil {
		return 0
	}
	return tok.OrgID
}
