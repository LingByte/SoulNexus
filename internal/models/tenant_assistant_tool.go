package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	AssistantToolKindHTTP     = "http"
	AssistantToolKindMCPStdio = "mcp_stdio"
	AssistantToolKindMCPSSE   = "mcp_sse"

	AssistantToolSourceCustom = "custom"
	AssistantToolSourceMarket = "market"
)

// TenantAssistantTool is a tenant-wide catalog entry for LLM function tools
// (HTTP webhooks, stdio MCP subprocesses, or MCP SSE endpoints).
type TenantAssistantTool struct {
	common.BaseModel

	TenantID    uint   `json:"tenantId,string" gorm:"index;not null;uniqueIndex:idx_tenant_assistant_tool_name"`
	Name        string `json:"name" gorm:"size:64;not null;uniqueIndex:idx_tenant_assistant_tool_name"`
	DisplayName string `json:"displayName" gorm:"size:128;not null;default:''"`
	Description string `json:"description,omitempty" gorm:"type:text"`
	Kind        string `json:"kind" gorm:"size:16;not null;default:'http'"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:true"`
	// Source: custom (tenant-defined) or market (activated from MCP marketplace).
	Source string `json:"source" gorm:"size:16;not null;default:'custom';index"`
	// MarketItemID links to mcp_market_items when Source=market.
	MarketItemID *uint `json:"marketItemId,string,omitempty" gorm:"index"`

	// HTTP tool fields (kind=http)
	Method         string         `json:"method,omitempty" gorm:"size:8;default:'POST'"`
	URL            string         `json:"url,omitempty" gorm:"size:2048"`
	HeadersJSON    datatypes.JSON `json:"headers,omitempty" gorm:"column:headers_json;type:json"`
	BodyTemplate   string         `json:"bodyTemplate,omitempty" gorm:"column:body_template;type:text"`
	TimeoutMS      int            `json:"timeoutMs" gorm:"not null;default:15000"`
	ParametersJSON datatypes.JSON `json:"parameters,omitempty" gorm:"column:parameters_json;type:json"`

	// MCP stdio fields (kind=mcp_stdio)
	McpCommand  string         `json:"mcpCommand,omitempty" gorm:"column:mcp_command;size:512"`
	McpArgsJSON datatypes.JSON `json:"mcpArgs,omitempty" gorm:"column:mcp_args_json;type:json"`
	McpEnvsJSON datatypes.JSON `json:"mcpEnvs,omitempty" gorm:"column:mcp_envs_json;type:json"`

	// MCP SSE fields (kind=mcp_sse)
	McpSSEURL string `json:"mcpSseUrl,omitempty" gorm:"column:mcp_sse_url;size:2048"`
	// DiscoveredToolsJSON caches tools/list results for UI (authority remains on the MCP server).
	DiscoveredToolsJSON datatypes.JSON `json:"discoveredTools,omitempty" gorm:"column:discovered_tools_json;type:json"`
}

func (TenantAssistantTool) TableName() string {
	return constants.TENANT_ASSISTANT_TOOL_TABLE_NAME
}

func NormalizeAssistantToolKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case AssistantToolKindMCPStdio:
		return AssistantToolKindMCPStdio
	case AssistantToolKindMCPSSE:
		return AssistantToolKindMCPSSE
	default:
		return AssistantToolKindHTTP
	}
}

func NormalizeAssistantToolName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func ValidateTenantAssistantTool(row *TenantAssistantTool) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.Kind = NormalizeAssistantToolKind(row.Kind)
	row.Name = NormalizeAssistantToolName(row.Name)
	if row.Source == "" {
		row.Source = AssistantToolSourceCustom
	}
	if row.Source != AssistantToolSourceMarket {
		row.Source = AssistantToolSourceCustom
	}
	if row.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if row.TenantID == 0 {
		return fmt.Errorf("tenantId is required")
	}
	switch row.Kind {
	case AssistantToolKindHTTP:
		if strings.TrimSpace(row.URL) == "" {
			return fmt.Errorf("url is required for http tools")
		}
		row.Method = normalizeHTTPMethod(row.Method)
	case AssistantToolKindMCPStdio:
		if strings.TrimSpace(row.McpCommand) == "" {
			return fmt.Errorf("mcpCommand is required for mcp_stdio tools")
		}
	case AssistantToolKindMCPSSE:
		if strings.TrimSpace(row.McpSSEURL) == "" {
			return fmt.Errorf("mcpSseUrl is required for mcp_sse tools")
		}
	}
	if row.TimeoutMS <= 0 {
		row.TimeoutMS = 8000
	}
	return nil
}

