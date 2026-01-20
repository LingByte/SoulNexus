package bootstrap

import (
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

type SeedService struct {
	db *gorm.DB
}

func (s *SeedService) SeedAll() error {
	if err := s.seedConfigs(); err != nil {
		return err
	}

	if err := s.seedAdminUsers(); err != nil {
		return err
	}

	if err := s.seedAssistants(); err != nil {
		return err
	}

	if err := s.seedPromptModels(); err != nil {
		return err
	}

	if err := s.seedPromptArgs(); err != nil {
		return err
	}

	return nil
}

func (s *SeedService) seedConfigs() error {
	apiPrefix := config.GlobalConfig.APIPrefix
	defaults := []utils.Config{
		{Key: constants.KEY_SITE_URL, Desc: "Site URL", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerUrl != "" {
				return config.GlobalConfig.ServerUrl
			}
			return "https://lingecho.com"
		}()},
		{Key: constants.KEY_SITE_NAME, Desc: "Site Name", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerName != "" {
				return config.GlobalConfig.ServerName
			}
			return "LingEcho"
		}()},
		{Key: constants.KEY_SITE_LOGO_URL, Desc: "Site Logo", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerLogo != "" {
				return config.GlobalConfig.ServerLogo
			}
			return "/static/img/favicon.png"
		}()},
		{Key: constants.KEY_SITE_DESCRIPTION, Desc: "Site Description", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerDesc != "" {
				return config.GlobalConfig.ServerDesc
			}
			return "LingEcho - Intelligent Voice Customer Service Platform"
		}()},
		{Key: constants.KEY_SITE_TERMS_URL, Desc: "Terms of Service", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerTermsUrl != "" {
				return config.GlobalConfig.ServerTermsUrl
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
		// Search configuration
		{Key: constants.KEY_SEARCH_ENABLED, Desc: "Search Feature Enabled", Autoload: true, Public: true, Format: "bool", Value: func() string {
			if config.GlobalConfig.SearchEnabled {
				return "true"
			}
			return "false"
		}()},
		{Key: constants.KEY_SEARCH_PATH, Desc: "Search Index Path", Autoload: true, Public: false, Format: "text", Value: func() string {
			if config.GlobalConfig.SearchPath != "" {
				return config.GlobalConfig.SearchPath
			}
			return "./search"
		}()},
		{Key: constants.KEY_SEARCH_BATCH_SIZE, Desc: "Search Batch Size", Autoload: true, Public: false, Format: "int", Value: func() string {
			if config.GlobalConfig.SearchBatchSize > 0 {
				return strconv.Itoa(config.GlobalConfig.SearchBatchSize)
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

func (s *SeedService) seedAdminUsers() error {
	// 超级管理员权限（所有权限）
	allPermissions := `["*"]`

	defaultAdmins := []models.User{
		{
			Email:       "admin@lingecho.com",
			Password:    models.HashPassword("admin123"),
			IsStaff:     true,
			Role:        models.RoleSuperAdmin,
			Permissions: allPermissions,
			DisplayName: "Administrator",
			Enabled:     true,
		},
		{
			Email:       "19511899044@163.com",
			Password:    models.HashPassword("admin123"),
			IsStaff:     true,
			Role:        models.RoleSuperAdmin,
			Permissions: allPermissions,
			DisplayName: "Administrator",
			Enabled:     true,
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
		}
	}
	return nil
}

func (s *SeedService) seedAssistants() error {
	var count int64
	if err := s.db.Model(&models.Assistant{}).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil // Data already exists, skip
	}

	defaultAssistant := []models.Assistant{
		{
			UserID:       2,
			Name:         "Technical Support",
			Description:  "Provides technical support and answers various technical support questions",
			Icon:         "MessageCircle",
			SystemPrompt: "You are a professional technical support engineer, focused on helping users solve technology-related problems.",
			PersonaTag:   "support",
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			UserID:       2,
			Name:         "Smart Assistant",
			Description:  "Smart assistant providing various intelligent services",
			Icon:         "Bot",
			SystemPrompt: "You are a smart assistant, please answer user questions as an assistant.",
			PersonaTag:   "assistant",
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			UserID:       1,
			Name:         "Mentor",
			Description:  "Mentor providing various guidance services",
			Icon:         "Users",
			SystemPrompt: "You are a mentor, please answer user questions as a mentor.",
			PersonaTag:   "mentor",
			Temperature:  0.6,
			MaxTokens:    50,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			UserID:       1,
			Name:         "Assistant",
			Description:  "An assistant that you can use to answer your questions.",
			Icon:         "Zap",
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

func (s *SeedService) seedPromptModels() error {
	defaultPrompts := []models.PromptModel{
		{
			Name:        "summarize_article",
			Description: "Summarize the main content of an article, suitable for extracting summaries from long paragraphs or blog posts.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "translate_text",
			Description: "Translate input text to a specified language, suitable for scenarios like English-Chinese translation.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "generate_title",
			Description: "Generate a concise and attractive title based on article content.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "email_reply_generator",
			Description: "Automatically generate professional email replies based on email content and intent.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	var count int64
	if err := s.db.Model(models.PromptModel{}).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	return s.db.Model(models.PromptModel{}).Create(defaultPrompts).Error
}

func (s *SeedService) seedPromptArgs() error {
	defaultArgs := []models.PromptArgModel{
		// summarize_article
		{Name: "content", Description: "Article content to be summarized", Required: true, PromptID: 1},

		// translate_text
		{Name: "text", Description: "Text to be translated", Required: true, PromptID: 2},
		{Name: "target_language", Description: "Target language (e.g., en, zh)", Required: true, PromptID: 2},

		// generate_title
		{Name: "article", Description: "Article content", Required: true, PromptID: 3},

		// email_reply_generator
		{Name: "email_body", Description: "Original email content", Required: true, PromptID: 4},
		{Name: "tone", Description: "Reply tone (e.g., formal, casual)", Required: false, PromptID: 4},
	}
	return s.db.Model(models.PromptArgModel{}).Create(defaultArgs).Error
}
