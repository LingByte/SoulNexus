package sms

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

const (
	SmsStatusAccepted = "accepted"
	SmsStatusFailed   = "failed"
)

// SMSLog is a persisted record of an outbound SMS (optional when DB is wired).
type SMSLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	OrgID       uint      `gorm:"index;not null;default:0" json:"org_id"`
	UserID      uint      `gorm:"index;not null;default:0" json:"user_id"`
	Provider    string    `gorm:"size:32;index" json:"provider"`
	ChannelName string    `gorm:"size:128;index" json:"channel_name"`
	ToPhone     string    `gorm:"size:64;index" json:"to_phone"`
	Template    string    `gorm:"size:128;index" json:"template"`
	Content     string    `gorm:"type:text" json:"content"`
	Status      string    `gorm:"size:32;index" json:"status"`
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	MessageID   string    `gorm:"type:varchar(255);index" json:"message_id"`
	Raw         string    `gorm:"type:longtext" json:"raw"`
	IPAddress   string    `gorm:"size:64" json:"ip_address"`
	SentAt      time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SMSLog) TableName() string { return "sms_logs" }

func CreateSMSLog(db *gorm.DB, orgID, userID uint, provider, channelName, toPhone, template, content, messageID, status, raw, ip string) (*SMSLog, error) {
	row := &SMSLog{
		ID:          uint(utils.SnowflakeUtil.NextID()),
		OrgID:       orgID,
		UserID:      userID,
		Provider:    provider,
		ChannelName: channelName,
		ToPhone:     toPhone,
		Template:    template,
		Content:     content,
		Status:      status,
		MessageID:   messageID,
		Raw:         raw,
		IPAddress:   ip,
		SentAt:      time.Now(),
	}
	if err := db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func CreateFailedSMSLog(db *gorm.DB, orgID, userID uint, provider, channelName, toPhone, template, content, errMsg, raw, ip string) (*SMSLog, error) {
	row := &SMSLog{
		ID:          uint(utils.SnowflakeUtil.NextID()),
		OrgID:       orgID,
		UserID:      userID,
		Provider:    provider,
		ChannelName: channelName,
		ToPhone:     toPhone,
		Template:    template,
		Content:     content,
		Status:      SmsStatusFailed,
		ErrorMsg:    errMsg,
		Raw:         raw,
		IPAddress:   ip,
		SentAt:      time.Time{},
	}
	if err := db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}
