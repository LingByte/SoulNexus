package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
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
	SearchHandler     *search.SearchHandlers
	ipLocationService *utils.IPLocationService
}

func NewHandlers(db *gorm.DB) *Handlers {
	wsConfig := websocket.LoadConfigFromEnv()
	wsHub := websocket.NewHub(wsConfig)
	var searchHandler *search.SearchHandlers

	// Read search configuration from config table
	searchEnabled := config.GlobalConfig.Features.SearchEnabled
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
			logger.Warn("Failed to initialize search engine", zap.Error(err))
			searchHandler = search.NewSearchHandlers(nil)
		} else {
			searchHandler = search.NewSearchHandlers(engine)
		}
		if searchHandler != nil {
			searchHandler.SetDB(db)
		}
	} else {
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
		SearchHandler:     searchHandler,
		ipLocationService: ipLocationService,
	}
}

func (h *Handlers) Register(engine *gin.Engine) {
	r := engine.Group(config.GlobalConfig.Server.APIPrefix)

	// Register Global Singleton DB
	r.Use(middleware.InjectDB(h.db))
	r.Use(middleware.MutatingRequestTrustedOrigin())

	// Apply global middlewares (rate limiting, timeout, circuit breaker, operation log)
	middleware.ApplyGlobalMiddlewares(r)

	// cmd/voice VOICE_DIALOG_WS：兼容业务对话面 ws://host:port/ws/call（无 /api 前缀）
	wsCall := engine.Group("")
	wsCall.Use(middleware.InjectDB(h.db))
	wsCall.Use(middleware.MutatingRequestTrustedOrigin())
	wsCall.GET("/ws/call", h.handleConnection)

	// Register routes regardless of whether search is enabled, check in handler methods
	// If handler is nil, try to initialize
	if h.SearchHandler == nil {
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
			h.SearchHandler = search.NewSearchHandlers(nil)
		} else {
			h.SearchHandler = search.NewSearchHandlers(engine)
		}
	}

	// Register routes regardless of whether search is enabled, check in handler methods
	if h.SearchHandler == nil {
		// If handler is still nil, create an empty one for route registration
		logger.Info("Search handler is nil, creating empty handler for route registration")
		h.SearchHandler = search.NewSearchHandlers(nil)
	}

	// Set database connection for configuration checking
	if h.SearchHandler != nil {
		h.SearchHandler.SetDB(h.db)
		logger.Info("Registering search routes")
		h.SearchHandler.RegisterSearchRoutes(r)
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
	h.registerSendCloudWebhookRoutes(r)
	h.registerGroupRoutes(r)
	h.registerAnnouncementRoutes(r)
	h.registerWebSocketRoutes(r)
	h.registerAgentRoutes(r)
	h.registerKnowledgeRoutes(r)
	h.registerChatRoutes(r)
	h.registerCredentialsRoutes(r)
	h.registerVolcengineTTSRoutes(r)
	h.registerVoiceTrainingRoutes(r)
	h.registerJSTemplateRoutes(r)
	h.registerBillingRoutes(r)
	h.registerMiddlewareRoutes(r)
	h.registerAdminManagementRoutes(r)
	h.registerWorkflowRoutes(r)
	h.registerWorkflowPluginRoutes(r) // Add workflow plugin routes
	h.registerNodePluginRoutes(r)     // Add node plugin routes
	h.registerPresetRoutes(r)         // Preset template system
	h.registerMarketRoutes(r)         // Character card market
	h.registerOpenAPIRoutes(r)        // Open API (apiKey + apiSecret auth)
	h.registerLLMRelayRoutes(engine)  // OpenAI/Anthropic 兼容对外网关 /v1/*
	h.RegisterPublicWorkflowRoutes(r)

	h.registerAuthIntegratedRoutes(engine)
}

