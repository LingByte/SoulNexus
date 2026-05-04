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

// EnsureAccessDefaults 写入内置权限、角色、关联并同步 user_roles（可重复执行）。
func EnsureAccessDefaults(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	perms := []Permission{
		{Key: PermWildcard, Name: "全部权限", Description: "超级管理员通配", Resource: "system"},
		{Key: PermAdminAccess, Name: "管理后台", Description: "登录并使用管理后台", Resource: "admin"},
		{Key: PermManageRoles, Name: "角色与权限管理", Description: "管理角色、权限与用户授权", Resource: "access"},
	}
	keyToID := map[string]uint{}
	for _, p := range perms {
		var existing Permission
		err := db.Where("`key` = ? AND is_deleted = ?", p.Key, SoftDeleteStatusActive).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&p).Error; err != nil {
				return err
			}
			keyToID[p.Key] = p.ID
		} else if err != nil {
			return err
		} else {
			keyToID[p.Key] = existing.ID
		}
	}

	roles := []Role{
		{Name: "超级管理员", Slug: RoleSuperAdmin, Description: "系统超级管理员", IsSystem: true},
		{Name: "管理员", Slug: RoleAdmin, Description: "后台管理员", IsSystem: true},
		{Name: "普通用户", Slug: RoleUser, Description: "默认用户", IsSystem: true},
	}
	slugToID := map[string]uint{}
	for _, r := range roles {
		var existing Role
		err := db.Where("slug = ? AND is_deleted = ?", r.Slug, SoftDeleteStatusActive).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&r).Error; err != nil {
				return err
			}
			slugToID[r.Slug] = r.ID
		} else if err != nil {
			return err
		} else {
			slugToID[r.Slug] = existing.ID
		}
	}

	link := func(roleSlug, permKey string) error {
		rid, okR := slugToID[roleSlug]
		pid, okP := keyToID[permKey]
		if !okR || !okP {
			return nil
		}
		var n int64
		_ = db.Model(&RolePermission{}).Where("role_id = ? AND permission_id = ?", rid, pid).Count(&n).Error
		if n > 0 {
			return nil
		}
		return db.Create(&RolePermission{RoleID: rid, PermissionID: pid}).Error
	}

	_ = link(RoleSuperAdmin, PermWildcard)
	_ = link(RoleAdmin, PermAdminAccess)
	_ = link(RoleAdmin, PermManageRoles)

	return SyncAllUserRolesFromLegacy(db)
}

// EnsureRBACDefaults 兼容旧名称，请改用 EnsureAccessDefaults。
func EnsureRBACDefaults(db *gorm.DB) error {
	return EnsureAccessDefaults(db)
}
