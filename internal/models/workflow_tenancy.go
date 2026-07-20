package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"

	"gorm.io/gorm"
)

// Workflow tenancy helpers — SoulNexus used GroupID; SoulNexus maps GroupID to TenantID
// via tenant_users (user id = TenantUser.ID, tenant = TenantUser.TenantID).

// MemberGroupIDs returns tenant IDs the user belongs to (compat name for SoulNexus).
func MemberGroupIDs(db *gorm.DB, userID uint) ([]uint, error) {
	if db == nil || userID == 0 {
		return nil, nil
	}
	var tu TenantUser
	if err := db.Select("tenant_id").Where("id = ?", userID).First(&tu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if tu.TenantID == 0 {
		return nil, nil
	}
	return []uint{tu.TenantID}, nil
}

// UserIsGroupMember reports whether the user belongs to the tenant (groupID == tenantID).
func UserIsGroupMember(db *gorm.DB, userID, groupID uint) bool {
	if db == nil || userID == 0 || groupID == 0 {
		return false
	}
	var n int64
	_ = db.Model(&TenantUser{}).Where("id = ? AND tenant_id = ?", userID, groupID).Limit(1).Count(&n).Error
	return n > 0
}

// ResolveWriteGroupID picks the tenant id for creating workflow resources.
// If requestedGroupID is set it must match the user's tenant; otherwise uses the user's tenant.
func ResolveWriteGroupID(db *gorm.DB, userID uint, requestedGroupID *uint) (uint, error) {
	if db == nil || userID == 0 {
		return 0, errors.New("user required")
	}
	var tu TenantUser
	if err := db.Select("id, tenant_id").Where("id = ?", userID).First(&tu).Error; err != nil {
		return 0, err
	}
	if tu.TenantID == 0 {
		return 0, errors.New("user has no tenant")
	}
	if requestedGroupID != nil {
		gid := *requestedGroupID
		if gid != 0 && gid != tu.TenantID {
			return 0, errors.New("not a member of the target organization")
		}
		if gid == 0 {
			return tu.TenantID, nil
		}
		return gid, nil
	}
	return tu.TenantID, nil
}

// CanManageTenantResource reports whether the user may manage a resource in the tenant.
func CanManageTenantResource(db *gorm.DB, userID, groupID, createdBy uint) bool {
	if !UserIsGroupMember(db, userID, groupID) {
		return false
	}
	if createdBy == 0 || createdBy == userID {
		return true
	}
	// Same-tenant members can manage (SoulNexus tenant RBAC is enforced at route layer).
	return true
}
