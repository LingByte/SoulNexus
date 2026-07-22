package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/dialog/chat"
	stagespeaker "github.com/LingByte/SoulNexus/pkg/dialog/stages/speaker"
	"github.com/LingByte/SoulNexus/pkg/humax"
	knconfig "github.com/LingByte/SoulNexus/pkg/knowledge/config"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	knworker "github.com/LingByte/SoulNexus/pkg/knowledge/worker"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/nlu"
	"github.com/LingByte/SoulNexus/pkg/websocket"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handlers struct {
	db       *gorm.DB
	kb       *knowledge.Service
	kbWorker *knworker.DocumentWorker
	wsHub    *websocket.Hub
}

func NewHandlers(db *gorm.DB) *Handlers {
	chat.WireSessionBridge(db)
	h := &Handlers{db: db, wsHub: websocket.NewHub()}
	h.wsHub.Start()
	stagespeaker.SetDB(db)
	cfg := knconfig.LoadConfig()
	svc, err := knowledge.NewService(context.Background(), cfg)
	if err != nil {
		logger.Warn("knowledge base init failed", zap.Error(err))
	} else {
		h.kb = svc
		h.kbWorker = knworker.NewDocumentWorker(db, cfg.IngestWorkers, h.processKnowledgeDocumentJob)
		knowledge.RegisterSyncEnqueue(h.enqueueKnowledgeSyncJob)
		knowledge.StartSyncCron(db, 30*time.Minute)
		logger.Info("knowledge document worker started", zap.Int("workers", cfg.IngestWorkers))
	}
	workflowdef.SetKnowledgeRecaller(workflowdef.NewServiceKnowledgeRecaller(db, h.kb))
	InitVoiceSessionModule()
	models.ReloadAKSKRoutePolicy(db)
	nlu.Load(db)
	models.SetAIInvocationDB(db)
	tenantcfg.SetLoader(func(ctx context.Context, tenantID uint, callID string) (tenantcfg.VoiceConfigBundle, bool) {
		return models.LoadCallVoiceConfigBundle(ctx, db, tenantID, callID)
	})
	return h
}

// Register wires the full HTTP router onto the gin engine and documents
// routes on the Huma OpenAPI surface via humax.Group.
func (h *Handlers) Register(engine *gin.Engine, api huma.API) {
	RegisterOpenAPIBodies()
	engine.GET("/.well-known/jwks.json", h.JWKSHandler)
	r := humax.NewGroup(api, engine, config.GlobalConfig.Server.APIPrefix)
	r.Use(middleware.InjectDB(h.db))

	// Public — no JWT
	h.registerSystemRoutes(r)
	h.registerTenantPublicRoutes(r)
	h.registerSendCloudWebhookRoutes(r)
	h.registerEmbedRoutes(r)
	h.RegisterPublicWorkflowRoutes(r)
	h.registerDialogPublicRoutes(r)

	protected := r.Group("")
	protected.Use(middleware.RequireTenantJWTOrAPIKey())
	protected.Use(middleware.UserRateLimit())
	protected.Use(h.credentialOperationAudit())

	// Authenticated — any valid JWT or API Key
	h.registerAccountRoutes(protected)
	h.registerSystemProtectedRoutes(protected)
	h.registerVoiceSessionRoutes(protected)
	h.registerDialogProtectedRoutes(protected)

	// Tenant RBAC — enforced inside each registrar
	h.registerTenantUserRoutes(protected)
	h.registerAssistantRoutes(protected)
	h.registerCredentialRoutes(protected)
	h.registerTenantWorkspaceRoutes(protected)
	h.registerJSTemplateRoutes(protected)
	h.registerWorkflowRoutes(protected)
	h.registerWorkflowPluginRoutes(protected)
	h.registerNodePluginRoutes(protected)
	h.registerWebhookRoutes(protected)
	h.registerIMChannelRoutes(protected)
	h.registerKnowledgeBaseRoutes(protected)
	h.registerNotificationRoutes(protected)
	h.registerAIReportRoutes(protected)
	h.registerOperationLogRoutes(protected)
	h.registerAIInvocationLogRoutes(protected)

	// Platform admin — RequirePlatformAdmin inside each registrar
	h.registerPlatformTenantRoutes(protected)
	h.registerPlatformAdminRoutes(protected)
	h.registerSystemConfigRoutes(protected)
	h.registerPlatformSystemRoutes(protected)
	h.registerPlatformNotificationAdminRoutes(protected)
	h.registerPlatformEmailTemplateAdminRoutes(protected)
	h.registerPlatformMailLogAdminRoutes(protected)
	h.registerPlatformSMSAdminRoutes(protected)
	h.registerPlatformExecutionTaskRoutes(protected)
	h.registerPlatformVoiceRoutes(protected)
	h.registerPlatformAIPoolRoutes(protected)
	h.registerTenantAIPoolGrantRoutes(protected)
	h.registerPlatformWorkflowMarketRoutes(protected)
	h.registerPlatformNLURoutes(protected)
	h.registerPlatformAssistantToolRoutes(protected)
	h.registerPlatformMcpMarketRoutes(protected)
}

