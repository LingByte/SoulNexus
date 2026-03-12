package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func InitUserListeners() {
	logger.Info("Initializing user listeners...")

	// Handle after user registration success
	utils.Sig().Connect(constants.SigUserCreate, func(sender any, params ...any) {
		if len(params) < 2 {
			return
		}
		user, ok := sender.(*models.User)
		if !ok {
			return
		}

		db, ok := params[0].(*gorm.DB)
		if !ok {
			return
		}

		logger.Info("User registered successfully", zap.Uint("userId", user.ID), zap.String("email", user.Email))

		// Send welcome email
		go sendWelcomeEmail(user, db)

		// Log user registration event
		logUserEvent(user, "user_created", "User registered successfully")
	})

	// Handle after user login
	utils.Sig().Connect(constants.SigUserLogin, func(sender any, params ...any) {
		user, ok := sender.(*models.User)
		if !ok {
			return
		}

		logger.Info("User logged in", zap.Uint("userId", user.ID), zap.String("email", user.Email))

		// Send login notification
		go sendWelcomeEmail(user, params[0].(*gorm.DB))

		notification.NewInternalNotificationService(params[0].(*gorm.DB)).Send(user.ID,
			"Welcome back",
			"Dear "+user.DisplayName+", welcome back to LingEcho AI voice platform! You have successfully logged into the system.")

		// Log login event
		logUserEvent(user, "user_login", "User login")
	})

	// Handle after user logout
	utils.Sig().Connect(constants.SigUserLogout, func(sender any, params ...any) {
		if len(params) < 1 {
			return
		}
		user, ok := params[0].(*models.User)
		if !ok {
			return
		}

		logger.Info("User logged out", zap.Uint("userId", user.ID), zap.String("email", user.Email))

		// Log logout event
		go logUserEvent(user, "user_logout", "User logout")
	})

	// User email verification
	utils.Sig().Connect(constants.SigUserVerifyEmail, func(sender any, params ...any) {
		logger.Info("SigUserVerifyEmail signal received")
		if len(params) < 4 {
			logger.Warn("SigUserVerifyEmail: insufficient parameters", zap.Int("paramCount", len(params)))
			return
		}
		user, ok := sender.(*models.User)
		if !ok {
			logger.Warn("SigUserVerifyEmail: invalid user type")
			return
		}
		hash, ok := params[0].(string)
		if !ok {
			logger.Warn("SigUserVerifyEmail: invalid hash type")
			return
		}
		clientIp, ok := params[1].(string)
		if !ok {
			logger.Warn("SigUserVerifyEmail: invalid clientIp type")
			return
		}
		userAgent, ok := params[2].(string)
		if !ok {
			logger.Warn("SigUserVerifyEmail: invalid userAgent type")
			return
		}
		db, ok := params[3].(*gorm.DB)
		if !ok {
			logger.Warn("SigUserVerifyEmail: invalid db type")
			return
		}

		logger.Info("Sending email verification", zap.Uint("userId", user.ID), zap.String("email", user.Email))

		// Send email verification
		go sendEmailVerification(user, hash, clientIp, userAgent, db)
	})

	// User password reset
	utils.Sig().Connect(constants.SigUserResetPassword, func(sender any, params ...any) {
		if len(params) < 5 {
			return
		}
		user, ok := params[0].(*models.User)
		if !ok {
			return
		}
		hash, ok := params[1].(string)
		if !ok {
			return
		}
		clientIp, ok := params[2].(string)
		if !ok {
			return
		}
		userAgent, ok := params[3].(string)
		if !ok {
			return
		}
		db, ok := params[4].(*gorm.DB)
		if !ok {
			return
		}

		logger.Info("Sending password reset email", zap.Uint("userId", user.ID), zap.String("email", user.Email))

		// Send password reset email
		sendPasswordResetEmail(user, hash, clientIp, userAgent, db)
	})

	// Handle new device login alert
	utils.Sig().Connect(constants.SigUserNewDeviceLogin, func(sender any, params ...any) {
		logger.Info("SigUserNewDeviceLogin signal received")
		if len(params) < 2 {
			logger.Warn("SigUserNewDeviceLogin: insufficient parameters", zap.Int("paramCount", len(params)))
			return
		}
		user, ok := sender.(*models.User)
		if !ok {
			logger.Warn("SigUserNewDeviceLogin: invalid user type")
			return
		}
		deviceInfo, ok := params[0].(map[string]interface{})
		if !ok {
			logger.Warn("SigUserNewDeviceLogin: invalid deviceInfo type")
			return
		}
		db, ok := params[1].(*gorm.DB)
		if !ok {
			logger.Warn("SigUserNewDeviceLogin: invalid db type")
			return
		}

		logger.Info("Sending new device login alert", zap.Uint("userId", user.ID), zap.String("email", user.Email))

		// Send new device login alert email
		go sendNewDeviceLoginAlert(user, deviceInfo, db)
	})

	logger.Info("User module listeners initialized successfully")
}

