package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWorkflowPluginTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = MigrateWorkflowPluginTables(db)
	require.NoError(t, err)

	// 迁移依赖表
	err = db.AutoMigrate(&User{}, &Group{}, &WorkflowDefinition{})
	require.NoError(t, err)

	return db
}

func createTestUserForWorkflowPlugin(t *testing.T, db *gorm.DB) *User {
	user := &User{
		Email:    "workflow@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func createTestWorkflowDefinition(t *testing.T, db *gorm.DB, userID uint) *WorkflowDefinition {
	workflow := &WorkflowDefinition{
		UserID:      userID,
		Name:        "Test Workflow",
		Description: "A test workflow",
		Status:      "active",
	}
	err := db.Create(workflow).Error
	require.NoError(t, err)
	return workflow
}

func TestWorkflowPluginIOSchema_Value(t *testing.T) {
	schema := WorkflowPluginIOSchema{
		Parameters: []WorkflowPluginParameter{
			{
				Name:        "input",
				Type:        "string",
				Required:    true,
				Description: "Input parameter",
			},
		},
	}

	value, err := schema.Value()
	assert.NoError(t, err)
	assert.NotNil(t, value)

	// 测试空参数
	emptySchema := WorkflowPluginIOSchema{}
	value, err = emptySchema.Value()
	assert.NoError(t, err)
	assert.Nil(t, value)
}

func TestWorkflowPluginIOSchema_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected WorkflowPluginIOSchema
		wantErr  bool
	}{
		{
			name:  "valid JSON bytes",
			input: []byte(`{"parameters":[{"name":"test","type":"string","required":true}]}`),
			expected: WorkflowPluginIOSchema{
				Parameters: []WorkflowPluginParameter{
					{Name: "test", Type: "string", Required: true},
				},
			},
			wantErr: false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: WorkflowPluginIOSchema{},
			wantErr:  false,
		},
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: WorkflowPluginIOSchema{},
			wantErr:  false,
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schema WorkflowPluginIOSchema
			err := schema.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.input != "invalid" {
					assert.Equal(t, tt.expected, schema)
				}
			}
		})
	}
}

func TestWorkflowPlugin_CRUD(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)

	// 测试创建工作流插件
	plugin := &WorkflowPlugin{
		UserID:      user.ID,
		WorkflowID:  workflow.ID,
		Name:        "Data Processing Plugin",
		Slug:        "data-processing-plugin",
		DisplayName: "Data Processing Plugin",
		Description: "A plugin for processing data workflows",
		Category:    WorkflowPluginCategoryDataProcessing,
		Version:     "1.0.0",
		Status:      WorkflowPluginStatusDraft,
		Icon:        "https://example.com/icon.svg",
		Color:       "#2196F3",
		Tags:        StringArray{"data", "processing", "workflow"},
		InputSchema: WorkflowPluginIOSchema{
			Parameters: []WorkflowPluginParameter{
				{
					Name:        "data",
					Type:        "object",
					Required:    true,
					Description: "Input data to process",
					Example:     map[string]interface{}{"key": "value"},
				},
				{
					Name:        "format",
					Type:        "string",
					Required:    false,
					Default:     "json",
					Description: "Output format",
				},
			},
		},
		OutputSchema: WorkflowPluginIOSchema{
			Parameters: []WorkflowPluginParameter{
				{
					Name:        "result",
					Type:        "object",
					Required:    true,
					Description: "Processed data result",
				},
				{
					Name:        "metadata",
					Type:        "object",
					Required:    false,
					Description: "Processing metadata",
				},
			},
		},
		WorkflowSnapshot: WorkflowGraph{
			Nodes: []WorkflowNodeSchema{
				{
					ID:         "node1",
					Type:       "input",
					Properties: StringMap{"label": "Input"},
				},
				{
					ID:         "node2",
					Type:       "process",
					Properties: StringMap{"label": "Process"},
				},
			},
			Edges: []WorkflowEdgeSchema{
				{
					ID:     "edge1",
					Source: "node1",
					Target: "node2",
				},
			},
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

	// 测试读取工作流插件
	var retrieved WorkflowPlugin
	err = db.Preload("User").Preload("Workflow").Preload("Versions").Preload("Reviews").First(&retrieved, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, plugin.Name, retrieved.Name)
	assert.Equal(t, plugin.Slug, retrieved.Slug)
	assert.Equal(t, WorkflowPluginCategoryDataProcessing, retrieved.Category)
	assert.Equal(t, WorkflowPluginStatusDraft, retrieved.Status)
	assert.Len(t, retrieved.Tags, 3)
	assert.Contains(t, retrieved.Tags, "data")
	assert.Len(t, retrieved.InputSchema.Parameters, 2)
	assert.Len(t, retrieved.OutputSchema.Parameters, 2)
	assert.Len(t, retrieved.WorkflowSnapshot.Nodes, 2)
	assert.Len(t, retrieved.WorkflowSnapshot.Edges, 1)
	assert.Equal(t, user.Email, retrieved.User.Email)
	assert.Equal(t, workflow.Name, retrieved.Workflow.Name)

	// 测试更新工作流插件
	retrieved.Status = WorkflowPluginStatusPublished
	retrieved.Version = "1.1.0"
	retrieved.DownloadCount = 50
	retrieved.StarCount = 10
	retrieved.Rating = 4.2
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated WorkflowPlugin
	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, WorkflowPluginStatusPublished, updated.Status)
	assert.Equal(t, "1.1.0", updated.Version)
	assert.Equal(t, 50, updated.DownloadCount)
	assert.Equal(t, 10, updated.StarCount)
	assert.Equal(t, 4.2, updated.Rating)

	// 测试软删除
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	var deleted WorkflowPlugin
	err = db.First(&deleted, plugin.ID).Error
	assert.Error(t, err)

	// 测试查询包含软删除的记录
	err = db.Unscoped().First(&deleted, plugin.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt)
}

