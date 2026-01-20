package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	GroupRoleAdmin               = "admin"
	GroupRoleMember              = "member"
	SigInitSystemConfig          = "system.init"
	SoftDeleteStatusActive  int8 = 0 // 未删除
	SoftDeleteStatusDeleted int8 = 1 // 已删除
)

type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" gorm:"autoUpdateTime;comment:更新时间"`
	IsDeleted int8      `json:"isDeleted,omitempty" gorm:"default:0;index;comment:软删除标记(0:未删除,1:已删除)"`
	CreateBy  string    `json:"createBy,omitempty" gorm:"size:128;index;comment:创建人"`
	UpdateBy  string    `json:"updateBy,omitempty" gorm:"size:128;index;comment:更新人"`
}

// BeforeCreate GORM hook: 创建前自动设置创建时间
func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	if m.IsDeleted == 0 {
		m.IsDeleted = SoftDeleteStatusActive
	}
	return nil
}

// BeforeUpdate GORM hook: 更新前自动设置更新时间
func (m *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// IsSoftDeleted 检查是否已软删除
func (m *BaseModel) IsSoftDeleted() bool {
	return m.IsDeleted == SoftDeleteStatusDeleted
}

// SoftDelete 执行软删除
func (m *BaseModel) SoftDelete(operator string) {
	m.IsDeleted = SoftDeleteStatusDeleted
	m.UpdateBy = operator
	m.UpdatedAt = time.Now()
}

// Restore 恢复软删除的记录
func (m *BaseModel) Restore(operator string) {
	m.IsDeleted = SoftDeleteStatusActive
	m.UpdateBy = operator
	m.UpdatedAt = time.Now()
}

// SetCreateInfo 设置创建信息
func (m *BaseModel) SetCreateInfo(operator string) {
	m.CreateBy = operator
	m.UpdateBy = operator
}

// SetUpdateInfo 设置更新信息
func (m *BaseModel) SetUpdateInfo(operator string) {
	m.UpdateBy = operator
}

// GetCreatedAtString 获取格式化的创建时间字符串
func (m *BaseModel) GetCreatedAtString() string {
	return m.CreatedAt.Format("2006-01-02 15:04:05")
}

// GetUpdatedAtString 获取格式化的更新时间字符串
func (m *BaseModel) GetUpdatedAtString() string {
	if m.UpdatedAt.IsZero() {
		return ""
	}
	return m.UpdatedAt.Format("2006-01-02 15:04:05")
}

// GetCreatedAtUnix 获取创建时间的Unix时间戳
func (m *BaseModel) GetCreatedAtUnix() int64 {
	return m.CreatedAt.Unix()
}

// GetUpdatedAtUnix 获取更新时间的Unix时间戳
func (m *BaseModel) GetUpdatedAtUnix() int64 {
	if m.UpdatedAt.IsZero() {
		return 0
	}
	return m.UpdatedAt.Unix()
}
