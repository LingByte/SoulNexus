package notification

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&InternalNotification{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func TestNewInternalNotificationService(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.DB)
}

func TestInternalNotificationService_Send(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	err := service.Send(1, "Test Title", "Test Content")
	assert.NoError(t, err)

	// Verify notification was created
	var notification InternalNotification
	err = db.Where("user_id = ?", 1).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Title", notification.Title)
	assert.Equal(t, "Test Content", notification.Content)
	assert.False(t, notification.Read)
}

func TestInternalNotificationService_GetUnreadNotifications(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create read and unread notifications
	service.Send(1, "Unread 1", "Content 1")
	service.Send(1, "Unread 2", "Content 2")

	// Mark one as read
	var notification InternalNotification
	db.Where("user_id = ? AND title = ?", 1, "Unread 1").First(&notification)
	service.MarkAsRead(notification.ID)

	// Get unread notifications
	notifications, err := service.GetUnreadNotifications(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(notifications))
	assert.Equal(t, "Unread 2", notifications[0].Title)
}

func TestInternalNotificationService_GetUnreadNotificationsCount(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create notifications
	service.Send(1, "Title 1", "Content 1")
	service.Send(1, "Title 2", "Content 2")

	count, err := service.GetUnreadNotificationsCount(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Mark one as read
	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)
	service.MarkAsRead(notification.ID)

	count, err = service.GetUnreadNotificationsCount(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestInternalNotificationService_MarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create notification
	service.Send(1, "Test Title", "Test Content")

	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)
	assert.False(t, notification.Read)

	// Mark as read
	err := service.MarkAsRead(notification.ID)
	assert.NoError(t, err)

	// Verify it's marked as read
	db.First(&notification, notification.ID)
	assert.True(t, notification.Read)
}

func TestInternalNotificationService_MarkAllAsRead(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create multiple notifications
	service.Send(1, "Title 1", "Content 1")
	service.Send(1, "Title 2", "Content 2")
	service.Send(2, "Title 3", "Content 3") // Different user

	// Mark all as read for user 1
	err := service.MarkAllAsRead(1)
	assert.NoError(t, err)

	// Verify user 1's notifications are read
	count, _ := service.GetUnreadNotificationsCount(1)
	assert.Equal(t, int64(0), count)

	// Verify user 2's notification is still unread
	count, _ = service.GetUnreadNotificationsCount(2)
	assert.Equal(t, int64(1), count)
}

func TestInternalNotificationService_GetPaginatedNotifications(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create multiple notifications
	for i := 0; i < 5; i++ {
		service.Send(1, "Title", "Content")
	}

	notifications, total, unread, read, err := service.GetPaginatedNotifications(1, 1, 2, "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Equal(t, int64(5), unread)
	assert.Equal(t, int64(0), read)
	assert.Equal(t, 2, len(notifications))
}

func TestInternalNotificationService_GetPaginatedNotifications_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create notifications
	service.Send(1, "Title 1", "Content 1")
	service.Send(1, "Title 2", "Content 2")

	// Mark one as read
	var notification InternalNotification
	db.Where("user_id = ? AND title = ?", 1, "Title 1").First(&notification)
	service.MarkAsRead(notification.ID)

	// Filter by read
	notifications, total, unread, read, err := service.GetPaginatedNotifications(1, 1, 10, "read", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, int64(1), unread)
	assert.Equal(t, int64(1), read)
	assert.Equal(t, 1, len(notifications))
	assert.True(t, notifications[0].Read)

	// Filter by unread
	notifications, _, _, _, err = service.GetPaginatedNotifications(1, 1, 10, "unread", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(notifications))
	assert.False(t, notifications[0].Read)
}

func TestInternalNotificationService_GetPaginatedNotifications_WithTitleKeyword(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Important Title", "Content")
	service.Send(1, "Other Title", "Content")

	notifications, _, _, _, err := service.GetPaginatedNotifications(1, 1, 10, "", "Important", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(notifications))
	assert.Equal(t, "Important Title", notifications[0].Title)
}