func TestWorkflowPluginVersion_CRUD(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)

	// 创建工作流插件
	plugin := &WorkflowPlugin{
		UserID:      user.ID,
		WorkflowID:  workflow.ID,
		Name:        "Versioned Plugin",
		Slug:        "versioned-plugin",
		DisplayName: "Versioned Plugin",
		Category:    WorkflowPluginCategoryUtility,
		Version:     "1.0.0",
	}
	err := db.Create(plugin).Error
	require.NoError(t, err)

	// 测试创建插件版本
	version := &WorkflowPluginVersion{
		PluginID: plugin.ID,
		Version:  "1.1.0",
		WorkflowSnapshot: WorkflowGraph{
			Nodes: []WorkflowNodeSchema{
				{
					ID:         "node1",
					Type:       "start",
					Properties: StringMap{"label": "Start"},
				},
			},
		},
		InputSchema: WorkflowPluginIOSchema{
			Parameters: []WorkflowPluginParameter{
				{
					Name:     "input",
					Type:     "string",
					Required: true,
				},
			},
		},
		OutputSchema: WorkflowPluginIOSchema{
			Parameters: []WorkflowPluginParameter{
				{
					Name:     "output",
					Type:     "string",
					Required: true,
				},
			},
		},
		ChangeLog: "Added new features and improved performance",
	}

	err = db.Create(version).Error
	assert.NoError(t, err)
	assert.NotZero(t, version.ID)
	assert.NotZero(t, version.CreatedAt)

	// 测试读取插件版本
	var retrieved WorkflowPluginVersion
	err = db.Preload("Plugin").First(&retrieved, version.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, plugin.ID, retrieved.PluginID)
	assert.Equal(t, "1.1.0", retrieved.Version)
	assert.Len(t, retrieved.WorkflowSnapshot.Nodes, 1)
	assert.Equal(t, "start", retrieved.WorkflowSnapshot.Nodes[0].Type)
	assert.Len(t, retrieved.InputSchema.Parameters, 1)
	assert.Len(t, retrieved.OutputSchema.Parameters, 1)
	assert.Equal(t, "Added new features and improved performance", retrieved.ChangeLog)
	assert.Equal(t, plugin.Name, retrieved.Plugin.Name)

	// 测试更新版本
	retrieved.ChangeLog = "Updated changelog with bug fixes"
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated WorkflowPluginVersion
	err = db.First(&updated, version.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated changelog with bug fixes", updated.ChangeLog)
}

