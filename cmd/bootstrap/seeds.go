package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultPlatformAdminEmail       = "admin@lingecho.com"
	defaultPlatformAdminPassword    = "admin123"
	defaultPlatformAdminDisplayName = "Platform Admin"
	envPlatformAdminEmail           = "PLATFORM_ADMIN_EMAIL"
	envPlatformAdminPassword        = "PLATFORM_ADMIN_PASSWORD"
	envPlatformAdminDisplayName     = "PLATFORM_ADMIN_DISPLAY_NAME"
)

func platformAdminSeedEmail() string {
	if v := strings.TrimSpace(utils.GetEnv(envPlatformAdminEmail)); v != "" {
		return strings.ToLower(v)
	}
	return defaultPlatformAdminEmail
}

func platformAdminSeedPassword() string {
	if v := utils.GetEnv(envPlatformAdminPassword); strings.TrimSpace(v) != "" {
		return v
	}
	return defaultPlatformAdminPassword
}

func platformAdminSeedPasswordAllowed(password string) error {
	mode := ""
	if config.GlobalConfig != nil {
		mode = strings.ToLower(strings.TrimSpace(config.GlobalConfig.Server.Mode))
	}
	if mode != pkgconst.ENV_PROD && mode != "production" {
		return nil
	}
	pw := strings.TrimSpace(password)
	if pw == "" || pw == defaultPlatformAdminPassword || len(pw) < 10 {
		return errors.New("production seed requires PLATFORM_ADMIN_PASSWORD (min 10 chars, not admin123)")
	}
	return nil
}

func platformAdminSeedDisplayName() string {
	if v := strings.TrimSpace(utils.GetEnv(envPlatformAdminDisplayName)); v != "" {
		return v
	}
	return defaultPlatformAdminDisplayName
}

type SeedService struct {
	db *gorm.DB
}

func (s *SeedService) SeedAll() error {
	if err := s.seedConfigs(); err != nil {
		return err
	}
	if err := s.seedPermissions(); err != nil {
		return err
	}
	if err := s.seedPlatformAdmin(); err != nil {
		return err
	}
	if err := s.seedMailTemplates(); err != nil {
		return err
	}
	return nil
}

// seedPermissions syncs the global permission catalog and binds every system「管理员」role
// to all catalog rows (idempotent). Safe to call after init-sql tenant/role inserts.
func (s *SeedService) seedPermissions() error {
	if s == nil || s.db == nil {
		return nil
	}
	if err := models.SyncPermissionCatalog(s.db); err != nil {
		return err
	}
	return models.BackfillSystemTenantAdminPermissions(s.db, "seed")
}

func (s *SeedService) seedConfigs() error {
	defaults := []utils.Config{
		{Key: constants.KEY_SITE_URL, Desc: "站点地址", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.URL != "" {
				return config.GlobalConfig.Server.URL
			}
			return pkgconst.DefaultSiteURL
		}()},
		{Key: constants.KEY_SITE_NAME, Desc: "站点名称", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.Name != "" {
				return config.GlobalConfig.Server.Name
			}
			return "SoulNexus"
		}()},
		{Key: constants.KEY_SITE_DESCRIPTION, Desc: "站点描述", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.Desc != "" {
				return config.GlobalConfig.Server.Desc
			}
			return "SoulNexus - Intelligent Voice Customer Service Platform"
		}()},
		{Key: constants.KEY_SITE_LOGO, Desc: "站点 Logo", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.Server.Logo != "" {
				return config.GlobalConfig.Server.Logo
			}
			return constants.DefaultSiteLogoPath
		}()},
		{Key: constants.KEY_SITE_TERMS_URL, Desc: "服务条款链接", Autoload: true, Public: true, Format: "text", Value: func() string {
			return config.GlobalConfig.Server.TermsURL
		}()},
		{
			Key: constants.KEY_API_AKSK_ROUTE_POLICY, Autoload: true, Public: false, Format: "json",
			Value: `{"enabled":false,"routeIds":[]}`,
			Desc:  "API Key 路由白名单。enabled=true 时 routeIds 为接口 catalog id 列表（见 GET /system-configs/route-policy/catalog）；默认全部关闭。",
		},
		{
			Key: constants.KEY_TENANT_SELF_REGISTER, Autoload: true, Public: true, Format: "bool", Value: "false",
			Desc: "是否开放租户自助注册（POST /api/register）。true=开放；false=关闭，需平台管理员在租户管理中开通。修改后立即生效，无需重启。",
		},
		{
			Key: constants.KEY_NLU_MODEL, Autoload: true, Public: false, Format: "text", Value: "",
			Desc: "ONNX 意图分类模型路径（.onnx）。需同时在 .env 设置 NLU_ENABLED=true。",
		},
		{
			Key: constants.KEY_NLU_TOKENIZER, Autoload: true, Public: false, Format: "text", Value: "data/nlu/tokenizer.json",
			Desc: "Hugging Face tokenizer.json 路径，与训练导出模型配套。",
		},
		{
			Key: constants.KEY_NLU_INTENTS_CONFIG, Autoload: true, Public: false, Format: "text", Value: "",
			Desc: "意图配置 JSON 路径（可选）；留空使用内置 default_intents.json。",
		},
		{
			Key: constants.KEY_NLU_MIN_CONFIDENCE, Autoload: true, Public: false, Format: "float", Value: "0.22",
			Desc: "意图置信度下限（0~1）。低于该值时实验室解析结果 channel=llm（不展示固定话术）。",
		},
		{
			Key: constants.KEY_CONTENT_CENSOR_ENABLED, Autoload: true, Public: false, Format: "bool", Value: "false",
			Desc: "平台级内容审核总开关。false=全局关闭；true 时仍需租户开启 autoCensorOnCallEnd。文本/音频厂商凭证：ALIYUN_CENSOR_* / QCLOUD_CENSOR_* / 七牛 AK/SK。",
		},
		{
			Key: constants.KEY_CONTENT_CENSOR_PROVIDER, Autoload: true, Public: false, Format: "text", Value: "qiniu",
			Desc: "默认审核厂商：qiniu | aliyun | qcloud。租户可覆盖。音频分别走七牛 AI、阿里云 VoiceModeration、腾讯云 AMS。",
		},
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

