package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// MCPMarketplaceItem MCP 广场项目
type MCPMarketplaceItem struct {
	ID            uint            `gorm:"primaryKey" json:"id"`
	Name          string          `gorm:"index" json:"name"`                             // MCP 名称
	Description   string          `json:"description"`                                   // 描述
	Category      string          `gorm:"index" json:"category"`                         // 分类（内置、社区、企业）
	Version       string          `json:"version"`                                       // 版本
	Author        string          `json:"author"`                                        // 作者
	Icon          string          `json:"icon"`                                          // 图标 URL
	Repository    string          `json:"repository"`                                    // 仓库地址
	Documentation string          `json:"documentation"`                                 // 文档链接
	Tags          datatypes.JSON  `gorm:"type:json" json:"tags"`                         // 标签
	Features      datatypes.JSON  `gorm:"type:json" json:"features"`                     // 功能列表
	Requirements  MCPRequirements `gorm:"type:json;serializer:json" json:"requirements"` // 系统要求
	Config        datatypes.JSON  `gorm:"type:json" json:"config"`                       // 默认配置

	// 统计信息
	Downloads int64   `json:"downloads"` // 下载次数
	Rating    float64 `json:"rating"`    // 评分（0-5）
	Reviews   int64   `json:"reviews"`   // 评论数

	// 状态
	Status     string `gorm:"index" json:"status"` // 状态（published、draft、deprecated）
	IsOfficial bool   `json:"is_official"`         // 是否官方
	IsFeatured bool   `json:"is_featured"`         // 是否推荐

	// 时间戳
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
}

// MCPRequirements MCP 系统要求
type MCPRequirements struct {
	MinGoVersion     string   `json:"min_go_version,omitempty"`
	MinMemory        int64    `json:"min_memory,omitempty"`        // MB
	SupportedOS      []string `json:"supported_os,omitempty"`      // linux, darwin, windows
	Dependencies     []string `json:"dependencies,omitempty"`      // 依赖的其他 MCP
	ExternalServices []string `json:"external_services,omitempty"` // 需要的外部服务
}

// Scan 实现 sql.Scanner 接口
func (r *MCPRequirements) Scan(value interface{}) error {
	bytes, _ := value.([]byte)
	return json.Unmarshal(bytes, &r)
}

// Value 实现 driver.Valuer 接口
func (r MCPRequirements) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// MCPUserInstallation 用户安装的 MCP
type MCPUserInstallation struct {
	ID            uint                `gorm:"primaryKey" json:"id"`
	UserID        uint                `gorm:"index" json:"user_id"`
	MarketplaceID uint                `gorm:"index" json:"marketplace_id"`
	Marketplace   *MCPMarketplaceItem `gorm:"foreignKey:MarketplaceID" json:"marketplace,omitempty"`

	// 安装信息
	InstalledVersion string         `json:"installed_version"`
	Config           datatypes.JSON `gorm:"type:json" json:"config"`
	Enabled          bool           `json:"enabled"`

	// 使用统计
	LastUsedAt *time.Time `json:"last_used_at"`
	UsageCount int64      `json:"usage_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MCPReview MCP 评论
type MCPReview struct {
	ID            uint `gorm:"primaryKey" json:"id"`
	UserID        uint `gorm:"index" json:"user_id"`
	MarketplaceID uint `gorm:"index" json:"marketplace_id"`

	Rating  int    `json:"rating"` // 1-5
	Title   string `json:"title"`
	Content string `json:"content"`
	Helpful int64  `json:"helpful"` // 有帮助的数量

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MCPCategory MCP 分类
type MCPCategory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);uniqueIndex" json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Order       int       `json:"order"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定表名
func (MCPMarketplaceItem) TableName() string {
	return "mcp_marketplace_items"
}

func (MCPUserInstallation) TableName() string {
	return "mcp_user_installations"
}

func (MCPReview) TableName() string {
	return "mcp_reviews"
}

func (MCPCategory) TableName() string {
	return "mcp_categories"
}

// CreateMCPMarketplaceItemRequest 创建 MCP 广场项目请求
type CreateMCPMarketplaceItemRequest struct {
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description" binding:"required"`
	Category      string                 `json:"category" binding:"required"`
	Version       string                 `json:"version" binding:"required"`
	Author        string                 `json:"author" binding:"required"`
	Icon          string                 `json:"icon"`
	Repository    string                 `json:"repository"`
	Documentation string                 `json:"documentation"`
	Tags          []string               `json:"tags"`
	Features      []string               `json:"features"`
	Requirements  MCPRequirements        `json:"requirements"`
	Config        map[string]interface{} `json:"config"`
}

// InstallMCPRequest 安装 MCP 请求
type InstallMCPRequest struct {
	MarketplaceID uint                   `json:"marketplace_id"`
	Config        map[string]interface{} `json:"config"`
}

// SearchMCPRequest 搜索 MCP 请求
type SearchMCPRequest struct {
	Query    string   `json:"query"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	SortBy   string   `json:"sort_by"` // downloads, rating, created_at
	Order    string   `json:"order"`   // asc, desc
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// MCPMarketplaceResponse 广场响应
type MCPMarketplaceResponse struct {
	Items      []MCPMarketplaceItem `json:"items"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	Categories []MCPCategory        `json:"categories,omitempty"`
}

// ReviewMCPRequest 评论 MCP 请求
type ReviewMCPRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}
