package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"

	SoulNexus "github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

type SeedService struct {
	db *gorm.DB
}

func (s *SeedService) SeedAll() error {
	if err := s.SeedAuth(); err != nil {
		return err
	}
	return s.SeedServer()
}

// SeedAuth seeds roles, admin users, and shared site configs (cmd/auth only).
func (s *SeedService) SeedAuth() error {
	if err := s.seedMinimalRolesIfEmpty(); err != nil {
		return err
	}
	if err := s.seedBootstrapRBAC(); err != nil {
		return err
	}
	if err := s.seedConfigs(); err != nil {
		return err
	}
	if err := s.seedAdminUsers(); err != nil {
		return err
	}
	return nil
}

// SeedServer seeds business demo data (cmd/server only).
func (s *SeedService) SeedServer() error {
	if err := s.seedConfigs(); err != nil {
		return err
	}
	if err := s.seedAssistants(); err != nil {
		return err
	}
	if err := s.seedMailTemplates(); err != nil {
		return err
	}
	if err := s.seedPresetTemplates(); err != nil {
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
		{notification.TmplWelcome, "欢迎邮件", "欢迎加入 SoulNexus", SoulNexus.WelcomeHTML, "用户注册成功欢迎邮件"},
		{notification.TmplVerification, "通用验证码", "您的 SoulNexus 验证码", SoulNexus.VerificationHTML, "通用 6 位验证码邮件"},
		{notification.TmplEmailVerification, "邮箱验证", "请验证您的邮箱地址", SoulNexus.EmailVerificationHTML, "注册后邮箱地址验证邮件"},
		{notification.TmplPasswordReset, "密码重置", "密码重置请求", SoulNexus.PasswordResetHTML, "密码重置链接邮件"},
		{notification.TmplDeviceVerification, "设备验证码", "设备验证码", SoulNexus.DeviceVerificationHTML, "新设备登录二次验证邮件"},
		{notification.TmplGroupInvitation, "组织邀请", "您收到了来自 {{.InviterName}} 的组织邀请", SoulNexus.GroupInvitationHTML, "组织 / 团队邀请邮件"},
		{notification.TmplNewDeviceLogin, "新设备登录提醒", "{{if .IsSuspicious}}可疑登录警告{{else}}新设备登录提醒{{end}}", SoulNexus.NewDeviceLoginHTML, "新设备 / 异地登录提醒"},
	}
	for _, d := range defs {
		var n int64
		if err := s.db.Model(&svcmodels.MailTemplate{}).
			Where("org_id = ? AND code = ? AND locale = ?", 0, d.code, "").
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		tpl := &svcmodels.MailTemplate{
			OrgID:       0,
			Code:        d.code,
			Name:        d.name,
			Subject:     d.subject,
			Description: d.desc,
			Locale:      "",
			Enabled:     true,
		}
		svcmodels.ApplyMailTemplateHTMLDerivedFields(tpl, d.html, "")
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
		{Key: constants.KEY_SERVER_WEBSOCKET, Desc: "SERVER WEBSOCKET", Autoload: true, Public: false, Format: "text", Value: "ws://localhost:7080/voice/lingecho/v1/"},
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
	if err := s.db.Model(&auth.Role{}).Where("is_deleted = ?", models.SoftDeleteStatusActive).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return s.db.Create(&auth.Role{Name: "Member", Slug: "member", IsSystem: true}).Error
}

// seedBootstrapRBAC ensures admin console + RBAC permissions and an admin role for JWT claims.
func (s *SeedService) seedBootstrapRBAC() error {
	permDefs := []auth.Permission{
		{Key: auth.PermAdminAccess, Name: "Admin console", Description: "Access management UI", Resource: "admin"},
		{Key: auth.PermManageRoles, Name: "Manage RBAC", Description: "Manage roles and permissions", Resource: "rbac"},
	}
	permIDs := make([]uint, 0, len(permDefs))
	for _, def := range permDefs {
		var p auth.Permission
		err := s.db.Where("`key` = ? AND is_deleted = ?", def.Key, models.SoftDeleteStatusActive).First(&p).Error
		if err != nil {
			if err := s.db.Create(&def).Error; err != nil {
				return err
			}
			permIDs = append(permIDs, def.ID)
			continue
		}
		permIDs = append(permIDs, p.ID)
	}

	var adminRole auth.Role
	err := s.db.Where("slug = ? AND is_deleted = ?", "admin", models.SoftDeleteStatusActive).First(&adminRole).Error
	if err != nil {
		adminRole = auth.Role{Name: "Administrator", Slug: "admin", Description: "Full admin access", IsSystem: true}
		if err := s.db.Create(&adminRole).Error; err != nil {
			return err
		}
	}
	for _, pid := range permIDs {
		var n int64
		if err := s.db.Model(&auth.RolePermission{}).Where("role_id = ? AND permission_id = ?", adminRole.ID, pid).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			if err := s.db.Create(&auth.RolePermission{RoleID: adminRole.ID, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SeedService) seedAdminUsers() error {
	defaultAdmins := []auth.User{
		{
			Email:    "admin@lingecho.com",
			Password: auth.HashPassword("admin123"),
			Status:   auth.UserStatusActive,
			Source:   auth.UserSourceAdmin,
		},
		{
			Email:    "19511899044@163.com",
			Password: auth.HashPassword("admin123"),
			Status:   auth.UserStatusActive,
			Source:   auth.UserSourceAdmin,
		},
	}

	for _, user := range defaultAdmins {
		var count int64
		err := s.db.Model(&auth.User{}).Where("`email` = ?", user.Email).Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			if err := s.db.Create(&user).Error; err != nil {
				return err
			}
			_ = auth.UpdateUserProfileFields(s.db, user.ID, map[string]any{"display_name": "Administrator"})
			if err := auth.AssignUserSingleRoleBySlug(s.db, user.ID, "admin"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SeedService) seedAssistants() error {
	var count int64
	if err := s.db.Model(&svcmodels.Agent{}).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil // Data already exists, skip
	}

	g2, err := svcmodels.EnsurePersonalGroupForUser(s.db, 2)
	if err != nil {
		return err
	}
	g1, err := svcmodels.EnsurePersonalGroupForUser(s.db, 1)
	if err != nil {
		return err
	}

	defaultAssistant := []svcmodels.Agent{
		{
			GroupID:      g2.ID,
			CreatedBy:    2,
			Name:         "Technical Support",
			SystemPrompt: "You are a professional technical support engineer, focused on helping users solve technology-related problems.",
			PersonaTag:   "support",
			Temperature:  0.6,
			MaxTokens:    50,
			Description:  "A knowledgeable technical support engineer ready to debug any issue.",
			Personality:  "Patient, analytical, methodical, and detail-oriented. Always explains solutions step by step.",
			Scenario:     "You work at a tech company's help desk, assisting users with software and hardware issues.",
			Tags:         "tech,support,debugging,troubleshooting",
			Visibility:   "public",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			GroupID:      g2.ID,
			CreatedBy:    2,
			Name:         "Smart Assistant",
			SystemPrompt: "You are a smart assistant, please answer user questions as an assistant.",
			PersonaTag:   "assistant",
			Temperature:  0.6,
			MaxTokens:    50,
			Description:  "A versatile AI assistant that can help with a wide range of tasks.",
			Personality:  "Friendly, helpful, curious, and enthusiastic. Always eager to learn and assist.",
			Scenario:     "You are a general-purpose assistant ready to help users with any questions or tasks.",
			Tags:         "assistant,general,helpful",
			Visibility:   "public",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			GroupID:      g1.ID,
			CreatedBy:    1,
			Name:         "Mentor",
			SystemPrompt: "You are a mentor, please answer user questions as a mentor.",
			PersonaTag:   "mentor",
			Temperature:  0.6,
			MaxTokens:    50,
			Description:  "A wise and experienced mentor who guides users through learning and growth.",
			Personality:  "Wise, patient, encouraging, and insightful. Believes in the potential of every learner.",
			Scenario:     "You are a seasoned mentor helping students and professionals navigate their career paths.",
			Tags:         "mentor,education,coaching,growth",
			Visibility:   "public",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			GroupID:      g1.ID,
			CreatedBy:    1,
			Name:         "Assistant",
			SystemPrompt: "You are an assistant, please answer user questions as an assistant.",
			PersonaTag:   "assistant",
			JsSourceID:   strconv.Itoa(1),
			Temperature:  0.6,
			MaxTokens:    50,
			Description:  "A reliable everyday assistant for your daily needs.",
			Personality:  "Professional, efficient, courteous, and reliable.",
			Scenario:     "You assist users with their daily tasks, from scheduling to information lookup.",
			Tags:         "assistant,daily,productive",
			Visibility:   "group",
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

// seedPresetTemplates 写入系统内置预设模板（idempotent：按 name+type 去重）。
func (s *SeedService) seedPresetTemplates() error {
	type seedPreset struct {
		Name        string
		Description string
		Type        string
		Category    string
		Tags        string
		Content     string
	}

	presets := []seedPreset{
		// ========== system_prompt 模板 ==========
		{
			Name:        "通用助手",
			Description: "适合大多数场景的通用助手提示词，可用于客服、问答、通用对话",
			Type:        svcmodels.PresetTypeSystemPrompt,
			Category:    "通用",
			Tags:        "助手,通用,入门",
			Content: `{
  "systemPrompt": "你是一个友好、专业的AI助手。请用简洁清晰的语言回答用户问题。\n\n你的职责：\n1. 准确理解用户意图\n2. 提供有帮助、准确的信息\n3. 保持礼貌和耐心\n4. 如果不确定，诚实告知\n\n请始终使用{{language}}回复。",
  "personaTag": "assistant",
  "variables": [
    {"name": "language", "label": "回复语言", "defaultVal": "中文", "description": "期望的回复语言", "required": false}
  ]
}`,
		},
		{
			Name:        "技术客服",
			Description: "适用于技术支持、IT 帮助台等场景的专业客服提示词",
			Type:        svcmodels.PresetTypeSystemPrompt,
			Category:    "客服",
			Tags:        "技术支持,客服,IT,专业",
			Content: `{
  "systemPrompt": "你是一名{{company_name}}的资深技术支持工程师。\n\n你的职责：\n- 帮助用户诊断和解决技术问题\n- 提供清晰的分步操作指导\n- 必要时引导用户提供更多诊断信息\n- 对无法解决的问题，正确升级给后端团队\n\n请保持专业、耐心，使用用户能理解的语言。\n\n当前产品版本：{{product_version}}",
  "personaTag": "support",
  "variables": [
    {"name": "company_name", "label": "公司名称", "defaultVal": "LingByte", "description": "你的公司名称", "required": true},
    {"name": "product_version", "label": "产品版本", "defaultVal": "v3.0", "description": "当前产品版本号", "required": false}
  ]
}`,
		},
		{
			Name:        "销售顾问",
			Description: "适用于产品推荐、销售转化的顾问式提示词",
			Type:        svcmodels.PresetTypeSystemPrompt,
			Category:    "销售",
			Tags:        "销售,转化,推荐,电商",
			Content: `{
  "systemPrompt": "你是一名经验丰富的{{industry}}销售顾问。\n\n核心能力：\n1. 了解客户需求，精准推荐产品\n2. 用FAB法则介绍产品（特性-优势-利益）\n3. 巧妙处理客户异议\n4. 适时推进成交\n\n沟通风格：\n- 热情但不压迫\n- 专业但不生硬\n- 关注客户利益\n\n当前推荐产品线：{{product_line}}",
  "personaTag": "sales",
  "variables": [
    {"name": "industry", "label": "行业", "defaultVal": "SaaS", "description": "销售所属行业", "required": true},
    {"name": "product_line", "label": "产品线", "defaultVal": "智能语音客服", "description": "推荐的产品线", "required": false}
  ]
}`,
		},
		{
			Name:        "面试官",
			Description: "模拟面试官角色，可用于模拟面试练习场景",
			Type:        svcmodels.PresetTypeSystemPrompt,
			Category:    "角色扮演",
			Tags:        "面试,HR,招聘,模拟",
			Content: `{
  "systemPrompt": "你是一位经验丰富的{{position}}面试官。\n\n面试流程：\n1. 开场寒暄，营造轻松氛围\n2. 了解候选人背景\n3. 提出与{{position}}相关的技术/行为问题\n4. 给予候选人提问机会\n5. 结束后给出建设性反馈\n\n面试风格：\n- 专业友好\n- 考察深度与广度并重\n- 关注解决问题的思路而非死记硬背\n- 适当追问以深入了解",
  "personaTag": "mentor",
  "variables": [
    {"name": "position", "label": "面试岗位", "defaultVal": "后端工程师", "description": "面试的目标岗位", "required": true}
  ]
}`,
		},
		{
			Name:        "翻译助手",
			Description: "专注于多语言翻译的提示词模板",
			Type:        svcmodels.PresetTypeSystemPrompt,
			Category:    "工具",
			Tags:        "翻译,多语言,本地化",
			Content: `{
  "systemPrompt": "你是一个专业的翻译助手。\n\n翻译规则：\n1. 将用户输入的内容从{{source_lang}}翻译为{{target_lang}}\n2. 保持原文的语气、风格和格式\n3. 专业术语使用行业标准译法\n4. 如果遇到无法翻译的内容，标注 [无法翻译]\n5. 翻译后附上简要的翻译说明（可选）\n\n输出格式：\n---\n[翻译结果]\n---\n[说明]（如有）",
  "personaTag": "assistant",
  "variables": [
    {"name": "source_lang", "label": "源语言", "defaultVal": "中文", "description": "输入内容的语言", "required": true},
    {"name": "target_lang", "label": "目标语言", "defaultVal": "英文", "description": "翻译目标语言", "required": true}
  ]
}`,
		},
		{
			Name:        "教育导师",
			Description: "适用于在线教育、学习辅导的提示词",
			Type:        svcmodels.PresetTypeSystemPrompt,
			Category:    "教育",
			Tags:        "教育,导师,学习,辅导",
			Content: `{
  "systemPrompt": "你是一位{{subject}}领域的教育导师，擅长用苏格拉底式提问引导学生独立思考。\n\n教学原则：\n1. 不要直接给出答案，而是引导思考\n2. 用生活中的例子解释抽象概念\n3. 根据学生水平调整讲解深度\n4. 鼓励学生，建立学习信心\n5. 定期总结知识点，帮助巩固\n\n学生当前水平：{{level}}",
  "personaTag": "mentor",
  "variables": [
    {"name": "subject", "label": "学科", "defaultVal": "编程", "description": "教学领域", "required": true},
    {"name": "level", "label": "学生水平", "defaultVal": "初级", "description": "学生的当前水平", "required": false}
  ]
}`,
		},

		// ========== voice 模板 ==========
		{
			Name:        "标准女声配置",
			Description: "使用默认标准女声，VAD开启，适合通用对话场景",
			Type:        svcmodels.PresetTypeVoice,
			Category:    "标准",
			Tags:        "女声,标准,通用,VAD",
			Content: `{
  "speaker": "101016",
  "ttsProvider": "volcengine",
  "enableVAD": true,
  "vadThreshold": 500,
  "vadConsecutiveFrames": 2
}`,
		},
		{
			Name:        "高灵敏度打断",
			Description: "降低 VAD 阈值，支持更灵敏的语音打断，适合快节奏对话",
			Type:        svcmodels.PresetTypeVoice,
			Category:    "进阶",
			Tags:        "打断,VAD,高灵敏度,快节奏",
			Content: `{
  "speaker": "101016",
  "ttsProvider": "volcengine",
  "enableVAD": true,
  "vadThreshold": 300,
  "vadConsecutiveFrames": 1
}`,
		},
		{
			Name:        "低延迟配置",
			Description: "优化语音延迟，适合实时对话场景",
			Type:        svcmodels.PresetTypeVoice,
			Category:    "进阶",
			Tags:        "低延迟,实时,优化",
			Content: `{
  "speaker": "101016",
  "ttsProvider": "volcengine",
  "enableVAD": true,
  "vadThreshold": 500,
  "vadConsecutiveFrames": 2,
  "speed": 1.1,
  "volume": 1.0,
  "pitch": 1.0
}`,
		},

		// ========== knowledge 模板 ==========
		{
			Name:        "产品FAQ知识库",
			Description: "适用于产品常见问题解答的知识库配置，使用 Qdrant 向量库",
			Type:        svcmodels.PresetTypeKnowledge,
			Category:    "FAQ",
			Tags:        "FAQ,产品,常见问题,Qdrant",
			Content: `{
  "namespace": "product_faq",
  "name": "产品FAQ",
  "description": "产品常见问题与解答知识库",
  "vectorProvider": "qdrant",
  "embedModel": "text-embedding-ada-002",
  "vectorDim": 1536,
  "chunkSize": 512,
  "chunkOverlap": 50,
  "fileTypes": "pdf,txt,md,csv"
}`,
		},
		{
			Name:        "技术文档库",
			Description: "适用于技术文档、API文档的知识库配置",
			Type:        svcmodels.PresetTypeKnowledge,
			Category:    "技术",
			Tags:        "技术文档,API,开发",
			Content: `{
  "namespace": "tech_docs",
  "name": "技术文档",
  "description": "技术文档与API参考知识库",
  "vectorProvider": "qdrant",
  "embedModel": "text-embedding-ada-002",
  "vectorDim": 1536,
  "chunkSize": 800,
  "chunkOverlap": 100,
  "fileTypes": "pdf,md,rst,txt,html"
}`,
		},
		{
			Name:        "客服话术库",
			Description: "适用于客服标准话术、回复模板的知识库配置",
			Type:        svcmodels.PresetTypeKnowledge,
			Category:    "客服",
			Tags:        "客服,话术,模板,回复",
			Content: `{
  "namespace": "cs_scripts",
  "name": "客服话术库",
  "description": "客服标准回复话术与处理流程",
  "vectorProvider": "qdrant",
  "embedModel": "text-embedding-ada-002",
  "vectorDim": 1536,
  "chunkSize": 256,
  "chunkOverlap": 30,
  "fileTypes": "txt,md,csv,json"
}`,
		},

		// ========== agent 模板 ==========
		{
			Name:        "智能客服Agent",
			Description: "开箱即用的智能客服Agent模板，包含完整的系统提示词、语音和模型配置",
			Type:        svcmodels.PresetTypeAgent,
			Category:    "客服",
			Tags:        "客服,开箱即用,完整配置",
			Content: `{
  "name": "智能客服",
  "systemPrompt": "你是一位专业的智能客服代表。请礼貌、耐心地解答用户问题，提供准确的信息和解决方案。如遇到无法解决的问题，请引导用户联系人工客服。",
  "openingStatement": "您好！我是智能客服小灵，很高兴为您服务。请问有什么可以帮助您的？",
  "personaTag": "support",
  "temperature": 0.6,
  "maxTokens": 150,
  "llmModel": "gpt-4o-mini",
  "speaker": "101016",
  "ttsProvider": "volcengine",
  "enableVAD": true,
  "vadThreshold": 500,
  "vadConsecutiveFrames": 2
}`,
		},
		{
			Name:        "学习助手Agent",
			Description: "专为学习场景设计的Agent模板，导师风格，适合教育类应用",
			Type:        svcmodels.PresetTypeAgent,
			Category:    "教育",
			Tags:        "学习,教育,导师,耐心",
			Content: `{
  "name": "学习助手",
  "systemPrompt": "你是一位充满耐心的学习导师。请用引导式提问帮助学生思考，而不是直接给出答案。善于用比喻和生活中的例子来解释复杂概念。鼓励学生提问，培养他们的学习兴趣。",
  "openingStatement": "你好！我是你的学习伙伴。无论你在学习什么，我都会陪伴你一起探索。今天想学习什么呢？",
  "personaTag": "mentor",
  "temperature": 0.7,
  "maxTokens": 200,
  "llmModel": "gpt-4o-mini",
  "speaker": "101016",
  "ttsProvider": "volcengine",
  "enableVAD": true,
  "vadThreshold": 500,
  "vadConsecutiveFrames": 2
}`,
		},
		{
			Name:        "创意写作伙伴",
			Description: "适合创意写作、头脑风暴场景的Agent模板，富有创造力",
			Type:        svcmodels.PresetTypeAgent,
			Category:    "创作",
			Tags:        "写作,创意,头脑风暴,灵感",
			Content: `{
  "name": "创意伙伴",
  "systemPrompt": "你是一位充满想象力的创意写作伙伴。你擅长：\n1. 头脑风暴和创意发散\n2. 故事构思和情节设计\n3. 文案润色和风格调整\n4. 提供多种角度的创意建议\n\n请保持开放、鼓励的态度，帮助用户释放创造力。",
  "openingStatement": "嘿！创意的大门已经打开。无论你想写故事、改文案，还是寻找灵感，我都在这里陪你一起创作！",
  "personaTag": "assistant",
  "temperature": 0.9,
  "maxTokens": 300,
  "llmModel": "gpt-4o",
  "speaker": "101016",
  "ttsProvider": "volcengine",
  "enableVAD": true,
  "vadThreshold": 500,
  "vadConsecutiveFrames": 2
}`,
		},
	}

	for _, p := range presets {
		var count int64
		if err := s.db.Model(&svcmodels.PresetTemplate{}).
			Where("name = ? AND type = ? AND is_builtin = ?", p.Name, p.Type, true).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		tpl := svcmodels.PresetTemplate{
			GroupID:     0, // system
			CreatedBy:   0, // system
			Name:        p.Name,
			Description: p.Description,
			Type:        p.Type,
			Category:    p.Category,
			Tags:        p.Tags,
			Visibility:  svcmodels.PresetVisibilityPublic,
			Content:     p.Content,
			IsBuiltin:   true,
			Status:      "active",
		}
		if err := s.db.Create(&tpl).Error; err != nil {
			return err
		}
	}

	return nil
}
