package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// ErrInvalidOrgReference indicates an id list did not resolve to valid catalog rows.
var ErrInvalidOrgReference = errors.New("invalid organization reference")

// Permission is a global capability code (shared RBAC catalog across tenants).
type Permission struct {
	common.BaseModel

	Code        string `json:"code" gorm:"size:128;uniqueIndex;not null;comment:权限编码"`
	Name        string `json:"name" gorm:"size:256;not null;comment:权限名称"`
	Description string `json:"description,omitempty" gorm:"size:512;comment:描述"`
	Kind        string `json:"kind" gorm:"size:32;index;not null;default:menu;comment:类型(module/menu/button/api/data)"`
	ParentCode  string `json:"parentCode,omitempty" gorm:"size:128;index;comment:父级编码"`
	Resource    string `json:"resource,omitempty" gorm:"size:128;index;comment:资源标识"`
	Action      string `json:"action,omitempty" gorm:"size:64;index;comment:动作标识"`
}

func (Permission) TableName() string {
	return constants2.PERMISSION_TABLE_NAME
}

// TenantRolePermission assigns permissions to a tenant role.
type TenantRolePermission struct {
	common.BaseModel

	RoleID       uint `json:"roleId" gorm:"index;not null;uniqueIndex:idx_role_permission;comment:角色ID"`
	PermissionID uint `json:"permissionId" gorm:"index;not null;uniqueIndex:idx_role_permission;comment:权限ID"`
}

func (TenantRolePermission) TableName() string {
	return constants2.TENANT_ROLE_PERMISSION_TABLE_NAME
}

// ListAllPermissions returns the global permission catalog (active rows).
func ListAllPermissions(db *gorm.DB) ([]Permission, error) {
	var rows []Permission
	err := db.
		Order(`CASE kind 
			WHEN '` + constants.PermissionKindModule + `' THEN 0 
			WHEN '` + constants.PermissionKindMenu + `' THEN 1 
			WHEN '` + constants.PermissionKindButton + `' THEN 2 
			WHEN '` + constants.PermissionKindAPI + `' THEN 3 
			WHEN '` + constants.PermissionKindData + `' THEN 4 
			ELSE 5 END, parent_code ASC, code ASC`).
		Find(&rows).Error
	return rows, err
}

// ListPermissionIDsForRole returns permission ids bound to a role.
func ListPermissionIDsForRole(db *gorm.DB, roleID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&TenantRolePermission{}).
		Where("role_id = ?", roleID).
		Pluck("permission_id", &ids).Error
	return ids, err
}