func TestInternalNotificationService_GetPaginatedNotifications_WithContentKeyword(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Title", "Important Content")
	service.Send(1, "Title", "Other Content")

	notifications, _, _, _, err := service.GetPaginatedNotifications(1, 1, 10, "", "", "Important", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(notifications))
	assert.Contains(t, notifications[0].Content, "Important")
}

func TestInternalNotificationService_GetPaginatedNotifications_WithTimeRange(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	service.Send(1, "Title", "Content")

	// Test with time range
	notifications, _, _, _, err := service.GetPaginatedNotifications(1, 1, 10, "", "", "", past, future)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(notifications), 1)

	// Test with only start time
	notifications, _, _, _, err = service.GetPaginatedNotifications(1, 1, 10, "", "", "", past, time.Time{})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(notifications), 1)

	// Test with only end time
	notifications, _, _, _, err = service.GetPaginatedNotifications(1, 1, 10, "", "", "", time.Time{}, future)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(notifications), 1)
}

func TestInternalNotificationService_GetPaginatedNotifications_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Query for non-existent user
	notifications, total, unread, read, err := service.GetPaginatedNotifications(999, 1, 10, "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, int64(0), unread)
	assert.Equal(t, int64(0), read)
	assert.Equal(t, 0, len(notifications))
}

func TestInternalNotificationService_GetPaginatedNotifications_Pagination(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create 5 notifications
	for i := 0; i < 5; i++ {
		service.Send(1, "Title", "Content")
	}

	// First page
	notifications, _, _, _, err := service.GetPaginatedNotifications(1, 1, 2, "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(notifications))

	// Second page
	notifications, _, _, _, err = service.GetPaginatedNotifications(1, 2, 2, "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(notifications))

	// Third page
	notifications, _, _, _, err = service.GetPaginatedNotifications(1, 3, 2, "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(notifications))
}

func TestInternalNotificationService_GetOne(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Test Title", "Test Content")

	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)

	result, err := service.GetOne(1, notification.ID)
	assert.NoError(t, err)
	assert.Equal(t, notification.ID, result.ID)
	assert.Equal(t, "Test Title", result.Title)
}

func TestInternalNotificationService_GetOne_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	_, err := service.GetOne(1, 999)
	assert.Error(t, err)
}

func TestInternalNotificationService_GetOne_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Title", "Content")

	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)

	// Try to get with wrong user ID
	_, err := service.GetOne(2, notification.ID)
	assert.Error(t, err)
}

func TestInternalNotificationService_Delete(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Title", "Content")

	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)

	err := service.Delete(1, notification.ID)
	assert.NoError(t, err)

	// Verify deleted
	err = db.First(&notification, notification.ID).Error
	assert.Error(t, err)
}

func TestInternalNotificationService_Delete_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Title", "Content")

	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)

	// Try to delete with wrong user ID
	err := service.Delete(2, notification.ID)
	assert.NoError(t, err) // Should not error, but should not delete

	// Verify not deleted
	err = db.First(&notification, notification.ID).Error
	assert.NoError(t, err)
}

func TestInternalNotificationService_BatchDelete(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	// Create notifications
	service.Send(1, "Title 1", "Content 1")
	service.Send(1, "Title 2", "Content 2")
	service.Send(1, "Title 3", "Content 3")

	var notifications []InternalNotification
	db.Where("user_id = ?", 1).Find(&notifications)

	ids := []uint{notifications[0].ID, notifications[1].ID}

	count, err := service.BatchDelete(1, ids)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify remaining count
	var remaining int64
	db.Model(&InternalNotification{}).Where("user_id = ?", 1).Count(&remaining)
	assert.Equal(t, int64(1), remaining)
}

func TestInternalNotificationService_BatchDelete_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	count, err := service.BatchDelete(1, []uint{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestInternalNotificationService_BatchDelete_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewInternalNotificationService(db)

	service.Send(1, "Title", "Content")

	var notification InternalNotification
	db.Where("user_id = ?", 1).First(&notification)

	// Try to delete with wrong user ID
	count, err := service.BatchDelete(2, []uint{notification.ID})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count) // Should not delete anything
}
