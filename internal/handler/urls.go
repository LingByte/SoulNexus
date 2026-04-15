package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"log"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/service"
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
	// Register Business Module Routes
	h.registerAuthRoutes(r)
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
	h.registerWorkflowRoutes(r)
	h.registerWorkflowPluginRoutes(r) // Add workflow plugin routes
	h.registerNodePluginRoutes(r)     // Add node plugin routes
	h.registerMCPRoutes(r)            // Add MCP routes
	h.registerMCPMarketplaceRoutes(r) // Add MCP marketplace routes
	h.registerOpenAPIRoutes(r)        // Open API (apiKey + apiSecret auth)
	// Register public workflow routes (no auth required)
	h.RegisterPublicWorkflowRoutes(r)
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

		// Batch delete notifications
		notificationGroup.POST("/batch-delete", models.AuthRequired, h.handleBatchDeleteNotifications)

		// Get all notification IDs (for select all functionality)
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

		// System initialization route (no auth required)
		system.GET("/init", h.SystemInit)

		// Voice clone configuration routes
		system.POST("/voice-clone/config", models.AuthRequired, h.SaveVoiceCloneConfig)

		// Voiceprint configuration routes
		system.POST("/voiceprint/config", models.AuthRequired, h.SaveVoiceprintConfig)

		// Audio file upload route
		system.POST("/upload/audio", h.UploadAudio)

		// Search configuration routes
		system.GET("/search/status", h.GetSearchStatus)
		system.PUT("/search/config", models.AuthRequired, h.UpdateSearchConfig)
		system.POST("/search/enable", models.AuthRequired, h.EnableSearch)
		system.POST("/search/disable", models.AuthRequired, h.DisableSearch)
	}

	// Voiceprint management routes (separate from system group)
	voiceprint := r.Group("/voiceprint")
	{
		voiceprint.GET("", models.AuthRequired, h.GetVoiceprints)               // 获取声纹列表
		voiceprint.POST("", models.AuthRequired, h.CreateVoiceprint)            // 创建声纹记录
		voiceprint.POST("/register", models.AuthRequired, h.RegisterVoiceprint) // 注册声纹（上传音频）
		voiceprint.POST("/identify", models.AuthRequired, h.IdentifyVoiceprint) // 声纹识别
		voiceprint.POST("/verify", models.AuthRequired, h.VerifyVoiceprint)     // 声纹验证
		voiceprint.PUT("/:id", models.AuthRequired, h.UpdateVoiceprint)         // 更新声纹记录
		voiceprint.DELETE("/:id", models.AuthRequired, h.DeleteVoiceprint)      // 删除声纹记录
	}
}

// registerOTARoutes OTA Module
func (h *Handlers) registerOTARoutes(r *gin.RouterGroup) {
	ota := r.Group("ota")
	{
		// OTA version check and device activation (no auth required for device registration)
		ota.POST("/", h.HandleOTACheck)

		// Quick device activation check
		ota.POST("/activate", h.HandleOTAActivate)

		// OTA health check
		ota.GET("/", h.HandleOTAGet)
	}
}

