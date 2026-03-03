package models

import (
	"time"

	"gorm.io/gorm"
)

// AICallSession AI通话会话记录
type AICallSession struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联信息
	CallID      string `json:"callId" gorm:"size:128;index;not null"` // Call-ID
	AssistantID int64  `json:"assistantId" gorm:"index"`              // 关联的AI助手

	// 会话状态
	Status    string     `json:"status" gorm:"size:20;index"` // active, ended, error
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime,omitempty"`

	// 对话历史（JSON格式）
	Messages string `json:"messages" gorm:"type:text"` // 消息历史

	// 统计信息
	TurnCount int `json:"turnCount" gorm:"default:0"` // 对话轮数
	Duration  int `json:"duration" gorm:"default:0"`  // 通话时长（秒）

	// 关联对象
	Assistant Assistant `json:"assistant,omitempty" gorm:"foreignKey:AssistantID"`
}

// TableName 指定表名
func (AICallSession) TableName() string {
	return "ai_call_sessions"
}

// CreateAICallSession 创建AI通话会话
func CreateAICallSession(db *gorm.DB, session *AICallSession) error {
	return db.Create(session).Error
}

// GetAICallSessionByCallID 根据CallID获取会话
func GetAICallSessionByCallID(db *gorm.DB, callID string) (*AICallSession, error) {
	var session AICallSession
	err := db.Where("call_id = ?", callID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateAICallSession 更新AI通话会话
func UpdateAICallSession(db *gorm.DB, session *AICallSession) error {
	return db.Save(session).Error
}

// GetAICallSessions 获取AI通话会话列表
func GetAICallSessions(db *gorm.DB, assistantID *int64, limit int) ([]AICallSession, error) {
	var sessions []AICallSession
	query := db.Order("created_at DESC")

	if assistantID != nil {
		query = query.Where("assistant_id = ?", *assistantID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Preload("Assistant").Find(&sessions).Error
	return sessions, err
}

// GetActiveAICallSessions 获取活跃的AI通话会话
func GetActiveAICallSessions(db *gorm.DB) ([]AICallSession, error) {
	var sessions []AICallSession
	err := db.Where("status = ?", "active").
		Preload("Assistant").
		Find(&sessions).Error
	return sessions, err
}
