package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"log"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/search"
	"github.com/LingByte/SoulNexus/pkg/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handlers struct {
	db                *gorm.DB
	wsHub             *websocket.Hub
	searchHandler     *search.SearchHandlers
	ipLocationService *utils.IPLocationService
}

// GetSearchHandler gets the search handler (for scheduled tasks)
func (h *Handlers) GetSearchHandler() *search.SearchHandlers {
	return h.searchHandler
}

func NewHandlers(db *gorm.DB) *Handlers {
	wsConfig := websocket.LoadConfigFromEnv()
	wsHub := websocket.NewHub(wsConfig)
	var searchHandler *search.SearchHandlers

	// Read search configuration from config table
	searchEnabled := utils.GetBoolValue(db, constants.KEY_SEARCH_ENABLED)
	// If not configured in config table, use environment variables
	if !searchEnabled && config.GlobalConfig != nil {
		searchEnabled = config.GlobalConfig.Features.SearchEnabled
	}

	if searchEnabled {
		searchPath := utils.GetValue(db, constants.KEY_SEARCH_PATH)
		if searchPath == "" && config.GlobalConfig != nil {
			searchPath = config.GlobalConfig.Features.SearchPath
		}
		if searchPath == "" {
			searchPath = "./search"
		}

		batchSize := utils.GetIntValue(db, constants.KEY_SEARCH_BATCH_SIZE, 100)
		if batchSize == 0 && config.GlobalConfig != nil {
			batchSize = config.GlobalConfig.Features.SearchBatchSize
		}
		if batchSize == 0 {
			batchSize = 100
		}

		engine, err := search.New(
			search.Config{
				IndexPath:    searchPath,
				QueryTimeout: 5 * time.Second,
				BatchSize:    batchSize,
			},
			search.BuildIndexMapping(""),
		)
		if err != nil {
			log.Printf("Failed to initialize search engine: %v", err)
			// Even if initialization fails, create an empty handler for route registration
			searchHandler = search.NewSearchHandlers(nil)
		} else {
			searchHandler = search.NewSearchHandlers(engine)
		}
		// Set database connection for configuration checking
		if searchHandler != nil {
			searchHandler.SetDB(db)
		}
	} else {
		// Even if search is not enabled, create an empty handler for route registration
		searchHandler = search.NewSearchHandlers(nil)
		if searchHandler != nil {
			searchHandler.SetDB(db)
		}
	}

	// Initialize IP geolocation service
	ipLocationService := utils.NewIPLocationService(logger.Lg)

	return &Handlers{
		db:                db,
		wsHub:             wsHub,
		searchHandler:     searchHandler,
		ipLocationService: ipLocationService,
	}
}

// NewUserServiceHandlers returns handlers for the standalone user (auth) service binary.
// It omits WebSocket hub and search engine wiring that auth routes do not use.
func NewUserServiceHandlers(db *gorm.DB) *Handlers {
	return &Handlers{
		db:                db,
		wsHub:             nil,
		searchHandler:     nil,
		ipLocationService: utils.NewIPLocationService(logger.Lg),
	}
}

// RegisterUserServiceRoutes registers auth routes and a minimal system health check under the API prefix.
func (h *Handlers) RegisterUserServiceRoutes(engine *gin.Engine) {
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r := engine.Group(apiPrefix)
	r.Use(middleware.InjectDB(h.db))
	middleware.ApplyGlobalMiddlewares(r)

	sys := r.Group("system")
	{
		sys.GET("/health", h.HealthCheck)
	}

	h.registerAuthRoutes(r)

	// Rendered login page for browser access in user-service.
	engine.GET("/login", h.RenderSigninPage)
}

func (h *Handlers) Register(engine *gin.Engine) {

	r := engine.Group(config.GlobalConfig.Server.APIPrefix)

	// Register Global Singleton DB
	r.Use(middleware.InjectDB(h.db))

	// Apply global middlewares (rate limiting, timeout, circuit breaker, operation log)
	middleware.ApplyGlobalMiddlewares(r)

	// Register routes regardless of whether search is enabled, check in handler methods
	// If handler is nil, try to initialize
	if h.searchHandler == nil {
		searchPath := utils.GetValue(h.db, constants.KEY_SEARCH_PATH)
		if searchPath == "" && config.GlobalConfig != nil {
			searchPath = config.GlobalConfig.Features.SearchPath
		}
		if searchPath == "" {
			searchPath = "./search"
		}

		batchSize := utils.GetIntValue(h.db, constants.KEY_SEARCH_BATCH_SIZE, 100)
		if batchSize == 0 && config.GlobalConfig != nil {
			batchSize = config.GlobalConfig.Features.SearchBatchSize
		}
		if batchSize == 0 {
			batchSize = 100
		}

		engine, err := search.New(
			search.Config{
				IndexPath:    searchPath,
				QueryTimeout: 5 * time.Second,
				BatchSize:    batchSize,
			},
			search.BuildIndexMapping(""),
		)
		if err != nil {
			logger.Warn("Failed to initialize search engine in Register", zap.Error(err))
			// Even if initialization fails, create an empty handler for route registration
			h.searchHandler = search.NewSearchHandlers(nil)
		} else {
			h.searchHandler = search.NewSearchHandlers(engine)
		}
	}

	// Register routes regardless of whether search is enabled, check in handler methods
	if h.searchHandler == nil {
		// If handler is still nil, create an empty one for route registration
		logger.Info("Search handler is nil, creating empty handler for route registration")
		h.searchHandler = search.NewSearchHandlers(nil)
	}

	// Set database connection for configuration checking
	if h.searchHandler != nil {
		h.searchHandler.SetDB(h.db)
		logger.Info("Registering search routes")
		h.searchHandler.RegisterSearchRoutes(r)
		logger.Info("Search routes registered successfully")
	} else {
		logger.Warn("Search handler is still nil after initialization, routes not registered")
	}
	// Register System Module Routes
	h.registerSystemRoutes(r)
	// Register OTA routes
	h.registerOTARoutes(r)
	// Register Device routes
	h.registerDeviceRoutes(r)
	h.registerNotificationRoutes(r)
	h.registerEmailLogRoutes(r)
	h.registerSendCloudWebhookRoutes(r)
	h.registerGroupRoutes(r)
	h.registerQuotaRoutes(r)
	h.registerAlertRoutes(r)
	h.registerWebSocketRoutes(r)
	h.registerAssistantRoutes(r)
	h.registerChatRoutes(r)
	h.registerCredentialsRoutes(r)
	h.registerKnowledgeBaseRoutes(r)
	h.registerXunfeiTTSRoutes(r)
	h.registerVolcengineTTSRoutes(r)
	h.registerVoiceTrainingRoutes(r)
	h.registerJSTemplateRoutes(r)
	h.registerBillingRoutes(r)
	h.registerMiddlewareRoutes(r)
	h.registerAdminManagementRoutes(r)
	h.registerWorkflowRoutes(r)
	h.registerWorkflowPluginRoutes(r) // Add workflow plugin routes
	h.registerNodePluginRoutes(r)     // Add node plugin routes
	h.registerMCPRoutes(r)            // Add MCP routes
	h.registerMCPMarketplaceRoutes(r) // Add MCP marketplace routes
	h.registerOpenAPIRoutes(r)        // Open API (apiKey + apiSecret auth)
	h.RegisterPublicWorkflowRoutes(r)
}