// registerDeviceRoutes Device Module (completely consistent with xiaozhi-esp32)
func (h *Handlers) registerDeviceRoutes(r *gin.RouterGroup) {
	device := r.Group("device")

	// Get device configuration interface (no authentication required, for xiaozhi-server calls)
	device.GET("/config/:deviceId", h.GetDeviceConfig)

	device.Use(models.AuthRequired) // Requires user login
	{
		// Bind device (activate device) - completely consistent with xiaozhi-esp32 path
		device.POST("/bind/:agentId/:deviceCode", h.BindDevice)

		// Get bound devices
		device.GET("/bind/:agentId", h.GetUserDevices)

		// Unbind device
		device.POST("/unbind", h.UnbindDevice)

		// Update device information
		device.PUT("/update/:id", h.UpdateDeviceInfo)

		// Manually add device
		device.POST("/manual-add", h.ManualAddDevice)

		// Device monitoring and management
		device.GET("/:deviceId", h.GetDeviceDetail)                       // Get device detail
		device.GET("/:deviceId/error-logs", h.GetDeviceErrorLogs)         // Get device error logs
		device.POST("/error-logs/:errorId/resolve", h.ResolveDeviceError) // Resolve device error
		device.GET("/call-recordings", h.GetCallRecordings)               // Get call recordings
		device.GET("/call-recordings/:id", h.GetCallRecordingDetail)      // Get call recording detail

		// AI分析相关路由
		device.POST("/call-recordings/:id/analyze", h.AnalyzeCallRecording)         // 分析单个录音
		device.POST("/call-recordings/batch-analyze", h.BatchAnalyzeCallRecordings) // 批量分析录音
		device.GET("/call-recordings/:id/analysis", h.GetCallRecordingAnalysis)     // 获取分析结果

		// Device status updates (for hardware devices to report status)
		device.POST("/status", h.UpdateDeviceStatus) // Update device status
		device.POST("/error", h.LogDeviceError)      // Log device error

		// Recording file access
		device.GET("/recordings/*filepath", h.ServeRecordingFile) // Serve recording files
	}
}

// registerGroupRoutes Group Module
func (h *Handlers) registerGroupRoutes(r *gin.RouterGroup) {
	group := r.Group("group")
	group.Use(models.AuthRequired)
	{
		// Organization management
		group.POST("", h.CreateGroup)
		group.GET("", h.ListGroups)

		// Search users - must be before /:id
		group.GET("/search-users", h.SearchUsers)

		// Invitation management - must be before /:id, otherwise will be matched as id=invitations
		group.GET("/invitations", h.ListInvitations)
		group.POST("/invitations/:id/accept", h.AcceptInvitation)
		group.POST("/invitations/:id/reject", h.RejectInvitation)

		// Overview configuration management - must be registered before /:id to avoid route conflicts
		group.GET("/:id/overview/config", h.GetOverviewConfig)
		group.POST("/:id/overview/config", h.SaveOverviewConfig)
		group.PUT("/:id/overview/config", h.SaveOverviewConfig)
		group.DELETE("/:id/overview/config", h.DeleteOverviewConfig)

		// Organization statistics - must be registered before /:id
		group.GET("/:id/statistics", h.GetGroupStatistics)

		// Organization member management - must be registered before /:id
		group.POST("/:id/leave", h.LeaveGroup)
		group.DELETE("/:id/members/:memberId", h.RemoveMember)
		group.PUT("/:id/members/:memberId/role", h.UpdateMemberRole)

		// Invite users - must be registered before /:id
		group.POST("/:id/invite", h.InviteUser)

		// Get organization shared resources - must be registered before /:id
		group.GET("/:id/resources", h.GetGroupSharedResources)

		// Upload organization avatar - must be registered before /:id
		group.POST("/:id/avatar", h.UploadGroupAvatar)

		// Organization advanced operations - must be registered before /:id
		group.POST("/:id/archive", h.ArchiveGroup)
		group.POST("/:id/restore", h.RestoreGroup)
		group.POST("/:id/clone", h.CloneGroup)
		group.GET("/:id/export", h.ExportGroup)
		group.GET("/:id/activity-logs", h.GetGroupActivityLogs)

		// Organization details and management - parameter routes at the end
		group.GET("/:id", h.GetGroup)
		group.PUT("/:id", h.UpdateGroup)
		group.DELETE("/:id", h.DeleteGroup)
	}
}

