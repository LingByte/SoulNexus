// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package mail

import (
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// isFourByteEncodingErr 判断 DB 错误是否为 utf8mb3 列遇到 4 字节字符（emoji 等）所致。
// 命中后调用方应去掉 4 字节字符再重试，避免日志彻底丢失。
func isFourByteEncodingErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "Error 1366") { // Incorrect string value
		return true
	}
	if strings.Contains(msg, "Error 3988") { // Conversion from collation ... impossible
		return true
	}
	if strings.Contains(msg, "Incorrect string value") {
		return true
	}
	return false
}

// sanitizeForMailLog 兜底去掉 4 字节字符（emoji 等）以适配 utf8mb3 列。
func sanitizeForMailLog(log *MailLog) {
	log.Subject = utils.StripFourByteRunes(log.Subject)
	log.HtmlBody = utils.StripFourByteRunes(log.HtmlBody)
	log.ErrorMsg = utils.StripFourByteRunes(log.ErrorMsg)
	log.ChannelName = utils.StripFourByteRunes(log.ChannelName)
}

// alterAttempted 保证一次进程只会尝试一次表字符集升级，避免对 DB 高频 ALTER。
var alterAttempted bool

// ensureMailLogsUtf8mb4 在遇到 utf8mb3 兼容错误时，尝试把 mail_logs 升级为 utf8mb4。
// 该操作幂等，但只在 MySQL 上执行。失败也不报错，由调用方自行 fallback 到 sanitize 重写。
func ensureMailLogsUtf8mb4(db *gorm.DB) {
	if db == nil || alterAttempted {
		return
	}
	alterAttempted = true
	if db.Dialector == nil || db.Dialector.Name() != "mysql" {
		return
	}
	_ = db.Exec("ALTER TABLE mail_logs CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
}

// MailLog is a persisted record of an outbound email (optional when DB is wired).
type MailLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	OrgID       uint      `gorm:"index;not null;default:0" json:"org_id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Provider    string    `gorm:"size:32;index" json:"provider"`      // smtp | sendcloud | multi
	ChannelName string    `gorm:"size:128;index" json:"channel_name"` // MailConfig.Name when set
	ToEmail     string    `gorm:"index" json:"to_email"`
	Subject     string    `json:"subject"`
	Status      string    `gorm:"index" json:"status"`
	HtmlBody    string    `gorm:"type:longtext" json:"html_body"` // 邮件 HTML，管理端可预览
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	MessageID   string    `gorm:"type:varchar(255);index" json:"message_id"`
	IPAddress   string    `gorm:"size:64" json:"ip_address"`
	RetryCount  int       `json:"retry_count"`
	SentAt      time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName returns the GORM table name.
func (MailLog) TableName() string {
	return "mail_logs"
}

// CreateMailLog records a successful send (or send accepted by provider).
func CreateMailLog(db *gorm.DB, orgID uint, userID uint, provider, channelName, toEmail, subject, htmlBody, messageID, status string, ip string) (*MailLog, error) {
	log := &MailLog{
		ID:          uint(utils.SnowflakeUtil.NextID()),
		OrgID:       orgID,
		UserID:      userID,
		Provider:    provider,
		ChannelName: channelName,
		ToEmail:     toEmail,
		Subject:     subject,
		HtmlBody:    htmlBody,
		Status:      status,
		MessageID:   messageID,
		IPAddress:   ip,
		SentAt:      time.Now(),
	}
	if err := db.Create(log).Error; err != nil {
		if isFourByteEncodingErr(err) {
			ensureMailLogsUtf8mb4(db)
			if err2 := db.Create(log).Error; err2 == nil {
				return log, nil
			}
			sanitizeForMailLog(log)
			if err2 := db.Create(log).Error; err2 == nil {
				return log, nil
			}
		}
		return nil, err
	}
	return log, nil
}

// CreateFailedMailLog records a send that failed after all retries.
func CreateFailedMailLog(db *gorm.DB, orgID uint, userID uint, provider, channelName, toEmail, subject, htmlBody, errMsg string, retries int, ip string) (*MailLog, error) {
	log := &MailLog{
		ID:          uint(utils.SnowflakeUtil.NextID()),
		OrgID:       orgID,
		UserID:      userID,
		Provider:    provider,
		ChannelName: channelName,
		ToEmail:     toEmail,
		Subject:     subject,
		HtmlBody:    htmlBody,
		Status:      StatusFailed,
		ErrorMsg:    errMsg,
		RetryCount:  retries,
		IPAddress:   ip,
		SentAt:      time.Now(),
	}
	if err := db.Create(log).Error; err != nil {
		if isFourByteEncodingErr(err) {
			ensureMailLogsUtf8mb4(db)
			if err2 := db.Create(log).Error; err2 == nil {
				return log, nil
			}
			sanitizeForMailLog(log)
			if err2 := db.Create(log).Error; err2 == nil {
				return log, nil
			}
		}
		return nil, err
	}
	return log, nil
}

// UpdateMailLogStatusByMessageID updates status for SendCloud (and any provider keyed by message id).
// Only rows with matching provider are updated when provider is non-empty.
func UpdateMailLogStatusByMessageID(db *gorm.DB, messageID, provider, status, errorMsg string) error {
	if messageID == "" {
		return nil
	}
	q := db.Model(&MailLog{}).Where("message_id = ?", messageID)
	if provider != "" {
		q = q.Where("provider = ?", provider)
	}
	return q.Updates(map[string]interface{}{
		"status":    status,
		"error_msg": errorMsg,
	}).Error
}

// GetMailLogByMessageID returns a log by provider message id.
func GetMailLogByMessageID(db *gorm.DB, messageID string) (*MailLog, error) {
	var log MailLog
	if err := db.Where("message_id = ?", messageID).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetMailLogs returns paginated logs for a user.
func GetMailLogs(db *gorm.DB, userID uint, page, pageSize int) ([]MailLog, int64, error) {
	var logs []MailLog
	var total int64
	base := db.Model(&MailLog{}).Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// GetMailLogStats aggregates counts by status for a user.
func GetMailLogStats(db *gorm.DB, userID uint) (map[string]int64, error) {
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	if err := db.Model(&MailLog{}).Select("status, count(*) as cnt").Where("user_id = ?", userID).Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{
		"total": 0,
	}
	for _, r := range rows {
		out[r.Status] = r.Cnt
		out["total"] += r.Cnt
	}
	return out, nil
}
