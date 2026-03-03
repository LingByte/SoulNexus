package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupNodePluginTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = MigrateNodePluginTables(db)
	require.NoError(t, err)

	// 迁移依赖表
	err = db.AutoMigrate(&User{}, &Group{})
	require.NoError(t, err)

	return db
}

func createTestUserForNodePlugin(t *testing.T, db *gorm.DB) *User {
	user := &User{
		Email:    "plugin@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestNodePlugin_CRUD(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)

	// 测试创建插件
	plugin := &NodePlugin{
		UserID:      user.ID,
		Name:        "Test API Plugin",
		Slug:        "test-api-plugin",
		DisplayName: "Test API Plugin",
		Description: "A test plugin for API integration",
		Category:    NodePluginCategoryAPI,
		Version:     "1.0.0",
		Status:      NodePluginStatusDraft,
		Icon:        "https://example.com/icon.svg",
		Color:       "#FF5722",
		Tags:        StringArray{"api", "integration", "test"},
		Definition: NodePluginDefinition{
			Type: "api-call",
			Inputs: []NodePluginPort{
				{
					Name:        "url",
					Type:        "string",
					Required:    true,
					Description: "API endpoint URL",
				},
				{
					Name:        "method",
					Type:        "string",
					Required:    false,
					Default:     "GET",
					Description: "HTTP method",
				},
			},
			Outputs: []NodePluginPort{
				{
					Name:        "response",
					Type:        "object",
					Required:    true,
					Description: "API response data",
				},
			},
			Runtime: NodePluginRuntime{
				Type:    "http",
				Config:  JSONMap{"timeout": 30},
				Timeout: 30,
				Retry:   3,
			},
			UI: NodePluginUI{
				ConfigForm: []NodePluginFormField{
					{
						Name:        "apiKey",
						Type:        "text",
						Label:       "API Key",
						Description: "Your API key",
						Required:    true,
						Validation: FormFieldValidation{
							Message: "API key is required",
						},
					},
				},
				Help: "This plugin makes HTTP API calls",
			},
		},
		Schema: NodePluginSchema{
			Properties: map[string]SchemaProperty{
				"apiKey": {
					Type:        "string",
					Description: "API authentication key",
				},
			},
			Required: []string{"apiKey"},
		},
		Author:     "Test Author",
		Homepage:   "https://example.com",
		Repository: "https://github.com/example/plugin",
		License:    "MIT",
	}

	err := db.Create(plugin).Error
	assert.NoError(t, err)
	assert.NotZero(t, plugin.ID)
	assert.NotZero(t, plugin.CreatedAt)

	// 测试读取插件
	var retrieved NodePlugin
	err = db.Preload("Versions").Preload("Reviews").First(&retrieved, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, plugin.Name, retrieved.Name)
	assert.Equal(t, plugin.Slug, retrieved.Slug)
	assert.Equal(t, NodePluginCategoryAPI, retrieved.Category)
	assert.Equal(t, NodePluginStatusDraft, retrieved.Status)
	assert.Len(t, retrieved.Tags, 3)
	assert.Contains(t, retrieved.Tags, "api")
	assert.Equal(t, "api-call", retrieved.Definition.Type)
	assert.Len(t, retrieved.Definition.Inputs, 2)
	assert.Len(t, retrieved.Definition.Outputs, 1)

	// 测试更新插件
	retrieved.Status = NodePluginStatusPublished
	retrieved.Version = "1.1.0"
	retrieved.DownloadCount = 100
	retrieved.StarCount = 25
	retrieved.Rating = 4.5
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated NodePlugin
	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, NodePluginStatusPublished, updated.Status)
	assert.Equal(t, "1.1.0", updated.Version)
	assert.Equal(t, uint(100), updated.DownloadCount)
	assert.Equal(t, uint(25), updated.StarCount)
	assert.Equal(t, 4.5, updated.Rating)

	// 测试软删除
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	var deleted NodePlugin
	err = db.First(&deleted, plugin.ID).Error
	assert.Error(t, err)

	// 测试查询包含软删除的记录
	err = db.Unscoped().First(&deleted, plugin.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt)
}

