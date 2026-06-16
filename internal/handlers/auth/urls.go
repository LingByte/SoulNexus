package handlers

import (
	authmodel "github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type Handlers struct {
	db                *gorm.DB
	ipLocationService *utils.IPLocationService
}

// NewUserServiceHandlers returns handlers for the standalone user (auth) service binary.
// It omits WebSocket hub and search engine wiring that auth routes do not use.
func NewUserServiceHandlers(db *gorm.DB) *Handlers {
	return &Handlers{
		db:                db,
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
	h.registerAuthAdminManagementRoutes(r)

	// Browser HTML pages need DB (GetRenderPageContext); API group prefix is not used here.
	browser := engine.Group("")
	browser.Use(middleware.InjectDB(h.db))
	{
		browser.GET("/login", h.RenderSigninPage)
		browser.GET("/login/revoke-account-deletion", h.RenderAccountDeletionRevokePage)
	}
}

// registerAuthAdminManagementRoutes registers user/role/permission/passkey and related
// admin surfaces on the standalone auth service (cmd/auth).
func (h *Handlers) registerAuthAdminManagementRoutes(r *gin.RouterGroup) {
	adminGuard := []gin.HandlerFunc{authmodel.AuthRequired, middleware.RequireAdmin}

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

	mePasskeys := r.Group("me/passkeys", authmodel.AuthRequired)
	{
		mePasskeys.GET("", h.handleMeListPasskeys)
		mePasskeys.POST("/registration/begin", h.handleMePasskeyRegistrationBegin)
		mePasskeys.POST("/registration/finish", h.handleMePasskeyRegistrationFinish)
		mePasskeys.PUT("/:id", h.handleMeUpdatePasskey)
		mePasskeys.DELETE("/:id", h.handleMeDeletePasskey)
	}

	meTwoFA := r.Group("me/twofa", authmodel.AuthRequired)
	{
		meTwoFA.GET("/status", h.handleMeTwoFAStatus)
		meTwoFA.POST("/backup-codes/regenerate", h.handleMeTwoFABackupCodesRegenerate)
		meTwoFA.POST("/backup-codes/use", h.handleMeTwoFABackupCodeUse)
		meTwoFA.POST("/reset", h.handleMeTwoFAReset)
	}

	security := r.Group("security", adminGuard...)
	{
		security.GET("/operation-logs", h.handleAdminListOperationLogs)
		security.GET("/operation-logs/:id", h.handleAdminGetOperationLog)
		security.GET("/login-history", h.handleAdminListLoginHistory)
		security.GET("/login-history/:id", h.handleAdminGetLoginHistory)
		security.GET("/account-locks", h.handleAdminListAccountLocks)
		security.POST("/account-locks/:id/unlock", h.handleAdminUnlockAccount)
	}
}

func (h *Handlers) registerAccessAdminRoutes(r *gin.RouterGroup) {
	guard := []gin.HandlerFunc{authmodel.AuthRequired, middleware.RequireAdmin, h.requireAccessManage}

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
		auth.GET("/wechat/bind/code", authmodel.AuthRequired, h.handleWechatBindCode)
		auth.GET("/wechat/bind/status", authmodel.AuthRequired, h.handleWechatBindStatus)
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
		auth.GET("/info", authmodel.AuthRequired, h.handleUserInfo)

		auth.GET("/reset-password", h.handleUserResetPasswordPage)
		auth.POST("/reset-password", h.handleResetPassword)
		auth.POST("/reset-password/confirm", h.handleResetPasswordConfirm)
		auth.POST("/change-password", authmodel.AuthRequired, h.handleChangePassword)
		auth.POST("/change-password/email", authmodel.AuthRequired, h.handleChangePasswordByEmail)

		auth.GET("/devices", authmodel.AuthRequired, h.handleGetUserDevices)
		auth.DELETE("/devices/:deviceId", authmodel.AuthRequired, h.handleDeleteUserDevice)
		auth.POST("/devices/trust", authmodel.AuthRequired, h.handleTrustUserDevice)
		auth.POST("/devices/untrust", authmodel.AuthRequired, h.handleUntrustUserDevice)

		auth.POST("/devices/verify", h.handleVerifyDeviceForLogin)
		auth.POST("/devices/send-verification", h.handleSendDeviceVerificationCode)

		auth.POST("/email/send-current-code", authmodel.AuthRequired, h.handleSendCurrentEmailCode)
		auth.POST("/email/bind", authmodel.AuthRequired, h.handleBindEmail)
		auth.POST("/email/change", authmodel.AuthRequired, h.handleChangeEmail)

		auth.POST("/verify-phone", authmodel.AuthRequired, h.handleVerifyPhone)
		auth.POST("/send-phone-verification", authmodel.AuthRequired, h.handleSendPhoneVerification)

		auth.PUT("/update", authmodel.AuthRequired, h.handleUserUpdate)
		auth.PUT("/update/preferences", authmodel.AuthRequired, h.handleUserUpdatePreferences)
		auth.POST("/update/basic/info", authmodel.AuthRequired, h.handleUserUpdateBasicInfo)

		auth.PUT("/notification-settings", authmodel.AuthRequired, h.handleUpdateNotificationSettings)

		auth.PUT("/user-preferences", authmodel.AuthRequired, h.handleUpdateUserPreferences)

		auth.GET("/stats", authmodel.AuthRequired, h.handleGetUserStats)

		auth.POST("/avatar/upload", authmodel.AuthRequired, h.handleUploadAvatar)

		auth.POST("/two-factor/setup", authmodel.AuthRequired, h.handleTwoFactorSetup)
		auth.POST("/two-factor/enable", authmodel.AuthRequired, h.handleTwoFactorEnable)
		auth.POST("/two-factor/disable", authmodel.AuthRequired, h.handleTwoFactorDisable)
		auth.GET("/two-factor/status", authmodel.AuthRequired, h.handleTwoFactorStatus)

		// 无密码 / Passkey 登录：discoverable，浏览器经 navigator.credentials.get 完成。
		auth.POST("/passkey/begin", h.handleAuthPasskeyBegin)
		auth.POST("/passkey/finish", h.handleAuthPasskeyFinish)

		auth.GET("/activity", authmodel.AuthRequired, h.handleGetUserActivity)

		auth.POST("/account-deletion/send-cancel-code", h.handleAccountDeletionSendCancelCode)
		auth.POST("/account-deletion/cancel-by-email", h.handleAccountDeletionCancelByEmail)

		auth.GET("/account-deletion/eligibility", authmodel.AuthRequired, h.handleAccountDeletionEligibility)
		auth.POST("/account-deletion/send-email-code", authmodel.AuthRequired, h.handleAccountDeletionSendEmailCode)
		auth.POST("/account-deletion/request", authmodel.AuthRequired, h.handleAccountDeletionRequest)
		auth.POST("/account-deletion/cancel", authmodel.AuthRequired, h.handleAccountDeletionCancel)
		auth.DELETE("/bindings/github", authmodel.AuthRequired, h.handleUnbindGitHub)
		auth.DELETE("/bindings/wechat", authmodel.AuthRequired, h.handleUnbindWechat)

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

func operatorEmail(c *gin.Context) string {
	if cur := authmodel.CurrentUser(c); cur != nil && cur.Email != "" {
		return cur.Email
	}
	return "system"
}