// ReplaceTenantRolePermissions replaces all permission bindings for a role (hard reset pivot rows).
func ReplaceTenantRolePermissions(db *gorm.DB, roleID uint, permissionIDs []uint, operator string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		permissionIDs = utils.DedupeUint(permissionIDs)
		if len(permissionIDs) > 0 {
			var n int64
			if err := tx.Model(&Permission{}).
				Where("id IN ?", permissionIDs).
				Count(&n).Error; err != nil {
				return err
			}
			if int(n) != len(permissionIDs) {
				return ErrInvalidOrgReference
			}
		}
		if err := tx.Unscoped().Where("role_id = ?", roleID).Delete(&TenantRolePermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			rp := &TenantRolePermission{RoleID: roleID, PermissionID: pid}
			rp.SetCreateInfo(operator)
			if err := tx.Create(rp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AttachAllPermissionsToRole binds every active catalog permission to the role (tenant admin bootstrap).
func AttachAllPermissionsToRole(tx *gorm.DB, roleID uint, operator string) error {
	var ids []uint
	if err := tx.Model(&Permission{}).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, pid := range ids {
		rp := &TenantRolePermission{RoleID: roleID, PermissionID: pid}
		rp.SetCreateInfo(operator)
		if err := tx.Create(rp).Error; err != nil {
			return err
		}
	}
	return nil
}

// builtinPermissions defines the default RBAC permission catalog seeded on startup.
var builtinPermissions = []struct {
	Code       string
	Name       string
	Kind       string
	ParentCode string
}{
	{constants.PermAPIAssistantsRead, "AI 智能体查看", constants.PermissionKindAPI, "api.assistants"},
	{constants.PermAPIAssistantsWrite, "AI 智能体管理", constants.PermissionKindAPI, "api.assistants"},
	// Compatibility aliases for pre-migration role bindings / API keys.
	{constants.PermAPIWorkflowRead, "工作流查看", constants.PermissionKindAPI, "api.workflow"},
	{constants.PermAPIWorkflowWrite, "工作流管理", constants.PermissionKindAPI, "api.workflow"},
	{constants.PermAPITenantOrgRead, "组织架构查看", constants.PermissionKindAPI, "api.tenant"},
	{constants.PermAPITenantOrgWrite, "组织架构管理", constants.PermissionKindAPI, "api.tenant"},
	{constants.PermAPITenantUsersRead, "成员查看", constants.PermissionKindAPI, "api.tenant"},
	{constants.PermAPITenantUsersWrite, "成员管理", constants.PermissionKindAPI, "api.tenant"},
	{constants.PermAPICredentialsRead, "访问密钥查看", constants.PermissionKindAPI, "api.credentials"},
	{constants.PermAPICredentialsWrite, "访问密钥管理", constants.PermissionKindAPI, "api.credentials"},
	{constants.PermAPIVoiceRead, "音色克隆查看", constants.PermissionKindAPI, "api.voice"},
	{constants.PermAPIVoiceWrite, "音色克隆管理", constants.PermissionKindAPI, "api.voice"},
	{constants.PermAPIOperationLogsRead, "操作日志查看", constants.PermissionKindAPI, "api.audit"},
	{constants.PermAPIAIInvocationsRead, "AI 调用记录查看", constants.PermissionKindAPI, "api.audit"},
	{constants.PermAPIReportsRead, "AI 报表查看", constants.PermissionKindAPI, "api.reports"},
	{constants.PermAPIKBRead, "知识库查看", constants.PermissionKindAPI, "api.kb"},
	{constants.PermAPIKBWrite, "知识库管理", constants.PermissionKindAPI, "api.kb"},
	{constants.PermAPIBillingRead, "账单查看", constants.PermissionKindAPI, "api.billing"},
	{constants.PermAPIBillingWrite, "账单管理", constants.PermissionKindAPI, "api.billing"},
	{constants.PermAPIWebhooksRead, "Webhook 查看", constants.PermissionKindAPI, "api.webhooks"},
	{constants.PermAPIWebhooksWrite, "Webhook 管理", constants.PermissionKindAPI, "api.webhooks"},
	{constants.PermAPINluRead, "NLU 查看", constants.PermissionKindAPI, "api.nlu"},
	{constants.PermAPINluWrite, "NLU 管理", constants.PermissionKindAPI, "api.nlu"},
	{constants.PermMenuWorkspaceOverview, "工作台菜单", constants.PermissionKindMenu, "menu.workspace"},
	{constants.PermMenuResAssistant, "AI 智能体菜单", constants.PermissionKindMenu, "menu.res"},
	{constants.PermMenuResNlu, "NLU 实验室菜单", constants.PermissionKindMenu, "menu.res"},
	{constants.PermMenuResWorkflow, "工作流菜单", constants.PermissionKindMenu, "menu.res"},
	{constants.PermMenuResVoice, "音色管理菜单", constants.PermissionKindMenu, "menu.res"},
	{constants.PermMenuAccKeys, "访问管理菜单", constants.PermissionKindMenu, "menu.acc"},
	{constants.PermMenuAccBilling, "账单菜单", constants.PermissionKindMenu, "menu.acc"},
	{constants.PermMenuAccUsageMetrics, "用量与指标菜单", constants.PermissionKindMenu, "menu.acc"},
	{constants.PermMenuProfileReports, "AI 报表菜单", constants.PermissionKindMenu, "menu.profile"},
	{constants.PermMenuProfileLogs, "操作日志菜单", constants.PermissionKindMenu, "menu.profile"},
	{constants.PermMenuProfileAIInvoc, "AI 调用记录菜单", constants.PermissionKindMenu, "menu.profile"},
	{constants.PermMenuOrgMembers, "成员管理菜单", constants.PermissionKindMenu, "menu.org"},
	{constants.PermMenuOrgDept, "部门菜单", constants.PermissionKindMenu, "menu.org"},
	{constants.PermMenuOrgRole, "角色与权限菜单", constants.PermissionKindMenu, "menu.org"},
	{constants.PermMenuKBRead, "知识库菜单", constants.PermissionKindMenu, "menu.kb"},
}

// BackfillSystemTenantAdminPermissions ensures every system「管理员」role is bound to the
// full current catalog. Idempotent; only inserts missing pivot rows. Used to heal tenants
// created before new permission codes (e.g. menu.* codes) were introduced.
func BackfillSystemTenantAdminPermissions(db *gorm.DB, operator string) error {
	_, err := BackfillSystemTenantAdminPermissionsStats(db, operator)
	return err
}

// BackfillSystemTenantAdminPermissionsStats is like BackfillSystemTenantAdminPermissions
// but returns the number of tenant_role_permissions rows inserted.
func BackfillSystemTenantAdminPermissionsStats(db *gorm.DB, operator string) (int, error) {
	var permIDs []uint
	if err := db.Model(&Permission{}).Pluck("id", &permIDs).Error; err != nil {
		return 0, err
	}
	if len(permIDs) == 0 {
		return 0, nil
	}
	var adminRoleIDs []uint
	if err := db.Model(&TenantRole{}).
		Where("is_system = ? AND name = ?", true, constants.TenantAdminRoleName).
		Pluck("id", &adminRoleIDs).Error; err != nil {
		return 0, err
	}
	added := 0
	for _, roleID := range adminRoleIDs {
		var bound []uint
		if err := db.Model(&TenantRolePermission{}).
			Where("role_id = ?", roleID).
			Pluck("permission_id", &bound).Error; err != nil {
			return added, err
		}
		have := make(map[uint]struct{}, len(bound))
		for _, id := range bound {
			have[id] = struct{}{}
		}
		for _, pid := range permIDs {
			if _, ok := have[pid]; ok {
				continue
			}
			rp := &TenantRolePermission{RoleID: roleID, PermissionID: pid}
			rp.SetCreateInfo(operator)
			if err := db.Create(rp).Error; err != nil {
				return added, err
			}
			added++
		}
	}
	return added, nil
}

// SyncPermissionCatalog upserts the built-in permission rows; existing rows are updated, missing ones created.
func SyncPermissionCatalog(db *gorm.DB) error {
	for _, p := range builtinPermissions {
		var existing Permission
		err := db.Unscoped().Where("code = ?", p.Code).First(&existing).Error
		if err == nil {
			updates := map[string]any{
				"name":        p.Name,
				"kind":        p.Kind,
				"parent_code": p.ParentCode,
			}
			if existing.ID > 0 {
				db.Where("id = ?", existing.ID).Updates(updates)
			} else {
				db.Where("code = ?", p.Code).Updates(updates)
			}
			continue
		}
		row := Permission{
			Code:       p.Code,
			Name:       p.Name,
			Kind:       p.Kind,
			ParentCode: p.ParentCode,
		}
		row.SetCreateInfo("system")
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