func TestWorkflowPluginReview_CRUD(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)
	reviewer := &User{
		Email:    "reviewer@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(reviewer).Error
	require.NoError(t, err)

	// 创建工作流插件
	plugin := &WorkflowPlugin{
		UserID:      user.ID,
		WorkflowID:  workflow.ID,
		Name:        "Reviewed Plugin",
		Slug:        "reviewed-plugin",
		DisplayName: "Reviewed Plugin",
		Category:    WorkflowPluginCategoryUtility,
		Version:     "1.0.0",
		Status:      WorkflowPluginStatusPublished,
	}
	err = db.Create(plugin).Error
	require.NoError(t, err)

	// 测试创建评价
	review := &WorkflowPluginReview{
		PluginID: plugin.ID,
		UserID:   reviewer.ID,
		Rating:   5,
		Comment:  "Excellent workflow plugin! Very useful and well-designed.",
	}

	err = db.Create(review).Error
	assert.NoError(t, err)
	assert.NotZero(t, review.ID)
	assert.NotZero(t, review.CreatedAt)

	// 测试读取评价
	var retrieved WorkflowPluginReview
	err = db.Preload("Plugin").Preload("User").First(&retrieved, review.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, plugin.ID, retrieved.PluginID)
	assert.Equal(t, reviewer.ID, retrieved.UserID)
	assert.Equal(t, 5, retrieved.Rating)
	assert.Equal(t, "Excellent workflow plugin! Very useful and well-designed.", retrieved.Comment)
	assert.Equal(t, plugin.Name, retrieved.Plugin.Name)
	assert.Equal(t, reviewer.Email, retrieved.User.Email)

	// 测试更新评价
	retrieved.Rating = 4
	retrieved.Comment = "Good plugin, but could use some improvements in the UI."
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated WorkflowPluginReview
	err = db.First(&updated, review.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 4, updated.Rating)
	assert.Equal(t, "Good plugin, but could use some improvements in the UI.", updated.Comment)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestWorkflowPluginInstallation_CRUD(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)
	installer := &User{
		Email:    "installer@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(installer).Error
	require.NoError(t, err)

	// 创建工作流插件
	plugin := &WorkflowPlugin{
		UserID:      user.ID,
		WorkflowID:  workflow.ID,
		Name:        "Installable Plugin",
		Slug:        "installable-plugin",
		DisplayName: "Installable Plugin",
		Category:    WorkflowPluginCategoryUtility,
		Version:     "1.0.0",
		Status:      WorkflowPluginStatusPublished,
	}
	err = db.Create(plugin).Error
	require.NoError(t, err)

	// 测试创建安装记录
	installation := &WorkflowPluginInstallation{
		UserID:   installer.ID,
		PluginID: plugin.ID,
		Version:  "1.0.0",
		Status:   "active",
		Config: JSONMap{
			"apiEndpoint": "https://api.example.com",
			"timeout":     30,
			"retryCount":  3,
			"enabled":     true,
		},
	}

	err = db.Create(installation).Error
	assert.NoError(t, err)
	assert.NotZero(t, installation.ID)
	assert.NotZero(t, installation.CreatedAt)

	// 测试读取安装记录
	var retrieved WorkflowPluginInstallation
	err = db.Preload("User").Preload("Plugin").First(&retrieved, installation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, installer.ID, retrieved.UserID)
	assert.Equal(t, plugin.ID, retrieved.PluginID)
	assert.Equal(t, "1.0.0", retrieved.Version)
	assert.Equal(t, "active", retrieved.Status)
	assert.Equal(t, "https://api.example.com", retrieved.Config["apiEndpoint"])
	assert.Equal(t, float64(30), retrieved.Config["timeout"])
	assert.Equal(t, installer.Email, retrieved.User.Email)
	assert.Equal(t, plugin.Name, retrieved.Plugin.Name)

	// 测试更新安装记录
	retrieved.Status = "inactive"
	retrieved.Config["timeout"] = 60
	retrieved.Config["reason"] = "temporarily disabled"
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated WorkflowPluginInstallation
	err = db.First(&updated, installation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "inactive", updated.Status)
	assert.Equal(t, float64(60), updated.Config["timeout"])
	assert.Equal(t, "temporarily disabled", updated.Config["reason"])
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestWorkflowPlugin_Categories(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)

	categories := []WorkflowPluginCategory{
		WorkflowPluginCategoryDataProcessing,
		WorkflowPluginCategoryAPIIntegration,
		WorkflowPluginCategoryAIService,
		WorkflowPluginCategoryNotification,
		WorkflowPluginCategoryUtility,
		WorkflowPluginCategoryBusiness,
		WorkflowPluginCategoryCustom,
	}

	// 创建不同分类的插件
	for i, category := range categories {
		plugin := &WorkflowPlugin{
			UserID:      user.ID,
			WorkflowID:  workflow.ID,
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
	var dataPlugins []WorkflowPlugin
	err := db.Where("category = ?", WorkflowPluginCategoryDataProcessing).Find(&dataPlugins).Error
	assert.NoError(t, err)
	assert.Len(t, dataPlugins, 1)
	assert.Equal(t, WorkflowPluginCategoryDataProcessing, dataPlugins[0].Category)

	// 查询所有插件
	var allPlugins []WorkflowPlugin
	err = db.Find(&allPlugins).Error
	assert.NoError(t, err)
	assert.Len(t, allPlugins, len(categories))
}

func TestWorkflowPlugin_StatusTransitions(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)

	plugin := &WorkflowPlugin{
		UserID:      user.ID,
		WorkflowID:  workflow.ID,
		Name:        "Status Test Plugin",
		Slug:        "status-test-plugin",
		DisplayName: "Status Test Plugin",
		Category:    WorkflowPluginCategoryUtility,
		Version:     "1.0.0",
		Status:      WorkflowPluginStatusDraft,
	}
	err := db.Create(plugin).Error
	require.NoError(t, err)

	// 测试状态转换：草稿 -> 发布
	plugin.Status = WorkflowPluginStatusPublished
	err = db.Save(plugin).Error
	assert.NoError(t, err)

	var updated WorkflowPlugin
	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, WorkflowPluginStatusPublished, updated.Status)

	// 测试状态转换：发布 -> 归档
	updated.Status = WorkflowPluginStatusArchived
	err = db.Save(&updated).Error
	assert.NoError(t, err)

	err = db.First(&updated, plugin.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, WorkflowPluginStatusArchived, updated.Status)
}

func TestWorkflowPlugin_ComplexWorkflowSnapshot(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)

	// 测试复杂的工作流快照
	complexWorkflow := WorkflowGraph{
		Nodes: []WorkflowNodeSchema{
			{
				ID:   "start",
				Type: "trigger",
				Properties: StringMap{
					"label":       "HTTP Trigger",
					"method":      "POST",
					"path":        "/webhook",
					"description": "Receives HTTP requests",
				},
				Position: &Point{X: 100, Y: 100},
			},
			{
				ID:   "validate",
				Type: "validator",
				Properties: StringMap{
					"label":  "Data Validator",
					"schema": `{"type": "object"}`,
					"rules":  `["required", "format"]`,
				},
				Position: &Point{X: 300, Y: 100},
			},
			{
				ID:   "process",
				Type: "processor",
				Properties: StringMap{
					"label":     "Data Processor",
					"algorithm": "transform",
					"config":    `{"format": "json"}`,
				},
				Position: &Point{X: 500, Y: 100},
			},
			{
				ID:   "notify",
				Type: "notification",
				Properties: StringMap{
					"label":    "Send Notification",
					"channel":  "email",
					"template": "success",
				},
				Position: &Point{X: 700, Y: 100},
			},
		},
		Edges: []WorkflowEdgeSchema{
			{
				ID:     "edge1",
				Source: "start",
				Target: "validate",
				Type:   "default",
			},
			{
				ID:     "edge2",
				Source: "validate",
				Target: "process",
				Type:   "success",
			},
			{
				ID:     "edge3",
				Source: "process",
				Target: "notify",
				Type:   "default",
			},
		},
	}

	plugin := &WorkflowPlugin{
		UserID:           user.ID,
		WorkflowID:       workflow.ID,
		Name:             "Complex Workflow Plugin",
		Slug:             "complex-workflow-plugin",
		DisplayName:      "Complex Workflow Plugin",
		Category:         WorkflowPluginCategoryBusiness,
		Version:          "1.0.0",
		WorkflowSnapshot: complexWorkflow,
	}

	err := db.Create(plugin).Error
	assert.NoError(t, err)

	// 验证复杂工作流的存储和读取
	var retrieved WorkflowPlugin
	err = db.First(&retrieved, plugin.ID).Error
	assert.NoError(t, err)

	assert.Len(t, retrieved.WorkflowSnapshot.Nodes, 4)
	assert.Len(t, retrieved.WorkflowSnapshot.Edges, 3)

	// 验证节点数据
	startNode := retrieved.WorkflowSnapshot.Nodes[0]
	assert.Equal(t, "start", startNode.ID)
	assert.Equal(t, "trigger", startNode.Type)
	assert.Equal(t, "HTTP Trigger", startNode.Properties["label"])
	assert.Equal(t, "POST", startNode.Properties["method"])
	assert.Equal(t, float64(100), startNode.Position.X)
	assert.Equal(t, float64(100), startNode.Position.Y)

	// 验证边数据
	edge1 := retrieved.WorkflowSnapshot.Edges[0]
	assert.Equal(t, "edge1", edge1.ID)
	assert.Equal(t, "start", edge1.Source)
	assert.Equal(t, "validate", edge1.Target)
}