// registerQuotaRoutes registers quota routes
func (h *Handlers) registerQuotaRoutes(r *gin.RouterGroup) {
	quota := r.Group("quota")
	quota.Use(models.AuthRequired)
	{
		// User quota management
		quota.GET("/user", h.ListUserQuotas)
		quota.GET("/user/:type", h.GetUserQuota)
		quota.POST("/user", h.CreateUserQuota)
		quota.PUT("/user/:type", h.UpdateUserQuota)
		quota.DELETE("/user/:type", h.DeleteUserQuota)

		// Organization quota management
		quota.GET("/group/:id", h.ListGroupQuotas)
		quota.GET("/group/:id/:type", h.GetGroupQuota)
		quota.POST("/group/:id", h.CreateGroupQuota)
		quota.PUT("/group/:id/:type", h.UpdateGroupQuota)
		quota.DELETE("/group/:id/:type", h.DeleteGroupQuota)
	}
}

// registerAlertRoutes registers alert routes
func (h *Handlers) registerAlertRoutes(r *gin.RouterGroup) {
	alert := r.Group("alert")
	alert.Use(models.AuthRequired)
	{
		// Alert rule management
		alert.POST("/rules", h.CreateAlertRule)
		alert.GET("/rules", h.ListAlertRules)
		alert.GET("/rules/:id", h.GetAlertRule)
		alert.PUT("/rules/:id", h.UpdateAlertRule)
		alert.DELETE("/rules/:id", h.DeleteAlertRule)

		// 告警管理
		alert.GET("", h.ListAlerts)
		alert.GET("/:id", h.GetAlert)
		alert.POST("/:id/resolve", h.ResolveAlert)
		alert.POST("/:id/mute", h.MuteAlert)
	}
}

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

		// 版本管理
		jsTemplate.GET("/:id/versions", h.ListJSTemplateVersions)
		jsTemplate.GET("/:id/versions/:versionId", h.GetJSTemplateVersion)
		jsTemplate.POST("/:id/versions/:versionId/rollback", h.RollbackJSTemplateVersion)
		jsTemplate.POST("/:id/versions/:versionId/publish", h.PublishJSTemplateVersion)
	}

	// Webhook路由（不需要认证，使用签名验证）
	webhook := r.Group("js-templates/webhook")
	{
		webhook.POST("/:jsSourceId", h.TriggerJSTemplateWebhook)
	}
}

