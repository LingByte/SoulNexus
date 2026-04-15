package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gin-gonic/gin"
)

// registerAssistantRoutes Assistant Module
func (h *Handlers) registerAssistantRoutes(r *gin.RouterGroup) {
	assistant := r.Group("assistant")
	{
		assistant.POST("add", models.AuthRequired, h.CreateAssistant)

		assistant.GET("", models.AuthRequired, h.ListAssistants)

		assistant.GET("/:id", models.AuthRequired, h.GetAssistant)

		assistant.GET("/:id/graph", models.AuthRequired, h.GetAssistantGraphData)

		assistant.PUT("/:id", models.AuthRequired, h.UpdateAssistant)

		assistant.DELETE("/:id", models.AuthRequired, h.DeleteAssistant)

		assistant.GET("/lingecho/client/:id/loader.js", h.ServeVoiceSculptorLoaderJS)

	}
}

// registerJSTemplateRoutes JSTemplate Module
func (h *Handlers) registerJSTemplateRoutes(r *gin.RouterGroup) {
	jsTemplate := r.Group("js-templates")
	jsTemplate.Use(models.AuthRequired)
	{
		jsTemplate.POST("", h.CreateJSTemplate)
		jsTemplate.GET("/:id", h.GetJSTemplate)
		jsTemplate.GET("/name/:name", h.GetJSTemplateByName)
		jsTemplate.GET("", h.ListJSTemplates)
		jsTemplate.PUT("/:id", h.UpdateJSTemplate)
		jsTemplate.DELETE("/:id", h.DeleteJSTemplate)
		jsTemplate.GET("/default", h.ListDefaultJSTemplates)
		jsTemplate.GET("/custom", h.ListCustomJSTemplates)
		jsTemplate.GET("/search", h.SearchJSTemplates)

		jsTemplate.GET("/:id/versions", h.ListJSTemplateVersions)
		jsTemplate.GET("/:id/versions/:versionId", h.GetJSTemplateVersion)
		jsTemplate.POST("/:id/versions/:versionId/rollback", h.RollbackJSTemplateVersion)
		jsTemplate.POST("/:id/versions/:versionId/publish", h.PublishJSTemplateVersion)
	}

	webhook := r.Group("js-templates/webhook")
	{
		webhook.POST("/:jsSourceId", h.TriggerJSTemplateWebhook)
	}
}

// registerChatRoutes Chat Module
func (h *Handlers) registerChatRoutes(r *gin.RouterGroup) {
	chat := r.Group("chat")

	chat.GET("call", h.handleConnection)

	chat.Use(models.AuthApiRequired)
	{
		chat.GET("chat-session-log", h.getChatSessionLog)

		chat.GET("chat-session-log/:id", h.getChatSessionLogDetail)

		chat.GET("chat-session-log/by-session/:sessionId", h.getChatSessionLogsBySession)

		chat.GET("chat-session-log/by-assistant/:assistantId", h.getChatSessionLogByAssistant)
	}
}

// registerCredentialsRoutes Credentials Module
func (h *Handlers) registerCredentialsRoutes(r *gin.RouterGroup) {
	credential := r.Group("credentials")
	{
		credential.POST("/", models.AuthRequired, h.handleCreateCredential)

		credential.GET("/", models.AuthRequired, h.handleGetCredential)

		credential.POST("/by-key", h.handleGetCredentialByKey)

		credential.DELETE("/:id", models.AuthRequired, h.handleDeleteCredential)
	}
}

// registerKnowledgeBaseRoutes Knowledge Base Module
func (h *Handlers) registerKnowledgeBaseRoutes(r *gin.RouterGroup) {
	kb := r.Group("knowledge-base")
	{
		kb.GET("", models.AuthRequired, h.ListKnowledgeBases)
		kb.GET("/supported-document-types", models.AuthRequired, h.ListKnowledgeDocumentFormats)
		kb.POST("", models.AuthRequired, h.CreateKnowledgeBase)
		kb.GET("/:id", models.AuthRequired, h.GetKnowledgeBase)
		kb.PUT("/:id", models.AuthRequired, h.UpdateKnowledgeBase)
		kb.DELETE("/:id", models.AuthRequired, h.DeleteKnowledgeBase)
		kb.GET("/:id/documents", models.AuthRequired, h.ListKnowledgeDocuments)
		kb.POST("/:id/documents/upload", models.AuthRequired, h.UploadKnowledgeDocument)
		kb.DELETE("/:id/documents", models.AuthRequired, h.DeleteKnowledgeDocument)
		kb.POST("/:id/recall-test", models.AuthRequired, h.RecallTestKnowledgeBase)
	}
}