// registerOperationLogRoutes mounts /operation-logs:
//   - /mine — tenant users see only their own rows (no tenantId)
//   - / — platform admins see all rows (with tenantId)
func (h *Handlers) registerOperationLogRoutes(r *humax.Group) {
	mine := r.Group("operation-logs")
	mine.Use(middleware.RequireHumanJWTUser())
	mine.GET("/mine", h.listMyOperationLogs)

	tenant := r.Group("operation-logs")
	tenant.Use(middleware.RequireTenantPermissionAll(constants.PermAPIOperationLogsRead))
	tenant.GET("/tenant", h.listTenantOperationLogs)

	admin := r.Group("operation-logs")
	admin.Use(middleware.RequirePlatformAdmin())
	admin.GET("", h.listOperationLogsPlatform)
}

// registerCredentialRoutes mounts /credentials for tenant-scoped
func (h *Handlers) registerCredentialRoutes(r *humax.Group) {
	cr := r.Group("credentials")
	cr.Use(middleware.RequireHumanJWTUser())

	crRead := cr.Group("")
	crRead.Use(middleware.RequireTenantPermissionAll("api.credentials.read"))
	{
		crRead.GET("", h.listCredentials)
		crRead.GET("/external-api", h.getCredentialExternalAPI)
		crRead.GET("/aksk-route-catalog", h.getCredentialAKSKRouteCatalog)
	}
	crWrite := cr.Group("")
	crWrite.Use(middleware.RequireTenantPermissionAll("api.credentials.write"))
	{
		crWrite.POST("", h.createCredential)
		crWrite.POST("/llm-test", h.testCredentialLLMStream)
		crWrite.PUT("/:id", h.updateCredential)
		crWrite.POST("/:id/regenerate", h.regenerateCredential)
		crWrite.POST("/:id/disable", h.disableCredential)
		crWrite.POST("/:id/enable", h.enableCredential)
		crWrite.DELETE("/:id", h.deleteCredential)
	}
}

func (h *Handlers) registerWebhookRoutes(r *humax.Group) {
	g := r.Group("webhooks")
	g.Use(middleware.RequireHumanJWTUser())
	read := g.Group("")
	read.Use(middleware.RequireTenantPermissionAll("api.webhooks.read"))
	{
		read.GET("", h.listTenantWebhooks)
		read.GET("/events", h.listWebhookEvents)
		read.GET("/:id/deliveries", h.listTenantWebhookDeliveries)
	}
	write := g.Group("")
	write.Use(middleware.RequireTenantPermissionAll("api.webhooks.write"))
	{
		write.POST("", h.createTenantWebhook)
		write.PUT("/:id", h.updateTenantWebhook)
		write.DELETE("/:id", h.deleteTenantWebhook)
		write.POST("/:id/test", h.testTenantWebhook)
	}
}

