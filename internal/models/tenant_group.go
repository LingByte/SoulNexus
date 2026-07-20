package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// TenantGroup is a team / department within a tenant. Mark IsDefault for the bucket new users join by default.
type TenantGroup struct {
	common.BaseModel

	TenantID  uint   `json:"tenantId" gorm:"index;not null;comment:所属租户ID"`
	Name      string `json:"name" gorm:"size:128;index;not null;comment:部门名称"`
	IsDefault bool   `json:"isDefault" gorm:"index;not null;default:0;comment:是否默认部门"`
}

func (TenantGroup) TableName() string {
	return constants.TENANT_GROUP_TABLE_NAME
}

// TenantUserGroup links users to groups (many-to-many).
type TenantUserGroup struct {
	common.BaseModel

	TenantUserID uint `json:"tenantUserId" gorm:"index;not null;uniqueIndex:idx_user_group;comment:租户用户ID"`
	GroupID      uint `json:"groupId" gorm:"index;not null;uniqueIndex:idx_user_group;comment:部门ID"`
}

func (TenantUserGroup) TableName() string {
	return constants.TENANT_USER_GROUP_TABLE_NAME
}

// ListTenantGroupsForTenant lists departments for a tenant.
func ListTenantGroupsForTenant(db *gorm.DB, tenantID uint) ([]TenantGroup, error) {
	var rows []TenantGroup
	err := db.Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Find(&rows).Error
	return rows, err
}

// ListTenantGroupsForUser lists departments linked to a user (active memberships).
func ListTenantGroupsForUser(db *gorm.DB, tenantUserID uint) ([]TenantGroup, error) {
	tg := constants.TENANT_GROUP_TABLE_NAME
	tugTbl := constants.TENANT_USER_GROUP_TABLE_NAME
	var rows []TenantGroup
	err := db.Model(&TenantGroup{}).
		Joins("INNER JOIN "+tugTbl+" AS tug ON tug.group_id = "+tg+".id AND tug.deleted_at IS NULL").
		Where("tug.tenant_user_id = ? AND "+tg+".deleted_at IS NULL", tenantUserID).
		Order(tg + ".name ASC").
		Find(&rows).Error
	return rows, err
}

// CreateTenantGroupRecord persists a new tenant group.
func CreateTenantGroupRecord(db *gorm.DB, g *TenantGroup) error {
	return db.Create(g).Error
}

// ReplaceTenantUserGroups replaces group memberships for a tenant user.
func ReplaceTenantUserGroups(db *gorm.DB, tenantID, tenantUserID uint, groupIDs []uint, operator string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		groupIDs = utils.DedupeUint(groupIDs)
		if len(groupIDs) > 0 {
			var n int64
			if err := tx.Model(&TenantGroup{}).
				Where("tenant_id = ? AND id IN ?", tenantID, groupIDs).
				Count(&n).Error; err != nil {
				return err
			}
			if int(n) != len(groupIDs) {
				return ErrInvalidOrgReference
			}
		}
		if err := tx.Unscoped().Where("tenant_user_id = ?", tenantUserID).Delete(&TenantUserGroup{}).Error; err != nil {
			return err
		}
		for _, gid := range groupIDs {
			tug := &TenantUserGroup{TenantUserID: tenantUserID, GroupID: gid}
			tug.SetCreateInfo(operator)
			if err := tx.Create(tug).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SoftDeleteTenantGroup soft-deletes a group and marks its memberships deleted.
func SoftDeleteTenantGroup(db *gorm.DB, tenantID, groupID uint, updateBy string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var g TenantGroup
		if err := tx.Where("id = ? AND tenant_id = ?", groupID, tenantID).
			First(&g).Error; err != nil {
			return err
		}
		meta := common.BaseModel{}
		meta.SoftDelete(updateBy)
		if err := tx.Model(&TenantUserGroup{}).
			Where("group_id = ?", groupID).
			Updates(map[string]any{
				"updated_at": meta.UpdatedAt,
				"update_by":  meta.UpdateBy,
				"deleted_at": meta.DeletedAt,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&TenantGroup{}).Where("id = ?", groupID).Updates(map[string]any{
			"updated_at": meta.UpdatedAt,
			"update_by":  meta.UpdateBy,
			"deleted_at": meta.DeletedAt,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}
