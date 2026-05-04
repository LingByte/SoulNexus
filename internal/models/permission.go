package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// 内置权限 key（数据库中存这些字符串）
const (
	PermWildcard    = "*"
	PermAdminAccess = "admin.access"
	PermManageRoles = "rbac.manage" // 管理角色、权限与用户授权
)

// Permission 表示一条可授权的权限点。
type Permission struct {
	BaseModel
	Key         string `json:"key" gorm:"size:128;uniqueIndex;not null"`
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	Resource    string `json:"resource" gorm:"size:64;index"`
	// 关联（仅查询时使用，非持久化列）
	Roles []Role `json:"roles,omitempty" gorm:"many2many:role_permissions;joinForeignKey:PermissionID;joinReferences:RoleID"`
}

func (Permission) TableName() string { return "permissions" }

// UserPermission 用户直接附加的权限（user_permissions / user_perms）。
type UserPermission struct {
	UserID       uint `json:"userId" gorm:"primaryKey"`
	PermissionID uint `json:"permissionId" gorm:"primaryKey"`
}

func (UserPermission) TableName() string { return "user_permissions" }
