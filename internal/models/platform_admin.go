package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// PlatformAdmin is a global operator (not under any tenant).
type PlatformAdmin struct {
	common.BaseModel

	Email                    string     `json:"email" gorm:"size:255;uniqueIndex;not null;comment:登录邮箱"`
	PasswordHash             string     `json:"-" gorm:"size:255;not null;column:password_hash;comment:密码哈希"`
	DisplayName              string     `json:"displayName" gorm:"size:128;comment:显示名"`
	Status                   string     `json:"status" gorm:"size:24;index;not null;default:active;comment:账号状态"`
	GitHubID                 string     `json:"-" gorm:"size:64;index;column:github_id;comment:GitHub用户ID"`
	GitHubLogin              string     `json:"githubLogin,omitempty" gorm:"size:128;column:github_login;comment:GitHub用户名"`
	TOTPSecret               string     `json:"-" gorm:"size:128;column:totp_secret;comment:TOTP密钥"`
	TOTPEnabled              bool       `json:"totpEnabled" gorm:"column:totp_enabled;not null;default:0;comment:是否启用TOTP"`
	TOTPRecoveryHashes       string     `json:"-" gorm:"column:totp_recovery_hashes;type:text;comment:TOTP恢复码哈希JSON"`
	ReceiveEmailNotify       bool       `json:"receiveEmailNotify" gorm:"column:receive_email_notify;not null;default:0;comment:接收营销/通知类邮件"`
	RequireDeviceVerify      bool       `json:"requireDeviceVerify" gorm:"column:require_device_verify;not null;default:1;comment:新设备需邮箱验证"`
	TrustDeviceLoginEnabled  bool       `json:"trustDeviceLoginEnabled" gorm:"column:trust_device_login_enabled;not null;default:1;comment:7天免登录验证"`
	RequireRemoteLoginVerify bool       `json:"requireRemoteLoginVerify" gorm:"column:require_remote_login_verify;not null;default:1;comment:异地登录保护"`
	PrimaryLoginCity         string     `json:"primaryLoginCity,omitempty" gorm:"column:primary_login_city;size:64;comment:常用登录城市"`
	KnownLoginCitiesJSON     string     `json:"-" gorm:"column:known_login_cities;type:text;comment:已知登录城市JSON"`
	SessionIdleTimeoutHours  int        `json:"sessionIdleTimeoutHours" gorm:"column:session_idle_timeout_hours;not null;default:12;comment:无操作退出小时数"`
	SessionMaxLifetimeHours  int        `json:"sessionMaxLifetimeHours" gorm:"column:session_max_lifetime_hours;not null;default:48;comment:最大登录保持小时数"`
	DeletionRequestedAt      *time.Time `json:"deletionRequestedAt,omitempty" gorm:"column:deletion_requested_at;comment:注销申请时间"`
}

// TableName returns the database table name for platform admin records.
func (PlatformAdmin) TableName() string {
	return constants2.PLATFORM_ADMIN_TABLE_NAME
}

// GetActivePlatformAdminByEmail fetches a platform admin with active status by email.
// Returns gorm.ErrRecordNotFound when no matching active admin exists.
func GetActivePlatformAdminByEmail(db *gorm.DB, email string) (PlatformAdmin, error) {
	var row PlatformAdmin
	err := ActivePlatformAdmins(db).Where("email = ?", email).First(&row).Error
	return row, err
}

// GetActivePlatformAdminByGitHubID fetches a platform admin with active status by GitHub user ID.
// Returns gorm.ErrRecordNotFound when githubID is empty or no matching active admin exists.
func GetActivePlatformAdminByGitHubID(db *gorm.DB, githubID string) (PlatformAdmin, error) {
	if githubID == "" {
		return PlatformAdmin{}, gorm.ErrRecordNotFound
	}
	var row PlatformAdmin
	err := ActivePlatformAdmins(db).Where("github_id = ?", githubID).First(&row).Error
	return row, err
}

// ActivePlatformAdmins returns a scoped query that filters only active platform admins.
func ActivePlatformAdmins(db *gorm.DB) *gorm.DB {
	return db.Model(&PlatformAdmin{}).Where("status = ?", constants.PlatformAdminStatusActive)
}

// CountPlatformAdmins returns the total number of active platform admin accounts.
func CountPlatformAdmins(db *gorm.DB) (int64, error) {
	var n int64
	err := ActivePlatformAdmins(db).Count(&n).Error
	return n, err
}

