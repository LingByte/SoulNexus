package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/websocket"
	"github.com/gin-gonic/gin"
)

// registerWebSocketRoutes registers WebSocket routes
func (h *Handlers) registerWebSocketRoutes(r *gin.RouterGroup) {
	wsHandler := websocket.NewHandler(h.wsHub)

	r.GET("/ws", models.AuthRequired, wsHandler.HandleWebSocket)

	r.GET("/voice/websocket", h.HandleWebSocketVoice)

	wsGroup := r.Group("/ws")
	wsGroup.Use(models.AuthRequired)
	{
		wsGroup.GET("/stats", wsHandler.GetStats)
		wsGroup.GET("/health", wsHandler.HealthCheck)
		wsGroup.GET("/user/:user_id", wsHandler.GetUserStats)
		wsGroup.GET("/group/:group", wsHandler.GetGroupStats)
		wsGroup.POST("/message", wsHandler.SendMessage)
		wsGroup.POST("/broadcast", wsHandler.BroadcastMessage)
		wsGroup.DELETE("/user/:user_id", wsHandler.DisconnectUser)
		wsGroup.DELETE("/group/:group", wsHandler.DisconnectGroup)
	}
}

// registerXunfeiTTSRoutes 注册讯飞TTS路由
func (h *Handlers) registerXunfeiTTSRoutes(r *gin.RouterGroup) {
	xunfei := r.Group("/xunfei")
	xunfei.Use(models.AuthRequired)
	{
		xunfei.POST("/synthesize", h.XunfeiSynthesize)

		xunfei.POST("/task/create", h.XunfeiCreateTask)
		xunfei.POST("/task/submit-audio", h.XunfeiSubmitAudio)
		xunfei.POST("/task/query", h.XunfeiQueryTask)

		xunfei.GET("/training-texts", h.XunfeiGetTrainingTexts)
	}
}

// registerVolcengineTTSRoutes 注册火山引擎TTS路由
func (h *Handlers) registerVolcengineTTSRoutes(r *gin.RouterGroup) {
	volcengine := r.Group("/volcengine")
	volcengine.Use(models.AuthRequired)
	{
		volcengine.POST("/synthesize", h.VolcengineSynthesize)

		volcengine.POST("/task/submit-audio", h.VolcengineSubmitAudio)

		volcengine.POST("/task/query", h.VolcengineQueryTask)
	}
}

// registerVoiceTrainingRoutes 注册音色训练路由
func (h *Handlers) registerVoiceTrainingRoutes(r *gin.RouterGroup) {
	voice := r.Group("/voice")

	voice.GET("/lingecho/v1/", h.HandleHardwareWebSocketVoice)
	voice.POST("/simple_text_chat", h.SimpleTextChat)

	voice.Use(models.AuthRequired)
	{
		voice.POST("/training/create", h.CreateTrainingTask)
		voice.POST("/training/submit-audio", h.SubmitAudio)
		voice.POST("/training/query", h.QueryTaskStatus)

		voice.GET("/clones", h.GetUserVoiceClones)
		voice.GET("/clones/:id", h.GetVoiceClone)
		voice.POST("/clones/update", h.UpdateVoiceClone)
		voice.POST("/clones/delete", h.DeleteVoiceClone)

		voice.POST("/synthesize", h.SynthesizeWithVoice)

		voice.GET("/synthesis/history", h.GetSynthesisHistory)
		voice.POST("/synthesis/delete", h.DeleteSynthesisRecord)

		voice.GET("/training-texts", h.GetTrainingTexts)

		voice.POST("/oneshot_text", h.OneShotText)

		voice.POST("/plain_text", h.PlainText)

		voice.GET("/audio_status", h.GetAudioStatus)

		voice.GET("/options", h.GetVoiceOptions)
		voice.GET("/language-options", h.GetLanguageOptions)
	}
}

// registerBillingRoutes 注册计费路由
func (h *Handlers) registerBillingRoutes(r *gin.RouterGroup) {
	billing := r.Group("billing")
	billing.Use(models.AuthRequired)
	{
		billing.GET("/statistics", h.GetUsageStatistics)
		billing.GET("/daily-usage", h.GetDailyUsageData)

		billing.GET("/usage-records", h.GetUsageRecords)
		billing.GET("/usage-records/export", h.ExportUsageRecords)

		billing.POST("/bills", h.GenerateBill)
		billing.GET("/bills", h.GetBills)
		billing.GET("/bills/:id", h.GetBill)
		billing.PUT("/bills/:id", h.UpdateBill)
		billing.DELETE("/bills/:id", h.DeleteBill)
		billing.POST("/bills/:id/archive", h.ArchiveBill)
		billing.PUT("/bills/:id/notes", h.UpdateBillNotes)
		billing.GET("/bills/:id/export", h.ExportBill)
	}
}