func TestNodePluginVersion_CRUD(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)

	// 创建插件
	plugin := &NodePlugin{
		UserID:      user.ID,
		Name:        "Versioned Plugin",
		Slug:        "versioned-plugin",
		DisplayName: "Versioned Plugin",
		Category:    NodePluginCategoryUtility,
		Version:     "1.0.0",
	}
	err := db.Create(plugin).Error
	require.NoError(t, err)

	// 测试创建版本
	version := &NodePluginVersion{
		PluginID: plugin.ID,
		Version:  "1.1.0",
		Definition: NodePluginDefinition{
			Type: "utility",
			Inputs: []NodePluginPort{
				{
					Name:     "input",
					Type:     "string",
					Required: true,
				},
			},
			Outputs: []NodePluginPort{
				{
					Name:     "output",
					Type:     "string",
					Required: true,
				},
			},
			Runtime: NodePluginRuntime{
				Type:   "script",
				Config: JSONMap{"language": "javascript"},
			},
		},
		Schema: NodePluginSchema{
			Properties: map[string]SchemaProperty{
				"config": {
					Type:        "string",
					Description: "Configuration value",
				},
			},
		},
		ChangeLog: "Added new features and bug fixes",
	}

	err = db.Create(version).Error
	assert.NoError(t, err)
	assert.NotZero(t, version.ID)
	assert.NotZero(t, version.CreatedAt)

	// 测试读取版本
	var retrieved NodePluginVersion
	err = db.Preload("Plugin").First(&retrieved, version.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, plugin.ID, retrieved.PluginID)
	assert.Equal(t, "1.1.0", retrieved.Version)
	assert.Equal(t, "utility", retrieved.Definition.Type)
	assert.Equal(t, "Added new features and bug fixes", retrieved.ChangeLog)
	assert.Equal(t, plugin.Name, retrieved.Plugin.Name)

	// 测试更新版本
	retrieved.ChangeLog = "Updated changelog"
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated NodePluginVersion
	err = db.First(&updated, version.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated changelog", updated.ChangeLog)
}

