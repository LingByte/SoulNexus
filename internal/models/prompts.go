package models

import "time"

type PromptModel struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:100;not null" json:"name"` // 模板唯一名称
	Description string    `gorm:"type:text" json:"description"`              // 描述
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PromptArgModel struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	PromptID    uint   `gorm:"index" json:"prompt_id"`        // 不加 not null，避免强制约束
	Name        string `gorm:"size:100;not null" json:"name"` // 参数名
	Description string `gorm:"type:text" json:"description"`  // 参数说明
	Required    bool   `gorm:"default:false" json:"required"` // 是否必填
}
