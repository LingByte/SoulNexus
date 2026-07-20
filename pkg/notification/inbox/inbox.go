// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package inbox provides in-app (站内信) notifications backed by internal_notifications table.
package inbox

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Message is one in-app notification row.
type Message struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null;comment:接收用户 ID"`
	Title     string    `json:"title" gorm:"size:255;not null;comment:标题"`
	Content   string    `json:"content" gorm:"type:text;not null;comment:正文"`
	ActionURL   string  `json:"action_url" gorm:"size:512;comment:跳转路径或 URL"`
	ActionLabel string  `json:"action_label" gorm:"size:64;comment:跳转按钮文案"`
	Read      bool      `json:"read" gorm:"default:false;index;comment:是否已读"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

func (Message) TableName() string { return "internal_notifications" }

// Service is the generic in-app notification facade.
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// SendOptions optional deep-link for inbox messages.
type SendOptions struct {
	ActionURL   string
	ActionLabel string
}

// Send delivers one in-app message to a user.
func (s *Service) Send(userID uint, title, content string) error {
	return s.SendWith(userID, title, content, SendOptions{})
}

// SendWith delivers one in-app message with optional action link.
func (s *Service) SendWith(userID uint, title, content string, opt SendOptions) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	return s.db.Create(&Message{
		UserID:      userID,
		Title:       title,
		Content:     content,
		ActionURL:   strings.TrimSpace(opt.ActionURL),
		ActionLabel: strings.TrimSpace(opt.ActionLabel),
		Read:        false,
	}).Error
}

func (s *Service) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := s.db.Model(&Message{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&count).Error
	return count, err
}

// MarkRead marks one notification as read for the owning user (single UPDATE).
func (s *Service) MarkRead(userID, notificationID uint) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	res := s.db.Model(&Message{}).
		Where("user_id = ? AND id = ? AND `read` = ?", userID, notificationID, false).
		Update("`read`", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	// Already read or missing — verify ownership without a second write.
	_, err := s.GetOne(userID, notificationID)
	return err
}

func (s *Service) MarkAllRead(userID uint) error {
	return s.db.Model(&Message{}).
		Where("user_id = ? AND `read` = ?", userID, false).
		Update("`read`", true).Error
}

func (s *Service) GetOne(userID, notificationID uint) (Message, error) {
	var n Message
	return n, s.db.Where("user_id = ? AND id = ?", userID, notificationID).First(&n).Error
}

func (s *Service) Delete(userID, notificationID uint) error {
	return s.db.Where("user_id = ? AND id = ?", userID, notificationID).Delete(&Message{}).Error
}

func (s *Service) BatchDelete(userID uint, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := s.db.Where("user_id = ? AND id IN ?", userID, ids).Delete(&Message{})
	return res.RowsAffected, res.Error
}

type PageResult struct {
	List        []Message
	Total       int64
	TotalUnread int64
	TotalRead   int64
}

func (s *Service) ListPage(
	userID uint,
	page, size int,
	filter, titleKeyword, contentKeyword string,
	startTime, endTime time.Time,
) (PageResult, error) {
	var out PageResult
	base := s.db.Model(&Message{}).Where("user_id = ?", userID)
	_ = base.Count(&out.Total)
	_ = s.db.Model(&Message{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&out.TotalUnread)
	_ = s.db.Model(&Message{}).Where("user_id = ? AND `read` = ?", userID, true).Count(&out.TotalRead)

	q := s.db.Model(&Message{}).Where("user_id = ?", userID)
	if filter == "read" {
		q = q.Where("`read` = ?", true)
	} else if filter == "unread" {
		q = q.Where("`read` = ?", false)
	}
	if titleKeyword != "" {
		q = q.Where("title LIKE ?", "%"+titleKeyword+"%")
	}
	if contentKeyword != "" {
		q = q.Where("content LIKE ?", "%"+contentKeyword+"%")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		q = q.Where("created_at BETWEEN ? AND ?", startTime, endTime)
	} else if !startTime.IsZero() {
		q = q.Where("created_at >= ?", startTime)
	} else if !endTime.IsZero() {
		q = q.Where("created_at <= ?", endTime)
	}
	var filtered int64
	if err := q.Count(&filtered).Error; err != nil {
		return out, err
	}
	_ = filtered
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&out.List).Error
	return out, err
}

func (s *Service) AllIDs(userID uint, filter, titleKeyword, contentKeyword string, startTime, endTime time.Time) ([]uint, error) {
	var ids []uint
	q := s.db.Model(&Message{}).Where("user_id = ?", userID)
	if filter == "read" {
		q = q.Where("`read` = ?", true)
	} else if filter == "unread" {
		q = q.Where("`read` = ?", false)
	}
	if titleKeyword != "" {
		q = q.Where("title LIKE ?", "%"+titleKeyword+"%")
	}
	if contentKeyword != "" {
		q = q.Where("content LIKE ?", "%"+contentKeyword+"%")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		q = q.Where("created_at BETWEEN ? AND ?", startTime, endTime)
	} else if !startTime.IsZero() {
		q = q.Where("created_at >= ?", startTime)
	} else if !endTime.IsZero() {
		q = q.Where("created_at <= ?", endTime)
	}
	err := q.Pluck("id", &ids).Error
	return ids, err
}

func (s *Service) CleanOldUnread(before time.Time) (int64, error) {
	res := s.db.Where("`read` = ? AND created_at < ?", false, before).Delete(&Message{})
	return res.RowsAffected, res.Error
}
