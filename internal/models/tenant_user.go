package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"time"

	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// TenantUser is a login identity scoped to exactly one tenant (SaaS member).
type TenantUser struct {
	common.BaseModel

	TenantID                 uint       `json:"tenantId" gorm:"index;not null;comment:所属租户ID"`
	Email                    string     `json:"email" gorm:"size:256;uniqueIndex:idx_global_tenant_user_email;not null;comment:登录邮箱"`
	Phone                    string     `json:"phone" gorm:"size:32;index;comment:手机号"`
	Username                 string     `json:"username" gorm:"size:128;comment:用户名"`
	PasswordHash             string     `json:"-" gorm:"size:256;comment:密码哈希"`
	DisplayName              string     `json:"displayName" gorm:"size:128;comment:显示名"`
	AvatarURL                string     `json:"avatarUrl" gorm:"size:512;comment:头像URL"`
	Status                   string     `json:"status" gorm:"size:32;index;not null;default:active;comment:账号状态"`
	LastLogin                *time.Time `json:"lastLogin,omitempty" gorm:"column:last_login_at;comment:最后登录时间"`
	LastLoginIP              string     `json:"-" gorm:"size:128;column:last_login_ip;comment:最后登录IP"`
	LastLoginCity            string     `json:"lastLoginCity,omitempty" gorm:"size:64;column:last_login_city;comment:最后登录城市"`
	LastLoginLocation        string     `json:"lastLoginLocation,omitempty" gorm:"size:256;column:last_login_location;comment:最后登录地理位置"`
	Source                   string     `json:"source" gorm:"size:64;index;default:register;comment:来源"`
	GitHubID                 string     `json:"-" gorm:"size:64;index;column:github_id;comment:GitHub用户ID"`
	GitHubLogin              string     `json:"githubLogin,omitempty" gorm:"size:128;column:github_login;comment:GitHub用户名"`
	LoginCount               int        `json:"loginCount" gorm:"default:0;comment:登录次数"`
	TOTPSecret               string     `json:"-" gorm:"size:128;column:totp_secret;comment:TOTP密钥"`
	TOTPEnabled              bool       `json:"totpEnabled" gorm:"column:totp_enabled;not null;default:0;comment:是否启用TOTP"`
	TOTPRecoveryHashes       string     `json:"-" gorm:"column:totp_recovery_hashes;type:text;comment:TOTP恢复码哈希JSON"`
	ReceiveEmailNotify       bool       `json:"receiveEmailNotify" gorm:"column:receive_email_notify;not null;default:0;comment:接收营销/通知类邮件"`
	RequireDeviceVerify      bool       `json:"requireDeviceVerify" gorm:"column:require_device_verify;not null;default:0;comment:新设备需邮箱验证"`
	TrustDeviceLoginEnabled  bool       `json:"trustDeviceLoginEnabled" gorm:"column:trust_device_login_enabled;not null;default:1;comment:7天免登录验证"`
	RequireRemoteLoginVerify bool       `json:"requireRemoteLoginVerify" gorm:"column:require_remote_login_verify;not null;default:0;comment:异地登录保护"`
	PrimaryLoginCity         string     `json:"primaryLoginCity,omitempty" gorm:"column:primary_login_city;size:64;comment:常用登录城市"`
	KnownLoginCitiesJSON     string     `json:"-" gorm:"column:known_login_cities;type:text;comment:已知登录城市JSON"`
	SessionIdleTimeoutHours  int        `json:"sessionIdleTimeoutHours" gorm:"column:session_idle_timeout_hours;not null;default:12;comment:无操作退出小时数"`
	SessionMaxLifetimeHours  int        `json:"sessionMaxLifetimeHours" gorm:"column:session_max_lifetime_hours;not null;default:48;comment:最大登录保持小时数"`
	VoiceprintID             *uint      `json:"voiceprintId,omitempty" gorm:"index;comment:绑定的账号声纹 voiceprints.id"`
	WelcomeNotifiedAt        *time.Time `json:"-" gorm:"column:welcome_notified_at;comment:欢迎通知已发送"`
	DeletionRequestedAt      *time.Time `json:"deletionRequestedAt,omitempty" gorm:"column:deletion_requested_at;comment:注销申请时间"`
}