func TestNodePluginReview_CRUD(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)
	reviewer := &User{
		Email:    "reviewer@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(reviewer).Error
	require.NoError(t, err)

	// 创建插件
	plugin := &NodePlugin{
		UserID:      user.ID,
		Name:        "Reviewed Plugin",
		Slug:        "reviewed-plugin",
		DisplayName: "Reviewed Plugin",
		Category:    NodePluginCategoryUtility,
		Version:     "1.0.0",
	}
	err = db.Create(plugin).Error
	require.NoError(t, err)

	// 测试创建评价
	review := &NodePluginReview{
		PluginID: plugin.ID,
		UserID:   reviewer.ID,
		Rating:   5,
		Comment:  "Excellent plugin! Very useful and well-documented.",
	}

	err = db.Create(review).Error
	assert.NoError(t, err)
	assert.NotZero(t, review.ID)
	assert.NotZero(t, review.CreatedAt)

	// 测试读取评价
	var retrieved NodePluginReview
	err = db.Preload("Plugin").Preload("User").First(&retrieved, review.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, plugin.ID, retrieved.PluginID)
	assert.Equal(t, reviewer.ID, retrieved.UserID)
	assert.Equal(t, 5, retrieved.Rating)
	assert.Equal(t, "Excellent plugin! Very useful and well-documented.", retrieved.Comment)
	assert.Equal(t, plugin.Name, retrieved.Plugin.Name)
	assert.Equal(t, reviewer.Email, retrieved.User.Email)

	// 测试更新评价
	retrieved.Rating = 4
	retrieved.Comment = "Good plugin, but could use some improvements."
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated NodePluginReview
	err = db.First(&updated, review.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 4, updated.Rating)
	assert.Equal(t, "Good plugin, but could use some improvements.", updated.Comment)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestNodePluginInstallation_CRUD(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)
	installer := &User{
		Email:    "installer@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(installer).Error
	require.NoError(t, err)

	// 创建插件
	plugin := &NodePlugin{
		UserID:      user.ID,
		Name:        "Installable Plugin",
		Slug:        "installable-plugin",
		DisplayName: "Installable Plugin",
		Category:    NodePluginCategoryUtility,
		Version:     "1.0.0",
		Status:      NodePluginStatusPublished,
	}
	err = db.Create(plugin).Error
	require.NoError(t, err)

	// 测试创建安装记录
	installation := &NodePluginInstallation{
		UserID:   installer.ID,
		PluginID: plugin.ID,
		Version:  "1.0.0",
		Status:   "active",
		Config: JSONMap{
			"apiKey":     "test-key-123",
			"timeout":    30,
			"retryCount": 3,
		},
	}

	err = db.Create(installation).Error
	assert.NoError(t, err)
	assert.NotZero(t, installation.ID)
	assert.NotZero(t, installation.InstalledAt)

	// 测试读取安装记录
	var retrieved NodePluginInstallation
	err = db.Preload("User").Preload("Plugin").First(&retrieved, installation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, installer.ID, retrieved.UserID)
	assert.Equal(t, plugin.ID, retrieved.PluginID)
	assert.Equal(t, "1.0.0", retrieved.Version)
	assert.Equal(t, "active", retrieved.Status)
	assert.Equal(t, "test-key-123", retrieved.Config["apiKey"])
	assert.Equal(t, installer.Email, retrieved.User.Email)
	assert.Equal(t, plugin.Name, retrieved.Plugin.Name)

	// 测试更新安装记录
	retrieved.Status = "inactive"
	retrieved.Config["timeout"] = 60
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated NodePluginInstallation
	err = db.First(&updated, installation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "inactive", updated.Status)
	assert.Equal(t, float64(60), updated.Config["timeout"])
	assert.True(t, updated.UpdatedAt.After(updated.InstalledAt))
}

func TestNodePlugin_Categories(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)

	categories := []NodePluginCategory{
		NodePluginCategoryAPI,
		NodePluginCategoryData,
		NodePluginCategoryAI,
		NodePluginCategoryNotification,
		NodePluginCategoryUtility,
		NodePluginCategoryCustom,
	}

	// 创建不同分类的插件
	for i, category := range categories {
		plugin := &NodePlugin{
			UserID:      user.ID,
			Name:        "Plugin " + string(category),
			Slug:        "plugin-" + string(category),
			DisplayName: "Plugin " + string(category),
			Category:    category,
			Version:     "1.0.0",
		}
		err := db.Create(plugin).Error
		require.NoError(t, err, "Failed to create plugin %d", i)
	}

	// 按分类查询
	var apiPlugins []NodePlugin
	err := db.Where("category = ?", NodePluginCategoryAPI).Find(&apiPlugins).Error
	assert.NoError(t, err)
	assert.Len(t, apiPlugins, 1)
	assert.Equal(t, NodePluginCategoryAPI, apiPlugins[0].Category)

	// 查询所有插件
	var allPlugins []NodePlugin
	err = db.Find(&allPlugins).Error
	assert.NoError(t, err)
	assert.Len(t, allPlugins, len(categories))
}

func TestNodePlugin_StatusTransitions(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)

	plugin := &NodePlugin{
		UserID:      user.ID,
		Name:        "Status Test Plugin",
		Slug:        "status-test-plugin",
		DisplayName: "Status Test Plugin",
		Category:    NodePluginCategoryUtility,
		Version:     "1.0.0",
		Status:      NodePluginStatusDraft,
	}
	err := db.Create(plugin).Error
	require.NoError(t, err)

	// 测试状态转换：草稿 -> 发布
	plugin.Status = NodePluginStatusPublished
	err = db.Save(plugin).Error
	assert.NoError(t, err)

	var updated NodePlugin
	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, NodePluginStatusPublished, updated.Status)

	// 测试状态转换：发布 -> 归档
	updated.Status = NodePluginStatusArchived
	err = db.Save(&updated).Error
	assert.NoError(t, err)

	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, NodePluginStatusArchived, updated.Status)

	// 测试状态转换：归档 -> 禁用
	updated.Status = NodePluginStatusBanned
	err = db.Save(&updated).Error
	assert.NoError(t, err)

	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, NodePluginStatusBanned, updated.Status)
}

