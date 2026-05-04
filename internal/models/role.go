package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Role 角色（如 superadmin / admin / user）。
type Role struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Slug        string `json:"slug" gorm:"size:64;uniqueIndex;not null"`
	Description string `json:"description" gorm:"size:512"`
	IsSystem    bool   `json:"isSystem" gorm:"default:false"`
	// 关联：角色拥有的权限（role_permissions）
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;joinForeignKey:RoleID;joinReferences:PermissionID"`
}

func (Role) TableName() string { return "roles" }

// RolePermission 角色与权限的多对多中间表。
type RolePermission struct {
	RoleID       uint `json:"roleId" gorm:"primaryKey"`
	PermissionID uint `json:"permissionId" gorm:"primaryKey"`
}

func (RolePermission) TableName() string { return "role_permissions" }

// UserRole 用户与角色的多对多中间表。
type UserRole struct {
	UserID uint `json:"userId" gorm:"primaryKey"`
	RoleID uint `json:"roleId" gorm:"primaryKey"`
}

func (UserRole) TableName() string { return "user_roles" }
