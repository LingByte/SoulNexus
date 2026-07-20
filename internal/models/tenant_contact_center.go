package models

import (
	"encoding/json"

	"github.com/LingByte/SoulNexus/internal/constants"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TenantAutomationConfig holds tenant-scoped automation toggles.
// Persisted in tenants.contact_center_config (legacy column name).
type TenantAutomationConfig struct{}

// ParseTenantAutomationConfig decodes the tenant JSON column (empty → zero value).
func ParseTenantAutomationConfig(raw datatypes.JSON) TenantAutomationConfig {
	var cfg TenantAutomationConfig
	if len(raw) == 0 {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	return cfg
}

// PatchTenantAutomationConfig replaces the tenant automation JSON blob.
func PatchTenantAutomationConfig(db *gorm.DB, tenantID uint, cfg TenantAutomationConfig, operator string) error {
	if db == nil || tenantID == 0 {
		return gorm.ErrInvalidDB
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"contact_center_config": datatypes.JSON(b),
	}
	if operator != "" {
		updates["update_by"] = operator
	}
	return db.Model(&Tenant{}).Where("id = ?", tenantID).Updates(updates).Error
}

// ListTenantAdminUserIDs returns tenant user IDs bound to the system「管理员」role.
func ListTenantAdminUserIDs(db *gorm.DB, tenantID uint) ([]uint, error) {
	if db == nil || tenantID == 0 {
		return nil, nil
	}
	var roleIDs []uint
	if err := db.Model(&TenantRole{}).
		Where("tenant_id = ? AND is_system = ? AND name = ?", tenantID, true, constants.TenantAdminRoleName).
		Pluck("id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var ids []uint
	err := db.Model(&TenantUserRole{}).
		Where("role_id IN ?", roleIDs).
		Distinct().
		Pluck("tenant_user_id", &ids).Error
	return ids, err
}