// seedPlatformAdmin 在 platform_admins 表为空时插入一个默认平台管理员账号。
func (s *SeedService) seedPlatformAdmin() error {
	if s == nil || s.db == nil {
		return nil
	}
	n, err := models.CountPlatformAdmins(s.db)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	email := platformAdminSeedEmail()
	password := platformAdminSeedPassword()
	if err := platformAdminSeedPasswordAllowed(password); err != nil {
		return err
	}
	displayName := platformAdminSeedDisplayName()
	hash, err := access.HashPassword(password)
	if err != nil {
		return err
	}
	row := &models.PlatformAdmin{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Status:       constants.PlatformAdminStatusActive,
	}
	row.SetCreateInfo("seed")
	if err := s.db.Create(row).Error; err != nil {
		return err
	}
	logger.Info("Initialized Platform Admin (change password after login)",
		zap.String("email", row.Email),
		zap.Uint("id", row.ID),
	)
	return nil
}

func (s *SeedService) seedMailTemplates() error {
	if s == nil || s.db == nil {
		return nil
	}
	if s.db.Dialector.Name() == "mysql" {
		for _, tbl := range []string{"mail_templates", "mail_logs", "sms_logs"} {
			_ = s.db.Exec("ALTER TABLE " + tbl + " CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
		}
	}
	type tplDef struct {
		code, name, subject, html, desc string
	}
	siteName := "SoulNexus"
	if config.GlobalConfig != nil && config.GlobalConfig.Server.Name != "" {
		siteName = config.GlobalConfig.Server.Name
	}
	defs := []tplDef{
		{notification.TmplWelcome, "欢迎邮件", "欢迎加入 " + siteName, SoulNexus.WelcomeHTML, "用户注册成功欢迎邮件"},
		{notification.TmplVerification, "通用验证码", "您的 " + siteName + " 验证码", SoulNexus.VerificationHTML, "通用 6 位验证码邮件"},
		{notification.TmplEmailVerification, "邮箱验证", "请验证您的邮箱地址", SoulNexus.EmailVerificationHTML, "注册后邮箱地址验证邮件"},
		{notification.TmplPasswordReset, "密码重置", "密码重置请求", SoulNexus.PasswordResetHTML, "密码重置链接邮件"},
		{notification.TmplDeviceVerification, "设备验证码", "设备验证码", SoulNexus.DeviceVerificationHTML, "新设备登录二次验证邮件"},
		{notification.TmplGroupInvitation, "组织邀请", "您收到了来自 {{.InviterName}} 的组织邀请", SoulNexus.GroupInvitationHTML, "组织 / 团队邀请邮件"},
		{notification.TmplNewDeviceLogin, "新设备登录提醒", "{{if .IsSuspicious}}可疑登录警告{{else}}新设备登录提醒{{end}}", SoulNexus.NewDeviceLoginHTML, "新设备 / 异地登录提醒"},
		{notification.TmplAIReportDaily, "AI 运营日报", "{{.Title}}", SoulNexus.AIReportDailyHTML, "AI 运营日报邮件"},
		{notification.TmplAIReportWeekly, "AI 运营周报", "{{.Title}}", SoulNexus.AIReportWeeklyHTML, "AI 运营周报邮件"},
	}
	for _, d := range defs {
		var existing models.MailTemplate
		// Unscoped: soft-deleted rows still hold the unique (code, locale) key in MySQL.
		err := s.db.Unscoped().
			Where("code = ? AND COALESCE(locale, '') = ?", d.code, "").
			First(&existing).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			tpl := &models.MailTemplate{
				Code:        d.code,
				Name:        d.name,
				Subject:     d.subject,
				Description: d.desc,
				Locale:      "",
				Enabled:     true,
			}
			models.ApplyMailTemplateHTMLDerivedFields(tpl, d.html, "")
			if err := s.db.Create(tpl).Error; err != nil {
				// Race or legacy row: fall through to update by unique key.
				if !isDuplicateKeyError(err) {
					return err
				}
				if err := s.db.Unscoped().
					Where("code = ? AND COALESCE(locale, '') = ?", d.code, "").
					First(&existing).Error; err != nil {
					return err
				}
			} else {
				continue
			}
		}
		existing.DeletedAt = gorm.DeletedAt{}
		existing.Name = d.name
		existing.Subject = d.subject
		existing.Description = d.desc
		existing.Locale = ""
		existing.Enabled = true
		models.ApplyMailTemplateHTMLDerivedFields(&existing, d.html, existing.Variables)
		if err := s.db.Unscoped().Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, pkgconst.ErrSubstrDuplicateEntry) || strings.Contains(msg, pkgconst.ErrSubstrUniqueConstraint)
}
