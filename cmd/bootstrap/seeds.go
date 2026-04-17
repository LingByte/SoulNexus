package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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
	if err := s.seedMCPMarketplace(); err != nil {
		return err
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

func (s *SeedService) seedAdminUsers() error {
	defaultAdmins := []models.User{
		{
			Email:       "admin@lingecho.com",
			Password:    models.HashPassword("admin123"),
			Role:        models.RoleSuperAdmin,
			DisplayName: "Administrator",
			Status:      models.UserStatusActive,
			Source:      models.UserSourceAdmin,
		},
		{
			Email:       "19511899044@163.com",
			Password:    models.HashPassword("admin123"),
			Role:        models.RoleSuperAdmin,
			DisplayName: "Administrator",
			Status:      models.UserStatusActive,
			Source:      models.UserSourceAdmin,
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

func (s *SeedService) seedMCPMarketplace() error {
	var count int64
	if err := s.db.Model(&models.MCPMarketplaceItem{}).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil // Data already exists, skip
	}

	// Seed categories first
	categories := []models.MCPCategory{
		{
			Name:        "Database",
			Description: "Database query and management tools",
			Order:       1,
		},
		{
			Name:        "File System",
			Description: "File operations and management tools",
			Order:       2,
		},
		{
			Name:        "Network",
			Description: "Network and HTTP request tools",
			Order:       3,
		},
		{
			Name:        "Utilities",
			Description: "Utility and helper tools",
			Order:       4,
		},
		{
			Name:        "System",
			Description: "System information and management tools",
			Order:       5,
		},
	}

	for i := range categories {
		if err := s.db.Create(&categories[i]).Error; err != nil {
			return err
		}
	}

	// Seed marketplace items
	items := []models.MCPMarketplaceItem{
		{
			Name:          "Database Query MCP",
			Description:   "Provides database query tools, supports SQL queries, data export and other functions",
			Version:       "1.0.0",
			Author:        "SoulNexus Team",
			Repository:    "https://github.com/soulmcp/mcp-database",
			Documentation: "https://docs.soulmcp.com/mcp-database",
			Category:      "Database",
			Tags:          []byte(`["database", "query", "sql"]`),
			Downloads:     1250,
			Rating:        4.8,
			Status:        "published",
			IsFeatured:    true,
		},
		{
			Name:          "File Operations MCP",
			Description:   "Provides file operation tools, supports read, write, list, delete and other operations",
			Version:       "1.0.0",
			Author:        "SoulNexus Team",
			Repository:    "https://github.com/soulmcp/mcp-file",
			Documentation: "https://docs.soulmcp.com/mcp-file",
			Category:      "File System",
			Tags:          []byte(`["file", "filesystem", "io"]`),
			Downloads:     980,
			Rating:        4.6,
			Status:        "published",
			IsFeatured:    true,
		},
		{
			Name:          "HTTP API MCP",
			Description:   "Provides HTTP request tools, supports GET, POST, PUT, DELETE and other methods",
			Version:       "1.0.0",
			Author:        "SoulNexus Team",
			Repository:    "https://github.com/soulmcp/mcp-http",
			Documentation: "https://docs.soulmcp.com/mcp-http",
			Category:      "Network",
			Tags:          []byte(`["http", "api", "rest"]`),
			Downloads:     750,
			Rating:        4.5,
			Status:        "published",
			IsFeatured:    true,
		},
		{
			Name:          "System Information MCP",
			Description:   "Provides system information tools, supports OS info, CPU, memory and other metrics",
			Version:       "1.0.0",
			Author:        "SoulNexus Team",
			Repository:    "https://github.com/soulmcp/mcp-system",
			Documentation: "https://docs.soulmcp.com/mcp-system",
			Category:      "System",
			Tags:          []byte(`["system", "info", "metrics"]`),
			Downloads:     650,
			Rating:        4.7,
			Status:        "published",
			IsFeatured:    true,
		},
		{
			Name:          "Utility Tools MCP",
			Description:   "Provides utility tools, supports hash, encode, decode, validate and other functions",
			Version:       "1.0.0",
			Author:        "SoulNexus Team",
			Repository:    "https://github.com/soulmcp/mcp-utils",
			Documentation: "https://docs.soulmcp.com/mcp-utils",
			Category:      "Utilities",
			Tags:          []byte(`["utility", "hash", "encode"]`),
			Downloads:     520,
			Rating:        4.4,
			Status:        "published",
			IsFeatured:    false,
		},
		{
			Name:          "Network Tools MCP",
			Description:   "Provides network tools, supports ping, DNS lookup, traceroute and other functions",
			Version:       "1.0.0",
			Author:        "SoulNexus Team",
			Repository:    "https://github.com/soulmcp/mcp-network",
			Documentation: "https://docs.soulmcp.com/mcp-network",
			Category:      "Network",
			Tags:          []byte(`["network", "ping", "dns"]`),
			Downloads:     380,
			Rating:        4.3,
			Status:        "published",
			IsFeatured:    false,
		},
	}

	for i := range items {
		if err := s.db.Create(&items[i]).Error; err != nil {
			return err
		}
	}

	return nil
}
