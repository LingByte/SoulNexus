package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/service"
	"github.com/gin-gonic/gin"
)

// registerNodePluginRoutes Node Plugin Module
func (h *Handlers) registerNodePluginRoutes(r *gin.RouterGroup) {
	pluginHandler := NewNodePluginHandler(h.db)

	plugins := r.Group("node-plugins")
	{
		plugins.GET("", pluginHandler.ListPlugins)
		plugins.GET("/:id", pluginHandler.GetPlugin)
	}

	pluginsAuth := r.Group("node-plugins")
	pluginsAuth.Use(models.AuthRequired)
	{
		pluginsAuth.POST("", pluginHandler.CreatePlugin)
		pluginsAuth.PUT("/:id", pluginHandler.UpdatePlugin)
		pluginsAuth.DELETE("/:id", pluginHandler.DeletePlugin)
		pluginsAuth.POST("/:id/publish", pluginHandler.PublishPlugin)
		pluginsAuth.POST("/:id/install", pluginHandler.InstallPlugin)
		pluginsAuth.GET("/installed", pluginHandler.ListInstalledPlugins)
	}
}

// registerWorkflowPluginRoutes Workflow Plugin Module
func (h *Handlers) registerWorkflowPluginRoutes(r *gin.RouterGroup) {
	pluginHandler := NewWorkflowPluginHandler(h.db)

	plugins := r.Group("workflow-plugins")
	{
		plugins.GET("", pluginHandler.ListWorkflowPlugins)
		plugins.GET("/:id", pluginHandler.GetWorkflowPlugin)
	}

	pluginsAuth := r.Group("workflow-plugins")
	pluginsAuth.Use(models.AuthRequired)
	{
		pluginsAuth.POST("/publish/:workflowId", pluginHandler.PublishWorkflowAsPlugin)

		pluginsAuth.GET("/workflow/:workflowId/published", pluginHandler.GetWorkflowPublishedPlugin)

		pluginsAuth.PUT("/:id", pluginHandler.UpdateWorkflowPlugin)
		pluginsAuth.DELETE("/:id", pluginHandler.DeleteWorkflowPlugin)
		pluginsAuth.POST("/:id/publish", pluginHandler.PublishWorkflowPlugin)
		pluginsAuth.POST("/:id/install", pluginHandler.InstallWorkflowPlugin)

		pluginsAuth.GET("/installed", pluginHandler.ListInstalledWorkflowPlugins)
		pluginsAuth.GET("/my-plugins", pluginHandler.GetUserWorkflowPlugins)
	}
}

// registerMCPRoutes MCP Server Management Module
func (h *Handlers) registerMCPRoutes(r *gin.RouterGroup) {
	mcpManager := service.NewMCPManager(h.db)
	mcpHandler := NewMCPHandler(mcpManager)

	mcp := r.Group("mcp")
	mcp.Use(models.AuthRequired)
	{
		mcp.GET("/servers", mcpHandler.ListMCPServers)
		mcp.GET("/servers/:id", mcpHandler.GetMCPServer)
		mcp.POST("/servers", mcpHandler.CreateMCPServer)
		mcp.PATCH("/servers/:id", mcpHandler.UpdateMCPServer)
		mcp.DELETE("/servers/:id", mcpHandler.DeleteMCPServer)
		mcp.POST("/servers/:id/enable", mcpHandler.EnableMCPServer)
		mcp.POST("/servers/:id/disable", mcpHandler.DisableMCPServer)

		mcp.GET("/servers/:id/tools", mcpHandler.GetMCPTools)
		mcp.POST("/servers/:id/call-tool", mcpHandler.CallMCPTool)

		mcp.GET("/servers/:id/logs", mcpHandler.GetMCPLogs)
	}
}

// registerMCPMarketplaceRoutes MCP Marketplace Module
func (h *Handlers) registerMCPMarketplaceRoutes(r *gin.RouterGroup) {
	marketplaceService := service.NewMCPMarketplaceService(h.db)
	marketplaceHandler := NewMCPMarketplaceHandler(marketplaceService)

	marketplace := r.Group("mcp/marketplace")
	{
		marketplace.GET("", marketplaceHandler.ListMarketplace)
		marketplace.GET("/:id", marketplaceHandler.GetMarketplaceItem)
		marketplace.GET("/categories", marketplaceHandler.GetCategories)
		marketplace.GET("/featured", marketplaceHandler.GetFeaturedMCPs)
		marketplace.GET("/trending", marketplaceHandler.GetTrendingMCPs)
		marketplace.GET("/search/tag/:tag", marketplaceHandler.SearchByTag)
		marketplace.GET("/:id/reviews", marketplaceHandler.GetMCPReviews)

		marketplace.POST("/:id/install", models.AuthRequired, marketplaceHandler.InstallMCP)
		marketplace.DELETE("/installations/:id", models.AuthRequired, marketplaceHandler.UninstallMCP)
		marketplace.GET("/my-installations", models.AuthRequired, marketplaceHandler.GetUserInstalledMCPs)
		marketplace.PATCH("/installations/:id/config", models.AuthRequired, marketplaceHandler.UpdateInstallationConfig)
		marketplace.POST("/:id/reviews", models.AuthRequired, marketplaceHandler.ReviewMCP)
	}
}
