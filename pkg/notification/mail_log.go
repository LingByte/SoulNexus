package notification

import (
	"time"

	"gorm.io/gorm"
)

// MailLog email log record
type MailLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	ToEmail   string    `gorm:"index" json:"to_email"`
	Subject   string    `json:"subject"`
	Status    string    `gorm:"index" json:"status"` // sent, delivered, failed, bounced, spam, etc.
	ErrorMsg  string    `json:"error_msg"`
	MessageID string    `gorm:"type:varchar(255);index" json:"message_id"` // SendCloud message ID (can be empty if not obtained)
	IPAddress string    `json:"ip_address"`                                // IP address for tracking when no user context
	SentAt    time.Time `json:"sent_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (MailLog) TableName() string {
	return "mail_logs"
}

// CreateMailLog creates a new mail log record
func CreateMailLog(db *gorm.DB, userID uint, toEmail, subject, messageID string) (*MailLog, error) {
	return CreateMailLogWithIP(db, userID, toEmail, subject, messageID, "")
}

// CreateMailLogWithIP creates a new mail log record with IP address
func CreateMailLogWithIP(db *gorm.DB, userID uint, toEmail, subject, messageID, ipAddress string) (*MailLog, error) {
	log := &MailLog{
		UserID:    userID,
		ToEmail:   toEmail,
		Subject:   subject,
		Status:    "sent",
		MessageID: messageID,
		IPAddress: ipAddress,
		SentAt:    time.Now(),
	}

	if err := db.Create(log).Error; err != nil {
		return nil, err
	}

	return log, nil
}

// UpdateMailLogStatus updates the status of a mail log
func UpdateMailLogStatus(db *gorm.DB, messageID, status, errorMsg string) error {
	return db.Model(&MailLog{}).
		Where("message_id = ?", messageID).
		Updates(map[string]interface{}{
			"status":    status,
			"error_msg": errorMsg,
		}).Error
}

// GetMailLogByMessageID gets mail log by message ID
func GetMailLogByMessageID(db *gorm.DB, messageID string) (*MailLog, error) {
	var log MailLog
	if err := db.Where("message_id = ?", messageID).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetMailLogs gets mail logs with pagination
func GetMailLogs(db *gorm.DB, userID uint, page, pageSize int) ([]MailLog, int64, error) {
	var logs []MailLog
	var total int64

	query := db.Where("user_id = ?", userID)
	if err := query.Model(&MailLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetMailLogsWithStatus gets mail logs with pagination and status filter
func GetMailLogsWithStatus(db *gorm.DB, userID uint, page, pageSize int, status string) ([]MailLog, int64, error) {
	var logs []MailLog
	var total int64

	query := db.Where("user_id = ?", userID)

	// Filter by status if not "all"
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Model(&MailLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetMailLogByID gets mail log by ID and user ID
func GetMailLogByID(db *gorm.DB, userID, logID uint) (*MailLog, error) {
	var log MailLog
	if err := db.Where("id = ? AND user_id = ?", logID, userID).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetMailLogStats gets mail log statistics
func GetMailLogStats(db *gorm.DB, userID uint) (map[string]interface{}, error) {
	var stats struct {
		Total        int64
		Sent         int64
		Delivered    int64
		Failed       int64
		Bounced      int64
		Spam         int64
		Clicked      int64
		Opened       int64
		Unsubscribed int64
	}

	query := db.Where("user_id = ?", userID)

	query.Model(&MailLog{}).Count(&stats.Total)
	query.Where("status = ?", "sent").Count(&stats.Sent)
	query.Where("status = ?", "delivered").Count(&stats.Delivered)
	query.Where("status = ?", "failed").Count(&stats.Failed)
	query.Where("status = ?", "bounced").Count(&stats.Bounced)
	query.Where("status = ?", "spam").Count(&stats.Spam)
	query.Where("status = ?", "clicked").Count(&stats.Clicked)
	query.Where("status = ?", "opened").Count(&stats.Opened)
	query.Where("status = ?", "unsubscribed").Count(&stats.Unsubscribed)

	return map[string]interface{}{
		"total":        stats.Total,
		"sent":         stats.Sent,
		"delivered":    stats.Delivered,
		"failed":       stats.Failed,
		"bounced":      stats.Bounced,
		"spam":         stats.Spam,
		"clicked":      stats.Clicked,
		"opened":       stats.Opened,
		"unsubscribed": stats.Unsubscribed,
	}, nil
}
