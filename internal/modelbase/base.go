package modelbase

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"gorm.io/gorm"
)

const (
	GroupRoleAdmin          = "admin"
	GroupRoleMember         = "member"
	SigInitSystemConfig     = "system.init"
	SoftDeleteStatusActive  int8 = 0
	SoftDeleteStatusDeleted int8 = 1
)

type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime;comment:Creation time"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" gorm:"autoUpdateTime;comment:Update time"`
	IsDeleted int8      `json:"isDeleted,omitempty" gorm:"default:0;index;comment:Soft delete flag (0:not deleted, 1:deleted)"`
	CreateBy  string    `json:"createBy,omitempty" gorm:"size:128;index;comment:Creator"`
	UpdateBy  string    `json:"updateBy,omitempty" gorm:"size:128;index;comment:Updater"`
}

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
	if m.CreateBy == "" {
		m.CreateBy = "system"
	}
	if m.UpdateBy == "" {
		m.UpdateBy = m.CreateBy
	}
	return nil
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	if m.UpdateBy == "" {
		m.UpdateBy = "system"
	}
	return nil
}

func (m *BaseModel) IsSoftDeleted() bool {
	return m.IsDeleted == SoftDeleteStatusDeleted
}

func (m *BaseModel) SoftDelete(operator string) {
	m.IsDeleted = SoftDeleteStatusDeleted
	m.UpdateBy = operator
	m.UpdatedAt = time.Now()
}

func (m *BaseModel) Restore(operator string) {
	m.IsDeleted = SoftDeleteStatusActive
	m.UpdateBy = operator
	m.UpdatedAt = time.Now()
}

func (m *BaseModel) SetCreateInfo(operator string) {
	m.CreateBy = operator
	m.UpdateBy = operator
}

func (m *BaseModel) SetUpdateInfo(operator string) {
	m.UpdateBy = operator
}

func (m *BaseModel) GetCreatedAtString() string {
	return m.CreatedAt.Format("2006-01-02 15:04:05")
}

func (m *BaseModel) GetUpdatedAtString() string {
	if m.UpdatedAt.IsZero() {
		return ""
	}
	return m.UpdatedAt.Format("2006-01-02 15:04:05")
}

func (m *BaseModel) GetCreatedAtUnix() int64 {
	return m.CreatedAt.Unix()
}

func (m *BaseModel) GetUpdatedAtUnix() int64 {
	if m.UpdatedAt.IsZero() {
		return 0
	}
	return m.UpdatedAt.Unix()
}
