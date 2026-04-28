package alert

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TriggerService 告警触发服务
type TriggerService struct {
	db *gorm.DB
}

// NewTriggerService 创建告警触发服务
func NewTriggerService(db *gorm.DB) *TriggerService {
	return &TriggerService{db: db}
}

// TriggerAlert 触发告警
func (s *TriggerService) TriggerAlert(
	userID uint,
	alertType models.AlertType,
	severity models.AlertSeverity,
	title, message string,
	data map[string]interface{},
) error {
	// 查找用户启用的告警规则
	var rules []models.AlertRule
	if err := s.db.Where("user_id = ? AND alert_type = ? AND enabled = ?", userID, alertType, true).Find(&rules).Error; err != nil {
		return err
	}

	// 如果没有规则，不触发告警
	if len(rules) == 0 {
		return nil
	}

	// 检查每个规则是否应该触发
	for _, rule := range rules {
		// 检查冷却期
		if rule.LastTriggerAt != nil {
			cooldownDuration := time.Duration(rule.Cooldown) * time.Second
			if time.Since(*rule.LastTriggerAt) < cooldownDuration {
				logger.Debug("告警规则仍在冷却期内，跳过",
					zap.Uint("ruleId", rule.ID),
					zap.Uint("userId", userID),
					zap.Duration("remaining", cooldownDuration-time.Since(*rule.LastTriggerAt)))
				continue // 还在冷却期内，跳过
			}
		}

		// 检查条件是否满足
		if !s.checkConditions(&rule, alertType, data) {
			continue
		}

		// 检查是否已有未解决的相同告警（避免重复创建告警记录）
		var existingAlert models.Alert
		err := s.db.Where("user_id = ? AND rule_id = ? AND status = ?", userID, rule.ID, models.AlertStatusActive).
			Order("created_at DESC").
			First(&existingAlert).Error

		// 如果已有未解决的告警，且冷却期已过，更新现有告警而不是创建新的
		if err == nil {
			// 更新现有告警的时间戳
			existingAlert.UpdatedAt = time.Now()
			s.db.Save(&existingAlert)

			// 更新规则的触发统计和最后触发时间
			now := time.Now()
			rule.TriggerCount++
			rule.LastTriggerAt = &now
			s.db.Save(&rule)

			logger.Info("更新现有告警记录",
				zap.Uint("alertId", existingAlert.ID),
				zap.Uint("ruleId", rule.ID),
				zap.Uint("userId", userID))

			// 不发送新通知，因为告警已经存在
			continue
		}

		// 创建新的告警记录
		alertDataJSON, _ := json.Marshal(data)
		alert := models.Alert{
			UserID:    userID,
			RuleID:    rule.ID,
			AlertType: alertType,
			Severity:  severity,
			Title:     title,
			Message:   message,
			Data:      string(alertDataJSON),
			Status:    models.AlertStatusActive,
		}

		if err := s.db.Create(&alert).Error; err != nil {
			logger.Error("创建告警记录失败", zap.Error(err))
			continue
		}

		// 更新规则的触发统计
		now := time.Now()
		rule.TriggerCount++
		rule.LastTriggerAt = &now
		s.db.Save(&rule)

		// 发送通知（只在创建新告警时发送）
		go s.sendNotifications(&alert, &rule)

		logger.Info("创建新告警记录并发送通知",
			zap.Uint("alertId", alert.ID),
			zap.Uint("ruleId", rule.ID),
			zap.Uint("userId", userID))
	}

	return nil
}

// checkConditions 检查告警条件是否满足
func (s *TriggerService) checkConditions(rule *models.AlertRule, alertType models.AlertType, data map[string]interface{}) bool {
	cond, err := rule.ParseConditions()
	if err != nil {
		logger.Warn("解析告警条件失败", zap.Error(err))
		return false
	}

	switch alertType {
	case models.AlertTypeQuotaExceeded:
		// 检查配额条件
		if cond.QuotaType != "" && cond.QuotaThreshold > 0 {
			quotaType := data["quotaType"]
			quotaUsed := data["quotaUsed"]
			quotaTotal := data["quotaTotal"]

			if quotaTypeStr, ok := quotaType.(string); ok && quotaTypeStr == cond.QuotaType {
				if used, ok := quotaUsed.(float64); ok {
					if total, ok := quotaTotal.(float64); ok && total > 0 {
						percentage := (used / total) * 100
						return percentage >= cond.QuotaThreshold
					}
				}
			}
		}
		return false // 如果没有设置条件或数据不完整，不触发

	case models.AlertTypeSystemError:
		// 检查系统错误条件
		if cond.ErrorCount > 0 {
			errorCount := data["errorCount"]
			if count, ok := errorCount.(float64); ok {
				return int(count) >= cond.ErrorCount
			}
		}
		return true

	case models.AlertTypeServiceError:
		// 检查服务异常条件
		if cond.ServiceName != "" {
			serviceName := data["serviceName"]
			if name, ok := serviceName.(string); ok && name == cond.ServiceName {
				if cond.FailureRate > 0 {
					failureRate := data["failureRate"]
					if rate, ok := failureRate.(float64); ok {
						return rate >= cond.FailureRate
					}
				}
				if cond.ResponseTime > 0 {
					responseTime := data["responseTime"]
					if rt, ok := responseTime.(float64); ok {
						return rt >= float64(cond.ResponseTime)
					}
				}
			}
		}
		return true

	case models.AlertTypeCustom:
		// 自定义条件（预留）
		return true

	default:
		return true
	}
}

