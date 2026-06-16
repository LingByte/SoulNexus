package auth

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
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
	`, userID, models.SoftDeleteStatusActive, userID, models.SoftDeleteStatusActive).Scan(&keys).Error
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
		ORDER BY r.slug ASC
	`, userID, models.SoftDeleteStatusActive).Scan(&slugs).Error
	return slugs, err
}

// HasPermissionKey reports whether the user carries a permission from JWT (no DB).
func (u *User) HasPermissionKey(key string) bool {
	if u == nil {
		return false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, k := range u.PermissionKeys {
		if k == PermWildcard || k == key {
			return true
		}
	}
	return false
}

// HasAdminAccess reports admin console access from JWT permission claims (no DB).
func (u *User) HasAdminAccess() bool {
	return u != nil && (u.HasPermissionKey(PermAdminAccess) || u.HasPermissionKey(PermWildcard))
}

// UserHasPermission 判断是否拥有某权限（含通配符 *）。仅 auth 服务应查库；业务服务请用 User.HasPermissionKey。
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

// AssignUserSingleRoleBySlug replaces all user_roles rows with a single role matching slug.
func AssignUserSingleRoleBySlug(db *gorm.DB, userID uint, slug string) error {
	if db == nil || userID == 0 {
		return errors.New("invalid user")
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return errors.New("role slug is required")
	}
	var role Role
	if err := db.Where("slug = ? AND is_deleted = ?", slug, models.SoftDeleteStatusActive).First(&role).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		return tx.Create(&UserRole{UserID: userID, RoleID: role.ID}).Error
	})
}

// EnsureUserHasOneRole assigns the lowest-id active role when the user has no role rows yet (e.g. after signup).
func EnsureUserHasOneRole(db *gorm.DB, userID uint) error {
	if db == nil || userID == 0 {
		return errors.New("invalid user")
	}
	var n int64
	if err := db.Model(&UserRole{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var role Role
	if err := db.Where("is_deleted = ?", models.SoftDeleteStatusActive).Order("id ASC").First(&role).Error; err != nil {
		return fmt.Errorf("no roles defined; create at least one role before registering users: %w", err)
	}
	return db.Create(&UserRole{UserID: userID, RoleID: role.ID}).Error
}

// PrimaryJWTClaimRole picks a stable single slug for JWT claims (lexicographically smallest non-empty slug).
func PrimaryJWTClaimRole(slugs []string) string {
	var cleaned []string
	for _, s := range slugs {
		u := strings.TrimSpace(strings.ToLower(s))
		if u != "" {
			cleaned = append(cleaned, u)
		}
	}
	if len(cleaned) == 0 {
		return ""
	}
	sort.Strings(cleaned)
	return cleaned[0]
}

// RoleSlugsFromJWTClaim maps a JWT role claim to RoleSlugs for context (best-effort).
func RoleSlugsFromJWTClaim(roleClaim string) []string {
	r := strings.TrimSpace(strings.ToLower(roleClaim))
	if r == "" {
		return nil
	}
	return []string{r}
}