func (TenantUser) TableName() string {
	return constants2.TENANT_USER_TABLE_NAME
}

// ActiveTenantUsers is the non-deleted tenant user scope.
func ActiveTenantUsers(db *gorm.DB) *gorm.DB {
	return db.Model(&TenantUser{})
}

// ListTenantUsersPage lists active tenant users with optional filters.
func ListTenantUsersPage(db *gorm.DB, tenantID uint, page, size int, status, search string) ([]TenantUser, int64, error) {
	q := ActiveTenantUsers(db)
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if s := strings.TrimSpace(status); s != "" {
		q = q.Where("status = ?", s)
	}
	if search = strings.TrimSpace(search); search != "" {
		q = q.Where("email LIKE ? OR username LIKE ? OR display_name LIKE ? OR phone LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	return utils.FindPage[TenantUser](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

// GetAuthenticatedTenantUser returns the active user when JWT user id matches tenant id.
func GetAuthenticatedTenantUser(db *gorm.DB, userID, tenantID uint) (TenantUser, error) {
	if userID == 0 || tenantID == 0 {
		return TenantUser{}, gorm.ErrRecordNotFound
	}
	u, err := GetActiveTenantUserByID(db, userID)
	if err != nil {
		return TenantUser{}, err
	}
	if u.TenantID != tenantID {
		return TenantUser{}, gorm.ErrRecordNotFound
	}
	return u, nil
}

// GetActiveTenantUserByID returns one active tenant user by primary key.
func GetActiveTenantUserByID(db *gorm.DB, id uint) (TenantUser, error) {
	var row TenantUser
	err := ActiveTenantUsers(db).Where("id = ?", id).First(&row).Error
	return row, err
}

// GetTenantUserByID returns a tenant user by ID (ignores soft-delete).
func GetTenantUserByID(db *gorm.DB, id uint) (TenantUser, error) {
	var row TenantUser
	err := db.First(&row, id).Error
	return row, err
}

// GetActiveTenantUserByEmailGlobal returns a non-deleted tenant user by email (unique across the system).
func GetActiveTenantUserByEmailGlobal(db *gorm.DB, email string) (TenantUser, error) {
	var row TenantUser
	err := ActiveTenantUsers(db).Where("email = ?", utils.TrimLower(email)).First(&row).Error
	return row, err
}

// GetActiveTenantUserByPhoneGlobal returns a non-deleted tenant user by normalized phone digits.
func GetActiveTenantUserByPhoneGlobal(db *gorm.DB, phone string) (TenantUser, error) {
	phone = utils.NormalizePhone(phone)
	if phone == "" {
		return TenantUser{}, gorm.ErrRecordNotFound
	}
	var row TenantUser
	q := ActiveTenantUsers(db).Where("phone = ?", phone)
	// Common China variants: +86 / 86 prefix stored inconsistently.
	if strings.HasPrefix(phone, "86") && len(phone) == 13 {
		q = q.Or("phone = ?", phone[2:])
	} else if len(phone) == 11 && strings.HasPrefix(phone, "1") {
		q = q.Or("phone = ?", "86"+phone).Or("phone = ?", "+86"+phone)
	}
	err := q.First(&row).Error
	return row, err
}

// CreateTenantUser creates a new tenant user.
func CreateTenantUser(db *gorm.DB, user *TenantUser) error {
	return db.Create(user).Error
}

// UpdateTenantUser updates a tenant user by ID.
func UpdateTenantUser(db *gorm.DB, id uint, updates map[string]any, updateBy string) (int64, error) {
	meta := common.BaseModel{}
	meta.SetUpdateInfo(updateBy)
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	res := db.Model(&TenantUser{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected, res.Error
}

// UpdateTenantUserStatus updates the status of a tenant user.
func UpdateTenantUserStatus(db *gorm.DB, id uint, status, updateBy string) (int64, error) {
	updates := map[string]any{"status": status}
	meta := common.BaseModel{}
	meta.SetUpdateInfo(updateBy)
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	res := db.Model(&TenantUser{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected, res.Error
}

// GetActiveTenantUserByGitHubID returns a tenant user bound to the given GitHub account id.
func GetActiveTenantUserByGitHubID(db *gorm.DB, githubID string) (TenantUser, error) {
	githubID = strings.TrimSpace(githubID)
	if githubID == "" {
		return TenantUser{}, gorm.ErrRecordNotFound
	}
	var row TenantUser
	err := ActiveTenantUsers(db).Where("github_id = ?", githubID).First(&row).Error
	return row, err
}

// RecordSuccessfulLoginCity appends the city to the user's known login cities.
func RecordSuccessfulLoginCity(db *gorm.DB, userID uint, city string) {
	city = strings.TrimSpace(city)
	if userID == 0 || city == "" {
		return
	}
	var u TenantUser
	if err := db.Select("known_login_cities", "primary_login_city").First(&u, userID).Error; err != nil {
		return
	}
	updates := map[string]any{
		"primary_login_city": city,
		"known_login_cities": utils.AddKnownLoginCity(u.KnownLoginCitiesJSON, city),
	}
	_ = db.Model(&TenantUser{}).Where("id = ?", userID).Updates(updates).Error
}

// RecordTenantUserLogin sets last login time, IP, geo and increments login_count.
func RecordTenantUserLogin(db *gorm.DB, id uint, ip, city, location string) error {
	now := time.Now()
	ip = strings.TrimSpace(ip)
	if len(ip) > 128 {
		ip = ip[:128]
	}
	city = strings.TrimSpace(city)
	if len(city) > 64 {
		city = city[:64]
	}
	location = strings.TrimSpace(location)
	if len(location) > 256 {
		location = location[:256]
	}
	updates := map[string]any{
		"last_login_at": &now,
		"last_login_ip": ip,
		"login_count":   gorm.Expr("login_count + ?", 1),
	}
	if city != "" {
		updates["last_login_city"] = city
	}
	if location != "" {
		updates["last_login_location"] = location
	}
	return db.Model(&TenantUser{}).Where("id = ?", id).Updates(updates).Error
}

// SoftDeleteTenantUserByID soft-deletes a tenant user by ID.
func SoftDeleteTenantUserByID(db *gorm.DB, id uint, updateBy string) (int64, error) {
	meta := common.BaseModel{}
	meta.SoftDelete(updateBy)
	u := map[string]any{
		"deleted_at": meta.DeletedAt,
		"updated_at": meta.UpdatedAt,
	}
	if meta.UpdateBy != "" {
		u["update_by"] = meta.UpdateBy
	}
	res := db.Model(&TenantUser{}).Where("id = ?", id).Updates(u)
	return res.RowsAffected, res.Error
}

// RestoreTenantUser restores a soft-deleted tenant user.
func RestoreTenantUser(db *gorm.DB, id uint, updateBy string) (int64, error) {
	meta := common.BaseModel{}
	meta.Restore(updateBy)
	u := map[string]any{
		"deleted_at": nil,
		"updated_at": meta.UpdatedAt,
	}
	if meta.UpdateBy != "" {
		u["update_by"] = meta.UpdateBy
	}
	res := db.Unscoped().Model(&TenantUser{}).Where("id = ?", id).Updates(u)
	return res.RowsAffected, res.Error
}

// CountTenantUsers counts total active users (optionally by tenant).
func CountTenantUsers(db *gorm.DB, tenantID uint) (int64, error) {
	q := ActiveTenantUsers(db)
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

// CountTenantUsersByStatus counts users by status.
func CountTenantUsersByStatus(db *gorm.DB, tenantID uint, status string) (int64, error) {
	q := ActiveTenantUsers(db).Where("status = ?", status)
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

// CheckTenantUserEmailExists checks if email is already used by an active user (globally unique).
func CheckTenantUserEmailExists(db *gorm.DB, email string, excludeID uint) (bool, error) {
	email = utils.TrimLower(email)
	q := ActiveTenantUsers(db).Where("email = ?", email)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

// CheckTenantUserPhoneExists checks if phone is already used globally (excluding empty phone).
func CheckTenantUserPhoneExists(db *gorm.DB, phone string, excludeID uint) (bool, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false, nil
	}
	q := ActiveTenantUsers(db).Where("phone = ?", phone)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

// CheckTenantUserUsernameExists checks if username is already used globally (excluding blank).
func CheckTenantUserUsernameExists(db *gorm.DB, username string, excludeID uint) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil
	}
	q := ActiveTenantUsers(db).Where("username = ?", username)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}
