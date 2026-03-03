package models

import (
	"time"

	"gorm.io/datatypes"
)

// MCPServer MCP 服务器配置
type MCPServer struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"index" json:"user_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`                    // builtin, custom, remote
	Status      string         `json:"status"`                  // active, inactive, error
	Command     string         `json:"command"`                 // 执行命令
	Args        datatypes.JSON `gorm:"type:json" json:"args"`   // 命令参数
	Env         datatypes.JSON `gorm:"type:json" json:"env"`    // 环境变量
	Config      datatypes.JSON `gorm:"type:json" json:"config"` // 配置
	Enabled     bool           `json:"enabled"`

	// 连接信息
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"` // stdio, http, sse

	// 统计信息
	ToolCount  int        `json:"tool_count"`
	LastUsedAt *time.Time `json:"last_used_at"`
	UsageCount int64      `json:"usage_count"`
	ErrorCount int64      `json:"error_count"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MCPTool MCP 工具
type MCPTool struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ServerID    uint           `gorm:"index" json:"server_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema datatypes.JSON `gorm:"type:json" json:"input_schema"`
	CreatedAt   time.Time      `json:"created_at"`
}

// MCPCallLog MCP 调用日志
type MCPCallLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index" json:"user_id"`
	ServerID  uint           `gorm:"index" json:"server_id"`
	ToolName  string         `json:"tool_name"`
	Arguments datatypes.JSON `gorm:"type:json" json:"arguments"`
	Result    datatypes.JSON `gorm:"type:json" json:"result"`
	Error     string         `json:"error"`
	Duration  int64          `json:"duration"` // 毫秒
	Status    string         `json:"status"`   // success, error
	CreatedAt time.Time      `json:"created_at"`
}

// TableName 指定表名
func (MCPServer) TableName() string {
	return "mcp_servers"
}

func (MCPTool) TableName() string {
	return "mcp_tools"
}

func (MCPCallLog) TableName() string {
	return "mcp_call_logs"
}

// CreateMCPServerRequest 创建 MCP 服务器请求
type CreateMCPServerRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Type        string                 `json:"type" binding:"required"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Env         map[string]string      `json:"env"`
	Config      map[string]interface{} `json:"config"`
	Protocol    string                 `json:"protocol"`
	Host        string                 `json:"host"`
	Port        int                    `json:"port"`
}

// UpdateMCPServerRequest 更新 MCP 服务器请求
type UpdateMCPServerRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Enabled     *bool                  `json:"enabled"`
}

// CallMCPToolRequest 调用 MCP 工具请求
type CallMCPToolRequest struct {
	ToolName  string                 `json:"tool_name" binding:"required"`
	Arguments map[string]interface{} `json:"arguments"`
}

// CallMCPToolResponse 调用 MCP 工具响应
type CallMCPToolResponse struct {
	Result   interface{} `json:"result"`
	Error    string      `json:"error,omitempty"`
	Duration int64       `json:"duration"`
}
