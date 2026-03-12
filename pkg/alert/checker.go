package alert

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// QuotaChecker 配额检查器
type QuotaChecker struct {
	db             *gorm.DB
	triggerService *TriggerService
	checkInterval  time.Duration // 检查间隔
}

// NewQuotaChecker 创建配额检查器
func NewQuotaChecker(db *gorm.DB, triggerService *TriggerService) *QuotaChecker {
	return &QuotaChecker{
		db:             db,
		triggerService: triggerService,
		checkInterval:  5 * time.Minute, // 默认5分钟检查一次
	}
}

// Start 启动配额检查定时任务
func (qc *QuotaChecker) Start() {
	ticker := time.NewTicker(qc.checkInterval)
	go func() {
		for range ticker.C {
			qc.CheckAllQuotaAlerts()
		}
	}()
	logger.Info("配额告警检查器已启动", zap.Duration("interval", qc.checkInterval))
}

// CheckAllQuotaAlerts 检查所有用户的配额告警
func (qc *QuotaChecker) CheckAllQuotaAlerts() {
	// 获取所有启用的配额告警规则
	var rules []models.AlertRule
	if err := qc.db.Where("alert_type = ? AND enabled = ?", models.AlertTypeQuotaExceeded, true).Find(&rules).Error; err != nil {
		logger.Error("获取告警规则失败", zap.Error(err))
		return
	}

	// 按用户分组规则
	userRules := make(map[uint][]models.AlertRule)
	for _, rule := range rules {
		if rule.UserID > 0 {
			userRules[rule.UserID] = append(userRules[rule.UserID], rule)
		}
	}

	// 对每个用户检查配额
	for userID, rules := range userRules {
		qc.CheckUserQuotaAlerts(userID, rules)
	}
}

// CheckUserQuotaAlerts 检查用户的配额告警
func (qc *QuotaChecker) CheckUserQuotaAlerts(userID uint, rules []models.AlertRule) {
	for _, rule := range rules {
		cond, err := rule.ParseConditions()
		if err != nil {
			logger.Warn("解析告警条件失败", zap.Error(err), zap.Uint("ruleId", rule.ID))
			continue
		}

		if cond.QuotaType == "" || cond.QuotaThreshold <= 0 {
			continue
		}

		// 检查冷却期
		if rule.LastTriggerAt != nil {
			cooldownDuration := time.Duration(rule.Cooldown) * time.Second
			if time.Since(*rule.LastTriggerAt) < cooldownDuration {
				continue
			}
		}

		// 获取有效配额
		quotaType := models.QuotaType(cond.QuotaType)
		totalQuota, usedQuota, err := models.GetEffectiveQuota(qc.db, userID, quotaType)
		if err != nil {
			logger.Warn("获取配额失败", zap.Error(err), zap.Uint("userId", userID), zap.String("quotaType", cond.QuotaType))
			continue
		}

		// 如果总配额为0，表示无限制，不触发告警
		if totalQuota == 0 {
			continue
		}

		// 计算使用率（usedQuota 已经从 GetEffectiveQuota 获取，是实时统计的）
		percentage := (float64(usedQuota) / float64(totalQuota)) * 100

		// 检查是否达到阈值
		if percentage >= cond.QuotaThreshold {
			// 更新使用量缓存（异步，不影响告警触发）
			go models.UpdateUserQuotaUsage(qc.db, userID, quotaType)

			// 触发告警（TriggerQuotaAlert 内部会处理冷却期和重复告警检查）
			qc.triggerService.TriggerQuotaAlert(userID, cond.QuotaType, float64(usedQuota), float64(totalQuota))
		}
	}
}

// CheckGroupQuotaAlerts 检查组织的配额告警（预留）
func (qc *QuotaChecker) CheckGroupQuotaAlerts(groupID uint) {
	// TODO: 实现组织级别的配额告警检查
}