// sendWelcomeEmail sends welcome email
func sendWelcomeEmail(user *models.User, db *gorm.DB) {
	if config.GlobalConfig.Services.Mail.APIUser == "" || config.GlobalConfig.Services.Mail.From == "" || config.GlobalConfig.Services.Mail.APIKey == "" {
		logger.Warn("Mail configuration not set, skipping sending login notification")
		return
	}

	if user.EmailNotifications {
		mailer := notification.NewMailNotificationWithDB(config.GlobalConfig.Services.Mail, db, user.ID)
		err := mailer.SendWelcomeEmail(
			user.Email,
			user.DisplayName,
			utils.GetValue(db, constants.KEY_SITE_URL), // Welcome page link
		)

		if err != nil {
			logger.Error("Failed to send welcome email", zap.Error(err), zap.String("email", user.Email))
		} else {
			logger.Info("Welcome email sent successfully", zap.String("email", user.Email))
		}
	}
}

// sendEmailVerification sends email verification
func sendEmailVerification(user *models.User, hash, clientIp, userAgent string, db *gorm.DB) {
	logger.Info("Starting email verification process",
		zap.String("email", user.Email),
		zap.String("hash", hash))

	if config.GlobalConfig.Services.Mail.APIUser == "" {
		logger.Warn("Mail configuration not set, skipping sending email verification")
		return
	}

	// Get site URL
	siteURL := utils.GetValue(db, constants.KEY_SITE_URL)
	if siteURL == "" {
		siteURL = "http://localhost:3000" // Default value
	}

	// Build verification URL
	verifyUrl := siteURL + "/verify-email?token=" + hash

	logger.Info("Preparing to send email verification",
		zap.String("email", user.Email),
		zap.String("verifyUrl", verifyUrl),
		zap.String("mailAPIUser", config.GlobalConfig.Services.Mail.APIUser))

	mailer := notification.NewMailNotificationWithDB(config.GlobalConfig.Services.Mail, db, user.ID)
	err := mailer.SendVerificationEmail(user.Email, user.DisplayName, verifyUrl)
	if err != nil {
		logger.Error("Failed to send email verification", zap.Error(err), zap.String("email", user.Email))
	} else {
		logger.Info("Email verification sent successfully", zap.String("email", user.Email), zap.String("verifyUrl", verifyUrl))
	}
}

// sendPasswordResetEmail sends password reset email
func sendPasswordResetEmail(user *models.User, hash, clientIp, userAgent string, db *gorm.DB) {
	if config.GlobalConfig.Services.Mail.APIUser == "" {
		logger.Warn("Mail configuration not set, skipping sending password reset email")
		return
	}

	// Get site URL
	siteURL := utils.GetValue(db, constants.KEY_SITE_URL)
	if siteURL == "" {
		siteURL = "http://localhost:3000" // Default value
	}

	// Build password reset URL
	resetUrl := siteURL + "/reset-password?token=" + hash

	mailer := notification.NewMailNotificationWithDB(config.GlobalConfig.Services.Mail, db, user.ID)
	err := mailer.SendPasswordResetEmail(user.Email, user.DisplayName, resetUrl)
	if err != nil {
		logger.Error("Failed to send password reset email", zap.Error(err), zap.String("email", user.Email))
	} else {
		logger.Info("Password reset email sent successfully", zap.String("email", user.Email), zap.String("resetUrl", resetUrl))
	}
}

// logUserEvent logs user events
func logUserEvent(user *models.User, eventType, description string) {
	// Here you can log user events to database or logging system
	logger.Info("User event recorded",
		zap.Uint("userId", user.ID),
		zap.String("eventType", eventType),
		zap.String("description", description),
	)
}

// sendNewDeviceLoginAlert sends new device login alert email
func sendNewDeviceLoginAlert(user *models.User, deviceInfo map[string]interface{}, db *gorm.DB) {
	if config.GlobalConfig.Services.Mail.APIUser == "" {
		logger.Warn("Mail configuration not set, skipping sending new device login alert")
		return
	}

	// Extract device information from the map
	clientIP, _ := deviceInfo["clientIP"].(string)
	location, _ := deviceInfo["location"].(string)
	deviceType, _ := deviceInfo["deviceType"].(string)
	os, _ := deviceInfo["os"].(string)
	browser, _ := deviceInfo["browser"].(string)
	isSuspicious, _ := deviceInfo["isSuspicious"].(bool)
	loginTime, _ := deviceInfo["loginTime"].(string)

	// Get display name
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Email
	}

	// Get URLs from configuration
	siteURL := utils.GetValue(db, constants.KEY_SITE_URL)
	if siteURL == "" {
		siteURL = "http://localhost:3000" // Default value
	}

	securityURL := siteURL + "/security"       // Security settings page
	changePasswordURL := siteURL + "/password" // Change password page

	// Send the alert email
	mailer := notification.NewMailNotificationWithDB(config.GlobalConfig.Services.Mail, db, user.ID)
	err := mailer.SendNewDeviceLoginAlert(
		user.Email,
		displayName,
		loginTime,
		clientIP,
		location,
		deviceType,
		os,
		browser,
		isSuspicious,
		securityURL,
		changePasswordURL,
	)

	if err != nil {
		logger.Error("Failed to send new device login alert email",
			zap.Error(err),
			zap.String("email", user.Email),
			zap.String("deviceID", deviceInfo["deviceID"].(string)))
	} else {
		logger.Info("New device login alert email sent",
			zap.String("email", user.Email),
			zap.String("deviceID", deviceInfo["deviceID"].(string)),
			zap.Bool("isSuspicious", isSuspicious))
	}
}
