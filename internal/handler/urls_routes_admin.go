package handlers

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registerAdminManagementRoutes(r *gin.RouterGroup) {
	adminGuard := []gin.HandlerFunc{models.AuthRequired, h.requireAdmin}

	users := r.Group("users", adminGuard...)
	{
		users.GET("", h.handleAdminListUsers)
		users.GET("/:id", h.handleAdminGetUser)
		users.POST("", h.handleAdminCreateUser)
		users.PUT("/:id", h.handleAdminUpdateUser)
		users.DELETE("/:id", h.handleAdminDeleteUser)
	}

	configs := r.Group("configs", adminGuard...)
	{
		configs.GET("", h.handleAdminListConfigs)
		configs.GET("/:key", h.handleAdminGetConfig)
		configs.POST("", h.handleAdminCreateConfig)
		configs.PUT("/:key", h.handleAdminUpdateConfig)
		configs.DELETE("/:key", h.handleAdminDeleteConfig)
	}

	oauthClients := r.Group("oauth-clients", adminGuard...)
	{
		oauthClients.GET("", h.handleAdminListOAuthClients)
		oauthClients.GET("/:id", h.handleAdminGetOAuthClient)
		oauthClients.POST("", h.handleAdminCreateOAuthClient)
		oauthClients.PUT("/:id", h.handleAdminUpdateOAuthClient)
		oauthClients.DELETE("/:id", h.handleAdminDeleteOAuthClient)
	}

	assistants := r.Group("admin/assistants", adminGuard...)
	{
		assistants.GET("", h.handleAdminListAssistants)
		assistants.GET("/:id", h.handleAdminGetAssistant)
		assistants.PUT("/:id", h.handleAdminUpdateAssistant)
		assistants.DELETE("/:id", h.handleAdminDeleteAssistant)
	}

	security := r.Group("security", adminGuard...)
	{
		security.GET("/operation-logs", h.handleAdminListOperationLogs)
		security.GET("/operation-logs/:id", h.handleAdminGetOperationLog)
		security.GET("/account-locks", h.handleAdminListAccountLocks)
		security.POST("/account-locks/:id/unlock", h.handleAdminUnlockAccount)
	}

	groups := r.Group("admin/groups", adminGuard...)
	{
		groups.GET("", h.handleAdminListGroups)
		groups.GET("/:id", h.handleAdminGetGroup)
		groups.PUT("/:id", h.handleAdminUpdateGroup)
		groups.DELETE("/:id", h.handleAdminDeleteGroup)
	}

	credentials := r.Group("admin/credentials", adminGuard...)
	{
		credentials.GET("", h.handleAdminListCredentials)
		credentials.GET("/:id", h.handleAdminGetCredential)
		credentials.PATCH("/:id/status", h.handleAdminUpdateCredentialStatus)
		credentials.DELETE("/:id", h.handleAdminDeleteCredential)
	}

	jsTemplates := r.Group("admin/js-templates", adminGuard...)
	{
		jsTemplates.GET("", h.handleAdminListJSTemplates)
		jsTemplates.GET("/:id", h.handleAdminGetJSTemplate)
		jsTemplates.PUT("/:id", h.handleAdminUpdateJSTemplate)
		jsTemplates.DELETE("/:id", h.handleAdminDeleteJSTemplate)
	}

	bills := r.Group("admin/bills", adminGuard...)
	{
		bills.GET("", h.handleAdminListBills)
		bills.GET("/:id", h.handleAdminGetBill)
		bills.PUT("/:id", h.handleAdminUpdateBill)
		bills.DELETE("/:id", h.handleAdminDeleteBill)
	}

	voiceTraining := r.Group("admin/voice-training", adminGuard...)
	{
		voiceTraining.GET("/tasks", h.handleAdminListVoiceTrainingTasks)
		voiceTraining.GET("/tasks/:id", h.handleAdminGetVoiceTrainingTask)
		voiceTraining.DELETE("/tasks/:id", h.handleAdminDeleteVoiceTrainingTask)
	}

	mcpServers := r.Group("admin/mcp-servers", adminGuard...)
	{
		mcpServers.GET("", h.handleAdminListMCPServers)
		mcpServers.GET("/:id", h.handleAdminGetMCPServer)
		mcpServers.DELETE("/:id", h.handleAdminDeleteMCPServer)
	}

	mcpMarketplace := r.Group("admin/mcp-marketplace", adminGuard...)
	{
		mcpMarketplace.GET("", h.handleAdminListMCPMarketplaceItems)
		mcpMarketplace.GET("/:id", h.handleAdminGetMCPMarketplaceItem)
		mcpMarketplace.DELETE("/:id", h.handleAdminDeleteMCPMarketplaceItem)
	}

	workflows := r.Group("admin/workflows", adminGuard...)
	{
		workflows.GET("", h.handleAdminListWorkflowDefinitions)
		workflows.GET("/:id", h.handleAdminGetWorkflowDefinition)
		workflows.DELETE("/:id", h.handleAdminDeleteWorkflowDefinition)
	}

	workflowPlugins := r.Group("admin/workflow-plugins", adminGuard...)
	{
		workflowPlugins.GET("", h.handleAdminListWorkflowPlugins)
		workflowPlugins.GET("/:id", h.handleAdminGetWorkflowPlugin)
		workflowPlugins.DELETE("/:id", h.handleAdminDeleteWorkflowPlugin)
	}

	nodePlugins := r.Group("admin/node-plugins", adminGuard...)
	{
		nodePlugins.GET("", h.handleAdminListNodePlugins)
		nodePlugins.GET("/:id", h.handleAdminGetNodePlugin)
		nodePlugins.DELETE("/:id", h.handleAdminDeleteNodePlugin)
	}

	alerts := r.Group("admin/alerts", adminGuard...)
	{
		alerts.GET("", h.handleAdminListAlerts)
		alerts.GET("/:id", h.handleAdminGetAlert)
		alerts.DELETE("/:id", h.handleAdminDeleteAlert)
	}

	notificationCenter := r.Group("admin/notifications", adminGuard...)
	{
		notificationCenter.GET("", h.handleAdminListInternalNotifications)
		notificationCenter.GET("/:id", h.handleAdminGetInternalNotification)
		notificationCenter.DELETE("/:id", h.handleAdminDeleteInternalNotification)
	}

	knowledgeBase := r.Group("admin/knowledge-bases", adminGuard...)
	{
		knowledgeBase.GET("", h.handleAdminListKnowledgeBases)
		knowledgeBase.GET("/:id", h.handleAdminGetKnowledgeBase)
		knowledgeBase.DELETE("/:id", h.handleAdminDeleteKnowledgeBase)
	}

	devices := r.Group("admin/devices", adminGuard...)
	{
		devices.GET("", h.handleAdminListDevices)
		devices.GET("/:id", h.handleAdminGetDevice)
		devices.DELETE("/:id", h.handleAdminDeleteDevice)
	}
}