// registerLLMRelayRoutes 注册对外 OpenAI/Anthropic 兼容 API 网关。
// 鉴权走 UserCredential.APIKey；不在 /api 前缀下，直接 /v1/* 与官方 SDK 对齐。
func (h *Handlers) registerLLMRelayRoutes(engine *gin.Engine) {
	v1 := engine.Group("/v1")
	v1.Use(middleware.InjectDB(h.db))
	{
		// LLM
		v1.GET("/models", h.handleRelayOpenAIModelsList)
		v1.POST("/chat/completions", h.handleRelayOpenAIChat)
		v1.POST("/messages", h.handleRelayAnthropicMessages)

		// Speech (OpenAI 兼容)
		v1.POST("/audio/transcriptions", h.handleRelayOpenAIAudioTranscriptions)
		v1.POST("/audio/speech", h.handleRelayOpenAIAudioSpeech)

		// Speech (LingVoice 兼容 + 流式)
		v1.POST("/speech/asr/transcribe", h.handleRelaySpeechASRTranscribe)
		v1.POST("/speech/tts/synthesize", h.handleRelaySpeechTTSSynthesize)
		v1.GET("/speech/asr/stream", h.handleRelaySpeechASRStream)
		v1.GET("/speech/tts/stream", h.handleRelaySpeechTTSStream)
	}
}

func (h *Handlers) registerAnnouncementRoutes(r *gin.RouterGroup) {
	ann := r.Group("announcements")
	{
		ann.GET("", h.handleListAnnouncements)
		ann.GET("/:id", h.handleGetAnnouncement)
	}
}

