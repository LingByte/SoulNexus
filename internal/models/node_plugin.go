package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// NodePluginStatus 插件状态
type NodePluginStatus string

const (
	NodePluginStatusDraft     NodePluginStatus = "draft"     // 草稿
	NodePluginStatusPublished NodePluginStatus = "published" // 已发布
	NodePluginStatusArchived  NodePluginStatus = "archived"  // 已归档
	NodePluginStatusBanned    NodePluginStatus = "banned"    // 已禁用
)

// NodePluginCategory 插件分类
type NodePluginCategory string

const (
	NodePluginCategoryAPI          NodePluginCategory = "api"          // API调用
	NodePluginCategoryData         NodePluginCategory = "data"         // 数据处理
	NodePluginCategoryAI           NodePluginCategory = "ai"           // AI服务
	NodePluginCategoryNotification NodePluginCategory = "notification" // 通知服务
	NodePluginCategoryUtility      NodePluginCategory = "utility"      // 工具类
	NodePluginCategoryCustom       NodePluginCategory = "custom"       // 自定义
)

// NodePlugin 节点插件定义
type NodePlugin struct {
	ID          uint               `json:"id" gorm:"primaryKey"`
	UserID      uint               `json:"userId" gorm:"index"`                   // 创建者ID
	GroupID     *uint              `json:"groupId,omitempty" gorm:"index"`        // 组织ID（可选）
	Name        string             `json:"name" gorm:"size:128;not null"`         // 插件名称
	Slug        string             `json:"slug" gorm:"size:128;uniqueIndex"`      // 插件标识符
	DisplayName string             `json:"displayName" gorm:"size:128;not null"`  // 显示名称
	Description string             `json:"description" gorm:"type:text"`          // 插件描述
	Category    NodePluginCategory `json:"category" gorm:"size:32;not null"`      // 插件分类
	Version     string             `json:"version" gorm:"size:32;not null"`       // 当前版本
	Status      NodePluginStatus   `json:"status" gorm:"size:32;default:'draft'"` // 插件状态

	// 插件配置
	Icon  string      `json:"icon" gorm:"size:256"`  // 图标URL或SVG
	Color string      `json:"color" gorm:"size:16"`  // 主题色
	Tags  StringArray `json:"tags" gorm:"type:json"` // 标签

	// 技术配置
	Definition NodePluginDefinition `json:"definition" gorm:"type:json"` // 插件定义
	Schema     NodePluginSchema     `json:"schema" gorm:"type:json"`     // 配置模式

	// 统计信息
	DownloadCount uint    `json:"downloadCount" gorm:"default:0"` // 下载次数
	StarCount     uint    `json:"starCount" gorm:"default:0"`     // 收藏次数
	Rating        float64 `json:"rating" gorm:"default:0"`        // 评分

	// 元数据
	Author     string `json:"author" gorm:"size:128"`     // 作者
	Homepage   string `json:"homepage" gorm:"size:512"`   // 主页
	Repository string `json:"repository" gorm:"size:512"` // 代码仓库
	License    string `json:"license" gorm:"size:64"`     // 许可证

	// 时间戳
	CreatedAt time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Versions []NodePluginVersion `json:"versions,omitempty" gorm:"foreignKey:PluginID"`
	Reviews  []NodePluginReview  `json:"reviews,omitempty" gorm:"foreignKey:PluginID"`
}

// NodePluginDefinition 插件定义
type NodePluginDefinition struct {
	// 节点类型信息
	Type    string           `json:"type"`    // 节点类型标识
	Inputs  []NodePluginPort `json:"inputs"`  // 输入端口定义
	Outputs []NodePluginPort `json:"outputs"` // 输出端口定义

	// 执行配置
	Runtime NodePluginRuntime `json:"runtime"` // 运行时配置

	// UI配置
	UI NodePluginUI `json:"ui"` // UI配置

	// 依赖配置
	Dependencies []string `json:"dependencies,omitempty"` // 依赖的其他插件
}

// Value implements driver.Valuer interface for GORM
func (npd NodePluginDefinition) Value() (driver.Value, error) {
	return json.Marshal(npd)
}

