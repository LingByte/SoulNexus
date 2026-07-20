package models

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	McpMarketStatusDraft     = "draft"
	McpMarketStatusPublished = "published"
	McpMarketStatusArchived  = "archived"

	McpMarketCategoryCRM     = "crm"
	McpMarketCategoryOrder   = "order"
	McpMarketCategoryUtility = "utility"
	McpMarketCategoryCustom  = "custom"
)

// McpMarketItem is a publishable MCP package in the tenant-facing marketplace.
type McpMarketItem struct {
	common.BaseModel

	Slug        string `json:"slug" gorm:"size:64;not null;uniqueIndex"`
	Name        string `json:"name" gorm:"size:64;not null"`
	DisplayName string `json:"displayName" gorm:"size:128;not null;default:''"`
	Description string `json:"description,omitempty" gorm:"type:text"`
	Category    string `json:"category" gorm:"size:32;not null;default:'utility'"`
	Icon        string `json:"icon,omitempty" gorm:"size:256"`
	LogoURL     string `json:"logoUrl,omitempty" gorm:"column:logo_url;size:512"`
	Tags        string `json:"tags,omitempty" gorm:"size:512"` // comma-separated labels
	Version     string `json:"version" gorm:"size:32;not null;default:'1.0.0'"`
	Status      string `json:"status" gorm:"size:16;not null;default:'draft';index"`
	Author      string `json:"author,omitempty" gorm:"size:128"`
	// AuthorTenantID is set when a tenant publishes their custom MCP; 0 = platform.
	AuthorTenantID uint `json:"authorTenantId,string,omitempty" gorm:"index;default:0"`
	// SourceToolID links to tenant_assistant_tools.id when a tenant publishes a custom MCP.
	SourceToolID *uint `json:"sourceToolId,string,omitempty" gorm:"index"`

	Kind        string         `json:"kind" gorm:"size:16;not null;default:'mcp_sse'"`
	McpSSEURL   string         `json:"mcpSseUrl" gorm:"column:mcp_sse_url;size:2048"`
	HeadersJSON datatypes.JSON `json:"headers,omitempty" gorm:"column:headers_json;type:json"`
	TimeoutMS   int            `json:"timeoutMs" gorm:"not null;default:15000"`

	InstallCount int `json:"installCount" gorm:"not null;default:0"`
}

func (McpMarketItem) TableName() string {
	return constants.MCP_MARKET_ITEM_TABLE_NAME
}

func NormalizeMcpMarketStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case McpMarketStatusPublished:
		return McpMarketStatusPublished
	case McpMarketStatusArchived:
		return McpMarketStatusArchived
	default:
		return McpMarketStatusDraft
	}
}

func NormalizeMcpMarketSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}

func ValidateMcpMarketItem(row *McpMarketItem) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.Slug = NormalizeMcpMarketSlug(row.Slug)
	row.Name = NormalizeAssistantToolName(row.Name)
	row.Status = NormalizeMcpMarketStatus(row.Status)
	row.Kind = NormalizeAssistantToolKind(row.Kind)
	if row.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if row.Name == "" {
		row.Name = strings.ReplaceAll(row.Slug, "-", "_")
	}
	if row.DisplayName == "" {
		row.DisplayName = row.Name
	}
	if row.Category == "" {
		row.Category = McpMarketCategoryUtility
	}
	if row.Version == "" {
		row.Version = "1.0.0"
	}
	if row.Kind != AssistantToolKindMCPSSE {
		return fmt.Errorf("market items currently support mcp_sse only")
	}
	if strings.TrimSpace(row.McpSSEURL) == "" {
		return fmt.Errorf("mcpSseUrl is required")
	}
	if row.TimeoutMS <= 0 {
		row.TimeoutMS = 15000
	}
	return nil
}

func ListPublishedMcpMarketItems(db *gorm.DB, category, keyword string) ([]McpMarketItem, error) {
	if db == nil {
		return nil, nil
	}
	q := db.Where("status = ?", McpMarketStatusPublished)
	if c := strings.TrimSpace(category); c != "" && c != "all" {
		q = q.Where("category = ?", c)
	}
	if k := strings.TrimSpace(keyword); k != "" {
		like := "%" + k + "%"
		q = q.Where(
			"display_name LIKE ? OR name LIKE ? OR description LIKE ? OR slug LIKE ? OR tags LIKE ?",
			like, like, like, like, like,
		)
	}
	var rows []McpMarketItem
	err := q.Order("install_count DESC, id DESC").Find(&rows).Error
	return rows, err
}