func normalizeHTTPMethod(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return strings.ToUpper(strings.TrimSpace(method))
	default:
		return "POST"
	}
}

func ListTenantAssistantTools(db *gorm.DB, tenantID uint) ([]TenantAssistantTool, error) {
	return ListTenantAssistantToolsFiltered(db, tenantID, "", false)
}

// ListTenantAssistantToolsFiltered filters by source (custom|market|"") and optionally enabledOnly.
func ListTenantAssistantToolsFiltered(db *gorm.DB, tenantID uint, source string, enabledOnly bool) ([]TenantAssistantTool, error) {
	if db == nil || tenantID == 0 {
		return nil, nil
	}
	q := db.Where("tenant_id = ?", tenantID)
	switch strings.ToLower(strings.TrimSpace(source)) {
	case AssistantToolSourceMarket:
		q = q.Where("source = ?", AssistantToolSourceMarket)
	case AssistantToolSourceCustom:
		q = q.Where("(source = ? OR source = '' OR source IS NULL)", AssistantToolSourceCustom)
	}
	if enabledOnly {
		q = q.Where("enabled = ?", true)
	}
	var rows []TenantAssistantTool
	err := q.Order("name ASC").Find(&rows).Error
	return rows, err
}

func ListEnabledTenantAssistantTools(ctx context.Context, db *gorm.DB, tenantID uint) ([]TenantAssistantTool, error) {
	if db == nil || tenantID == 0 {
		return nil, nil
	}
	var rows []TenantAssistantTool
	err := db.WithContext(ctx).
		Where("tenant_id = ? AND enabled = ?", tenantID, true).
		Order("name ASC").
		Find(&rows).Error
	return rows, err
}

func GetTenantAssistantTool(db *gorm.DB, tenantID, id uint) (TenantAssistantTool, error) {
	var row TenantAssistantTool
	err := db.Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return row, err
}

func GetTenantAssistantToolByID(db *gorm.DB, id uint) (TenantAssistantTool, error) {
	var row TenantAssistantTool
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

func CreateTenantAssistantTool(db *gorm.DB, row *TenantAssistantTool) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	if err := ValidateTenantAssistantTool(row); err != nil {
		return err
	}
	return db.Create(row).Error
}

func UpdateTenantAssistantTool(db *gorm.DB, tenantID, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	if kind, ok := updates["kind"].(string); ok {
		updates["kind"] = NormalizeAssistantToolKind(kind)
	}
	if name, ok := updates["name"].(string); ok {
		updates["name"] = NormalizeAssistantToolName(name)
	}
	if method, ok := updates["method"].(string); ok {
		updates["method"] = normalizeHTTPMethod(method)
	}
	return db.Model(&TenantAssistantTool{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates).Error
}

func DeleteTenantAssistantTool(db *gorm.DB, tenantID, id uint) error {
	return db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&TenantAssistantTool{}).Error
}

// CatalogToolRow converts the model row for providers runtime registration.
func (row TenantAssistantTool) CatalogToolRow() providers.CatalogToolRow {
	return providers.CatalogToolRow{
		ID:             row.ID,
		Name:           row.Name,
		DisplayName:    row.DisplayName,
		Description:    row.Description,
		Kind:           row.Kind,
		Enabled:        row.Enabled,
		Method:         row.Method,
		URL:            row.URL,
		HeadersJSON:    append([]byte(nil), row.HeadersJSON...),
		BodyTemplate:   row.BodyTemplate,
		TimeoutMS:      row.TimeoutMS,
		ParametersJSON: append([]byte(nil), row.ParametersJSON...),
		McpCommand:     row.McpCommand,
		McpArgsJSON:    append([]byte(nil), row.McpArgsJSON...),
		McpEnvsJSON:    append([]byte(nil), row.McpEnvsJSON...),
		McpSSEURL:      row.McpSSEURL,
	}
}

// DiscoveredMCPTools decodes the cached tools/list snapshot.
func (row TenantAssistantTool) DiscoveredMCPTools() []providers.DiscoveredMCPTool {
	if len(row.DiscoveredToolsJSON) == 0 {
		return nil
	}
	var out []providers.DiscoveredMCPTool
	if err := json.Unmarshal(row.DiscoveredToolsJSON, &out); err != nil {
		return nil
	}
	return out
}

func ListTenantAssistantToolsAdmin(db *gorm.DB, tenantID uint, page, size int) ([]TenantAssistantTool, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	q := db.Model(&TenantAssistantTool{})
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []TenantAssistantTool
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func ParseToolHeadersJSON(raw datatypes.JSON) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func ParseToolStringSliceJSON(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
