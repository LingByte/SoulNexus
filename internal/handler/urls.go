package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"log"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
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

func NewSIPServiceHandlers(db *gorm.DB) *Handlers {
	return &Handlers{
		db: db,
	}
}

// RegisterUserServiceRoutes registers auth routes and a minimal system health check under the API prefix.
func (h *Handlers) RegisterSIPServiceRoutes(engine *gin.Engine) {
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r := engine.Group(apiPrefix)
	r.Use(middleware.InjectDB(h.db))
	r.Use(middleware.MutatingRequestTrustedOrigin())
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
	engine.GET("/.well-known/jwks.json", h.JWKSHandler)
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r := engine.Group(apiPrefix)
	r.Use(middleware.InjectDB(h.db))
	r.Use(middleware.MutatingRequestTrustedOrigin())
	middleware.ApplyGlobalMiddlewares(r)

	sys := r.Group("system")
	{
		sys.GET("/health", h.HealthCheck)
	}

	h.registerAuthRoutes(r)

	// Browser HTML pages need DB (GetRenderPageContext); API group prefix is not used here.
	browser := engine.Group("")
	browser.Use(middleware.InjectDB(h.db))
	{
		browser.GET("/login", h.RenderSigninPage)
		browser.GET("/login/revoke-account-deletion", h.RenderAccountDeletionRevokePage)
	}
}