// sendNotifications 发送通知
func (s *TriggerService) sendNotifications(alert *models.Alert, rule *models.AlertRule) {
	channels := rule.GetChannels()
	user, err := s.getUser(alert.UserID)
	if err != nil {
		logger.Error("获取用户信息失败", zap.Error(err), zap.Uint("userId", alert.UserID))
		return
	}

	for _, channel := range channels {
		var notificationRecord models.AlertNotification
		notificationRecord.AlertID = alert.ID
		notificationRecord.Channel = channel

		var err error
		switch channel {
		case models.NotificationChannelEmail:
			err = s.sendEmailNotification(alert, rule, user)
			if err == nil {
				notificationRecord.Status = "success"
			} else {
				notificationRecord.Status = "failed"
				notificationRecord.Message = err.Error()
			}

		case models.NotificationChannelInternal:
			err = s.sendInternalNotification(alert, user)
			if err == nil {
				notificationRecord.Status = "success"
			} else {
				notificationRecord.Status = "failed"
				notificationRecord.Message = err.Error()
			}

		case models.NotificationChannelWebhook:
			err = s.sendWebhookNotification(alert, rule)
			if err == nil {
				notificationRecord.Status = "success"
			} else {
				notificationRecord.Status = "failed"
				notificationRecord.Message = err.Error()
			}

		case models.NotificationChannelSMS:
			// 短信通知（预留）
			notificationRecord.Status = "failed"
			notificationRecord.Message = "SMS notification not implemented"
		}

		now := time.Now()
		notificationRecord.SentAt = &now
		s.db.Create(&notificationRecord)

		if err != nil {
			logger.Error("发送告警通知失败",
				zap.String("channel", string(channel)),
				zap.Error(err),
				zap.Uint("alertId", alert.ID))
		}
	}

	// 更新告警的已通知状态
	now := time.Now()
	alert.Notified = true
	alert.NotifiedAt = &now
	s.db.Save(alert)
}

