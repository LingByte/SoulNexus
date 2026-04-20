package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
)

const (
	AccountDeletionStatusPending   = "pending"
	AccountDeletionStatusCancelled = "cancelled"
	AccountDeletionStatusCompleted = "completed"
)

// AccountDeletionSchedule 账户注销冷静期记录（到期后由定时任务执行永久删除）
type AccountDeletionSchedule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"uniqueIndex;not null" json:"userId"`
	RequestedAt time.Time `gorm:"not null" json:"requestedAt"`
	EffectiveAt time.Time `gorm:"index;not null" json:"effectiveAt"`
	Status      string    `gorm:"size:32;index;not null;default:'pending'" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (AccountDeletionSchedule) TableName() string {
	return constants.ACCOUNT_DELETION_SCHEDULE_TABLE_NAME
}

func GetPendingAccountDeletionByUserID(db *gorm.DB, userID uint) (*AccountDeletionSchedule, error) {
	var row AccountDeletionSchedule
	err := db.Where("user_id = ? AND status = ?", userID, AccountDeletionStatusPending).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateAccountDeletionSchedule(db *gorm.DB, userID uint, effectiveAt time.Time) (*AccountDeletionSchedule, error) {
	now := time.Now()
	row := AccountDeletionSchedule{
		UserID:      userID,
		RequestedAt: now,
		EffectiveAt: effectiveAt,
		Status:      AccountDeletionStatusPending,
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CancelPendingAccountDeletion(db *gorm.DB, userID uint) error {
	return db.Model(&AccountDeletionSchedule{}).
		Where("user_id = ? AND status = ?", userID, AccountDeletionStatusPending).
		Update("status", AccountDeletionStatusCancelled).Error
}

func ListDuePendingAccountDeletions(db *gorm.DB, before time.Time) ([]AccountDeletionSchedule, error) {
	var rows []AccountDeletionSchedule
	err := db.Where("status = ? AND effective_at <= ?", AccountDeletionStatusPending, before).
		Find(&rows).Error
	return rows, err
}

func MarkAccountDeletionCompleted(db *gorm.DB, id uint) error {
	return db.Model(&AccountDeletionSchedule{}).Where("id = ?", id).
		Update("status", AccountDeletionStatusCompleted).Error
}

// CountRecentSuspiciousSuccessfulLogins 最近若干天内是否存在「成功且标记可疑」的登录（用于注销风险拦截）
func CountRecentSuspiciousSuccessfulLogins(db *gorm.DB, userID uint, since time.Time) (int64, error) {
	var n int64
	err := db.Model(&LoginHistory{}).
		Where("user_id = ? AND success = ? AND is_suspicious = ? AND created_at >= ?", userID, true, true, since).
		Count(&n).Error
	return n, err
}
