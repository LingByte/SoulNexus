package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/gin-gonic/gin"
)

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
		auth.POST("/login", h.handleUserSignin)
		auth.POST("/login/password", h.handleUserSigninByPassword)
		auth.POST("/login/email", h.handleUserSigninByEmail)

		auth.GET("/logout", models.AuthRequired, h.handleUserLogout)
		auth.GET("/info", models.AuthRequired, h.handleUserInfo)

		auth.GET("/reset-password", h.handleUserResetPasswordPage)
		auth.POST("/reset-password", h.handleResetPassword)
		auth.POST("/reset-password/confirm", h.handleResetPasswordConfirm)
		auth.POST("/change-password", models.AuthRequired, h.handleChangePassword)
		auth.POST("/change-password/email", models.AuthRequired, h.handleChangePasswordByEmail)

		auth.GET("/devices", models.AuthRequired, h.handleGetUserDevices)
		auth.DELETE("/devices", models.AuthRequired, h.handleDeleteUserDevice)
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

		auth.GET("/activity", models.AuthRequired, h.handleGetUserActivity)
	}
}
