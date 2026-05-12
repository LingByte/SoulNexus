// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Relay Gateway 用量落库：监听 SignalLLMUsage 上的 *llm.LLMUsageSignalPayload。
// 与 in-process tracker 的 map[string]interface{} payload 共存，分流到不同 case。
package listeners

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InitLLMRelayListenerWithDB 注册 relay 用量落库；可与 InitLLMListenerWithDB 同时调用，
// 二者按 sender 类型分流：map → 旧 in-process；*LLMUsageSignalPayload → relay。
func InitLLMRelayListenerWithDB(db *gorm.DB) {
	utils.Sig().Connect(constants.SignalLLMUsage, func(sender any, params ...any) {
		p, ok := sender.(*llm.LLMUsageSignalPayload)
		if !ok || p == nil {
			return
		}
		if p.RequestID == "" {
			return
		}
		// 幂等：相同 RequestID 已写过即跳过
		var existing models.LLMUsage
		if err := db.Select("id").Where("request_id = ?", p.RequestID).First(&existing).Error; err == nil {
			return
		}
		row := &models.LLMUsage{
			ID:              utils.SnowflakeUtil.GenID(),
			RequestID:       p.RequestID,
			UserID:          p.UserID,
			Provider:        p.Provider,
			Model:           p.Model,
			BaseURL:         p.BaseURL,
			RequestType:     p.RequestType,
			InputTokens:     p.InputTokens,
			OutputTokens:    p.OutputTokens,
			TotalTokens:     p.TotalTokens,
			QuotaDelta:      p.QuotaDelta,
			LatencyMs:       p.LatencyMs,
			TTFTMs:          p.TTFTMs,
			TPS:             p.TPS,
			QueueTimeMs:     p.QueueTimeMs,
			RequestContent:  p.RequestContent,
			ResponseContent: p.ResponseContent,
			UserAgent:       p.UserAgent,
			IPAddress:       p.IPAddress,
			StatusCode:      p.StatusCode,
			Success:         p.Success,
			ErrorCode:       p.ErrorCode,
			ErrorMessage:    p.ErrorMessage,
			ChannelID:       p.ChannelID,
			ChannelAttempts: convertChannelAttempts(p.ChannelAttempts),
			RequestedAt:     toTimeFromUnixMillis(p.RequestedAtMs),
			StartedAt:       toTimeFromUnixMillis(p.StartedAtMs),
			FirstTokenAt:    toTimeFromUnixMillis(p.FirstTokenAtMs),
			CompletedAt:     toTimeFromUnixMillis(p.CompletedAtMs),
		}
		if err := db.Create(row).Error; err != nil {
			return
		}
		// 日聚合 upsert（UTC）
		date := time.Unix(p.RequestedAtMs/1000, 0).UTC().Format("2006-01-02")
		upsertLLMUsageDaily(db, p.UserID, date, p.TotalTokens, p.QuotaDelta, p.Success)
		upsertLLMUsageUMDaily(db, p.UserID, date, p.Model, p.TotalTokens, p.QuotaDelta, p.Success)
	})
}

func convertChannelAttempts(in []llm.UsageChannelAttempt) models.LLMUsageChannelAttempts {
	if len(in) == 0 {
		return nil
	}
	out := make(models.LLMUsageChannelAttempts, 0, len(in))
	for _, a := range in {
		out = append(out, models.LLMUsageChannelAttempt{
			Order:        a.Order,
			ChannelID:    a.ChannelID,
			BaseURL:      a.BaseURL,
			Success:      a.Success,
			StatusCode:   a.StatusCode,
			LatencyMs:    a.LatencyMs,
			TTFTMs:       a.TTFTMs,
			ErrorCode:    a.ErrorCode,
			ErrorMessage: a.ErrorMessage,
		})
	}
	return out
}

func toTimeFromUnixMillis(ms int64) time.Time {
	if ms <= 0 {
		return time.Now()
	}
	return time.Unix(ms/1000, (ms%1000)*int64(time.Millisecond))
}

// upsertLLMUsageDaily 增量更新 (user, date) 维度。
// 使用 ON CONFLICT / ON DUPLICATE KEY UPDATE，让 SQL 层完成原子累加。
func upsertLLMUsageDaily(db *gorm.DB, userID, date string, tokens, quota int, success bool) {
	if userID == "" || date == "" {
		return
	}
	successInc := 0
	if success {
		successInc = 1
	}
	row := models.LLMUsageUserDaily{
		UserID: userID, StatDate: date,
		RequestCount: 1, SuccessCount: int64(successInc),
		TokenSum: int64(tokens), QuotaSum: int64(quota),
	}
	_ = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count": gorm.Expr("request_count + ?", 1),
			"success_count": gorm.Expr("success_count + ?", successInc),
			"token_sum":     gorm.Expr("token_sum + ?", tokens),
			"quota_sum":     gorm.Expr("quota_sum + ?", quota),
		}),
	}).Create(&row).Error
}

func upsertLLMUsageUMDaily(db *gorm.DB, userID, date, model string, tokens, quota int, success bool) {
	if userID == "" || date == "" || model == "" {
		return
	}
	successInc := 0
	if success {
		successInc = 1
	}
	row := models.LLMUsageUserModelDaily{
		UserID: userID, StatDate: date, Model: model,
		RequestCount: 1, SuccessCount: int64(successInc),
		TokenSum: int64(tokens), QuotaSum: int64(quota),
	}
	_ = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "stat_date"}, {Name: "model"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count": gorm.Expr("request_count + ?", 1),
			"success_count": gorm.Expr("success_count + ?", successInc),
			"token_sum":     gorm.Expr("token_sum + ?", tokens),
			"quota_sum":     gorm.Expr("quota_sum + ?", quota),
		}),
	}).Create(&row).Error
}