// registerNodePluginRoutes Node Plugin Module
func (h *Handlers) registerNodePluginRoutes(r *gin.RouterGroup) {
	pluginHandler := NewNodePluginHandler(h.db)

	plugins := r.Group("node-plugins")
	{
		plugins.GET("", pluginHandler.ListPlugins)
		plugins.GET("/:id", pluginHandler.GetPlugin)
	}

	pluginsAuth := r.Group("node-plugins")
	pluginsAuth.Use(auth.AuthRequired)
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
	pluginsAuth.Use(auth.AuthRequired)
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

// registerWebSocketRoutes registers WebSocket routes
func (h *Handlers) registerWebSocketRoutes(r *gin.RouterGroup) {
	wsHandler := websocket.NewHandler(h.wsHub)

	r.GET("/ws", auth.AuthRequired, wsHandler.HandleWebSocket)

	wsGroup := r.Group("/ws")
	wsGroup.Use(auth.AuthRequired)
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

// registerVolcengineTTSRoutes 注册火山引擎TTS路由
func (h *Handlers) registerVolcengineTTSRoutes(r *gin.RouterGroup) {
	volcengine := r.Group("/volcengine")
	volcengine.Use(auth.AuthRequired)
	{
		volcengine.POST("/synthesize", h.VolcengineSynthesize)

		volcengine.POST("/task/submit-audio", h.VolcengineSubmitAudio)

		volcengine.POST("/task/query", h.VolcengineQueryTask)
	}
}

// registerVoiceTrainingRoutes 注册音色训练路由
func (h *Handlers) registerVoiceTrainingRoutes(r *gin.RouterGroup) {
	voice := r.Group("/voice")

	// Device-Id → dialog payload for cmd/voice hardware mount (X-Lingecho-Voice-Secret when LINGECHO_HARDWARE_BINDING_SECRET is set).
	voice.GET("/lingecho/binding", h.HandleSoulnexusHardwareBinding)
	voice.POST("/simple_text_chat", h.SimpleTextChat)

	voice.Use(auth.AuthRequired)
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

		voice.POST("/oneshot_text", h.OneShotText)

		voice.POST("/plain_text", h.PlainText)
		voice.POST("/parse_attachment", h.ParseAttachmentContent)

		voice.GET("/audio_status", h.GetAudioStatus)

		voice.GET("/options", h.GetVoiceOptions)
		voice.GET("/language-options", h.GetLanguageOptions)
	}
}

// registerBillingRoutes 注册计费路由
func (h *Handlers) registerBillingRoutes(r *gin.RouterGroup) {
	billing := r.Group("billing")
	billing.Use(auth.AuthRequired)
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

// registerGroupRoutes Group Module
func (h *Handlers) registerGroupRoutes(r *gin.RouterGroup) {
	group := r.Group("group")
	group.Use(auth.AuthRequired)
	{
		group.POST("", h.CreateGroup)
		group.GET("", h.ListGroups)

		group.GET("/search-users", h.SearchUsers)

		group.GET("/invitations", h.ListInvitations)
		group.POST("/invitations/:id/accept", h.AcceptInvitation)
		group.POST("/invitations/:id/reject", h.RejectInvitation)

		group.GET("/:id/statistics", h.GetGroupStatistics)

		group.POST("/:id/leave", h.LeaveGroup)
		group.DELETE("/:id/members/:memberId", h.RemoveMember)
		group.PUT("/:id/members/:memberId/role", h.UpdateMemberRole)

		group.POST("/:id/invite", h.InviteUser)

		group.GET("/:id/resources", h.GetGroupSharedResources)

		group.POST("/:id/avatar", h.UploadGroupAvatar)

		group.POST("/:id/archive", h.ArchiveGroup)
		group.POST("/:id/restore", h.RestoreGroup)
		group.POST("/:id/clone", h.CloneGroup)
		group.GET("/:id/export", h.ExportGroup)
		group.GET("/:id/activity-logs", h.GetGroupActivityLogs)

		group.GET("/:id", h.GetGroup)
		group.PUT("/:id", h.UpdateGroup)
		group.DELETE("/:id", h.DeleteGroup)
	}
}

// registerAgentRoutes Agent module (REST prefix /agents)
func (h *Handlers) registerAgentRoutes(r *gin.RouterGroup) {
	agents := r.Group("agents")
	{
		agents.POST("add", auth.AuthRequired, h.CreateAgent)

		agents.GET("", auth.AuthRequired, h.ListAgents)

		agents.GET("/:id", auth.AuthRequired, h.GetAgent)

		agents.PUT("/:id", auth.AuthRequired, h.UpdateAgent)

		agents.DELETE("/:id", auth.AuthRequired, h.DeleteAgent)

		agents.GET("/lingecho/client/:id/loader.js", h.ServeVoiceSculptorLoaderJS)

		// Character Card endpoints
		agents.POST("/import", auth.AuthRequired, h.handleImportCharacterCard)
		agents.POST("/:id/export", auth.AuthRequired, h.handleExportCharacterCard)
		agents.GET("/:id/export", auth.AuthRequired, h.handleExportCharacterCard)
		agents.POST("/:id/avatar", auth.AuthRequired, h.handleUploadAgentAvatar)

	}
}

// registerMarketRoutes Character Card Market (public)
func (h *Handlers) registerMarketRoutes(r *gin.RouterGroup) {
	market := r.Group("market/agents")
	{
		market.GET("", h.handleMarketListAgents)
		market.GET("/:id", h.handleMarketGetAgent)
		market.POST("/:id/fork", auth.AuthRequired, h.handleMarketForkAgent)
		market.POST("/:id/rate", auth.AuthRequired, h.handleMarketRateAgent)
		market.GET("/:id/share", h.handleMarketShareAgent)
	}
}

// registerJSTemplateRoutes JSTemplate Module
func (h *Handlers) registerJSTemplateRoutes(r *gin.RouterGroup) {
	// Public embed — third-party pages inject pet.js by template jsSourceId
	r.GET("/js-templates/embed/:jsSourceId/loader.js", h.ServeJSTemplatePetLoaderJS)
	r.GET("/js-templates/embed/:jsSourceId/file/*filepath", h.ServeJSTemplateEmbedFile)

	petPackages := r.Group("pet-packages")
	petPackages.Use(auth.AuthRequired)
	{
		petPackages.POST("/validate", h.ValidatePetPackage)
		petPackages.POST("/import", h.ImportPetPackage)
	}

	// Public pet marketplace (marketId — no jsSourceId)
	r.GET("/pet-market/:marketId/preview/loader.js", h.ServePetMarketPreviewLoaderJS)
	r.GET("/pet-market/:marketId/preview/file/*filepath", h.ServePetMarketPreviewFile)
	r.GET("/pet-market/listings", h.ListPetMarketListings)
	r.GET("/pet-market/listings/:marketId", h.GetPetMarketListing)
	r.GET("/pet-market/listings/:marketId/download.zip", h.DownloadPetMarketListingZip)

	petMarket := r.Group("pet-market")
	petMarket.Use(auth.AuthRequired)
	{
		petMarket.POST("/listings", h.PublishPetMarketListing)
		petMarket.POST("/listings/:marketId/fork", h.ForkPetMarketListing)
		petMarket.POST("/listings/:marketId/rate", h.RatePetMarketListing)
	}

	jsTemplate := r.Group("js-templates")
	jsTemplate.Use(auth.AuthRequired)
	{
		jsTemplate.POST("", h.CreateJSTemplate)
		jsTemplate.GET("/default", h.ListDefaultJSTemplates)
		jsTemplate.GET("/custom", h.ListCustomJSTemplates)
		jsTemplate.GET("/search", h.SearchJSTemplates)
		jsTemplate.GET("/name/:name", h.GetJSTemplateByName)
		jsTemplate.GET("/:id/export.zip", h.ExportJSTemplateProjectZip)
		jsTemplate.GET("/:id/pull", h.PullPetPackage)
		jsTemplate.PUT("/:id/push", h.PushPetPackage)
		jsTemplate.GET("/:id/versions", h.ListJSTemplateVersions)
		jsTemplate.GET("/:id/versions/:versionId", h.GetJSTemplateVersion)
		jsTemplate.POST("/:id/versions/:versionId/rollback", h.RollbackJSTemplateVersion)
		jsTemplate.POST("/:id/versions/:versionId/publish", h.PublishJSTemplateVersion)
		jsTemplate.GET("/:id", h.GetJSTemplate)
		jsTemplate.GET("", h.ListJSTemplates)
		jsTemplate.PUT("/:id", h.UpdateJSTemplate)
		jsTemplate.DELETE("/:id", h.DeleteJSTemplate)
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

	chat.Use(auth.AuthApiRequired)
	{
		chat.GET("chat-session-log", h.getChatSessionLog)

		chat.GET("chat-session-log/:id", h.getChatSessionLogDetail)

		chat.GET("chat-session-log/by-session/:sessionId", h.getChatSessionLogsBySession)

		chat.GET("chat-session-log/by-agent/:agentId", h.getChatSessionLogByAgent)

		// Swipe / Branching API
		chat.GET("messages/:id/alternatives", h.getMessageAlternatives)
		chat.POST("messages/:id/activate", h.activateMessageAlternative)
		chat.POST("messages/:id/regenerate", h.regenerateMessage)
		chat.POST("messages/:id/branch", h.branchFromMessage)

		// World Info / Lorebook API
		chat.GET("world-info", h.listWorldInfoEntries)
		chat.POST("world-info", h.createWorldInfoEntry)
		chat.PUT("world-info/:id", h.updateWorldInfoEntry)
		chat.DELETE("world-info/:id", h.deleteWorldInfoEntry)
		chat.POST("world-info/activate", h.activateWorldInfo)
		chat.POST("world-info/inject", h.injectWorldInfo)

		// User Persona API
		chat.GET("personas", h.listPersonas)
		chat.POST("personas", h.createPersona)
		chat.PUT("personas/:id", h.updatePersona)
		chat.DELETE("personas/:id", h.deletePersona)
		chat.PUT("personas/:id/default", h.setDefaultPersona)
		chat.GET("personas/:id/inject", h.injectPersona)
	}
}

// registerCredentialsRoutes Credentials Module
func (h *Handlers) registerCredentialsRoutes(r *gin.RouterGroup) {
	credential := r.Group("credentials")
	{
		credential.POST("/", auth.AuthRequired, h.handleCreateCredential)

		credential.GET("/", auth.AuthRequired, h.handleGetCredential)

		credential.POST("/by-key", h.handleGetCredentialByKey)

		credential.DELETE("/:id", auth.AuthRequired, h.handleDeleteCredential)
	}
}

// registerNotificationRoutes Notification Module
func (h *Handlers) registerNotificationRoutes(r *gin.RouterGroup) {
	notificationGroup := r.Group("notification")
	{
		notificationGroup.GET("unread-count", auth.AuthRequired, h.handleUnReadNotificationCount)

		notificationGroup.GET("", auth.AuthRequired, h.handleListNotifications)

		notificationGroup.POST("readAll", auth.AuthRequired, h.handleAllNotifications)

		notificationGroup.PUT("/read/:id", auth.AuthRequired, h.handleMarkNotificationAsRead)

		notificationGroup.DELETE("/:id", auth.AuthRequired, h.handleDeleteNotification)

		notificationGroup.POST("/batch-delete", auth.AuthRequired, h.handleBatchDeleteNotifications)

		notificationGroup.GET("/all-ids", auth.AuthRequired, h.handleGetAllNotificationIds)
	}
}

// registerSystemRoutes System Module
func (h *Handlers) registerSystemRoutes(r *gin.RouterGroup) {
	system := r.Group("system")
	{
		system.POST("/rate-limiter/config", h.UpdateRateLimiterConfig)

		system.GET("/health", h.HealthCheck)
		system.GET("/status", h.SystemStatus)
		system.GET("/dashboard/metrics", auth.AuthRequired, h.DashboardMetrics)

		system.GET("/init", h.SystemInit)

		system.POST("/voiceprint/config", auth.AuthRequired, h.SaveVoiceprintConfig)

		system.POST("/upload/audio", h.UploadAudio)

		system.GET("/search/status", h.GetSearchStatus)
		system.PUT("/search/config", auth.AuthRequired, h.UpdateSearchConfig)
		system.POST("/search/enable", auth.AuthRequired, h.EnableSearch)
		system.POST("/search/disable", auth.AuthRequired, h.DisableSearch)
	}

	voiceprint := r.Group("/voiceprint")
	{
		voiceprint.GET("", auth.AuthRequired, h.GetVoiceprints)
		voiceprint.POST("", auth.AuthRequired, h.CreateVoiceprint)
		voiceprint.POST("/register", auth.AuthRequired, h.RegisterVoiceprint)
		voiceprint.POST("/identify", auth.AuthRequired, h.IdentifyVoiceprint)
		voiceprint.POST("/verify", auth.AuthRequired, h.VerifyVoiceprint)
		voiceprint.PUT("/:id", auth.AuthRequired, h.UpdateVoiceprint)
		voiceprint.DELETE("/:id", auth.AuthRequired, h.DeleteVoiceprint)
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

	device.Use(auth.AuthRequired)
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

// registerSendCloudWebhookRoutes SendCloud Webhook Module
func (h *Handlers) registerSendCloudWebhookRoutes(r *gin.RouterGroup) {
	webhook := r.Group("webhooks/sendcloud")
	{
		webhook.POST("", h.handleSendCloudWebhook)
		webhook.POST("/batch", h.handleSendCloudWebhookBatch)
	}
}

func (h *Handlers) registerAdminManagementRoutes(r *gin.RouterGroup) {
	adminGuard := []gin.HandlerFunc{auth.AuthRequired, middleware.RequireAdmin}

	llmChannels := r.Group("admin/llm-channels", adminGuard...)
	{
		llmChannels.GET("", h.handleAdminListLLMChannels)
		llmChannels.GET("/:id", h.handleAdminGetLLMChannel)
		llmChannels.POST("", h.handleAdminCreateLLMChannel)
		llmChannels.PUT("/:id", h.handleAdminUpdateLLMChannel)
		llmChannels.DELETE("/:id", h.handleAdminDeleteLLMChannel)
		llmChannels.POST("/:id/sync-abilities", h.handleAdminSyncLLMChannelAbilities)
	}

	llmAbilities := r.Group("admin/llm-abilities", adminGuard...)
	{
		llmAbilities.GET("", h.handleAdminListLLMAbilities)
	}

	llmModelMetas := r.Group("admin/llm-model-metas", adminGuard...)
	{
		llmModelMetas.GET("", h.handleAdminListLLMModelMetas)
		llmModelMetas.POST("", h.handleAdminUpsertLLMModelMeta)
		llmModelMetas.PUT("/:id", h.handleAdminUpsertLLMModelMeta)
		llmModelMetas.DELETE("/:id", h.handleAdminDeleteLLMModelMeta)
	}

	asrChannels := r.Group("admin/asr-channels", adminGuard...)
	{
		asrChannels.GET("", h.handleAdminListASRChannels)
		asrChannels.GET("/:id", h.handleAdminGetASRChannel)
		asrChannels.POST("", h.handleAdminCreateASRChannel)
		asrChannels.PUT("/:id", h.handleAdminUpdateASRChannel)
		asrChannels.DELETE("/:id", h.handleAdminDeleteASRChannel)
	}

	ttsChannels := r.Group("admin/tts-channels", adminGuard...)
	{
		ttsChannels.GET("", h.handleAdminListTTSChannels)
		ttsChannels.GET("/:id", h.handleAdminGetTTSChannel)
		ttsChannels.POST("", h.handleAdminCreateTTSChannel)
		ttsChannels.PUT("/:id", h.handleAdminUpdateTTSChannel)
		ttsChannels.DELETE("/:id", h.handleAdminDeleteTTSChannel)
	}

	speechUsage := r.Group("admin/speech-usage", adminGuard...)
	{
		speechUsage.GET("", h.handleAdminListSpeechUsage)
		speechUsage.GET("/stats", h.handleAdminSpeechUsageStats)
		speechUsage.GET("/:id", h.handleAdminGetSpeechUsage)
	}

	meOrgs := r.Group("me/orgs", auth.AuthRequired)
	{
		meOrgs.GET("", h.handleMeListOrgs)
	}

	meSpeechUsage := r.Group("me/speech-usage")
	{
		meSpeechUsage.GET("", auth.AuthRequired, h.handleMeListSpeechUsage)
	}

	meLLMUsage := r.Group("me/llm-usage", auth.AuthRequired)
	{
		meLLMUsage.GET("", h.handleMeListLLMUsage)
		meLLMUsage.GET("/summary", h.handleMeLLMUsageSummary)
	}

	meLLMTokens := r.Group("me/llm-tokens", auth.AuthRequired, h.OrgScopeMiddleware())
	{
		meLLMTokens.GET("", h.handleMeListLLMTokens)
		meLLMTokens.POST("", h.handleMeCreateLLMToken)
		meLLMTokens.PUT("/:id", h.handleMeUpdateLLMToken)
		meLLMTokens.POST("/:id/regenerate", h.handleMeRegenerateLLMToken)
		meLLMTokens.DELETE("/:id", h.handleMeDeleteLLMToken)
	}

	llmTokens := r.Group("admin/llm-tokens", adminGuard...)
	{
		llmTokens.GET("", h.handleAdminListLLMTokens)
		llmTokens.GET("/:id", h.handleAdminGetLLMToken)
		llmTokens.POST("", h.handleAdminCreateLLMToken)
		llmTokens.PUT("/:id", h.handleAdminUpdateLLMToken)
		llmTokens.POST("/:id/regenerate", h.handleAdminRegenerateLLMToken)
		llmTokens.POST("/:id/reset-usage", h.handleAdminResetLLMTokenUsage)
		llmTokens.DELETE("/:id", h.handleAdminDeleteLLMToken)
	}

	announcements := r.Group("admin/announcements", adminGuard...)
	{
		announcements.GET("", h.handleAdminListAnnouncements)
		announcements.GET("/:id", h.handleAdminGetAnnouncement)
		announcements.POST("", h.handleAdminCreateAnnouncement)
		announcements.PUT("/:id", h.handleAdminUpdateAnnouncement)
		announcements.POST("/:id/publish", h.handleAdminPublishAnnouncement)
		announcements.POST("/:id/offline", h.handleAdminOfflineAnnouncement)
		announcements.DELETE("/:id", h.handleAdminDeleteAnnouncement)
	}

	notifChannels := r.Group("admin/notification-channels", adminGuard...)
	{
		notifChannels.GET("", h.handleListNotificationChannels)
		notifChannels.POST("", h.handleCreateNotificationChannel)
		notifChannels.GET("/:id", h.handleGetNotificationChannel)
		notifChannels.PUT("/:id", h.handleUpdateNotificationChannel)
		notifChannels.DELETE("/:id", h.handleDeleteNotificationChannel)
	}

	mailTemplates := r.Group("admin/mail-templates", adminGuard...)
	{
		mailTemplates.GET("", h.handleListMailTemplates)
		mailTemplates.POST("", h.handleCreateMailTemplate)
		mailTemplates.GET("/:id", h.handleGetMailTemplate)
		mailTemplates.PUT("/:id", h.handleUpdateMailTemplate)
		mailTemplates.DELETE("/:id", h.handleDeleteMailTemplate)
	}

	mailLogsAdmin := r.Group("admin/mail-logs", adminGuard...)
	{
		mailLogsAdmin.GET("", h.handleListMailLogs)
		mailLogsAdmin.GET("/:id", h.handleGetMailLogDetail)
		mailLogsAdmin.GET("/stats/summary", h.handleGetMailLogStats)
	}
	smsLogsAdmin := r.Group("admin/sms-logs", adminGuard...)
	{
		smsLogsAdmin.GET("", h.handleListSMSLogs)
		smsLogsAdmin.GET("/:id", h.handleGetSMSLogDetail)
	}
	smsAdmin := r.Group("admin/sms", adminGuard...)
	{
		smsAdmin.POST("/send", h.handleAdminSendSMS)
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

	adminAgents := r.Group("admin/agents", adminGuard...)
	{
		adminAgents.GET("", h.handleAdminListAgents)
		adminAgents.GET("/:id", h.handleAdminGetAgent)
		adminAgents.PUT("/:id", h.handleAdminUpdateAgent)
		adminAgents.DELETE("/:id", h.handleAdminDeleteAgent)
	}

	chatSessions := r.Group("admin/chat-sessions", adminGuard...)
	{
		chatSessions.GET("", h.handleAdminListChatSessions)
	}

	chatMessages := r.Group("admin/chat-messages", adminGuard...)
	{
		chatMessages.GET("", h.handleAdminListChatMessages)
	}

	llmUsage := r.Group("admin/llm-usage", adminGuard...)
	{
		llmUsage.GET("", h.handleAdminListLLMUsage)
	}

	voiceTraining := r.Group("admin/voice-training", adminGuard...)
	{
		voiceTraining.GET("/tasks", h.handleAdminListVoiceTrainingTasks)
		voiceTraining.GET("/tasks/:id", h.handleAdminGetVoiceTrainingTask)
		voiceTraining.DELETE("/tasks/:id", h.handleAdminDeleteVoiceTrainingTask)
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

	notificationCenter := r.Group("admin/notifications", adminGuard...)
	{
		notificationCenter.GET("", h.handleAdminListInternalNotifications)
		notificationCenter.GET("/:id", h.handleAdminGetInternalNotification)
		notificationCenter.DELETE("/:id", h.handleAdminDeleteInternalNotification)
	}

	devices := r.Group("admin/devices", adminGuard...)
	{
		devices.GET("", h.handleAdminListDevices)
		devices.GET("/:id", h.handleAdminGetDevice)
		devices.DELETE("/:id", h.handleAdminDeleteDevice)
	}

	worldInfo := r.Group("admin/world-info", adminGuard...)
	{
		worldInfo.GET("", h.handleAdminListWorldInfo)
		worldInfo.GET("/:id", h.handleAdminGetWorldInfo)
		worldInfo.PUT("/:id", h.handleAdminUpdateWorldInfo)
		worldInfo.DELETE("/:id", h.handleAdminDeleteWorldInfo)
	}

	personas := r.Group("admin/personas", adminGuard...)
	{
		personas.GET("", h.handleAdminListPersonas)
		personas.GET("/:id", h.handleAdminGetPersona)
		personas.PUT("/:id", h.handleAdminUpdatePersona)
		personas.DELETE("/:id", h.handleAdminDeletePersona)
	}
}
