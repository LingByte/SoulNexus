package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strconv"
	"time"

	LingEcho "github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

type SeedService struct {
	db *gorm.DB
}

func (s *SeedService) SeedAll() error {
	if err := s.seedMinimalRolesIfEmpty(); err != nil {
		return err
	}
	if err := s.seedConfigs(); err != nil {
		return err
	}
	if err := s.seedAdminUsers(); err != nil {
		return err
	}
	if err := s.seedAssistants(); err != nil {
		return err
	}
	if err := s.seedMailTemplates(); err != nil {
		return err
	}
	return nil
}

// seedMailTemplates 把 templates/email/*.html 内嵌模板写入 mail_templates 表（org_id=0 系统级）。
// 已存在 (org_id=0, code, locale="") 记录则跳过，便于管理员后续在后台修改。
func (s *SeedService) seedMailTemplates() error {
	// MySQL 下确保 mail 相关表使用 utf8mb4，避免旧 utf8mb3 列与 emoji 冲突 (Error 3988)。
	if s.db.Dialector.Name() == "mysql" {
		for _, tbl := range []string{"mail_templates", "mail_logs", "sms_logs"} {
			_ = s.db.Exec("ALTER TABLE " + tbl + " CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
		}
	}
	type tplDef struct {
		code, name, subject, html, desc string
	}
	defs := []tplDef{
		{notification.TmplWelcome, "欢迎邮件", "欢迎加入 LingEcho", LingEcho.WelcomeHTML, "用户注册成功欢迎邮件"},
		{notification.TmplVerification, "通用验证码", "您的 LingEcho 验证码", LingEcho.VerificationHTML, "通用 6 位验证码邮件"},
		{notification.TmplEmailVerification, "邮箱验证", "请验证您的邮箱地址", LingEcho.EmailVerificationHTML, "注册后邮箱地址验证邮件"},
		{notification.TmplPasswordReset, "密码重置", "密码重置请求", LingEcho.PasswordResetHTML, "密码重置链接邮件"},
		{notification.TmplDeviceVerification, "设备验证码", "设备验证码", LingEcho.DeviceVerificationHTML, "新设备登录二次验证邮件"},
		{notification.TmplGroupInvitation, "组织邀请", "您收到了来自 {{.InviterName}} 的组织邀请", LingEcho.GroupInvitationHTML, "组织 / 团队邀请邮件"},
		{notification.TmplNewDeviceLogin, "新设备登录提醒", "{{if .IsSuspicious}}可疑登录警告{{else}}新设备登录提醒{{end}}", LingEcho.NewDeviceLoginHTML, "新设备 / 异地登录提醒"},
	}
	for _, d := range defs {
		var n int64
		if err := s.db.Model(&models.MailTemplate{}).
			Where("org_id = ? AND code = ? AND locale = ?", 0, d.code, "").
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		tpl := &models.MailTemplate{
			OrgID:       0,
			Code:        d.code,
			Name:        d.name,
			Subject:     d.subject,
			Description: d.desc,
			Locale:      "",
			Enabled:     true,
		}
		models.ApplyMailTemplateHTMLDerivedFields(tpl, d.html, "")
		if err := s.db.Create(tpl).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SeedService) seedConfigs() error {
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	defaults := []utils.Config{
		{Key: constants.KEY_SITE_URL, Desc: "Site URL", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.URL != "" {
				return config.GlobalConfig.Server.URL
			}
			return "https://lingecho.com"
		}()},
		{Key: constants.KEY_SITE_NAME, Desc: "Site Name", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.Name != "" {
				return config.GlobalConfig.Server.Name
			}
			return "SoulNexus"
		}()},
		{Key: constants.KEY_SITE_LOGO_URL, Desc: "Site Logo", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.Logo != "" {
				return config.GlobalConfig.Server.Logo
			}
			return "/static/img/favicon.png"
		}()},
		{Key: constants.KEY_SITE_DESCRIPTION, Desc: "Site Description", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.Desc != "" {
				return config.GlobalConfig.Server.Desc
			}
			return "SoulNexus - Intelligent Voice Customer Service Platform"
		}()},
		{Key: constants.KEY_SITE_TERMS_URL, Desc: "Terms of Service", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.TermsURL != "" {
				return config.GlobalConfig.Server.TermsURL
			}
			return "https://lingecho.com"
		}()},
		{Key: constants.KEY_SITE_SIGNIN_URL, Desc: "Sign In Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/login"},
		{Key: constants.KEY_SITE_FAVICON_URL, Desc: "Favicon URL", Autoload: true, Public: true, Format: "text", Value: "/static/img/favicon.png"},
		{Key: constants.KEY_SITE_SIGNUP_URL, Desc: "Sign Up Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/register"},
		{Key: constants.KEY_SITE_LOGOUT_URL, Desc: "Logout Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/logout"},
		{Key: constants.KEY_SITE_RESET_PASSWORD_URL, Desc: "Reset Password Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/reset-password"},
		{Key: constants.KEY_SITE_SIGNIN_API, Desc: "Sign In API", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/login"},
		{Key: constants.KEY_SITE_SIGNUP_API, Desc: "Sign Up API", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/register"},
		{Key: constants.KEY_SITE_RESET_PASSWORD_DONE_API, Desc: "Reset Password API", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/reset-password-done"},
		{Key: constants.KEY_SITE_LOGIN_NEXT, Desc: "Login Redirect Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/admin/"},
		{Key: constants.KEY_SITE_USER_ID_TYPE, Desc: "User ID Type", Autoload: true, Public: true, Format: "text", Value: "email"},
		{Key: constants.KEY_SEARCH_ENABLED, Desc: "Search Feature Enabled", Autoload: true, Public: true, Format: "bool", Value: func() string {
			if config.GlobalConfig.Features.SearchEnabled {
				return "true"
			}
			return "false"
		}()},
		{Key: constants.KEY_SEARCH_PATH, Desc: "Search Index Path", Autoload: true, Public: false, Format: "text", Value: func() string {
			if config.GlobalConfig.Features.SearchPath != "" {
				return config.GlobalConfig.Features.SearchPath
			}
			return "./search"
		}()},
		{Key: constants.KEY_SEARCH_BATCH_SIZE, Desc: "Search Batch Size", Autoload: true, Public: false, Format: "int", Value: func() string {
			if config.GlobalConfig.Features.SearchBatchSize > 0 {
				return strconv.Itoa(config.GlobalConfig.Features.SearchBatchSize)
			}
			return "100"
		}()},
		{Key: constants.KEY_SEARCH_INDEX_SCHEDULE, Desc: "Search Index Schedule (Cron)", Autoload: true, Public: false, Format: "text", Value: "0 */6 * * *"}, // Execute every 6 hours
		{Key: constants.KEY_SERVER_WEBSOCKET, Desc: "SERVER WEBSOCKET", Autoload: true, Public: false, Format: "text", Value: "wss://lingecho.com/api/voice/websocket/voice/lingecho/v1/"},
		{Key: constants.KEY_STORAGE_KIND, Desc: "Storage Kind", Autoload: true, Public: true, Format: "text", Value: "qiniu"},
	}
	for _, cfg := range defaults {
		var count int64
		err := s.db.Model(&utils.Config{}).Where("`key` = ?", cfg.Key).Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			if err := s.db.Create(&cfg).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// seedMinimalRolesIfEmpty inserts one generic role when the table is empty so sign-up / seed users can satisfy user_roles.
func (s *SeedService) seedMinimalRolesIfEmpty() error {
	var n int64
	if err := s.db.Model(&models.Role{}).Where("is_deleted = ?", models.SoftDeleteStatusActive).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return s.db.Create(&models.Role{Name: "Member", Slug: "member", IsSystem: true}).Error
}

func (s *SeedService) seedAdminUsers() error {
	defaultAdmins := []models.User{
		{
			Email:    "admin@lingecho.com",
			Password: models.HashPassword("admin123"),
			Status:   models.UserStatusActive,
			Source:   models.UserSourceAdmin,
		},
		{
			Email:    "19511899044@163.com",
			Password: models.HashPassword("admin123"),
			Status:   models.UserStatusActive,
			Source:   models.UserSourceAdmin,
		},
	}

	for _, user := range defaultAdmins {
		var count int64
		err := s.db.Model(&models.User{}).Where("`email` = ?", user.Email).Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			if err := s.db.Create(&user).Error; err != nil {
				return err
			}
			_ = models.UpdateUserProfileFields(s.db, user.ID, map[string]any{"display_name": "Administrator"})
			if err := models.EnsureUserHasOneRole(s.db, user.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SeedService) seedAssistants() error {
	var count int64
	if err := s.db.Model(&models.Agent{}).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil // Data already exists, skip
	}

	g2, err := models.EnsurePersonalGroupForUser(s.db, 2)
	if err != nil {
		return err
	}
	g1, err := models.EnsurePersonalGroupForUser(s.db, 1)
	if err != nil {
		return err
	}

	defaultAssistant := []models.Agent{
		{
			GroupID:      g2.ID,
			CreatedBy:    2,
			Name:         "Technical Support",
			Description:  "Provides technical support and answers various technical support questions",
			SystemPrompt: "You are a professional technical support engineer, focused on helping users solve technology-related problems.",
			PersonaTag:   "support",
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			GroupID:      g2.ID,
			CreatedBy:    2,
			Name:         "Smart Assistant",
			Description:  "Smart assistant providing various intelligent services",
			SystemPrompt: "You are a smart assistant, please answer user questions as an assistant.",
			PersonaTag:   "assistant",
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			GroupID:      g1.ID,
			CreatedBy:    1,
			Name:         "Mentor",
			Description:  "Mentor providing various guidance services",
			SystemPrompt: "You are a mentor, please answer user questions as a mentor.",
			PersonaTag:   "mentor",
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			GroupID:      g1.ID,
			CreatedBy:    1,
			Name:         "Assistant",
			Description:  "An assistant that you can use to answer your questions.",
			SystemPrompt: "You are an assistant, please answer user questions as an assistant.",
			PersonaTag:   "assistant",
			JsSourceID:   strconv.Itoa(1),
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for i := range defaultAssistant {
		defaultAssistant[i].JsSourceID = strconv.FormatInt(utils.SnowflakeUtil.NextID(), 20)
		if err := s.db.Create(&defaultAssistant[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
