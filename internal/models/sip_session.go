package models

import (
	"time"

	"gorm.io/gorm"
)

// SipSessionStatus 会话状态
type SipSessionStatus string

const (
	SipSessionStatusPending   SipSessionStatus = "pending"   // 等待ACK
	SipSessionStatusActive    SipSessionStatus = "active"    // 活跃会话
	SipSessionStatusEnded     SipSessionStatus = "ended"     // 已结束
	SipSessionStatusCancelled SipSessionStatus = "cancelled" // 已取消
)

// SipSession SIP会话表（用于存储待确认和活跃的会话信息）
type SipSession struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"-" gorm:"index"`

	// 会话基本信息
	CallID        string           `json:"callId" gorm:"size:128;index;not null;uniqueIndex"` // SIP Call-ID
	Status        SipSessionStatus `json:"status" gorm:"size:20;index"`                       // 会话状态
	RemoteRTPAddr string           `json:"remoteRtpAddr" gorm:"size:128"`                     // 远程RTP地址
	LocalRTPAddr  string           `json:"localRtpAddr" gorm:"size:128"`                      // 本地RTP地址

	// 关联信息
	CallIDRef string `json:"callIdRef,omitempty" gorm:"size:128;index"` // 关联到 SipCall 的 CallID（可选）

	// 时间信息
	CreatedTime time.Time  `json:"createdTime"`          // 创建时间
	ActiveTime  *time.Time `json:"activeTime,omitempty"` // 激活时间（收到ACK）
	EndTime     *time.Time `json:"endTime,omitempty"`    // 结束时间

	// 元数据
	Metadata string `json:"metadata,omitempty" gorm:"type:text"` // JSON格式的额外信息
}

// TableName 指定表名
func (SipSession) TableName() string {
	return "sip_sessions"
}

// CreateSipSession 创建SIP会话记录
func CreateSipSession(db *gorm.DB, session *SipSession) error {
	return db.Create(session).Error
}

// GetSipSessionByCallID 根据CallID获取会话记录
func GetSipSessionByCallID(db *gorm.DB, callID string) (*SipSession, error) {
	var session SipSession
	err := db.Where("call_id = ?", callID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateSipSession 更新SIP会话记录
func UpdateSipSession(db *gorm.DB, session *SipSession) error {
	return db.Save(session).Error
}

// DeleteSipSessionByCallID 根据CallID删除会话记录
func DeleteSipSessionByCallID(db *gorm.DB, callID string) error {
	return db.Where("call_id = ?", callID).Delete(&SipSession{}).Error
}

// GetActiveSipSessions 获取所有活跃的会话
func GetActiveSipSessions(db *gorm.DB) ([]SipSession, error) {
	var sessions []SipSession
	err := db.Where("status = ?", SipSessionStatusActive).Find(&sessions).Error
	return sessions, err
}
