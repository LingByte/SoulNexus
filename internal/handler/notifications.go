package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetUnReadNotificationCount get user unread notification count
func (h *Handlers) handleUnReadNotificationCount(c *gin.Context) {
	user := models.CurrentUser(c)

	users, err := models.GetUserByEmail(h.db, user.Email)
	if err != nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	unreadNotificationCount, err := models.NewInternalNotificationService(h.db).GetUnreadNotificationsCount(users.ID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", unreadNotificationCount)
}

// ListNotifications list user notifications
func (h *Handlers) handleListNotifications(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
	}
	page := c.DefaultQuery("page", "1")
	size := c.DefaultQuery("size", "10")

	var (
		pageInt  int
		sizeInt  int
		filterBy = c.Query("filter")  // read / unread
		title    = c.Query("title")   // Query by title
		content  = c.Query("content") // Query by content
		layout   = "2006-01-02T15:04:05Z07:00"
		startStr = c.Query("start_time") // Start time
		endStr   = c.Query("end_time")   // End time
		start    time.Time
		end      time.Time
	)

	_, _ = fmt.Sscanf(page, "%d", &pageInt)
	_, _ = fmt.Sscanf(size, "%d", &sizeInt)

	if startStr != "" {
		start, _ = time.Parse(layout, startStr)
	}
	if endStr != "" {
		end, _ = time.Parse(layout, endStr)
	}

	service := models.NewInternalNotificationService(h.db)
	notifications, total, totalUnread, totalRead, err := service.GetPaginatedNotifications(
		user.ID,
		pageInt,
		sizeInt,
		filterBy,
		title,
		content,
		start,
		end,
	)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{
		"list":        notifications,
		"total":       total,
		"totalUnread": totalUnread,
		"totalRead":   totalRead,
		"page":        pageInt,
		"size":        sizeInt,
	})
}

// AllNotifications mark all notifications as read
func (h *Handlers) handleAllNotifications(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
	}
	err := models.NewInternalNotificationService(h.db).MarkAllAsRead(user.ID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "already mark all notifications", nil)
}

// handleMarkNotificationAsRead marks specified notification as read
func (h *Handlers) handleMarkNotificationAsRead(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// Get notification ID from path parameter
	idStr := c.Param("id")
	var notificationID uint
	_, err := fmt.Sscanf(idStr, "%d", &notificationID)
	if err != nil {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}

	_, err = models.NewInternalNotificationService(h.db).GetOne(user.ID, notificationID)
	if err != nil {
		response.Fail(c, "You don't have permission to flag this message.", nil)
		return
	}

	// Call service layer to mark as read
	err = models.NewInternalNotificationService(h.db).MarkAsRead(notificationID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Notification marked as read", nil)
}

func (h *Handlers) handleDeleteNotification(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}
	var notificationID uint
	_, err := fmt.Sscanf(c.Param("id"), "%d", &notificationID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusBadRequest, err)
		return
	}
	err = models.NewInternalNotificationService(h.db).Delete(user.ID, notificationID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "Notification deleted", nil)
}

// handleBatchDeleteNotifications batch deletes notifications
func (h *Handlers) handleBatchDeleteNotifications(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	var request struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, "Invalid request format", err)
		return
	}

	if len(request.IDs) == 0 {
		response.Fail(c, "No notification IDs provided", nil)
		return
	}

	service := models.NewInternalNotificationService(h.db)
	deletedCount, err := service.BatchDelete(user.ID, request.IDs)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Notifications deleted successfully", gin.H{
		"deletedCount":   deletedCount,
		"totalRequested": len(request.IDs),
	})
}

// handleGetAllNotificationIds gets all notification IDs (for select all functionality)
func (h *Handlers) handleGetAllNotificationIds(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	var (
		filterBy = c.Query("filter")  // read / unread
		title    = c.Query("title")   // Query by title
		content  = c.Query("content") // Query by content
		layout   = "2006-01-02T15:04:05Z07:00"
		startStr = c.Query("start_time") // Start time
		endStr   = c.Query("end_time")   // End time
		start    time.Time
		end      time.Time
	)

	if startStr != "" {
		start, _ = time.Parse(layout, startStr)
	}
	if endStr != "" {
		end, _ = time.Parse(layout, endStr)
	}

	service := models.NewInternalNotificationService(h.db)
	ids, err := service.GetAllNotificationIds(user.ID, filterBy, title, content, start, end)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", gin.H{
		"ids": ids,
	})
}
