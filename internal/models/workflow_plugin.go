package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// WorkflowPlugin 工作流插件 - 将工作流发布为可复用的插件
type WorkflowPlugin struct {
	ID               uint                   `json:"id" gorm:"primaryKey"`
	UserID           uint                   `json:"userId" gorm:"index"`                    // 创建者ID
	GroupID          *uint                  `json:"groupId,omitempty" gorm:"index"`         // 组织ID（可选）
	WorkflowID       uint                   `json:"workflowId" gorm:"index"`                // 源工作流ID
	Name             string                 `json:"name" gorm:"size:128;not null"`          // 插件名称
	Slug             string                 `json:"slug" gorm:"size:128;uniqueIndex"`       // 唯一标识符
	DisplayName      string                 `json:"displayName" gorm:"size:256;not null"`   // 显示名称
	Description      string                 `json:"description" gorm:"type:text"`           // 描述
	Category         WorkflowPluginCategory `json:"category" gorm:"size:32;not null"`       // 分类
	Version          string                 `json:"version" gorm:"size:32;default:'1.0.0'"` // 版本号
	Status           WorkflowPluginStatus   `json:"status" gorm:"size:32;default:'draft'"`  // 状态
	Icon             string                 `json:"icon" gorm:"size:256"`                   // 图标URL
	Color            string                 `json:"color" gorm:"size:16;default:'#6366f1'"` // 主题色
	Tags             StringArray            `json:"tags" gorm:"type:json"`                  // 标签
	InputSchema      WorkflowPluginIOSchema `json:"inputSchema" gorm:"type:json"`           // 输入参数定义
	OutputSchema     WorkflowPluginIOSchema `json:"outputSchema" gorm:"type:json"`          // 输出参数定义
	WorkflowSnapshot WorkflowGraph          `json:"workflowSnapshot" gorm:"type:json"`      // 工作流快照
	DownloadCount    int                    `json:"downloadCount" gorm:"default:0"`         // 下载次数
	StarCount        int                    `json:"starCount" gorm:"default:0"`             // 收藏次数
	Rating           float64                `json:"rating" gorm:"default:0"`                // 评分
	Author           string                 `json:"author" gorm:"size:128"`                 // 作者名称
	Homepage         string                 `json:"homepage" gorm:"size:512"`               // 主页链接
	Repository       string                 `json:"repository" gorm:"size:512"`             // 仓库链接
	License          string                 `json:"license" gorm:"size:64;default:'MIT'"`   // 许可证
	CreatedAt        time.Time              `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt        time.Time              `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt         `json:"-" gorm:"index"`

	// 关联
	User     *User                   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Group    *Group                  `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	Workflow *WorkflowDefinition     `json:"workflow,omitempty" gorm:"foreignKey:WorkflowID"`
	Versions []WorkflowPluginVersion `json:"versions,omitempty" gorm:"foreignKey:PluginID"`
	Reviews  []WorkflowPluginReview  `json:"reviews,omitempty" gorm:"foreignKey:PluginID"`
}

// WorkflowPluginCategory 工作流插件分类
type WorkflowPluginCategory string

const (
	WorkflowPluginCategoryDataProcessing WorkflowPluginCategory = "data_processing" // 数据处理
	WorkflowPluginCategoryAPIIntegration WorkflowPluginCategory = "api_integration" // API集成
	WorkflowPluginCategoryAIService      WorkflowPluginCategory = "ai_service"      // AI服务
	WorkflowPluginCategoryNotification   WorkflowPluginCategory = "notification"    // 通知服务
	WorkflowPluginCategoryUtility        WorkflowPluginCategory = "utility"         // 工具类
	WorkflowPluginCategoryBusiness       WorkflowPluginCategory = "business"        // 业务逻辑
	WorkflowPluginCategoryCustom         WorkflowPluginCategory = "custom"          // 自定义
)

// WorkflowPluginStatus 工作流插件状态
type WorkflowPluginStatus string

const (
	WorkflowPluginStatusDraft     WorkflowPluginStatus = "draft"     // 草稿
	WorkflowPluginStatusPublished WorkflowPluginStatus = "published" // 已发布
	WorkflowPluginStatusArchived  WorkflowPluginStatus = "archived"  // 已归档
)

// WorkflowPluginIOSchema 输入输出参数定义
type WorkflowPluginIOSchema struct {
	Parameters []WorkflowPluginParameter `json:"parameters"`
}

// WorkflowPluginParameter 参数定义
type WorkflowPluginParameter struct {
	Name        string      `json:"name"`                  // 参数名
	Type        string      `json:"type"`                  // 参数类型: string, number, boolean, object, array
	Required    bool        `json:"required"`              // 是否必需
	Default     interface{} `json:"default,omitempty"`     // 默认值
	Description string      `json:"description,omitempty"` // 描述
	Example     interface{} `json:"example,omitempty"`     // 示例值
}

// WorkflowPluginVersion 工作流插件版本
type WorkflowPluginVersion struct {
	ID               uint                   `json:"id" gorm:"primaryKey"`
	PluginID         uint                   `json:"pluginId" gorm:"index;not null"`
	Version          string                 `json:"version" gorm:"size:32;not null"`
	WorkflowSnapshot WorkflowGraph          `json:"workflowSnapshot" gorm:"type:json"`
	InputSchema      WorkflowPluginIOSchema `json:"inputSchema" gorm:"type:json"`
	OutputSchema     WorkflowPluginIOSchema `json:"outputSchema" gorm:"type:json"`
	ChangeLog        string                 `json:"changeLog" gorm:"type:text"`
	CreatedAt        time.Time              `json:"createdAt" gorm:"autoCreateTime"`
	DeletedAt        gorm.DeletedAt         `json:"-" gorm:"index"`

	// 关联
	Plugin *WorkflowPlugin `json:"plugin,omitempty" gorm:"foreignKey:PluginID"`
}

// WorkflowPluginReview 工作流插件评价
type WorkflowPluginReview struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	PluginID  uint           `json:"pluginId" gorm:"index;not null"`
	UserID    uint           `json:"userId" gorm:"index;not null"`
	Rating    int            `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"` // 1-5星评分
	Comment   string         `json:"comment" gorm:"type:text"`
	CreatedAt time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Plugin *WorkflowPlugin `json:"plugin,omitempty" gorm:"foreignKey:PluginID"`
	User   *User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// WorkflowPluginInstallation 工作流插件安装记录
type WorkflowPluginInstallation struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"userId" gorm:"index;not null"`
	PluginID  uint           `json:"pluginId" gorm:"index;not null"`
	Version   string         `json:"version" gorm:"size:32;not null"`
	Status    string         `json:"status" gorm:"size:32;default:'active'"` // active, inactive
	Config    JSONMap        `json:"config" gorm:"type:json"`                // 用户自定义配置
	CreatedAt time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	User   *User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Plugin *WorkflowPlugin `json:"plugin,omitempty" gorm:"foreignKey:PluginID"`
}

// Value implements driver.Valuer for WorkflowPluginIOSchema
func (s WorkflowPluginIOSchema) Value() (driver.Value, error) {
	if s.Parameters == nil {
		s.Parameters = []WorkflowPluginParameter{}
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner for WorkflowPluginIOSchema
func (s *WorkflowPluginIOSchema) Scan(value interface{}) error {
	if value == nil {
		*s = WorkflowPluginIOSchema{Parameters: []WorkflowPluginParameter{}}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("WorkflowPluginIOSchema: expected []byte, got %T", value)
	}
	if len(bytes) == 0 {
		*s = WorkflowPluginIOSchema{Parameters: []WorkflowPluginParameter{}}
		return nil
	}
	err := json.Unmarshal(bytes, s)
	if err == nil && s.Parameters == nil {
		s.Parameters = []WorkflowPluginParameter{}
	}
	return err
}

// MigrateWorkflowPluginTables 迁移工作流插件相关表
func MigrateWorkflowPluginTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&WorkflowPlugin{},
		&WorkflowPluginVersion{},
		&WorkflowPluginReview{},
		&WorkflowPluginInstallation{},
	)
}
