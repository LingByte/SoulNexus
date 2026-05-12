// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLMToken 是给 /v1/* Relay Gateway 用户专属调用凭证。
// 与 UserCredential 解耦：不带 API Secret、不混 ASR/TTS 配置、生命周期独立、按模型倍率扣费。
package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// LLMTokenStatus 状态
type LLMTokenStatus string

const (
	LLMTokenStatusActive   LLMTokenStatus = "active"
	LLMTokenStatusDisabled LLMTokenStatus = "disabled"
	LLMTokenStatusExpired  LLMTokenStatus = "expired"
)

// LLMTokenType 类型：决定该 token 可调用的 relay 协议族（OpenAI/Anthropic 聊天 vs 语音 ASR/TTS）。
type LLMTokenType string

const (
	LLMTokenTypeLLM LLMTokenType = "llm"
	LLMTokenTypeASR LLMTokenType = "asr"
	LLMTokenTypeTTS LLMTokenType = "tts"
)

// IsValidLLMTokenType 校验类型枚举。
func IsValidLLMTokenType(t string) bool {
	switch LLMTokenType(t) {
	case LLMTokenTypeLLM, LLMTokenTypeASR, LLMTokenTypeTTS:
		return true
	}
	return false
}

// LLMToken 用户调用 /v1/* 时的访问凭证。Relay 鉴权只看这张表。
type LLMToken struct {
	ID                      uint           `json:"id" gorm:"primaryKey"`
	UserID                  uint           `json:"user_id" gorm:"index;not null"`
	OrgID                   uint           `json:"org_id" gorm:"index;default:0;comment:租户隔离：0=个人级 token；>0 表示属于某 Group(组织) 的服务账号"`
	Name                    string         `json:"name" gorm:"size:128"`
	APIKey                  string         `json:"api_key" gorm:"uniqueIndex;size:128;not null"` // sk-... 格式
	Type                    LLMTokenType   `json:"type" gorm:"size:8;default:'llm';index;comment:llm/asr/tts"`
	Status                  LLMTokenStatus `json:"status" gorm:"size:16;default:'active';index"`
	Group                   string         `json:"group" gorm:"size:64;default:'default';index"` // 路由分组（与 llm_abilities.group 对齐）
	ModelWhitelist          string         `json:"model_whitelist" gorm:"type:text"`             // 逗号分隔；为空表示不限制
	OpenAPIModelCatalogJSON string         `json:"openapi_model_catalog_json,omitempty" gorm:"type:text;comment:GET /v1/models 用的固定列表 JSON：[{\"id\":\"x\",\"owned_by\":\"y\"}] 或 [\"x\"]"`
	UnlimitedQuota          bool           `json:"unlimited_quota" gorm:"default:false"`
	TokenQuota              int64          `json:"token_quota" gorm:"default:0"` // 总 Token 配额；0 + UnlimitedQuota=false 视为禁用
	TokenUsed               int64          `json:"token_used" gorm:"default:0"`
	QuotaUsed               int64          `json:"quota_used" gorm:"default:0"` // 按模型倍率换算后的"额度单位"累计
	RequestQuota            int64          `json:"request_quota" gorm:"default:0"`
	RequestUsed             int64          `json:"request_used" gorm:"default:0"`
	ExpiresAt               *time.Time     `json:"expires_at" gorm:"index"`
	LastUsedAt              *time.Time     `json:"last_used_at" gorm:"index"`
	CreatedAt               time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt               time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName GORM 表名。
func (LLMToken) TableName() string { return "llm_tokens" }

// IsActive 综合状态/过期/封禁判断。
func (t *LLMToken) IsActive(now time.Time) bool {
	if t == nil {
		return false
	}
	if t.Status != LLMTokenStatusActive {
		return false
	}
	if t.ExpiresAt != nil && !t.ExpiresAt.IsZero() && t.ExpiresAt.Before(now) {
		return false
	}
	return true
}

// QuotaExceeded 在请求前预检 Token / 请求总数额度。
// success 表示当前还可放行；reason 给客户端展示原因。
func (t *LLMToken) QuotaExceeded() (bool, string) {
	if t == nil {
		return true, "token not found"
	}
	if t.UnlimitedQuota {
		return false, ""
	}
	if t.TokenQuota > 0 && t.TokenUsed >= t.TokenQuota {
		return true, "token quota exhausted"
	}
	if t.RequestQuota > 0 && t.RequestUsed >= t.RequestQuota {
		return true, "request quota exhausted"
	}
	return false, ""
}

// AllowsModel 检查 model 是否在白名单内（白名单为空则放行）。
func (t *LLMToken) AllowsModel(model string) bool {
	if t == nil {
		return false
	}
	w := strings.TrimSpace(t.ModelWhitelist)
	if w == "" {
		return true
	}
	model = strings.TrimSpace(strings.ToLower(model))
	for _, m := range strings.Split(w, ",") {
		if strings.TrimSpace(strings.ToLower(m)) == model {
			return true
		}
	}
	return false
}

// LookupActiveLLMToken 按 APIKey 查找并校验状态/过期。返回 nil 时调用方应当返回 401。
func LookupActiveLLMToken(db *gorm.DB, apiKey string) (*LLMToken, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("empty api key")
	}
	var t LLMToken
	if err := db.Where("api_key = ?", apiKey).First(&t).Error; err != nil {
		return nil, err
	}
	if !t.IsActive(time.Now()) {
		return nil, errors.New("token not active")
	}
	return &t, nil
}

