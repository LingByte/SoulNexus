package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// EffectivePermissionKeys 返回用户最终权限 key（角色 ∪ 直接附加）。
func EffectivePermissionKeys(db *gorm.DB, userID uint) ([]string, error) {
	if db == nil || userID == 0 {
		return nil, nil
	}
	var keys []string
	err := db.Raw(`
		SELECT DISTINCT p.key FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		INNER JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ? AND p.is_deleted = ?
		UNION
		SELECT DISTINCT p.key FROM permissions p
		INNER JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ? AND p.is_deleted = ?
	`, userID, SoftDeleteStatusActive, userID, SoftDeleteStatusActive).Scan(&keys).Error
	return keys, err
}

// UserRoleSlugs 返回用户已绑定的角色 slug。
func UserRoleSlugs(db *gorm.DB, userID uint) ([]string, error) {
	if db == nil || userID == 0 {
		return nil, nil
	}
	var slugs []string
	err := db.Raw(`
		SELECT DISTINCT r.slug FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.is_deleted = ?
	`, userID, SoftDeleteStatusActive).Scan(&slugs).Error
	return slugs, err
}

// UserHasPermission 判断是否拥有某权限（含通配符 *）。
func UserHasPermission(db *gorm.DB, userID uint, key string) bool {
	if db == nil || userID == 0 || strings.TrimSpace(key) == "" {
		return false
	}
	keys, err := EffectivePermissionKeys(db, userID)
	if err != nil {
		return false
	}
	for _, k := range keys {
		if k == PermWildcard || k == key {
			return true
		}
	}
	return false
}

// UserHasAdminAccess 是否可进入管理后台（legacy 角色或 admin.access / *）。
func UserHasAdminAccess(db *gorm.DB, userID uint) bool {
	return UserHasPermission(db, userID, PermAdminAccess) || UserHasPermission(db, userID, PermWildcard)
}

// SyncUserRolesFromLegacyRole 根据 users.role 字符串重写 user_roles。
func SyncUserRolesFromLegacyRole(db *gorm.DB, user *User) error {
	if db == nil || user == nil || user.ID == 0 {
		return errors.New("invalid user")
	}
	slug := strings.TrimSpace(strings.ToLower(user.Role))
	if slug == "" {
		slug = RoleUser
	}
	var role Role
	if err := db.Where("slug = ? AND is_deleted = ?", slug, SoftDeleteStatusActive).First(&role).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", user.ID).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		return tx.Create(&UserRole{UserID: user.ID, RoleID: role.ID}).Error
	})
}

// SyncAllUserRolesFromLegacy 将所有用户的 user_roles 与 users.role 对齐。
func SyncAllUserRolesFromLegacy(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var users []User
	if err := db.Select("id", "role").Where("is_deleted = ?", SoftDeleteStatusActive).Find(&users).Error; err != nil {
		return err
	}
	for i := range users {
		if err := SyncUserRolesFromLegacyRole(db, &users[i]); err != nil {
			continue
		}
	}
	return nil
}
