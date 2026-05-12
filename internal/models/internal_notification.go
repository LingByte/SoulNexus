// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"time"

	"gorm.io/gorm"
)

// InternalNotification 站内通知
type InternalNotification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null;comment:接收用户 ID"`
	Title     string    `json:"title" gorm:"size:255;not null;comment:标题"`
	Content   string    `json:"content" gorm:"type:text;not null;comment:正文"`
	Read      bool      `json:"read" gorm:"default:false;index;comment:是否已读"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName GORM 表名
func (InternalNotification) TableName() string { return "internal_notifications" }

// InternalNotificationService 站内通知业务服务（保留与旧 pkg/notification 一致的 API，便于平迁）。
type InternalNotificationService struct {
	DB *gorm.DB
}

// NewInternalNotificationService 构造服务。
func NewInternalNotificationService(db *gorm.DB) *InternalNotificationService {
	return &InternalNotificationService{DB: db}
}

// Send 发送一条站内通知。
func (s *InternalNotificationService) Send(userID uint, title, content string) error {
	n := InternalNotification{
		UserID:    userID,
		Title:     title,
		Content:   content,
		Read:      false,
		CreatedAt: time.Now(),
	}
	return s.DB.Create(&n).Error
}

// GetUnreadNotifications 未读通知列表。
func (s *InternalNotificationService) GetUnreadNotifications(userID uint) ([]InternalNotification, error) {
	var ns []InternalNotification
	err := s.DB.Where("user_id = ? AND `read` = ?", userID, false).Find(&ns).Error
	return ns, err
}

// GetUnreadNotificationsCount 未读数。
func (s *InternalNotificationService) GetUnreadNotificationsCount(userID uint) (count int64, err error) {
	return count, s.DB.Model(&InternalNotification{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&count).Error
}

// MarkAsRead 标记单条已读。
func (s *InternalNotificationService) MarkAsRead(notificationID uint) error {
	return s.DB.Model(&InternalNotification{}).Where("id = ?", notificationID).Update("`read`", true).Error
}

// MarkAllAsRead 全部标记已读。
func (s *InternalNotificationService) MarkAllAsRead(userID uint) error {
	return s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID).Update("`read`", true).Error
}

// GetPaginatedNotifications 分页 + 全量统计。
func (s *InternalNotificationService) GetPaginatedNotifications(
	userID uint,
	page, size int,
	filter string,
	titleKeyword, contentKeyword string,
	startTime, endTime time.Time,
) ([]InternalNotification, int64, int64, int64, error) {
	var notifications []InternalNotification
	var total, totalUnread, totalRead int64

	s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID).Count(&total)
	s.DB.Model(&InternalNotification{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&totalUnread)
	s.DB.Model(&InternalNotification{}).Where("user_id = ? AND `read` = ?", userID, true).Count(&totalRead)

	db := s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID)
	if filter == "read" {
		db = db.Where("`read` = ?", true)
	} else if filter == "unread" {
		db = db.Where("`read` = ?", false)
	}
	if titleKeyword != "" {
		db = db.Where("title LIKE ?", "%"+titleKeyword+"%")
	}
	if contentKeyword != "" {
		db = db.Where("content LIKE ?", "%"+contentKeyword+"%")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		db = db.Where("created_at BETWEEN ? AND ?", startTime, endTime)
	} else if !startTime.IsZero() {
		db = db.Where("created_at >= ?", startTime)
	} else if !endTime.IsZero() {
		db = db.Where("created_at <= ?", endTime)
	}
	var filteredTotal int64
	if err := db.Count(&filteredTotal).Error; err != nil {
		return nil, 0, 0, 0, err
	}
	err := db.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&notifications).Error
	return notifications, total, totalUnread, totalRead, err
}

// GetOne 取单条（owner-scoped）。
func (s *InternalNotificationService) GetOne(userID uint, notificationID uint) (InternalNotification, error) {
	var n InternalNotification
	return n, s.DB.Where("user_id = ? AND id = ?", userID, notificationID).First(&n).Error
}

// Delete 删除单条（owner-scoped）。
func (s *InternalNotificationService) Delete(userID uint, notificationID uint) error {
	return s.DB.Where("user_id = ? AND id = ?", userID, notificationID).Delete(&InternalNotification{}).Error
}

// BatchDelete 批量删除。
func (s *InternalNotificationService) BatchDelete(userID uint, notificationIDs []uint) (int64, error) {
	if len(notificationIDs) == 0 {
		return 0, nil
	}
	result := s.DB.Where("user_id = ? AND id IN ?", userID, notificationIDs).Delete(&InternalNotification{})
	return result.RowsAffected, result.Error
}

// GetAllNotificationIds 用于全选。
func (s *InternalNotificationService) GetAllNotificationIds(
	userID uint,
	filter string,
	titleKeyword, contentKeyword string,
	startTime, endTime time.Time,
) ([]uint, error) {
	var ids []uint
	db := s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID)
	if filter == "read" {
		db = db.Where("`read` = ?", true)
	} else if filter == "unread" {
		db = db.Where("`read` = ?", false)
	}
	if titleKeyword != "" {
		db = db.Where("title LIKE ?", "%"+titleKeyword+"%")
	}
	if contentKeyword != "" {
		db = db.Where("content LIKE ?", "%"+contentKeyword+"%")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		db = db.Where("created_at BETWEEN ? AND ?", startTime, endTime)
	} else if !startTime.IsZero() {
		db = db.Where("created_at >= ?", startTime)
	} else if !endTime.IsZero() {
		db = db.Where("created_at <= ?", endTime)
	}
	err := db.Pluck("id", &ids).Error
	return ids, err
}