// GetPlatformAdminByID fetches a platform admin record by primary key (any status).
// Returns gorm.ErrRecordNotFound when the given id does not exist.
func GetPlatformAdminByID(db *gorm.DB, id uint) (PlatformAdmin, error) {
	var row PlatformAdmin
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

// ListActivePlatformAdmins returns all active platform admin accounts.
func ListActivePlatformAdmins(db *gorm.DB) ([]PlatformAdmin, error) {
	var rows []PlatformAdmin
	err := ActivePlatformAdmins(db).Find(&rows).Error
	return rows, err
}

// ListPlatformAdminsPage returns a paginated list of platform admins with optional search.
// When search is non-empty, results are filtered by email and display_name (LIKE match).
// The maximum page size is capped at MaxPageSize100.
func ListPlatformAdminsPage(db *gorm.DB, page, size int, search string) ([]PlatformAdmin, int64, error) {
	if db == nil {
		return nil, 0, nil
	}
	q := db.Model(&PlatformAdmin{})
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("email LIKE ? OR display_name LIKE ?", like, like)
	}
	return utils.FindPage[PlatformAdmin](q, page, size, "id DESC", utils.MaxPageSize100)
}

// UpdatePlatformAdminStatus updates the status of a platform admin (e.g. active/suspended).
// Records the operator as the update_by field when provided.
// Returns the number of rows affected and any error.
func UpdatePlatformAdminStatus(db *gorm.DB, id uint, status, operator string) (int64, error) {
	if status == "" || id == 0 {
		return 0, nil
	}
	updates := map[string]any{"status": status}
	if operator != "" {
		updates["update_by"] = operator
	}
	res := db.Model(&PlatformAdmin{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected, res.Error
}

// UpdatePlatformAdminProfile updates the email and/or display name of a platform admin.
// Empty values are ignored (no update). Records the operator as the update_by field when provided.
// Returns the number of rows affected and any error.
func UpdatePlatformAdminProfile(db *gorm.DB, id uint, email, displayName, operator string) (int64, error) {
	if id == 0 {
		return 0, nil
	}
	updates := map[string]any{}
	if email = utils.TrimLower(email); email != "" {
		updates["email"] = email
	}
	if displayName = strings.TrimSpace(displayName); displayName != "" {
		updates["display_name"] = displayName
	}
	if len(updates) == 0 {
		return 0, nil
	}
	if operator != "" {
		updates["update_by"] = operator
	}
	res := db.Model(&PlatformAdmin{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected, res.Error
}

// UpdatePlatformAdminPassword replaces the password hash for a platform admin.
// Does nothing when id is zero or passwordHash is empty.
func UpdatePlatformAdminPassword(db *gorm.DB, id uint, passwordHash string) error {
	if id == 0 || passwordHash == "" {
		return nil
	}
	return db.Model(&PlatformAdmin{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

// EnsureNotLastActivePlatformAdmin returns ErrLastActivePlatformAdmin when disabling or
// deleting the target platform admin would leave zero active platform admins.
// Only active targets are checked; already-inactive admins pass through.
func EnsureNotLastActivePlatformAdmin(db *gorm.DB, targetID uint) error {
	row, err := GetPlatformAdminByID(db, targetID)
	if err != nil {
		return err
	}
	if row.Status != constants.PlatformAdminStatusActive {
		return nil
	}
	n, err := CountPlatformAdmins(db)
	if err != nil {
		return err
	}
	if n <= 1 {
		return apperror.ErrLastActivePlatformAdmin
	}
	return nil
}

// SoftDeletePlatformAdmin marks a platform admin as deleted (status = deleted) and records
// the operator. Returns 1 on success. Fails with ErrLastActivePlatformAdmin when the target
// is the only remaining active admin.
func SoftDeletePlatformAdmin(db *gorm.DB, id uint, operator string) (int64, error) {
	var row PlatformAdmin
	if err := db.Where("id = ?", id).First(&row).Error; err != nil {
		return 0, err
	}
	if err := EnsureNotLastActivePlatformAdmin(db, id); err != nil {
		return 0, err
	}
	row.SoftDelete(operator)
	if err := db.Save(&row).Error; err != nil {
		return 0, err
	}
	return 1, nil
}