func TestNodePlugin_ComplexDefinition(t *testing.T) {
	db := setupNodePluginTestDB(t)
	user := createTestUserForNodePlugin(t, db)

	// 测试复杂的插件定义
	plugin := &NodePlugin{
		UserID:      user.ID,
		Name:        "Complex Plugin",
		Slug:        "complex-plugin",
		DisplayName: "Complex Plugin",
		Category:    NodePluginCategoryAI,
		Version:     "1.0.0",
		Definition: NodePluginDefinition{
			Type: "ai-processor",
			Inputs: []NodePluginPort{
				{
					Name:        "text",
					Type:        "string",
					Required:    true,
					Description: "Input text to process",
				},
				{
					Name:        "model",
					Type:        "string",
					Required:    false,
					Default:     "gpt-3.5-turbo",
					Description: "AI model to use",
				},
				{
					Name:        "temperature",
					Type:        "number",
					Required:    false,
					Default:     0.7,
					Description: "Model temperature",
				},
			},
			Outputs: []NodePluginPort{
				{
					Name:        "result",
					Type:        "string",
					Required:    true,
					Description: "Processed text result",
				},
				{
					Name:        "confidence",
					Type:        "number",
					Required:    false,
					Description: "Confidence score",
				},
			},
			Runtime: NodePluginRuntime{
				Type:    "http",
				Config:  JSONMap{"endpoint": "https://api.openai.com/v1/chat/completions"},
				Timeout: 60,
				Retry:   2,
			},
			UI: NodePluginUI{
				ConfigForm: []NodePluginFormField{
					{
						Name:        "apiKey",
						Type:        "password",
						Label:       "OpenAI API Key",
						Description: "Your OpenAI API key",
						Required:    true,
						Validation: FormFieldValidation{
							Pattern: "^sk-[a-zA-Z0-9]{48}$",
							Message: "Invalid API key format",
						},
					},
					{
						Name:        "maxTokens",
						Type:        "number",
						Label:       "Max Tokens",
						Description: "Maximum number of tokens to generate",
						Required:    false,
						Default:     1000,
						Validation: FormFieldValidation{
							Min:     func() *float64 { v := 1.0; return &v }(),
							Max:     func() *float64 { v := 4000.0; return &v }(),
							Message: "Must be between 1 and 4000",
						},
					},
					{
						Name:        "responseFormat",
						Type:        "select",
						Label:       "Response Format",
						Description: "Format of the response",
						Required:    false,
						Default:     "text",
						Options: []FormFieldOption{
							{Label: "Text", Value: "text"},
							{Label: "JSON", Value: "json"},
							{Label: "Markdown", Value: "markdown"},
						},
					},
				},
				Preview: "ai-processor-preview",
				Help:    "This plugin processes text using OpenAI's GPT models",
			},
			Dependencies: []string{"http-client", "json-parser"},
		},
		Schema: NodePluginSchema{
			Properties: map[string]SchemaProperty{
				"apiKey": {
					Type:        "string",
					Description: "OpenAI API key",
				},
				"maxTokens": {
					Type:        "number",
					Description: "Maximum tokens to generate",
					Default:     1000,
				},
				"responseFormat": {
					Type:        "string",
					Description: "Response format",
					Default:     "text",
					Enum:        []string{"text", "json", "markdown"},
				},
			},
			Required: []string{"apiKey"},
		},
	}

	err := db.Create(plugin).Error
	assert.NoError(t, err)

	// 验证复杂定义的存储和读取
	var retrieved NodePlugin
	err = db.First(&retrieved, plugin.ID).Error
	assert.NoError(t, err)

	assert.Equal(t, "ai-processor", retrieved.Definition.Type)
	assert.Len(t, retrieved.Definition.Inputs, 3)
	assert.Len(t, retrieved.Definition.Outputs, 2)
	assert.Equal(t, 60, retrieved.Definition.Runtime.Timeout)
	assert.Len(t, retrieved.Definition.UI.ConfigForm, 3)
	assert.Len(t, retrieved.Definition.Dependencies, 2)
	assert.Contains(t, retrieved.Definition.Dependencies, "http-client")
	assert.Contains(t, retrieved.Definition.Dependencies, "json-parser")

	// 验证表单字段
	apiKeyField := retrieved.Definition.UI.ConfigForm[0]
	assert.Equal(t, "apiKey", apiKeyField.Name)
	assert.Equal(t, "password", apiKeyField.Type)
	assert.True(t, apiKeyField.Required)
	assert.Equal(t, "^sk-[a-zA-Z0-9]{48}$", apiKeyField.Validation.Pattern)

	maxTokensField := retrieved.Definition.UI.ConfigForm[1]
	assert.Equal(t, "maxTokens", maxTokensField.Name)
	assert.Equal(t, "number", maxTokensField.Type)
	assert.False(t, maxTokensField.Required)
	assert.Equal(t, float64(1000), maxTokensField.Default)
	assert.NotNil(t, maxTokensField.Validation.Min)
	assert.Equal(t, 1.0, *maxTokensField.Validation.Min)

	formatField := retrieved.Definition.UI.ConfigForm[2]
	assert.Equal(t, "responseFormat", formatField.Name)
	assert.Equal(t, "select", formatField.Type)
	assert.Len(t, formatField.Options, 3)
	assert.Equal(t, "Text", formatField.Options[0].Label)
	assert.Equal(t, "text", formatField.Options[0].Value)
}

func TestMigrateNodePluginTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = MigrateNodePluginTables(db)
	assert.NoError(t, err)

	// 验证表是否创建成功
	tables := []string{
		"node_plugins",
		"node_plugin_versions",
		"node_plugin_reviews",
		"node_plugin_installations",
	}

	for _, table := range tables {
		var count int64
		err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count, "Table %s should exist", table)
	}
}

// Benchmark tests
func BenchmarkNodePlugin_Create(b *testing.B) {
	db := setupNodePluginTestDB(&testing.T{})
	user := createTestUserForNodePlugin(&testing.T{}, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := &NodePlugin{
			UserID:      user.ID,
			Name:        "Benchmark Plugin " + string(rune(i)),
			Slug:        "benchmark-plugin-" + string(rune(i)),
			DisplayName: "Benchmark Plugin " + string(rune(i)),
			Category:    NodePluginCategoryUtility,
			Version:     "1.0.0",
			Definition: NodePluginDefinition{
				Type: "utility",
				Runtime: NodePluginRuntime{
					Type:   "script",
					Config: JSONMap{"test": true},
				},
			},
		}
		db.Create(plugin)
	}
}

func BenchmarkNodePlugin_Query(b *testing.B) {
	db := setupNodePluginTestDB(&testing.T{})
	user := createTestUserForNodePlugin(&testing.T{}, db)

	// 创建测试数据
	for i := 0; i < 100; i++ {
		plugin := &NodePlugin{
			UserID:      user.ID,
			Name:        "Query Plugin " + string(rune(i)),
			Slug:        "query-plugin-" + string(rune(i)),
			DisplayName: "Query Plugin " + string(rune(i)),
			Category:    NodePluginCategoryUtility,
			Version:     "1.0.0",
		}
		db.Create(plugin)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var plugins []NodePlugin
		db.Where("category = ?", NodePluginCategoryUtility).Find(&plugins)
	}
}