func (h *Handlers) Register(engine *gin.Engine) {
	r := engine.Group(config.GlobalConfig.Server.APIPrefix)

	// Register Global Singleton DB
	r.Use(middleware.InjectDB(h.db))
	r.Use(middleware.MutatingRequestTrustedOrigin())

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
	h.registerSendCloudWebhookRoutes(r)
	h.registerGroupRoutes(r)
	h.registerAnnouncementRoutes(r)
	h.registerWebSocketRoutes(r)
	h.registerAgentRoutes(r)
	h.registerKnowledgeRoutes(r)
	h.registerChatRoutes(r)
	h.registerCredentialsRoutes(r)
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
	h.registerOpenAPIRoutes(r)        // Open API (apiKey + apiSecret auth)
	h.registerLLMRelayRoutes(engine)  // OpenAI/Anthropic 兼容对外网关 /v1/*
	h.RegisterPublicWorkflowRoutes(r)
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
		voice.POST("/parse_attachment", h.ParseAttachmentContent)

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

// registerGroupRoutes Group Module
func (h *Handlers) registerGroupRoutes(r *gin.RouterGroup) {
	group := r.Group("group")
	group.Use(models.AuthRequired)
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
		agents.POST("add", models.AuthRequired, h.CreateAgent)

		agents.GET("", models.AuthRequired, h.ListAgents)

		agents.GET("/:id", models.AuthRequired, h.GetAgent)

		agents.PUT("/:id", models.AuthRequired, h.UpdateAgent)

		agents.DELETE("/:id", models.AuthRequired, h.DeleteAgent)

		agents.GET("/lingecho/client/:id/loader.js", h.ServeVoiceSculptorLoaderJS)

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

		chat.GET("chat-session-log/by-agent/:agentId", h.getChatSessionLogByAgent)
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

// registerSendCloudWebhookRoutes SendCloud Webhook Module
func (h *Handlers) registerSendCloudWebhookRoutes(r *gin.RouterGroup) {
	webhook := r.Group("webhooks/sendcloud")
	{
		webhook.POST("", h.handleSendCloudWebhook)
		webhook.POST("/batch", h.handleSendCloudWebhookBatch)
	}
}

// registerAuthRoutes registers user authentication and profile routes (user module surface).
func (h *Handlers) registerAuthRoutes(r *gin.RouterGroup) {
	auth := r.Group(config.GlobalConfig.Server.AuthPrefix)
	{
		auth.GET("/register", h.handleUserSignupPage)
		auth.POST("/register", h.handleUserSignup)
		auth.POST("/register/email", h.handleUserSignupByEmail)
		auth.POST("/send/email", h.handleSendEmailCode)

		auth.GET("/captcha", h.handleGetCaptcha)
		auth.POST("/captcha/verify", h.handleVerifyCaptcha)

		auth.GET("/salt", h.handleGetSalt)

		auth.GET("/login", h.handleUserSigninPage)
		auth.POST("/login", h.handleUserSigninByPassword)
		auth.POST("/login/password", h.handleUserSigninByPassword)
		auth.POST("/login/email", h.handleUserSigninByEmail)
		auth.POST("/refresh", h.handleRefreshToken)
		auth.GET("/github/login", h.handleGitHubLogin)
		auth.GET("/github/callback", h.handleGitHubCallback)
		auth.GET("/wechat/login", h.handleWechatLogin)
		auth.GET("/wechat/config-check", h.handleWechatConfigCheck)
		auth.GET("/wechat/login-code", h.handleWechatLoginCode)
		auth.GET("/wechat/qrcode", h.handleWechatLoginCode)
		auth.GET("/wechat/bind/code", models.AuthRequired, h.handleWechatBindCode)
		auth.GET("/wechat/bind/status", models.AuthRequired, h.handleWechatBindStatus)
		auth.GET("/wechat/status", h.handleWechatLoginStatus)
		auth.GET("/wechat/check-login/:sceneId", h.handleWechatCheckLogin)
		auth.GET("/wechat/oauth/callback", h.handleWechatOAuthCallback)
		auth.GET("/wechat/callback", h.handleWechatLoginCallback)
		auth.POST("/wechat/callback", h.handleWechatLoginMessage)
		auth.POST("/wechat/mp/message", h.handleWechatLoginMessage)
		auth.GET("/wechat/mp/message", h.handleWechatLoginCallback)
		auth.GET("/oidc/authorize", h.handleOIDCAuthorize)
		auth.POST("/oidc/token", h.handleOIDCToken)
		auth.POST("/oidc/exchange", h.handleOIDCExchange)

		auth.GET("/logout", h.handleUserLogout)
		auth.GET("/info", models.AuthRequired, h.handleUserInfo)

		auth.GET("/reset-password", h.handleUserResetPasswordPage)
		auth.POST("/reset-password", h.handleResetPassword)
		auth.POST("/reset-password/confirm", h.handleResetPasswordConfirm)
		auth.POST("/change-password", models.AuthRequired, h.handleChangePassword)
		auth.POST("/change-password/email", models.AuthRequired, h.handleChangePasswordByEmail)

		auth.GET("/devices", models.AuthRequired, h.handleGetUserDevices)
		auth.DELETE("/devices/:deviceId", models.AuthRequired, h.handleDeleteUserDevice)
		auth.POST("/devices/trust", models.AuthRequired, h.handleTrustUserDevice)
		auth.POST("/devices/untrust", models.AuthRequired, h.handleUntrustUserDevice)

		auth.POST("/devices/verify", h.handleVerifyDeviceForLogin)
		auth.POST("/devices/send-verification", h.handleSendDeviceVerificationCode)

		auth.GET("/verify-email", h.handleVerifyEmail)
		auth.POST("/send-email-verification", models.AuthRequired, h.handleSendEmailVerification)

		auth.POST("/verify-phone", models.AuthRequired, h.handleVerifyPhone)
		auth.POST("/send-phone-verification", models.AuthRequired, h.handleSendPhoneVerification)

		auth.PUT("/update", models.AuthRequired, h.handleUserUpdate)
		auth.PUT("/update/preferences", models.AuthRequired, h.handleUserUpdatePreferences)
		auth.POST("/update/basic/info", models.AuthRequired, h.handleUserUpdateBasicInfo)

		auth.PUT("/notification-settings", models.AuthRequired, h.handleUpdateNotificationSettings)

		auth.PUT("/user-preferences", models.AuthRequired, h.handleUpdateUserPreferences)

		auth.GET("/stats", models.AuthRequired, h.handleGetUserStats)

		auth.POST("/avatar/upload", models.AuthRequired, h.handleUploadAvatar)

		auth.POST("/two-factor/setup", models.AuthRequired, h.handleTwoFactorSetup)
		auth.POST("/two-factor/enable", models.AuthRequired, h.handleTwoFactorEnable)
		auth.POST("/two-factor/disable", models.AuthRequired, h.handleTwoFactorDisable)
		auth.GET("/two-factor/status", models.AuthRequired, h.handleTwoFactorStatus)

		// 无密码 / Passkey 登录：discoverable，浏览器经 navigator.credentials.get 完成。
		auth.POST("/passkey/begin", h.handleAuthPasskeyBegin)
		auth.POST("/passkey/finish", h.handleAuthPasskeyFinish)

		auth.GET("/activity", models.AuthRequired, h.handleGetUserActivity)

		auth.POST("/account-deletion/send-cancel-code", h.handleAccountDeletionSendCancelCode)
		auth.POST("/account-deletion/cancel-by-email", h.handleAccountDeletionCancelByEmail)

		auth.GET("/account-deletion/eligibility", models.AuthRequired, h.handleAccountDeletionEligibility)
		auth.POST("/account-deletion/send-email-code", models.AuthRequired, h.handleAccountDeletionSendEmailCode)
		auth.POST("/account-deletion/request", models.AuthRequired, h.handleAccountDeletionRequest)
		auth.POST("/account-deletion/cancel", models.AuthRequired, h.handleAccountDeletionCancel)
		auth.DELETE("/bindings/github", models.AuthRequired, h.handleUnbindGitHub)
		auth.DELETE("/bindings/wechat", models.AuthRequired, h.handleUnbindWechat)

	}
}

func (h *Handlers) registerAdminManagementRoutes(r *gin.RouterGroup) {
	adminGuard := []gin.HandlerFunc{models.AuthRequired, h.requireAdmin}

	h.registerAccessAdminRoutes(r)

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

	meOrgs := r.Group("me/orgs", models.AuthRequired)
	{
		meOrgs.GET("", h.handleMeListOrgs)
	}

	meSpeechUsage := r.Group("me/speech-usage")
	{
		meSpeechUsage.GET("", models.AuthRequired, h.handleMeListSpeechUsage)
	}

	mePasskeys := r.Group("me/passkeys", models.AuthRequired)
	{
		mePasskeys.GET("", h.handleMeListPasskeys)
		mePasskeys.POST("/registration/begin", h.handleMePasskeyRegistrationBegin)
		mePasskeys.POST("/registration/finish", h.handleMePasskeyRegistrationFinish)
		mePasskeys.PUT("/:id", h.handleMeUpdatePasskey)
		mePasskeys.DELETE("/:id", h.handleMeDeletePasskey)
	}

	meTwoFA := r.Group("me/twofa", models.AuthRequired)
	{
		meTwoFA.GET("/status", h.handleMeTwoFAStatus)
		meTwoFA.POST("/backup-codes/regenerate", h.handleMeTwoFABackupCodesRegenerate)
		meTwoFA.POST("/backup-codes/use", h.handleMeTwoFABackupCodeUse)
		meTwoFA.POST("/reset", h.handleMeTwoFAReset)
	}

	meLLMTokens := r.Group("me/llm-tokens", models.AuthRequired, h.OrgScopeMiddleware())
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

	devices := r.Group("admin/devices", adminGuard...)
	{
		devices.GET("", h.handleAdminListDevices)
		devices.GET("/:id", h.handleAdminGetDevice)
		devices.DELETE("/:id", h.handleAdminDeleteDevice)
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
}

func (h *Handlers) registerAccessAdminRoutes(r *gin.RouterGroup) {
	guard := []gin.HandlerFunc{models.AuthRequired, h.requireAdmin, h.requireAccessManage}

	perms := r.Group("admin/permissions", guard...)
	{
		perms.GET("", h.handleAdminListPermissions)
		perms.POST("", h.handleAdminCreatePermission)
		perms.PUT("/:id", h.handleAdminUpdatePermission)
		perms.DELETE("/:id", h.handleAdminDeletePermission)
	}

	roles := r.Group("admin/roles", guard...)
	{
		roles.GET("", h.handleAdminListRoles)
		roles.POST("", h.handleAdminCreateRole)
		roles.PUT("/:id/permissions", h.handleAdminSetRolePermissions)
		roles.GET("/:id", h.handleAdminGetRole)
		roles.PUT("/:id", h.handleAdminUpdateRole)
		roles.DELETE("/:id", h.handleAdminDeleteRole)
	}

	userAccess := r.Group("admin/users", guard...)
	{
		userAccess.GET("/:id/access", h.handleAdminGetUserAccess)
		userAccess.PUT("/:id/access", h.handleAdminSetUserAccess)
	}
}

// JWKSHandler returns the JSON Web Key Set (JWKS) endpoint
func (h *Handlers) JWKSHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.Header("Cache-Control", "public, max-age=3600")
	km := utils.JWTKeyManager()
	if km == nil {
		km = bootstrap.GlobalKeyManager
	}
	if km == nil {
		c.JSON(500, gin.H{"error": "key manager not initialized"})
		return
	}
	jwksJSON, err := km.GetJWKSJSON()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate JWKS"})
		return
	}
	c.String(200, jwksJSON)
}
