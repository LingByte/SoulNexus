package models

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	WidgetMarketStatusDraft     = "draft"
	WidgetMarketStatusPublished = "published"
	WidgetMarketStatusArchived  = "archived"

	WidgetMarketCategoryDesktopPet = "desktop_pet"
	WidgetMarketCategoryChat       = "chat_widget"
	WidgetMarketCategoryLive2D     = "live2d"
	WidgetMarketCategoryUtility    = "utility"
	WidgetMarketCategoryCustom     = "custom"
)

// WidgetMarketItem is a published 网页挂件 listing (metadata only; script via embed.js).
type WidgetMarketItem struct {
	common.BaseModel

	Slug        string `json:"slug" gorm:"size:64;not null;uniqueIndex"`
	Name        string `json:"name" gorm:"size:128;not null"`
	DisplayName string `json:"displayName" gorm:"size:256;not null;default:''"`
	Description string `json:"description,omitempty" gorm:"type:text"`
	Category    string `json:"category" gorm:"size:32;not null;default:'utility';index"`
	AvatarURL   string `json:"avatarUrl,omitempty" gorm:"column:avatar_url;size:512"`
	Tags        string `json:"tags,omitempty" gorm:"size:512"`
	Version     string `json:"version" gorm:"size:32;not null;default:'1.0.0'"`
	Status      string `json:"status" gorm:"size:16;not null;default:'draft';index"`
	Author      string `json:"author,omitempty" gorm:"size:128"`
	AuthorTenantID uint `json:"authorTenantId,string,omitempty" gorm:"index;default:0"`

	JsSourceID         string `json:"jsSourceId" gorm:"column:js_source_id;size:64;not null;index"`
	SourceJSTemplateID uint   `json:"sourceJsTemplateId,string,omitempty" gorm:"column:source_js_template_id;index"`

	InstallCount int `json:"installCount" gorm:"not null;default:0"`
}

func (WidgetMarketItem) TableName() string {
	return constants.WIDGET_MARKET_ITEM_TABLE_NAME
}

func NormalizeWidgetMarketStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case WidgetMarketStatusPublished:
		return WidgetMarketStatusPublished
	case WidgetMarketStatusArchived:
		return WidgetMarketStatusArchived
	default:
		return WidgetMarketStatusDraft
	}
}

func NormalizeWidgetMarketSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}

func ValidateWidgetMarketItem(row *WidgetMarketItem) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.Slug = NormalizeWidgetMarketSlug(row.Slug)
	row.Status = NormalizeWidgetMarketStatus(row.Status)
	if row.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if strings.TrimSpace(row.JsSourceID) == "" {
		return fmt.Errorf("jsSourceId is required")
	}
	if row.DisplayName == "" {
		row.DisplayName = row.Name
	}
	if row.Category == "" {
		row.Category = WidgetMarketCategoryUtility
	}
	if row.Version == "" {
		row.Version = "1.0.0"
	}
	return nil
}

func ListPublishedWidgetMarketItemsPage(db *gorm.DB, category, keyword string, page, size int) ([]WidgetMarketItem, int64, error) {
	q := db.Model(&WidgetMarketItem{}).Where("status = ?", WidgetMarketStatusPublished)
	if c := strings.TrimSpace(category); c != "" && c != "all" {
		q = q.Where("category = ?", c)
	}
	if k := strings.TrimSpace(keyword); k != "" {
		like := "%" + k + "%"
		q = q.Where(
			"display_name LIKE ? OR name LIKE ? OR description LIKE ? OR slug LIKE ? OR tags LIKE ? OR js_source_id LIKE ?",
			like, like, like, like, like, like,
		)
	}
	return utils.FindPage[WidgetMarketItem](q, page, size, "install_count DESC, id DESC", utils.DefaultMaxPageSize)
}

func GetWidgetMarketItemPublished(db *gorm.DB, id uint) (WidgetMarketItem, error) {
	var row WidgetMarketItem
	err := db.Where("id = ? AND status = ?", id, WidgetMarketStatusPublished).First(&row).Error
	return row, err
}

func GetWidgetMarketItemByJsSourceIDPublished(db *gorm.DB, jsSourceID string) (WidgetMarketItem, error) {
	var row WidgetMarketItem
	err := db.Where("js_source_id = ? AND status = ?", strings.TrimSpace(jsSourceID), WidgetMarketStatusPublished).First(&row).Error
	return row, err
}

func GetWidgetMarketItemForTenantTemplate(db *gorm.DB, tenantID, templateID uint) (WidgetMarketItem, error) {
	var row WidgetMarketItem
	err := db.Where("author_tenant_id = ? AND source_js_template_id = ?", tenantID, templateID).First(&row).Error
	return row, err
}

func IncrementWidgetMarketInstallCount(db *gorm.DB, id uint) error {
	return db.Model(&WidgetMarketItem{}).Where("id = ?", id).
		UpdateColumn("install_count", gorm.Expr("install_count + 1")).Error
}

func CreateWidgetMarketItem(db *gorm.DB, row *WidgetMarketItem) error {
	if err := ValidateWidgetMarketItem(row); err != nil {
		return err
	}
	return db.Create(row).Error
}

func UpdateWidgetMarketItem(db *gorm.DB, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	if slug, ok := updates["slug"].(string); ok {
		updates["slug"] = NormalizeWidgetMarketSlug(slug)
	}
	if status, ok := updates["status"].(string); ok {
		updates["status"] = NormalizeWidgetMarketStatus(status)
	}
	return db.Model(&WidgetMarketItem{}).Where("id = ?", id).Updates(updates).Error
}
