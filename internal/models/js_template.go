package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	JSTemplateStatusActive = "active"
	JSTemplateStatusDraft  = "draft"
)

// JSTemplate is a tenant-editable embed JS snippet served by jsSourceId.
type JSTemplate struct {
	common.BaseModel
	TenantID   uint   `json:"tenantId,string" gorm:"index;not null"`
	JsSourceID string `json:"jsSourceId" gorm:"column:js_source_id;size:64;uniqueIndex;not null"`
	Name       string `json:"name" gorm:"size:128;not null;index"`
	AvatarURL  string `json:"avatarUrl,omitempty" gorm:"column:avatar_url;size:512"`
	Content    string `json:"content" gorm:"type:longtext;not null"`
	Usage      string `json:"usage,omitempty" gorm:"type:text"`
	Status     string `json:"status" gorm:"size:24;index;not null;default:active"` // active | draft
}

func (JSTemplate) TableName() string {
	return constants2.JS_TEMPLATE_TABLE_NAME
}

// BeforeCreate assigns snowflake ID and js_source_id when empty.
func (t *JSTemplate) BeforeCreate(tx *gorm.DB) error {
	if t == nil {
		return nil
	}
	if err := t.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}
	if strings.TrimSpace(t.Status) == "" {
		t.Status = JSTemplateStatusActive
	}
	if strings.TrimSpace(t.JsSourceID) != "" {
		return nil
	}
	id, err := generateUniqueJSSourceID(tx)
	if err != nil {
		return err
	}
	t.JsSourceID = id
	return nil
}

func generateUniqueJSSourceID(db *gorm.DB) (string, error) {
	for i := 0; i < 12; i++ {
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		id := fmt.Sprintf("js_%s", hex.EncodeToString(buf))
		var n int64
		if err := db.Model(&JSTemplate{}).Where("js_source_id = ?", id).Count(&n).Error; err != nil {
			return "", err
		}
		if n == 0 {
			return id, nil
		}
		time.Sleep(time.Microsecond)
	}
	return "", fmt.Errorf("failed to allocate js_source_id")
}

// ListJSTemplatesPage lists templates for a tenant.
func ListJSTemplatesPage(db *gorm.DB, tenantID uint, page, size int, nameContains string) ([]JSTemplate, int64, error) {
	q := db.Model(&JSTemplate{}).Where("tenant_id = ?", tenantID)
	if name := strings.TrimSpace(nameContains); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	return utils.FindPage[JSTemplate](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

// GetJSTemplateByIDForTenant loads one template scoped to tenant.
func GetJSTemplateByIDForTenant(db *gorm.DB, id, tenantID uint) (JSTemplate, error) {
	var row JSTemplate
	err := db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	return row, err
}

// GetActiveJSTemplateByJsSourceID loads an active template by public source id.
func GetActiveJSTemplateByJsSourceID(db *gorm.DB, jsSourceID string) (JSTemplate, error) {
	var row JSTemplate
	err := db.Where("js_source_id = ? AND status = ?", strings.TrimSpace(jsSourceID), JSTemplateStatusActive).First(&row).Error
	return row, err
}

// SoftDeleteJSTemplateByIDForTenant soft-deletes a template.
func SoftDeleteJSTemplateByIDForTenant(db *gorm.DB, id, tenantID uint, updateBy string) (int64, error) {
	meta := common.BaseModel{}
	meta.SoftDelete(updateBy)
	updates := map[string]any{
		"updated_at": meta.UpdatedAt,
		"deleted_at": meta.DeletedAt,
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	res := db.Model(&JSTemplate{}).Where("id = ? AND tenant_id = ?", id, tenantID).Updates(updates)
	return res.RowsAffected, res.Error
}
