package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	GroupRoleAdmin               = "admin"
	GroupRoleMember              = "member"
	SigInitSystemConfig          = "system.init"
	SoftDeleteStatusActive  int8 = 0 // Not deleted
	SoftDeleteStatusDeleted int8 = 1 // Deleted
)

type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime;comment:Creation time"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" gorm:"autoUpdateTime;comment:Update time"`
	IsDeleted int8      `json:"isDeleted,omitempty" gorm:"default:0;index;comment:Soft delete flag (0:not deleted, 1:deleted)"`
	CreateBy  string    `json:"createBy,omitempty" gorm:"size:128;index;comment:Creator"`
	UpdateBy  string    `json:"updateBy,omitempty" gorm:"size:128;index;comment:Updater"`
}

// BeforeCreate GORM hook: automatically set creation time before creating
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

// BeforeUpdate GORM hook: automatically set update time before updating
func (m *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// IsSoftDeleted checks if the record is soft deleted
func (m *BaseModel) IsSoftDeleted() bool {
	return m.IsDeleted == SoftDeleteStatusDeleted
}

// SoftDelete performs soft deletion
func (m *BaseModel) SoftDelete(operator string) {
	m.IsDeleted = SoftDeleteStatusDeleted
	m.UpdateBy = operator
	m.UpdatedAt = time.Now()
}

// Restore restores a soft deleted record
func (m *BaseModel) Restore(operator string) {
	m.IsDeleted = SoftDeleteStatusActive
	m.UpdateBy = operator
	m.UpdatedAt = time.Now()
}

// SetCreateInfo sets creation information
func (m *BaseModel) SetCreateInfo(operator string) {
	m.CreateBy = operator
	m.UpdateBy = operator
}

// SetUpdateInfo sets update information
func (m *BaseModel) SetUpdateInfo(operator string) {
	m.UpdateBy = operator
}

// GetCreatedAtString returns formatted creation time string
func (m *BaseModel) GetCreatedAtString() string {
	return m.CreatedAt.Format("2006-01-02 15:04:05")
}

// GetUpdatedAtString returns formatted update time string
func (m *BaseModel) GetUpdatedAtString() string {
	if m.UpdatedAt.IsZero() {
		return ""
	}
	return m.UpdatedAt.Format("2006-01-02 15:04:05")
}

// GetCreatedAtUnix returns creation time as Unix timestamp
func (m *BaseModel) GetCreatedAtUnix() int64 {
	return m.CreatedAt.Unix()
}

// GetUpdatedAtUnix returns update time as Unix timestamp
func (m *BaseModel) GetUpdatedAtUnix() int64 {
	if m.UpdatedAt.IsZero() {
		return 0
	}
	return m.UpdatedAt.Unix()
}