func TestWorkflowPlugin_WithGroup(t *testing.T) {
	db := setupWorkflowPluginTestDB(t)
	user := createTestUserForWorkflowPlugin(t, db)
	workflow := createTestWorkflowDefinition(t, db, user.ID)

	// 创建组织
	group := &Group{
		Name:      "Test Organization",
		Type:      "company",
		CreatorID: user.ID,
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// 创建组织的工作流插件
	plugin := &WorkflowPlugin{
		UserID:      user.ID,
		GroupID:     &group.ID,
		WorkflowID:  workflow.ID,
		Name:        "Organization Plugin",
		Slug:        "organization-plugin",
		DisplayName: "Organization Plugin",
		Category:    WorkflowPluginCategoryBusiness,
		Version:     "1.0.0",
	}
	err = db.Create(plugin).Error
	require.NoError(t, err)

	// 验证组织关联
	var retrieved WorkflowPlugin
	err = db.Preload("Group").First(&retrieved, plugin.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.GroupID)
	assert.Equal(t, group.ID, *retrieved.GroupID)
	assert.Equal(t, group.Name, retrieved.Group.Name)
}

func TestMigrateWorkflowPluginTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = MigrateWorkflowPluginTables(db)
	assert.NoError(t, err)

	// 验证表是否创建成功
	tables := []string{
		"workflow_plugins",
		"workflow_plugin_versions",
		"workflow_plugin_reviews",
		"workflow_plugin_installations",
	}

	for _, table := range tables {
		var count int64
		err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count, "Table %s should exist", table)
	}
}