// registerChatRoutes Chat Module
func (h *Handlers) registerChatRoutes(r *gin.RouterGroup) {
	chat := r.Group("chat")

	// WebSocket 连接不需要中间件，因为 handleConnection 内部已经做了验证
	chat.GET("call", h.handleConnection)

	// 其他路由需要认证
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

		credential.POST("/by-key", h.handleGetCredentialByKey) // 无需登录，通过 apikey/apisecret 查询

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

// registerWebSocketRoutes registers WebSocket routes
func (h *Handlers) registerWebSocketRoutes(r *gin.RouterGroup) {
	wsHandler := websocket.NewHandler(h.wsHub)

	// WebSocket连接端点
	r.GET("/ws", models.AuthRequired, wsHandler.HandleWebSocket)

	// 通用WebSocket语音端点（支持多服务商）
	r.GET("/voice/websocket", h.HandleWebSocketVoice)

	// WebSocket管理API端点
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
	xunfei.Use(models.AuthRequired) // 需要认证
	{
		// 语音合成
		xunfei.POST("/synthesize", h.XunfeiSynthesize)

		// 训练任务管理
		xunfei.POST("/task/create", h.XunfeiCreateTask)
		xunfei.POST("/task/submit-audio", h.XunfeiSubmitAudio)
		xunfei.POST("/task/query", h.XunfeiQueryTask)

		// 训练文本
		xunfei.GET("/training-texts", h.XunfeiGetTrainingTexts)
	}
}

// registerVolcengineTTSRoutes 注册火山引擎TTS路由
func (h *Handlers) registerVolcengineTTSRoutes(r *gin.RouterGroup) {
	volcengine := r.Group("/volcengine")
	volcengine.Use(models.AuthRequired) // 需要认证
	{
		// 语音合成
		volcengine.POST("/synthesize", h.VolcengineSynthesize)

		// 训练任务管理
		// 注意：火山引擎不需要 create task，speaker_id 从控制台获取
		volcengine.POST("/task/submit-audio", h.VolcengineSubmitAudio)

		volcengine.POST("/task/query", h.VolcengineQueryTask)
	}
}

// registerVoiceTrainingRoutes 注册音色训练路由
func (h *Handlers) registerVoiceTrainingRoutes(r *gin.RouterGroup) {
	voice := r.Group("/voice")

	// 无需认证的接口
	voice.GET("/lingecho/v1/", h.HandleHardwareWebSocketVoice)
	voice.POST("/simple_text_chat", h.SimpleTextChat) // 简单文本对话（无需token验证）

	// 需要认证的接口
	voice.Use(models.AuthRequired)
	{
		// 训练任务管理
		voice.POST("/training/create", h.CreateTrainingTask)
		voice.POST("/training/submit-audio", h.SubmitAudio)
		voice.POST("/training/query", h.QueryTaskStatus)

		// 音色管理
		voice.GET("/clones", h.GetUserVoiceClones)
		voice.GET("/clones/:id", h.GetVoiceClone)
		voice.POST("/clones/update", h.UpdateVoiceClone)
		voice.POST("/clones/delete", h.DeleteVoiceClone)

		// 语音合成
		voice.POST("/synthesize", h.SynthesizeWithVoice)

		// 合成历史
		voice.GET("/synthesis/history", h.GetSynthesisHistory)
		voice.POST("/synthesis/delete", h.DeleteSynthesisRecord)

		// 训练文本
		voice.GET("/training-texts", h.GetTrainingTexts)

		// 一句话模式
		voice.POST("/oneshot_text", h.OneShotText)

		voice.POST("/plain_text", h.PlainText)

		// 音频处理
		voice.GET("/audio_status", h.GetAudioStatus)

		// 获取音色选项列表（根据TTS Provider）
		voice.GET("/options", h.GetVoiceOptions)
		voice.GET("/language-options", h.GetLanguageOptions)
	}
}

// registerBillingRoutes 注册计费路由
func (h *Handlers) registerBillingRoutes(r *gin.RouterGroup) {
	billing := r.Group("billing")
	billing.Use(models.AuthRequired)
	{
		// 使用量统计
		billing.GET("/statistics", h.GetUsageStatistics)
		billing.GET("/daily-usage", h.GetDailyUsageData)

		// 使用量记录
		billing.GET("/usage-records", h.GetUsageRecords)
		billing.GET("/usage-records/export", h.ExportUsageRecords)

		// 账单管理
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

// registerNodePluginRoutes Node Plugin Module
func (h *Handlers) registerNodePluginRoutes(r *gin.RouterGroup) {
	pluginHandler := NewNodePluginHandler(h.db)

	plugins := r.Group("node-plugins")
	{
		// Public routes (no auth required for browsing)
		plugins.GET("", pluginHandler.ListPlugins)
		plugins.GET("/:id", pluginHandler.GetPlugin)
	}

	// Protected routes (require authentication)
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
		// Public routes (no auth required for browsing)
		plugins.GET("", pluginHandler.ListWorkflowPlugins)
		plugins.GET("/:id", pluginHandler.GetWorkflowPlugin)
	}

	// Protected routes (require authentication)
	pluginsAuth := r.Group("workflow-plugins")
	pluginsAuth.Use(models.AuthRequired)
	{
		// 发布工作流为插件
		pluginsAuth.POST("/publish/:workflowId", pluginHandler.PublishWorkflowAsPlugin)

		// 获取工作流已发布的插件信息
		pluginsAuth.GET("/workflow/:workflowId/published", pluginHandler.GetWorkflowPublishedPlugin)

		// 插件管理
		pluginsAuth.PUT("/:id", pluginHandler.UpdateWorkflowPlugin)
		pluginsAuth.DELETE("/:id", pluginHandler.DeleteWorkflowPlugin)
		pluginsAuth.POST("/:id/publish", pluginHandler.PublishWorkflowPlugin)
		pluginsAuth.POST("/:id/install", pluginHandler.InstallWorkflowPlugin)

		// 用户插件
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
		// MCP Server Management
		mcp.GET("/servers", mcpHandler.ListMCPServers)
		mcp.GET("/servers/:id", mcpHandler.GetMCPServer)
		mcp.POST("/servers", mcpHandler.CreateMCPServer)
		mcp.PATCH("/servers/:id", mcpHandler.UpdateMCPServer)
		mcp.DELETE("/servers/:id", mcpHandler.DeleteMCPServer)
		mcp.POST("/servers/:id/enable", mcpHandler.EnableMCPServer)
		mcp.POST("/servers/:id/disable", mcpHandler.DisableMCPServer)

		// MCP Tools
		mcp.GET("/servers/:id/tools", mcpHandler.GetMCPTools)
		mcp.POST("/servers/:id/call-tool", mcpHandler.CallMCPTool)

		// MCP Logs
		mcp.GET("/servers/:id/logs", mcpHandler.GetMCPLogs)
	}
}

// registerMCPMarketplaceRoutes MCP Marketplace Module
func (h *Handlers) registerMCPMarketplaceRoutes(r *gin.RouterGroup) {
	marketplaceService := service.NewMCPMarketplaceService(h.db)
	marketplaceHandler := NewMCPMarketplaceHandler(marketplaceService)

	marketplace := r.Group("mcp/marketplace")
	{
		// Public routes (no auth required for browsing)
		marketplace.GET("", marketplaceHandler.ListMarketplace)
		marketplace.GET("/:id", marketplaceHandler.GetMarketplaceItem)
		marketplace.GET("/categories", marketplaceHandler.GetCategories)
		marketplace.GET("/featured", marketplaceHandler.GetFeaturedMCPs)
		marketplace.GET("/trending", marketplaceHandler.GetTrendingMCPs)
		marketplace.GET("/search/tag/:tag", marketplaceHandler.SearchByTag)
		marketplace.GET("/:id/reviews", marketplaceHandler.GetMCPReviews)

		// Protected routes (require authentication)
		marketplace.POST("/:id/install", models.AuthRequired, marketplaceHandler.InstallMCP)
		marketplace.DELETE("/installations/:id", models.AuthRequired, marketplaceHandler.UninstallMCP)
		marketplace.GET("/my-installations", models.AuthRequired, marketplaceHandler.GetUserInstalledMCPs)
		marketplace.PATCH("/installations/:id/config", models.AuthRequired, marketplaceHandler.UpdateInstallationConfig)
		marketplace.POST("/:id/reviews", models.AuthRequired, marketplaceHandler.ReviewMCP)
	}
}

// registerEmailLogRoutes Email Log Module
func (h *Handlers) registerEmailLogRoutes(r *gin.RouterGroup) {
	emailLog := r.Group("email-logs")
	{
		// Get email logs with pagination
		emailLog.GET("", models.AuthRequired, h.handleGetEmailLogs)
		// Get email log detail
		emailLog.GET("/:id", models.AuthRequired, h.handleGetEmailLogDetail)
		// Get email statistics
		emailLog.GET("/stats/summary", models.AuthRequired, h.handleGetEmailStats)
	}
}

// registerSendCloudWebhookRoutes SendCloud Webhook Module
func (h *Handlers) registerSendCloudWebhookRoutes(r *gin.RouterGroup) {
	webhook := r.Group("webhooks/sendcloud")
	{
		// SendCloud webhook endpoint (no auth required)
		webhook.POST("", h.handleSendCloudWebhook)
		webhook.POST("/batch", h.handleSendCloudWebhookBatch)
	}
}
