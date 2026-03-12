package notification

import (
	"time"

	"gorm.io/gorm"
)

// InternalNotification represents internal notification
type InternalNotification struct {
	ID        uint      `json:"id" gorm:"primaryKey"` // Notification ID
	UserID    uint      `json:"user_id"`              // User ID
	Title     string    `json:"title"`                // Notification title
	Content   string    `json:"content"`              // Notification content
	Read      bool      `json:"read"`                 // Whether read
	CreatedAt time.Time `json:"created_at"`           // Creation time
}

// InternalNotificationService internal notification service
type InternalNotificationService struct {
	DB *gorm.DB // Database instance
}

// NewInternalNotificationService creates internal notification service instance
func NewInternalNotificationService(db *gorm.DB) *InternalNotificationService {
	return &InternalNotificationService{DB: db}
}

// Send sends internal notification
func (s *InternalNotificationService) Send(userID uint, title, content string) error {
	notification := InternalNotification{
		UserID:    userID,
		Title:     title,
		Content:   content,
		Read:      false,
		CreatedAt: time.Now(),
	}

	// 将通知存储到数据库
	return s.DB.Create(&notification).Error
}

// GetUnreadNotifications 获取用户的未读通知
func (s *InternalNotificationService) GetUnreadNotifications(userID uint) ([]InternalNotification, error) {
	var notifications []InternalNotification
	err := s.DB.Where("user_id = ? AND `read` = ?", userID, false).Find(&notifications).Error
	return notifications, err
}

func (s *InternalNotificationService) GetUnreadNotificationsCount(userID uint) (count int64, err error) {
	return count, s.DB.Model(&InternalNotification{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&count).Error
}

// MarkAsRead 将通知标记为已读
func (s *InternalNotificationService) MarkAsRead(notificationID uint) error {
	return s.DB.Model(&InternalNotification{}).Where("id = ?", notificationID).Update("`read`", true).Error
}

// MarkAsRead 将通知标记为已读
func (s *InternalNotificationService) MarkAllAsRead(userID uint) error {
	return s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID).Update("`read`", true).Error
}

// GetPaginatedNotifications 获取用户的分页通知，扩展返回未读和已读总数
func (s *InternalNotificationService) GetPaginatedNotifications(
	userID uint,
	page, size int,
	filter string,
	titleKeyword, contentKeyword string,
	startTime, endTime time.Time,
) ([]InternalNotification, int64, int64, int64, error) {
	var notifications []InternalNotification
	var total, totalUnread, totalRead int64

	// 1. 全量统计
	s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID).Count(&total)
	s.DB.Model(&InternalNotification{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&totalUnread)
	s.DB.Model(&InternalNotification{}).Where("user_id = ? AND `read` = ?", userID, true).Count(&totalRead)

	// 2. 列表/分页
	db := s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID)
	// 筛选只对列表有效，不影响上面统计
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

	// 返回全量统计而非filteredTotal
	return notifications, total, totalUnread, totalRead, err
}

func (s *InternalNotificationService) GetOne(userID uint, notificationID uint) (InternalNotification, error) {
	var notification InternalNotification
	return notification, s.DB.Where("user_id = ? AND id = ?", userID, notificationID).First(&notification).Error
}

func (s *InternalNotificationService) Delete(userID uint, notificationID uint) error {
	return s.DB.Where("user_id = ? AND id = ?", userID, notificationID).Delete(&InternalNotification{}).Error
}

// BatchDelete 批量删除通知
func (s *InternalNotificationService) BatchDelete(userID uint, notificationIDs []uint) (int64, error) {
	if len(notificationIDs) == 0 {
		return 0, nil
	}

	result := s.DB.Where("user_id = ? AND id IN ?", userID, notificationIDs).Delete(&InternalNotification{})
	return result.RowsAffected, result.Error
}

// GetAllNotificationIds 获取所有通知ID（用于全选功能）
func (s *InternalNotificationService) GetAllNotificationIds(
	userID uint,
	filter string,
	titleKeyword, contentKeyword string,
	startTime, endTime time.Time,
) ([]uint, error) {
	var ids []uint

	db := s.DB.Model(&InternalNotification{}).Where("user_id = ?", userID)

	// 应用筛选条件
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
