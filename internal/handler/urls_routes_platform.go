package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gin-gonic/gin"
)

// registerNotificationRoutes Notification Module
func (h *Handlers) registerNotificationRoutes(r *gin.RouterGroup) {
	notificationGroup := r.Group("notification")
	{
		notificationGroup.GET("unread-count", models.AuthRequired, h.handleUnReadNotificationCount)

		notificationGroup.GET("", models.AuthRequired, h.handleListNotifications)

		notificationGroup.POST("readAll", models.AuthRequired, h.handleAllNotifications)

		notificationGroup.PUT("/read/:id", models.AuthRequired, h.handleMarkNotificationAsRead)

		notificationGroup.DELETE("/:id", models.AuthRequired, h.handleDeleteNotification)

		notificationGroup.POST("/batch-delete", models.AuthRequired, h.handleBatchDeleteNotifications)

		notificationGroup.GET("/all-ids", models.AuthRequired, h.handleGetAllNotificationIds)
	}
}

// registerSystemRoutes System Module
func (h *Handlers) registerSystemRoutes(r *gin.RouterGroup) {
	system := r.Group("system")
	{
		system.POST("/rate-limiter/config", h.UpdateRateLimiterConfig)

		system.GET("/health", h.HealthCheck)
		system.GET("/status", h.SystemStatus)
		system.GET("/dashboard/metrics", models.AuthRequired, h.DashboardMetrics)

		system.GET("/init", h.SystemInit)

		system.POST("/voice-clone/config", models.AuthRequired, h.SaveVoiceCloneConfig)

		system.POST("/voiceprint/config", models.AuthRequired, h.SaveVoiceprintConfig)

		system.POST("/upload/audio", h.UploadAudio)

		system.GET("/search/status", h.GetSearchStatus)
		system.PUT("/search/config", models.AuthRequired, h.UpdateSearchConfig)
		system.POST("/search/enable", models.AuthRequired, h.EnableSearch)
		system.POST("/search/disable", models.AuthRequired, h.DisableSearch)
	}

	voiceprint := r.Group("/voiceprint")
	{
		voiceprint.GET("", models.AuthRequired, h.GetVoiceprints)
		voiceprint.POST("", models.AuthRequired, h.CreateVoiceprint)
		voiceprint.POST("/register", models.AuthRequired, h.RegisterVoiceprint)
		voiceprint.POST("/identify", models.AuthRequired, h.IdentifyVoiceprint)
		voiceprint.POST("/verify", models.AuthRequired, h.VerifyVoiceprint)
		voiceprint.PUT("/:id", models.AuthRequired, h.UpdateVoiceprint)
		voiceprint.DELETE("/:id", models.AuthRequired, h.DeleteVoiceprint)
	}
}

// registerOTARoutes OTA Module
func (h *Handlers) registerOTARoutes(r *gin.RouterGroup) {
	ota := r.Group("ota")
	{
		ota.POST("/", h.HandleOTACheck)

		ota.POST("/activate", h.HandleOTAActivate)

		ota.GET("/", h.HandleOTAGet)
	}
}

// registerDeviceRoutes Device Module (completely consistent with xiaozhi-esp32)
func (h *Handlers) registerDeviceRoutes(r *gin.RouterGroup) {
	device := r.Group("device")

	device.GET("/config/:deviceId", h.GetDeviceConfig)

	device.Use(models.AuthRequired)
	{
		device.POST("/bind/:agentId/:deviceCode", h.BindDevice)

		device.GET("/bind/:agentId", h.GetUserDevices)

		device.POST("/unbind", h.UnbindDevice)

		device.PUT("/update/:id", h.UpdateDeviceInfo)

		device.POST("/manual-add", h.ManualAddDevice)

		device.GET("/:deviceId", h.GetDeviceDetail)
		device.GET("/:deviceId/error-logs", h.GetDeviceErrorLogs)
		device.POST("/error-logs/:errorId/resolve", h.ResolveDeviceError)
		device.GET("/call-recordings", h.GetCallRecordings)
		device.GET("/call-recordings/:id", h.GetCallRecordingDetail)

		device.POST("/call-recordings/:id/analyze", h.AnalyzeCallRecording)
		device.POST("/call-recordings/batch-analyze", h.BatchAnalyzeCallRecordings)
		device.GET("/call-recordings/:id/analysis", h.GetCallRecordingAnalysis)

		device.POST("/status", h.UpdateDeviceStatus)
		device.POST("/error", h.LogDeviceError)

		device.GET("/recordings/*filepath", h.ServeRecordingFile)
	}
}

// registerEmailLogRoutes Email Log Module
func (h *Handlers) registerEmailLogRoutes(r *gin.RouterGroup) {
	emailLog := r.Group("email-logs")
	{
		emailLog.GET("", models.AuthRequired, h.handleGetEmailLogs)
		emailLog.GET("/:id", models.AuthRequired, h.handleGetEmailLogDetail)
		emailLog.GET("/stats/summary", models.AuthRequired, h.handleGetEmailStats)
	}
}

// registerSendCloudWebhookRoutes SendCloud Webhook Module
func (h *Handlers) registerSendCloudWebhookRoutes(r *gin.RouterGroup) {
	webhook := r.Group("webhooks/sendcloud")
	{
		webhook.POST("", h.handleSendCloudWebhook)
		webhook.POST("/batch", h.handleSendCloudWebhookBatch)
	}
}