// AddLLMTokenUsage 在 Relay 调用结束后原子累加用量。
// quotaDelta 为按模型倍率折算后的额度单位；调用方在所有渠道失败时应当传 0。
//
// 同时检查是否跨越 80%/100% 阈值——只在跨越的那一次发一条站内通知。
// 用 SELECT before/after 两次状态对比来判断跨越；通知发送失败不影响主流程。
func AddLLMTokenUsage(db *gorm.DB, tokenID uint, tokens int, quotaDelta int) error {
	if tokenID == 0 {
		return nil
	}
	// 1. 取出旧状态（用于阈值跨越判断）
	var before LLMToken
	if err := db.Where("id = ?", tokenID).First(&before).Error; err != nil {
		// 旧记录不存在不阻断（可能 token 被删了）
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"token_used":   gorm.Expr("token_used + ?", tokens),
		"quota_used":   gorm.Expr("quota_used + ?", quotaDelta),
		"request_used": gorm.Expr("request_used + ?", 1),
		"last_used_at": now,
	}
	if err := db.Model(&LLMToken{}).Where("id = ?", tokenID).Updates(updates).Error; err != nil {
		return err
	}
	// 2. 阈值通知（best-effort，吃异常）。不限额或未授额则跳过。
	if before.UnlimitedQuota || before.TokenQuota <= 0 {
		return nil
	}
	beforeUsed := before.TokenUsed
	afterUsed := beforeUsed + int64(tokens)
	emitTokenQuotaNotification(db, &before, beforeUsed, afterUsed)
	return nil
}

// emitTokenQuotaNotification 在 token_used 跨越 80% / 100% 阈值时发站内通知。
func emitTokenQuotaNotification(db *gorm.DB, t *LLMToken, beforeUsed, afterUsed int64) {
	defer func() { _ = recover() }()
	if t == nil || db == nil || t.TokenQuota <= 0 {
		return
	}
	thresholds := []struct {
		ratio float64
		label string
	}{
		{0.8, "80%"},
		{1.0, "100%"},
	}
	name := strings.TrimSpace(t.Name)
	if name == "" {
		name = "Token"
	}
	for _, th := range thresholds {
		boundary := int64(float64(t.TokenQuota) * th.ratio)
		if beforeUsed < boundary && afterUsed >= boundary {
			title := "LLM Token 用量预警 · " + th.label
			content := "您的 LLM Token「" + name + "」用量已达 " + th.label +
				"（已用 " + formatInt(afterUsed) + " / " + formatInt(t.TokenQuota) + "）。"
			if th.ratio >= 1.0 {
				content += " 已停止接受新的 /v1/* 请求，请及时充值或调整配额。"
			} else {
				content += " 接近上限请关注。"
			}
			_ = NewInternalNotificationService(db).Send(t.UserID, title, content)
		}
	}
}

func formatInt(v int64) string {
	// 简易千位分隔，避免 strconv + fmt 多次格式化。
	s := ""
	if v < 0 {
		s = "-"
		v = -v
	}
	digits := []byte{}
	if v == 0 {
		digits = []byte{'0'}
	}
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	// 加千位分隔
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, d)
	}
	return s + string(out)
}

// GenerateLLMTokenAPIKey 32 字节随机十六进制 + sk- 前缀。
func GenerateLLMTokenAPIKey() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(buf[:]), nil
}