// sendEmailNotification 发送邮件通知
func (s *TriggerService) sendEmailNotification(alert *models.Alert, rule *models.AlertRule, user *models.User) error {
	if !user.EmailNotifications || config.GlobalConfig.Services.Mail.APIKey == "" {
		return fmt.Errorf("用户未启用邮件通知或邮件配置未设置")
	}

	mailer := notification.NewMailNotification(config.GlobalConfig.Services.Mail)

	// 构建邮件内容
	subject := fmt.Sprintf("[告警] %s", alert.Title)
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2 style="color: #dc3545;">告警通知</h2>
			<p><strong>告警类型：</strong>%s</p>
			<p><strong>严重程度：</strong>%s</p>
			<p><strong>标题：</strong>%s</p>
			<p><strong>消息：</strong>%s</p>
			<p><strong>时间：</strong>%s</p>
			<hr>
			<p style="color: #666; font-size: 12px;">此告警由规则「%s」触发</p>
		</div>
	`, alert.AlertType, alert.Severity, alert.Title, alert.Message, alert.CreatedAt.Format("2006-01-02 15:04:05"), rule.Name)

	return mailer.SendHTML(user.Email, subject, body)
}

// sendInternalNotification 发送站内通知
func (s *TriggerService) sendInternalNotification(alert *models.Alert, user *models.User) error {
	notificationService := notification.NewInternalNotificationService(s.db)
	return notificationService.Send(user.ID, alert.Title, alert.Message)
}

// sendWebhookNotification 发送Webhook通知
func (s *TriggerService) sendWebhookNotification(alert *models.Alert, rule *models.AlertRule) error {
	if rule.WebhookURL == "" {
		return fmt.Errorf("Webhook URL未设置")
	}

	// 构建Webhook payload
	payload := map[string]interface{}{
		"alert": map[string]interface{}{
			"id":        alert.ID,
			"type":      alert.AlertType,
			"severity":  alert.Severity,
			"title":     alert.Title,
			"message":   alert.Message,
			"status":    alert.Status,
			"createdAt": alert.CreatedAt,
		},
		"rule": map[string]interface{}{
			"id":   rule.ID,
			"name": rule.Name,
		},
	}

	// 解析告警数据
	if alert.Data != "" {
		var alertData map[string]interface{}
		if err := json.Unmarshal([]byte(alert.Data), &alertData); err == nil {
			payload["data"] = alertData
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 发送HTTP请求
	method := rule.WebhookMethod
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, rule.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "LingEcho-Alert/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Webhook返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// getUser 获取用户信息
func (s *TriggerService) getUser(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// TriggerQuotaAlert 触发配额告警
// quotaType: 配额类型（storage, llm_tokens, llm_calls, api_calls, call_duration, call_count, asr_duration, asr_count, tts_duration, tts_count）
// quotaUsed: 已使用的配额值
// quotaTotal: 总配额值
//
// 计算逻辑：
// 使用率 (%) = (quotaUsed / quotaTotal) × 100
// 当使用率 >= 规则中设置的阈值时，触发告警
func (s *TriggerService) TriggerQuotaAlert(userID uint, quotaType string, quotaUsed, quotaTotal float64) error {
	if quotaTotal <= 0 {
		// 如果没有设置总配额，无法计算使用率，不触发告警
		return nil
	}

	percentage := (quotaUsed / quotaTotal) * 100

	data := map[string]interface{}{
		"quotaType":  quotaType,
		"quotaUsed":  quotaUsed,
		"quotaTotal": quotaTotal,
		"percentage": percentage,
	}

	// 根据使用率自动设置严重程度（如果规则中没有指定）
	severity := models.AlertSeverityMedium
	if percentage >= 95 {
		severity = models.AlertSeverityCritical
	} else if percentage >= 90 {
		severity = models.AlertSeverityHigh
	} else if percentage >= 75 {
		severity = models.AlertSeverityMedium
	} else {
		severity = models.AlertSeverityLow
	}

	// 格式化配额显示（根据类型选择单位）
	var quotaUsedStr, quotaTotalStr string
	switch quotaType {
	case "storage":
		quotaUsedStr = formatBytes(int64(quotaUsed))
		quotaTotalStr = formatBytes(int64(quotaTotal))
	case "llm_tokens", "llm_calls", "api_calls", "call_count", "asr_count", "tts_count":
		quotaUsedStr = formatNumber(int64(quotaUsed))
		quotaTotalStr = formatNumber(int64(quotaTotal))
	case "call_duration", "asr_duration", "tts_duration":
		quotaUsedStr = formatDuration(int64(quotaUsed))
		quotaTotalStr = formatDuration(int64(quotaTotal))
	default:
		quotaUsedStr = fmt.Sprintf("%.2f", quotaUsed)
		quotaTotalStr = fmt.Sprintf("%.2f", quotaTotal)
	}

	title := fmt.Sprintf("配额使用率告警 - %s", getQuotaTypeLabel(quotaType))
	message := fmt.Sprintf("您的%s配额使用率已达到%.2f%%，已使用%s，总配额%s",
		getQuotaTypeLabel(quotaType), percentage, quotaUsedStr, quotaTotalStr)

	return s.TriggerAlert(userID, models.AlertTypeQuotaExceeded, severity, title, message, data)
}

// getQuotaTypeLabel 获取配额类型的中文标签
func getQuotaTypeLabel(quotaType string) string {
	labels := map[string]string{
		"storage":       "存储空间",
		"llm_tokens":    "LLM Token",
		"llm_calls":     "LLM 调用次数",
		"api_calls":     "API 调用次数",
		"call_duration": "通话时长",
		"call_count":    "通话次数",
		"asr_duration":  "语音识别时长",
		"asr_count":     "语音识别次数",
		"tts_duration":  "语音合成时长",
		"tts_count":     "语音合成次数",
	}
	if label, ok := labels[quotaType]; ok {
		return label
	}
	return quotaType
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatNumber 格式化数字（添加千分位）
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.2fK", float64(n)/1000)
}

// formatDuration 格式化时长（秒）
func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.1f分钟", float64(seconds)/60)
	}
	return fmt.Sprintf("%.1f小时", float64(seconds)/3600)
}

// TriggerSystemErrorAlert 触发系统错误告警
func (s *TriggerService) TriggerSystemErrorAlert(userID uint, errorCount int, errorDetails []string) error {
	data := map[string]interface{}{
		"errorCount":   errorCount,
		"errorDetails": errorDetails,
	}

	severity := models.AlertSeverityMedium
	if errorCount >= 100 {
		severity = models.AlertSeverityCritical
	} else if errorCount >= 50 {
		severity = models.AlertSeverityHigh
	}

	title := "系统异常告警"
	message := fmt.Sprintf("系统在短时间内发生了%d个错误", errorCount)

	return s.TriggerAlert(userID, models.AlertTypeSystemError, severity, title, message, data)
}

// TriggerServiceErrorAlert 触发服务异常告警
func (s *TriggerService) TriggerServiceErrorAlert(userID uint, serviceName string, failureRate float64, responseTime int) error {
	data := map[string]interface{}{
		"serviceName":  serviceName,
		"failureRate":  failureRate,
		"responseTime": responseTime,
	}

	severity := models.AlertSeverityMedium
	if failureRate >= 50 || responseTime >= 5000 {
		severity = models.AlertSeverityCritical
	} else if failureRate >= 20 || responseTime >= 3000 {
		severity = models.AlertSeverityHigh
	}

	title := fmt.Sprintf("服务异常告警 - %s", serviceName)
	message := fmt.Sprintf("服务%s的失败率为%.2f%%，平均响应时间为%dms", serviceName, failureRate, responseTime)

	return s.TriggerAlert(userID, models.AlertTypeServiceError, severity, title, message, data)
}