func ListMcpMarketItemsAdmin(db *gorm.DB, status string, page, size int) ([]McpMarketItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	q := db.Model(&McpMarketItem{})
	if s := NormalizeMcpMarketStatus(status); status != "" && status != "all" {
		q = q.Where("status = ?", s)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []McpMarketItem
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func GetMcpMarketItem(db *gorm.DB, id uint) (McpMarketItem, error) {
	var row McpMarketItem
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

func GetMcpMarketItemBySlug(db *gorm.DB, slug string) (McpMarketItem, error) {
	var row McpMarketItem
	err := db.Where("slug = ?", NormalizeMcpMarketSlug(slug)).First(&row).Error
	return row, err
}

func CreateMcpMarketItem(db *gorm.DB, row *McpMarketItem) error {
	if err := ValidateMcpMarketItem(row); err != nil {
		return err
	}
	return db.Create(row).Error
}

func UpdateMcpMarketItem(db *gorm.DB, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	if slug, ok := updates["slug"].(string); ok {
		updates["slug"] = NormalizeMcpMarketSlug(slug)
	}
	if status, ok := updates["status"].(string); ok {
		updates["status"] = NormalizeMcpMarketStatus(status)
	}
	if name, ok := updates["name"].(string); ok {
		updates["name"] = NormalizeAssistantToolName(name)
	}
	return db.Model(&McpMarketItem{}).Where("id = ?", id).Updates(updates).Error
}

func DeleteMcpMarketItem(db *gorm.DB, id uint) error {
	return db.Where("id = ?", id).Delete(&McpMarketItem{}).Error
}

func IncrementMcpMarketInstallCount(db *gorm.DB, id uint) error {
	return db.Model(&McpMarketItem{}).Where("id = ?", id).
		UpdateColumn("install_count", gorm.Expr("install_count + 1")).Error
}

// EnsureDefaultMcpMarketItems seeds a demo LingMCP listing when the market is empty.
func EnsureDefaultMcpMarketItems(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var n int64
	if err := db.Model(&McpMarketItem{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	row := &McpMarketItem{
		Slug:        "lingmcp",
		Name:        "lingmcp",
		DisplayName: "LingMCP",
		Description: "官方示例 MCP：含 order_lookup / system_info。默认 SSE http://127.0.0.1:3920/sse",
		Category:    McpMarketCategoryOrder,
		Version:     "1.0.0",
		Status:      McpMarketStatusPublished,
		Author:      "SoulNexus",
		Kind:        AssistantToolKindMCPSSE,
		McpSSEURL:   "http://127.0.0.1:3920/sse",
		TimeoutMS:   15000,
	}
	return CreateMcpMarketItem(db, row)
}

// PublishTenantMcpMarketInput carries tenant publish/update fields for a custom MCP tool.
type PublishTenantMcpMarketInput struct {
	TenantID    uint
	ToolID      uint
	Slug        string
	Name        string
	DisplayName string
	Description string
	Category    string
	Version     string
	LogoURL     string
	Tags        string
	Publish     bool
	McpSSEURL   string
	HeadersJSON datatypes.JSON
	TimeoutMS   int
}

// PublishOrUpdateTenantMcpMarketItem creates or updates a tenant-owned market listing.
func PublishOrUpdateTenantMcpMarketItem(db *gorm.DB, in PublishTenantMcpMarketInput) (*McpMarketItem, error) {
	if db == nil || in.TenantID == 0 || in.ToolID == 0 {
		return nil, gorm.ErrInvalidData
	}
	status := McpMarketStatusDraft
	if in.Publish {
		status = McpMarketStatusPublished
	}
	var existing McpMarketItem
	err := db.Where("author_tenant_id = ? AND source_tool_id = ?", in.TenantID, in.ToolID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		slug := NormalizeMcpMarketSlug(in.Slug)
		err2 := db.Where("author_tenant_id = ? AND slug = ?", in.TenantID, slug).First(&existing).Error
		if err2 != nil && err2 != gorm.ErrRecordNotFound {
			return nil, err2
		}
		if err2 == gorm.ErrRecordNotFound {
			existing = McpMarketItem{}
		}
	}
	if existing.ID != 0 {
		updates := map[string]any{
			"display_name":   in.DisplayName,
			"description":    in.Description,
			"category":       in.Category,
			"mcp_sse_url":    in.McpSSEURL,
			"headers_json":   in.HeadersJSON,
			"timeout_ms":     in.TimeoutMS,
			"status":         status,
			"source_tool_id": in.ToolID,
		}
		if v := strings.TrimSpace(in.Version); v != "" {
			updates["version"] = v
		}
		if v := strings.TrimSpace(in.LogoURL); v != "" {
			updates["logo_url"] = v
		}
		if in.Tags != "" {
			updates["tags"] = strings.TrimSpace(in.Tags)
		}
		if err := UpdateMcpMarketItem(db, existing.ID, updates); err != nil {
			return nil, err
		}
		row, err := GetMcpMarketItem(db, existing.ID)
		if err != nil {
			return nil, err
		}
		return &row, nil
	}
	toolID := in.ToolID
	item := &McpMarketItem{
		Slug:           in.Slug,
		Name:           in.Name,
		DisplayName:    in.DisplayName,
		Description:    in.Description,
		Category:       in.Category,
		Version:        in.Version,
		LogoURL:        strings.TrimSpace(in.LogoURL),
		Tags:           strings.TrimSpace(in.Tags),
		Status:         status,
		Author:         "tenant",
		AuthorTenantID: in.TenantID,
		SourceToolID:   &toolID,
		Kind:           AssistantToolKindMCPSSE,
		McpSSEURL:      in.McpSSEURL,
		HeadersJSON:    in.HeadersJSON,
		TimeoutMS:      in.TimeoutMS,
	}
	if err := CreateMcpMarketItem(db, item); err != nil {
		return nil, err
	}
	return item, nil
}

// DelistTenantMcpMarketItem archives a published listing owned by the tenant for a source tool.
func DelistTenantMcpMarketItem(db *gorm.DB, tenantID, toolID uint) (*McpMarketItem, error) {
	if db == nil || tenantID == 0 || toolID == 0 {
		return nil, gorm.ErrInvalidData
	}
	item, err := GetPublishedMcpMarketItemForSourceTool(db, tenantID, toolID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("该 MCP 工具未发布到市场")
	}
	if err := UpdateMcpMarketItem(db, item.ID, map[string]any{"status": McpMarketStatusArchived}); err != nil {
		return nil, err
	}
	row, err := GetMcpMarketItem(db, item.ID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetPublishedMcpMarketItemForSourceTool returns the active published listing for a tenant tool, if any.
func GetPublishedMcpMarketItemForSourceTool(db *gorm.DB, tenantID, toolID uint) (*McpMarketItem, error) {
	if db == nil || tenantID == 0 || toolID == 0 {
		return nil, nil
	}
	var item McpMarketItem
	err := db.Where(
		"author_tenant_id = ? AND source_tool_id = ? AND status = ?",
		tenantID, toolID, McpMarketStatusPublished,
	).First(&item).Error
	if err == nil {
		return &item, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	tool, err := GetTenantAssistantTool(db, tenantID, toolID)
	if err != nil {
		return nil, nil
	}
	err = db.Where(
		"author_tenant_id = ? AND slug = ? AND status = ?",
		tenantID, NormalizeMcpMarketSlug(tool.Name), McpMarketStatusPublished,
	).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if item.SourceToolID == nil {
		_ = UpdateMcpMarketItem(db, item.ID, map[string]any{"source_tool_id": toolID})
		item.SourceToolID = &toolID
	}
	return &item, nil
}

// MapPublishedMcpMarketItemsBySourceTool batch-loads published listings for tenant tools.
func MapPublishedMcpMarketItemsBySourceTool(db *gorm.DB, tenantID uint, toolIDs []uint) (map[uint]McpMarketItem, error) {
	out := map[uint]McpMarketItem{}
	if db == nil || tenantID == 0 || len(toolIDs) == 0 {
		return out, nil
	}
	var rows []McpMarketItem
	err := db.Where(
		"author_tenant_id = ? AND source_tool_id IN ? AND status = ?",
		tenantID, toolIDs, McpMarketStatusPublished,
	).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.SourceToolID != nil {
			out[*row.SourceToolID] = row
		}
	}
	var tools []TenantAssistantTool
	if err := db.Where("tenant_id = ? AND id IN ?", tenantID, toolIDs).Find(&tools).Error; err != nil {
		return nil, err
	}
	for _, tool := range tools {
		if _, ok := out[tool.ID]; ok {
			continue
		}
		item, err := GetPublishedMcpMarketItemForSourceTool(db, tenantID, tool.ID)
		if err != nil {
			return nil, err
		}
		if item != nil {
			out[tool.ID] = *item
		}
	}
	return out, nil
}

// AssertTenantAssistantToolDeletable blocks delete when the tool is still published in the market.
func AssertTenantAssistantToolDeletable(db *gorm.DB, tenantID, toolID uint) error {
	item, err := GetPublishedMcpMarketItemForSourceTool(db, tenantID, toolID)
	if err != nil {
		return err
	}
	if item != nil {
		return fmt.Errorf("该 MCP 工具仍发布在市场中，请先下架后再删除")
	}
	return nil
}

// ActivateMcpMarketItemForTenant installs a published market item into the tenant catalog.
func ActivateMcpMarketItemForTenant(db *gorm.DB, tenantID, marketItemID uint) (*TenantAssistantTool, error) {
	if db == nil || tenantID == 0 || marketItemID == 0 {
		return nil, gorm.ErrInvalidData
	}
	item, err := GetMcpMarketItem(db, marketItemID)
	if err != nil {
		return nil, err
	}
	if item.Status != McpMarketStatusPublished {
		return nil, fmt.Errorf("mcp market item is not published")
	}
	var existing TenantAssistantTool
	err = db.Where("tenant_id = ? AND market_item_id = ?", tenantID, marketItemID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	name := NormalizeAssistantToolName("mkt_" + item.Slug)
	row := &TenantAssistantTool{
		TenantID:       tenantID,
		Name:           name,
		DisplayName:    item.DisplayName,
		Description:    item.Description,
		Kind:           AssistantToolKindMCPSSE,
		Enabled:        true,
		McpSSEURL:      item.McpSSEURL,
		HeadersJSON:    item.HeadersJSON,
		TimeoutMS:      item.TimeoutMS,
		Source:         AssistantToolSourceMarket,
		MarketItemID:   &marketItemID,
	}
	if err := CreateTenantAssistantTool(db, row); err != nil {
		return nil, err
	}
	_ = IncrementMcpMarketInstallCount(db, marketItemID)
	return row, nil
}