// Benchmark tests
func BenchmarkWorkflowPlugin_Create(b *testing.B) {
	db := setupWorkflowPluginTestDB(&testing.T{})
	user := createTestUserForWorkflowPlugin(&testing.T{}, db)
	workflow := createTestWorkflowDefinition(&testing.T{}, db, user.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := &WorkflowPlugin{
			UserID:      user.ID,
			WorkflowID:  workflow.ID,
			Name:        "Benchmark Plugin " + string(rune(i)),
			Slug:        "benchmark-plugin-" + string(rune(i)),
			DisplayName: "Benchmark Plugin " + string(rune(i)),
			Category:    WorkflowPluginCategoryUtility,
			Version:     "1.0.0",
			WorkflowSnapshot: WorkflowGraph{
				Nodes: []WorkflowNodeSchema{
					{
						ID:         "node1",
						Type:       "start",
						Properties: StringMap{"label": "Start"},
					},
				},
			},
		}
		db.Create(plugin)
	}
}

func BenchmarkWorkflowPlugin_Query(b *testing.B) {
	db := setupWorkflowPluginTestDB(&testing.T{})
	user := createTestUserForWorkflowPlugin(&testing.T{}, db)
	workflow := createTestWorkflowDefinition(&testing.T{}, db, user.ID)

	// 创建测试数据
	for i := 0; i < 100; i++ {
		plugin := &WorkflowPlugin{
			UserID:      user.ID,
			WorkflowID:  workflow.ID,
			Name:        "Query Plugin " + string(rune(i)),
			Slug:        "query-plugin-" + string(rune(i)),
			DisplayName: "Query Plugin " + string(rune(i)),
			Category:    WorkflowPluginCategoryUtility,
			Version:     "1.0.0",
		}
		db.Create(plugin)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var plugins []WorkflowPlugin
		db.Where("category = ?", WorkflowPluginCategoryUtility).Find(&plugins)
	}
}