// registerAccountRoutes: /me and logout — any authenticated tenant user or platform admin (no tenant RBAC codes).
func (h *Handlers) registerAccountRoutes(r *humax.Group) {
	r.GET("/me", h.getMe)
	r.PUT("/me", h.updateMe)
	r.POST("/me/voiceprint", h.enrollMyVoiceprint)
	r.GET("/me/voiceprint", h.getMyVoiceprint)
	r.DELETE("/me/voiceprint", h.deleteMyVoiceprint)
	r.PUT("/me/password", h.updateMyPassword)
	r.PUT("/me/email", h.changeMyEmail)
	r.GET("/me/account/deletion", h.getAccountDeletionStatus)
	r.POST("/me/account/deletion/request", h.requestAccountDeletion)
	r.POST("/me/account/deletion/cancel", h.cancelAccountDeletion)
	r.GET("/me/devices", h.listMyDevices)
	r.GET("/me/login-history", h.listMyLoginHistory)
	r.POST("/me/devices/:id/trust", h.trustMyDevice)
	r.POST("/me/devices/:id/revoke", h.revokeMyDevice)
	r.DELETE("/me/devices/:id", h.deleteMyDevice)
	r.POST("/me/sessions/revoke-all", h.revokeAllMySessions)
	r.POST("/me/avatar", h.uploadMeAvatar)
	r.POST("/me/totp/setup", h.setupTotp)
	r.POST("/me/totp/enable", h.enableTotp)
	r.POST("/me/totp/disable", h.disableTotp)
	r.PUT("/me/security-preferences", h.updateSecurityPreferences)
	r.GET("/me/oauth/github/bind", h.startGitHubOAuthBind)
	r.DELETE("/me/oauth/github", h.unbindGitHubOAuth)
	r.POST("/auth/logout", h.logout)
}

// registerTenantUserRoutes mounts /tenant-users for in-tenant user
func (h *Handlers) registerTenantUserRoutes(r *humax.Group) {
	g := r.Group("tenant-users")
	tuRead := g.Group("")
	tuRead.Use(middleware.RequireTenantPermissionAll("api.tenant_users.read"))
	{
		tuRead.GET("", h.listTenantUsers)
		tuRead.GET("/stats", h.getTenantUserStats)
		tuRead.GET("/:id", h.getTenantUser)
	}
	tuWrite := g.Group("")
	tuWrite.Use(middleware.RequireTenantPermissionAll("api.tenant_users.write"))
	{
		tuWrite.POST("", h.createTenantUser)
		tuWrite.PUT("/:id", h.updateTenantUser)
		tuWrite.PUT("/:id/status", h.updateTenantUserStatus)
		tuWrite.DELETE("/:id", h.deleteTenantUser)
		tuWrite.POST("/:id/restore", h.restoreTenantUser)
	}
}

func (h *Handlers) registerTenantPublicRoutes(r *humax.Group) {
	// Throttle credential-bearing endpoints by client IP to make
	// intentionally generous (10 req / 5 min, burst 10) so a human
	authLimit := middleware.AuthRateLimiter(10, 5*time.Minute, 10)
	r.POST("/register", authLimit, h.registerTenant)
	r.POST("/login", authLimit, h.tenantLogin)
	r.POST("/forgot-password", authLimit, h.forgotPassword)
	r.POST("/account/deletion/revoke", authLimit, h.revokeAccountDeletionPublic)
	r.GET("/oauth/github/login", authLimit, h.startGitHubOAuthLogin)
	r.GET("/oauth/github/callback", h.githubOAuthCallback)
	r.POST("/oauth/github/exchange", authLimit, h.exchangeGitHubOAuthTicket)
}

// registerPlatformTenantRoutes mounts /tenants — cross-tenant
func (h *Handlers) registerPlatformTenantRoutes(r *humax.Group) {
	g := r.Group("tenants")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("", h.listTenants)
		g.GET("/:id", h.getTenant)
		g.POST("", h.createTenantPlatform)
		g.PUT("/:id", h.updateTenantPlatform)
		g.POST("/:id/llm-test", h.testTenantLLMStream)
		g.GET("/:id/credentials", h.listTenantCredentialsPlatform)
		g.POST("/:id/credentials", h.createTenantCredentialPlatform)
		g.DELETE("/:id", h.deleteTenantPlatform)
	}
}

// registerPlatformSystemRoutes mounts /system/* read-only ops endpoints
// for platform admins. See handlers/system_status.go for the data source.
//
// We intentionally separate this from /metrics:
//   - /metrics speaks Prometheus text (counters/gauges/histograms) and
//     is for time-series scrapers; it has its own IP ACL.
//   - /system/status is JSON, includes runtime.MemStats + disk-cache
//     stats, and is meant for interactive ops dashboards / human eyes.
func (h *Handlers) registerPlatformSystemRoutes(r *humax.Group) {
	g := r.Group("system")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("/status", h.getSystemStatus)
	}
}

