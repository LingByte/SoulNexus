package models

import (
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// TenantRole is a named role within one tenant.
type TenantRole struct {
	common.BaseModel

	TenantID    uint   `json:"tenantId" gorm:"index;not null;comment:所属租户ID"`
	Name        string `json:"name" gorm:"size:128;index;not null;comment:角色名称"`
	Description string `json:"description,omitempty" gorm:"size:512;comment:描述"`
	IsSystem    bool   `json:"isSystem" gorm:"not null;default:0;comment:是否系统角色"`
}

func (TenantRole) TableName() string {
	return constants.TENANT_ROLE_TABLE_NAME
}

// TenantUserRole assigns roles to a tenant user.
type TenantUserRole struct {
	common.BaseModel

	TenantUserID uint `json:"tenantUserId" gorm:"index;not null;uniqueIndex:idx_tenant_user_role;comment:租户用户ID"`
	RoleID       uint `json:"roleId" gorm:"index;not null;uniqueIndex:idx_tenant_user_role;comment:角色ID"`
}

func (TenantUserRole) TableName() string {
	return constants.TENANT_USER_ROLE_TABLE_NAME
}

// CreateTenantRole inserts a tenant-scoped role.
func CreateTenantRole(db *gorm.DB, r *TenantRole) error {
	return db.Create(r).Error
}

// CreateTenantUserRole binds a tenant user to a role.
func CreateTenantUserRole(db *gorm.DB, tur *TenantUserRole) error {
	return db.Create(tur).Error
}

// TenantUserHasRoleName reports whether the user has an active role with the given name.
func TenantUserHasRoleName(db *gorm.DB, tenantUserID uint, roleName string) (bool, error) {
	var roleIDs []uint
	if err := db.Model(&TenantUserRole{}).Where("tenant_user_id = ?", tenantUserID).Pluck("role_id", &roleIDs).Error; err != nil {
		return false, err
	}
	if len(roleIDs) == 0 {
		return false, nil
	}
	var n int64
	err := db.Model(&TenantRole{}).
		Where("id IN ? AND name = ?", roleIDs, roleName).
		Count(&n).Error
	return n > 0, err
}

// ListTenantRolesByTenant lists named roles for one tenant.
func ListTenantRolesByTenant(db *gorm.DB, tenantID uint) ([]TenantRole, error) {
	var rows []TenantRole
	err := db.Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Find(&rows).Error
	return rows, err
}

// GetTenantRoleScoped returns a role owned by the tenant.
func GetTenantRoleScoped(db *gorm.DB, tenantID, roleID uint) (TenantRole, error) {
	var r TenantRole
	err := db.Where("id = ? AND tenant_id = ?", roleID, tenantID).
		First(&r).Error
	return r, err
}

// ListTenantRolesForUser returns roles assigned to a tenant user.
func ListTenantRolesForUser(db *gorm.DB, tenantUserID uint) ([]TenantRole, error) {
	var roleIDs []uint
	if err := db.Model(&TenantUserRole{}).
		Where("tenant_user_id = ?", tenantUserID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var rows []TenantRole
	err := db.Where("id IN ?", roleIDs).
		Order("name ASC").
		Find(&rows).Error
	return rows, err
}

// ListEffectivePermissionCodesForTenantUser returns distinct permission codes granted via roles.
func ListEffectivePermissionCodesForTenantUser(db *gorm.DB, tenantUserID uint) ([]string, error) {
	perm := constants.PERMISSION_TABLE_NAME
	trp := constants.TENANT_ROLE_PERMISSION_TABLE_NAME
	tr := constants.TENANT_ROLE_TABLE_NAME
	tur := constants.TENANT_USER_ROLE_TABLE_NAME
	var codes []string
	err := db.Table(perm).
		Select("DISTINCT "+perm+".code").
		Joins("INNER JOIN "+trp+" ON "+trp+".permission_id = "+perm+".id AND "+trp+".deleted_at IS NULL").
		Joins("INNER JOIN "+tr+" ON "+tr+".id = "+trp+".role_id AND "+tr+".deleted_at IS NULL").
		Joins("INNER JOIN "+tur+" ON "+tur+".role_id = "+tr+".id AND "+tur+".deleted_at IS NULL").
		Where(tur+".tenant_user_id = ? AND "+perm+".deleted_at IS NULL", tenantUserID).
		Order(perm+".code ASC").
		Pluck(perm+".code", &codes).Error
	return codes, err
}

// ListTenantUserIDsByRoleID returns distinct tenant user ids bound to a role.
func ListTenantUserIDsByRoleID(db *gorm.DB, roleID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&TenantUserRole{}).Where("role_id = ?", roleID).Distinct().Pluck("tenant_user_id", &ids).Error
	return ids, err
}

// ReplaceTenantUserRoles replaces role assignments for a tenant user.
func ReplaceTenantUserRoles(db *gorm.DB, tenantID, tenantUserID uint, roleIDs []uint, operator string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		roleIDs = utils.DedupeUint(roleIDs)
		if len(roleIDs) > 0 {
			var n int64
			if err := tx.Model(&TenantRole{}).
				Where("tenant_id = ? AND id IN ?", tenantID, roleIDs).
				Count(&n).Error; err != nil {
				return err
			}
			if int(n) != len(roleIDs) {
				return ErrInvalidOrgReference
			}
		}
		if err := tx.Unscoped().Where("tenant_user_id = ?", tenantUserID).Delete(&TenantUserRole{}).Error; err != nil {
			return err
		}
		for _, rid := range roleIDs {
			tur := &TenantUserRole{TenantUserID: tenantUserID, RoleID: rid}
			tur.SetCreateInfo(operator)
			if err := tx.Create(tur).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SoftDeleteTenantRole soft-deletes a custom tenant role (system roles are protected in the handler).
func SoftDeleteTenantRole(db *gorm.DB, tenantID, roleID uint, updateBy string) error {
	meta := common.BaseModel{}
	meta.SoftDelete(updateBy)
	updates := map[string]any{
		"updated_at": meta.UpdatedAt,
		"deleted_at": meta.DeletedAt,
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	if err := db.Model(&TenantRole{}).
		Where("id = ? AND tenant_id = ? AND is_system = ?", roleID, tenantID, false).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}