// Scan implements sql.Scanner interface for GORM
func (npd *NodePluginDefinition) Scan(value interface{}) error {
	if value == nil {
		*npd = NodePluginDefinition{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into NodePluginDefinition", value)
	}

	return json.Unmarshal(bytes, npd)
}

// NodePluginPort 端口定义
type NodePluginPort struct {
	Name        string      `json:"name"`                  // 端口名称
	Type        string      `json:"type"`                  // 数据类型
	Required    bool        `json:"required"`              // 是否必需
	Description string      `json:"description,omitempty"` // 描述
	Default     interface{} `json:"default,omitempty"`     // 默认值
}

// NodePluginRuntime 运行时配置
type NodePluginRuntime struct {
	Type    string  `json:"type"`              // 运行时类型: script, http, builtin
	Config  JSONMap `json:"config"`            // 运行时配置
	Timeout int     `json:"timeout,omitempty"` // 超时时间(秒)
	Retry   int     `json:"retry,omitempty"`   // 重试次数
}

// NodePluginUI UI配置
type NodePluginUI struct {
	ConfigForm []NodePluginFormField `json:"configForm"`        // 配置表单
	Preview    string                `json:"preview,omitempty"` // 预览组件
	Help       string                `json:"help,omitempty"`    // 帮助文档
}

// NodePluginFormField 表单字段
type NodePluginFormField struct {
	Name        string              `json:"name"`                  // 字段名
	Type        string              `json:"type"`                  // 字段类型: text, number, select, textarea, etc.
	Label       string              `json:"label"`                 // 显示标签
	Description string              `json:"description,omitempty"` // 字段描述
	Required    bool                `json:"required"`              // 是否必需
	Default     interface{}         `json:"default,omitempty"`     // 默认值
	Options     []FormFieldOption   `json:"options,omitempty"`     // 选项(用于select等)
	Validation  FormFieldValidation `json:"validation,omitempty"`  // 验证规则
}

// FormFieldOption 表单选项
type FormFieldOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// FormFieldValidation 验证规则
type FormFieldValidation struct {
	Min     *float64 `json:"min,omitempty"`     // 最小值
	Max     *float64 `json:"max,omitempty"`     // 最大值
	Pattern string   `json:"pattern,omitempty"` // 正则表达式
	Message string   `json:"message,omitempty"` // 错误消息
}

// NodePluginSchema 配置模式
type NodePluginSchema struct {
	Properties map[string]SchemaProperty `json:"properties"`         // 属性定义
	Required   []string                  `json:"required,omitempty"` // 必需属性
}

// Value implements driver.Valuer interface for GORM
func (nps NodePluginSchema) Value() (driver.Value, error) {
	return json.Marshal(nps)
}

// Scan implements sql.Scanner interface for GORM
func (nps *NodePluginSchema) Scan(value interface{}) error {
	if value == nil {
		*nps = NodePluginSchema{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into NodePluginSchema", value)
	}

	return json.Unmarshal(bytes, nps)
}

// SchemaProperty 属性定义
type SchemaProperty struct {
	Type        string      `json:"type"`                  // 属性类型
	Description string      `json:"description,omitempty"` // 属性描述
	Default     interface{} `json:"default,omitempty"`     // 默认值
	Enum        []string    `json:"enum,omitempty"`        // 枚举值
}

// NodePluginVersion 插件版本
type NodePluginVersion struct {
	ID         uint                 `json:"id" gorm:"primaryKey"`
	PluginID   uint                 `json:"pluginId" gorm:"index"`
	Version    string               `json:"version" gorm:"size:32;not null"`
	Definition NodePluginDefinition `json:"definition" gorm:"type:json"`
	Schema     NodePluginSchema     `json:"schema" gorm:"type:json"`
	ChangeLog  string               `json:"changeLog" gorm:"type:text"`
	CreatedAt  time.Time            `json:"createdAt" gorm:"autoCreateTime"`

	// 关联
	Plugin *NodePlugin `json:"plugin,omitempty" gorm:"foreignKey:PluginID"`
}

// NodePluginReview 插件评价
type NodePluginReview struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	PluginID  uint      `json:"pluginId" gorm:"index"`
	UserID    uint      `json:"userId" gorm:"index"`
	Rating    int       `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"` // 1-5星
	Comment   string    `json:"comment" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	// 关联
	Plugin *NodePlugin `json:"plugin,omitempty" gorm:"foreignKey:PluginID"`
	User   *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// NodePluginInstallation 插件安装记录
type NodePluginInstallation struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"userId" gorm:"index"`
	PluginID    uint      `json:"pluginId" gorm:"index"`
	Version     string    `json:"version" gorm:"size:32"`
	Status      string    `json:"status" gorm:"size:32;default:'active'"` // active, disabled
	Config      JSONMap   `json:"config" gorm:"type:json"`                // 用户自定义配置
	InstalledAt time.Time `json:"installedAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	// 关联
	Plugin *NodePlugin `json:"plugin,omitempty" gorm:"foreignKey:PluginID"`
	User   *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (NodePlugin) TableName() string {
	return "node_plugins"
}

func (NodePluginVersion) TableName() string {
	return "node_plugin_versions"
}

func (NodePluginReview) TableName() string {
	return "node_plugin_reviews"
}

func (NodePluginInstallation) TableName() string {
	return "node_plugin_installations"
}

// MigrateNodePluginTables 迁移插件相关表
func MigrateNodePluginTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&NodePlugin{},
		&NodePluginVersion{},
		&NodePluginReview{},
		&NodePluginInstallation{},
	)
}
