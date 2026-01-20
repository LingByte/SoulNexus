package handlers

import (
	"log"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/apidocs"
	"github.com/LingByte/SoulNexus/internal/models"
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
	sipHandler        *SipHandler
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
		searchEnabled = config.GlobalConfig.SearchEnabled
	}

	if searchEnabled {
		searchPath := utils.GetValue(db, constants.KEY_SEARCH_PATH)
		if searchPath == "" && config.GlobalConfig != nil {
			searchPath = config.GlobalConfig.SearchPath
		}
		if searchPath == "" {
			searchPath = "./search"
		}

		batchSize := utils.GetIntValue(db, constants.KEY_SEARCH_BATCH_SIZE, 100)
		if batchSize == 0 && config.GlobalConfig != nil {
			batchSize = config.GlobalConfig.SearchBatchSize
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

	// 初始化IP地理位置服务
	ipLocationService := utils.NewIPLocationService(logger.Lg)

	// 初始化SIP handler（SipServer可以通过SetSipServer方法设置）
	sipHandler := NewSipHandler(db, nil)

	return &Handlers{
		db:                db,
		wsHub:             wsHub,
		searchHandler:     searchHandler,
		ipLocationService: ipLocationService,
		sipHandler:        sipHandler,
	}
}

// SetSipServer 设置SIP服务器（用于依赖注入）
func (h *Handlers) SetSipServer(sipServer SipServerInterface) {
	if h.sipHandler != nil {
		h.sipHandler.sipServer = sipServer
	}
}

func (h *Handlers) Register(engine *gin.Engine) {

	r := engine.Group(config.GlobalConfig.APIPrefix)

	// Register Global Singleton DB
	r.Use(middleware.InjectDB(h.db))

	// Apply global middlewares (rate limiting, timeout, circuit breaker, operation log)
	middleware.ApplyGlobalMiddlewares(r)

	// Register routes regardless of whether search is enabled, check in handler methods
	// If handler is nil, try to initialize
	if h.searchHandler == nil {
		searchPath := utils.GetValue(h.db, constants.KEY_SEARCH_PATH)
		if searchPath == "" && config.GlobalConfig != nil {
			searchPath = config.GlobalConfig.SearchPath
		}
		if searchPath == "" {
			searchPath = "./search"
		}

		batchSize := utils.GetIntValue(h.db, constants.KEY_SEARCH_BATCH_SIZE, 100)
		if batchSize == 0 && config.GlobalConfig != nil {
			batchSize = config.GlobalConfig.SearchBatchSize
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
	// Register SIP routes (if SIP handler is available)
	if h.sipHandler != nil {
		h.registerSipRoutes(r)
	}
	// Register Business Module Routes
	h.registerAuthRoutes(r)
	h.registerNotificationRoutes(r)
	h.registerGroupRoutes(r)
	h.registerQuotaRoutes(r)
	h.registerAlertRoutes(r)
	h.registerWebSocketRoutes(r)
	h.registerAssistantRoutes(r)
	h.registerChatRoutes(r)
	h.registerCredentialsRoutes(r)
	h.registerKnowledgeRoutes(r)
	h.registerXunfeiTTSRoutes(r)
	h.registerVolcengineTTSRoutes(r)
	h.registerVoiceTrainingRoutes(r)
	h.registerJSTemplateRoutes(r)
	h.registerBillingRoutes(r)
	h.registerMiddlewareRoutes(r)
	h.registerWorkflowRoutes(r)
	// Register public workflow routes (no auth required)
	h.RegisterPublicWorkflowRoutes(r)
	objs := h.GetObjs()
	LingEcho.RegisterObjects(r, objs)
	if config.GlobalConfig.DocsPrefix != "" {
		var objDocs []apidocs.WebObjectDoc
		for _, obj := range objs {
			objDocs = append(objDocs, apidocs.GetWebObjectDocDefine(config.GlobalConfig.APIPrefix, obj))
		}
		apidocs.RegisterHandler(config.GlobalConfig.DocsPrefix, engine, h.GetDocs(), objDocs, h.db)
	}
	if config.GlobalConfig.AdminPrefix != "" {
		admin := r.Group(config.GlobalConfig.AdminPrefix)
		h.RegisterAdmin(admin)
	}
}

// registerAuthRoutes User Module
func (h *Handlers) registerAuthRoutes(r *gin.RouterGroup) {
	auth := r.Group(config.GlobalConfig.AuthPrefix)
	{
		// register
		auth.GET("/register", h.handleUserSignupPage)
		auth.POST("/register", h.handleUserSignup)
		auth.POST("/register/email", h.handleUserSignupByEmail)
		auth.POST("/send/email", h.handleSendEmailCode)

		// captcha
		auth.GET("/captcha", h.handleGetCaptcha)
		auth.POST("/captcha/verify", h.handleVerifyCaptcha)

		// password encryption salt
		auth.GET("/salt", h.handleGetSalt)

		// login
		auth.GET("/login", h.handleUserSigninPage)
		auth.POST("/login", h.handleUserSignin)
		auth.POST("/login/password", h.handleUserSigninByPassword)
		auth.POST("/login/email", h.handleUserSigninByEmail)

		// logout
		auth.GET("/logout", models.AuthRequired, h.handleUserLogout)
		auth.GET("/info", models.AuthRequired, h.handleUserInfo)

		// password management
		auth.GET("/reset-password", h.handleUserResetPasswordPage)
		auth.POST("/reset-password", h.handleResetPassword)
		auth.POST("/reset-password/confirm", h.handleResetPasswordConfirm)
		auth.POST("/change-password", models.AuthRequired, h.handleChangePassword)
		auth.POST("/change-password/email", models.AuthRequired, h.handleChangePasswordByEmail)

		// device management
		auth.GET("/devices", models.AuthRequired, h.handleGetUserDevices)
		auth.DELETE("/devices", models.AuthRequired, h.handleDeleteUserDevice)
		auth.POST("/devices/trust", models.AuthRequired, h.handleTrustUserDevice)
		auth.POST("/devices/untrust", models.AuthRequired, h.handleUntrustUserDevice)

		// device verification (no auth required for login flow)
		auth.POST("/devices/verify", h.handleVerifyDeviceForLogin)
		auth.POST("/devices/send-verification", h.handleSendDeviceVerificationCode)

		// email verification
		auth.GET("/verify-email", h.handleVerifyEmail)
		auth.POST("/send-email-verification", models.AuthRequired, h.handleSendEmailVerification)

		// phone verification
		auth.POST("/verify-phone", models.AuthRequired, h.handleVerifyPhone)
		auth.POST("/send-phone-verification", models.AuthRequired, h.handleSendPhoneVerification)

		// user management
		auth.PUT("/update", models.AuthRequired, h.handleUserUpdate)
		auth.PUT("/update/preferences", models.AuthRequired, h.handleUserUpdatePreferences)
		auth.POST("/update/basic/info", models.AuthRequired, h.handleUserUpdateBasicInfo)

		// notification settings
		auth.PUT("/notification-settings", models.AuthRequired, h.handleUpdateNotificationSettings)

		// user preferences
		auth.PUT("/user-preferences", models.AuthRequired, h.handleUpdateUserPreferences)

		// user stats
		auth.GET("/stats", models.AuthRequired, h.handleGetUserStats)

		// avatar upload (replace existing avatar)
		auth.POST("/avatar/upload", models.AuthRequired, h.handleUploadAvatar)

		// two-factor authentication
		auth.POST("/two-factor/setup", models.AuthRequired, h.handleTwoFactorSetup)
		auth.POST("/two-factor/enable", models.AuthRequired, h.handleTwoFactorEnable)
		auth.POST("/two-factor/disable", models.AuthRequired, h.handleTwoFactorDisable)
		auth.GET("/two-factor/status", models.AuthRequired, h.handleTwoFactorStatus)

		// user activity logs
		auth.GET("/activity", models.AuthRequired, h.handleGetUserActivity)
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

		// Search configuration routes
		system.GET("/search/status", h.GetSearchStatus)
		system.PUT("/search/config", models.AuthRequired, h.UpdateSearchConfig)
		system.POST("/search/enable", models.AuthRequired, h.EnableSearch)
		system.POST("/search/disable", models.AuthRequired, h.DisableSearch)
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

// registerDeviceRoutes Device Module (与 xiaozhi-esp32 完全一致)
func (h *Handlers) registerDeviceRoutes(r *gin.RouterGroup) {
	device := r.Group("device")

	// 获取设备配置接口（不需要认证，供xiaozhi-server调用）
	device.GET("/config/:deviceId", h.GetDeviceConfig)

	device.Use(models.AuthRequired) // 需要用户登录
	{
		// 绑定设备（激活设备）- 与 xiaozhi-esp32 路径完全一致
		device.POST("/bind/:agentId/:deviceCode", h.BindDevice)

		// 获取已绑定设备
		device.GET("/bind/:agentId", h.GetUserDevices)

		// 解绑设备
		device.POST("/unbind", h.UnbindDevice)

		// 更新设备信息
		device.PUT("/update/:id", h.UpdateDeviceInfo)

		// 手动添加设备
		device.POST("/manual-add", h.ManualAddDevice)
	}
}

// registerGroupRoutes Group Module
func (h *Handlers) registerGroupRoutes(r *gin.RouterGroup) {
	group := r.Group("group")
	group.Use(models.AuthRequired)
	{
		// 组织管理
		group.POST("", h.CreateGroup)
		group.GET("", h.ListGroups)

		// 搜索用户 - 必须在 /:id 之前
		group.GET("/search-users", h.SearchUsers)

		// 邀请管理 - 必须在 /:id 之前，否则会被匹配为 id=invitations
		group.GET("/invitations", h.ListInvitations)
		group.POST("/invitations/:id/accept", h.AcceptInvitation)
		group.POST("/invitations/:id/reject", h.RejectInvitation)

		// 概览配置管理 - 必须在 /:id 之前注册，避免路由冲突
		group.GET("/:id/overview/config", h.GetOverviewConfig)
		group.POST("/:id/overview/config", h.SaveOverviewConfig)
		group.PUT("/:id/overview/config", h.SaveOverviewConfig)
		group.DELETE("/:id/overview/config", h.DeleteOverviewConfig)

		// 组织统计数据 - 必须在 /:id 之前注册
		group.GET("/:id/statistics", h.GetGroupStatistics)

		// 组织成员管理 - 必须在 /:id 之前注册
		group.POST("/:id/leave", h.LeaveGroup)
		group.DELETE("/:id/members/:memberId", h.RemoveMember)
		group.PUT("/:id/members/:memberId/role", h.UpdateMemberRole)

		// 邀请用户 - 必须在 /:id 之前注册
		group.POST("/:id/invite", h.InviteUser)

		// 获取组织共享的资源 - 必须在 /:id 之前注册
		group.GET("/:id/resources", h.GetGroupSharedResources)

		// 上传组织头像 - 必须在 /:id 之前注册
		group.POST("/:id/avatar", h.UploadGroupAvatar)

		// 组织详情和管理 - 参数路由放在最后
		group.GET("/:id", h.GetGroup)
		group.PUT("/:id", h.UpdateGroup)
		group.DELETE("/:id", h.DeleteGroup)
	}
}

// registerQuotaRoutes 注册配额路由
func (h *Handlers) registerQuotaRoutes(r *gin.RouterGroup) {
	quota := r.Group("quota")
	quota.Use(models.AuthRequired)
	{
		// 用户配额管理
		quota.GET("/user", h.ListUserQuotas)
		quota.GET("/user/:type", h.GetUserQuota)
		quota.POST("/user", h.CreateUserQuota)
		quota.PUT("/user/:type", h.UpdateUserQuota)
		quota.DELETE("/user/:type", h.DeleteUserQuota)

		// 组织配额管理
		quota.GET("/group/:id", h.ListGroupQuotas)
		quota.GET("/group/:id/:type", h.GetGroupQuota)
		quota.POST("/group/:id", h.CreateGroupQuota)
		quota.PUT("/group/:id/:type", h.UpdateGroupQuota)
		quota.DELETE("/group/:id/:type", h.DeleteGroupQuota)
	}
}

// registerAlertRoutes 注册告警路由
func (h *Handlers) registerAlertRoutes(r *gin.RouterGroup) {
	alert := r.Group("alert")
	alert.Use(models.AuthRequired)
	{
		// 告警规则管理
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

		assistant.PUT("/:id/js", models.AuthRequired, h.UpdateAssistantJS)

		assistant.GET("/lingecho/client/:id/loader.js", h.ServeVoiceSculptorLoaderJS)

		// Assistant Tools management routes
		assistant.GET("/:id/tools", models.AuthRequired, h.ListAssistantTools)

		assistant.POST("/:id/tools", models.AuthRequired, h.CreateAssistantTool)

		assistant.PUT("/:id/tools/:toolId", models.AuthRequired, h.UpdateAssistantTool)

		assistant.DELETE("/:id/tools/:toolId", models.AuthRequired, h.DeleteAssistantTool)

		assistant.POST("/:id/tools/:toolId/test", models.AuthRequired, h.TestAssistantTool)
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

		credential.DELETE("/:id", models.AuthRequired, h.handleDeleteCredential)
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

// registerKnowledgeRoutes Knowledge Module
func (h *Handlers) registerKnowledgeRoutes(r *gin.RouterGroup) {
	knowledge := r.Group("/knowledge")
	knowledge.Use(models.AuthRequired)
	{
		//阿里创建知识库
		knowledge.POST("/create", models.AuthRequired, h.CreateKnowledgeBase)
		//阿里删除知识库
		knowledge.DELETE("/delete", models.AuthRequired, h.DeleteKnowledgeBase)
		//阿里获取知识库用户
		knowledge.GET("/get", models.AuthApiRequired, h.GetKnowledgeBase)
		//上传文件到知识库（支持多 provider）
		knowledge.POST("/upload", models.AuthRequired, h.UploadFileToKnowledgeBase)
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
	voice.GET("/lingecho/v1/", h.HandleHardwareWebSocketVoice)
	voice.Use(models.AuthRequired) // 需要认证
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

// registerSipRoutes SIP Module
func (h *Handlers) registerSipRoutes(r *gin.RouterGroup) {
	sip := r.Group("sip")
	{
		// SIP用户管理
		sip.GET("/users", models.AuthRequired, h.sipHandler.GetSipUsers)

		// 呼出相关
		sip.POST("/calls/outgoing", models.AuthRequired, h.sipHandler.MakeOutgoingCall)
		sip.GET("/calls/outgoing/:callId", models.AuthRequired, h.sipHandler.GetOutgoingCallStatus)
		sip.POST("/calls/outgoing/:callId/cancel", models.AuthRequired, h.sipHandler.CancelOutgoingCall)
		sip.POST("/calls/outgoing/:callId/hangup", models.AuthRequired, h.sipHandler.HangupOutgoingCall)

		// 通话历史
		sip.GET("/calls", models.AuthRequired, h.sipHandler.GetCallHistory)
	}
}