// registerKnowledgeBaseRoutes mounts /knowledge-namespaces for knowledge base management.
func (h *Handlers) registerKnowledgeBaseRoutes(r *humax.Group) {
	g := r.Group("knowledge-namespaces")

	read := g.Group("")
	read.Use(middleware.RequireTenantPermissionAll(constants.PermAPIKBRead))
	{
		read.GET("", h.listKnowledgeNamespaces)
		read.GET("/:id", h.getKnowledgeNamespace)
		read.GET("/:id/documents", h.listKnowledgeDocuments)
		read.GET("/:id/documents/:docId", h.getKnowledgeDocument)
		read.GET("/:id/documents/:docId/chunks", h.listKnowledgeDocumentChunks)
		read.GET("/:id/documents/:docId/chunks/:chunkIndex", h.getKnowledgeDocumentChunk)
		read.GET("/:id/documents/:docId/preview", h.getKnowledgeDocumentPreview)
		read.GET("/:id/documents/:docId/progress", h.getKnowledgeDocumentProgress)
		read.GET("/:id/documents/:docId/content", h.getKnowledgeDocumentContent)
		read.POST("/:id/recall", h.recallKnowledgeDocuments)
		read.GET("/:id/chunks", h.listKnowledgeChunks)
		read.GET("/:id/chunks/export", h.exportKnowledgeChunks)
		read.GET("/:id/unanswered-questions", h.listKnowledgeUnansweredQuestions)
		read.GET("/:id/unanswered-questions/count", h.countKnowledgeUnansweredQuestions)
		read.GET("/:id/hf-questions", h.listKnowledgeHFQuestions)
		read.GET("/:id/hf-questions/daily-summary", h.getKnowledgeHFDailySummary)
		read.GET("/:id/hf-questions/:typicalId/stats", h.getKnowledgeHFQuestionStats)
		read.GET("/:id/hf-questions/:typicalId/answers", h.listKnowledgeHFQuestionAnswers)
		read.POST("/:id/analytics/quote-rate", h.getKnowledgeQuoteRateReport)
		read.GET("/:id/sync-sources", h.listKnowledgeSyncSources)
		read.GET("/:id/worker/stats", h.getKnowledgeWorkerStats)
		read.GET("/:id/eval/datasets", h.listKnowledgeEvalDatasets)
		read.GET("/:id/eval/jobs/:jobId", h.getKnowledgeEvalJob)
	}

	write := g.Group("")
	write.Use(middleware.RequireTenantPermissionAll(constants.PermAPIKBWrite))
	{
		write.POST("", h.createKnowledgeNamespace)
		write.PUT("/:id", h.updateKnowledgeNamespace)
		write.DELETE("/:id", h.deleteKnowledgeNamespace)
		write.POST("/:id/documents", h.uploadKnowledgeDocument)
		write.POST("/:id/documents/:docId/confirm-index", h.confirmKnowledgeDocumentIndex)
		write.PUT("/:id/documents/:docId", h.updateKnowledgeDocument)
		write.DELETE("/:id/documents/:docId", h.deleteKnowledgeDocument)
		write.POST("/:id/chunks", h.createKnowledgeChunk)
		write.PUT("/:id/chunks/:chunkId", h.updateKnowledgeChunk)
		write.DELETE("/:id/chunks/:chunkId", h.deleteKnowledgeChunk)
		write.POST("/:id/unanswered-questions/:questionId/resolve", h.resolveKnowledgeUnansweredQuestion)
		write.POST("/:id/unanswered-questions/:questionId/draft-answer", h.draftKnowledgeUnansweredAnswer)
		write.DELETE("/:id/unanswered-questions/:questionId", h.deleteKnowledgeUnansweredQuestion)
		write.POST("/:id/eval/run", h.runKnowledgeEval)
		write.POST("/:id/eval/compare", h.compareKnowledgeEvalStrategies)
		write.POST("/:id/eval/datasets", h.createKnowledgeEvalDataset)
		write.DELETE("/:id/eval/datasets/:datasetId", h.deleteKnowledgeEvalDataset)
		write.POST("/:id/sync-sources", h.createKnowledgeSyncSource)
		write.PUT("/:id/sync-sources/:sourceId", h.updateKnowledgeSyncSource)
		write.DELETE("/:id/sync-sources/:sourceId", h.deleteKnowledgeSyncSource)
		write.POST("/:id/sync-sources/:sourceId/trigger", h.triggerKnowledgeSyncNow)
	}
}
