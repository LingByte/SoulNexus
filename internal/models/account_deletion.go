package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DefaultAccountDeletionCooldown 默认冷静期 72 小时（3 天），可通过环境变量 ACCOUNT_DELETION_COOLDOWN_HOURS 覆盖。
func DefaultAccountDeletionCooldown() time.Duration {
	h := 72
	if v := strings.TrimSpace(os.Getenv("ACCOUNT_DELETION_COOLDOWN_HOURS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 720 {
			h = n
		}
	}
	return time.Duration(h) * time.Hour
}

// AccountDeletionPending 是否在冷静期内（已申请、尚未到期、账号仍为正常态）。
func AccountDeletionPending(user *User) bool {
	if user == nil || user.AccountDeletionEffectiveAt == nil {
		return false
	}
	return time.Now().Before(*user.AccountDeletionEffectiveAt) && user.IsDeleted == SoftDeleteStatusActive
}

// ThirdPartyBindings 是否仍绑定 GitHub / 微信。
func ThirdPartyBindings(user *User) (github bool, wechat bool) {
	if user == nil {
		return false, false
	}
	github = strings.TrimSpace(user.GithubID) != ""
	wechat = strings.TrimSpace(user.WechatOpenID) != ""
	return github, wechat
}

// HasRecentSuspiciousLogins 近期是否存在标记为可疑的成功登录（用于注销前风控）。
func HasRecentSuspiciousLogins(db *gorm.DB, userID uint, lookback time.Duration) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	since := time.Now().Add(-lookback)
	var n int64
	err := db.Model(&LoginHistory{}).
		Where("user_id = ? AND is_suspicious = ? AND success = ? AND created_at >= ?", userID, true, true, since).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// AccountDeletionEligibilityReasons 返回不满足注销申请条件的原因（空切片表示通过基础校验）。
func AccountDeletionEligibilityReasons(db *gorm.DB, user *User, accountLocked bool, remoteLoginRisk bool, recentSuspicious bool) []string {
	var reasons []string
	if user == nil {
		return []string{"用户不存在"}
	}
	if err := CheckUserAllowLogin(db, user); err != nil {
		reasons = append(reasons, "账号状态不允许（未激活、已封禁或角色异常）")
	}
	if UserHasAdminAccess(db, user.ID) {
		reasons = append(reasons, "管理员账号不支持自助注销")
	}
	if accountLocked {
		reasons = append(reasons, "账号因多次失败登录处于锁定中")
	}
	if remoteLoginRisk {
		reasons = append(reasons, "当前网络环境存在异地登录风险，请稍后在常用环境再试")
	}
	if recentSuspicious {
		reasons = append(reasons, "近期存在异常登录记录")
	}
	gh, wx := ThirdPartyBindings(user)
	if gh {
		reasons = append(reasons, "仍绑定 GitHub，请先解绑")
	}
	if wx {
		reasons = append(reasons, "仍绑定微信，请先解绑")
	}
	return reasons
}

// ScheduleAccountDeletion 进入冷静期。
func ScheduleAccountDeletion(db *gorm.DB, userID uint, operator string) error {
	now := time.Now()
	effective := now.Add(DefaultAccountDeletionCooldown())
	vals := map[string]any{
		"account_deletion_requested_at": &now,
		"account_deletion_effective_at": &effective,
		"update_by":                     operator,
	}
	return db.Model(&User{}).Where("id = ? AND is_deleted = ?", userID, SoftDeleteStatusActive).Updates(vals).Error
}

// CancelAccountDeletion 用户主动撤回注销申请。
func CancelAccountDeletion(db *gorm.DB, userID uint, operator string) error {
	vals := map[string]any{
		"account_deletion_requested_at": nil,
		"account_deletion_effective_at": nil,
		"update_by":                     operator,
	}
	return db.Model(&User{}).Where("id = ? AND is_deleted = ?", userID, SoftDeleteStatusActive).Updates(vals).Error
}

// ListUsersDueForAccountDeletion 冷静期已结束、待执行永久注销的用户。
func ListUsersDueForAccountDeletion(db *gorm.DB, before time.Time) ([]User, error) {
	var list []User
	err := db.Where("account_deletion_effective_at IS NOT NULL AND account_deletion_effective_at <= ?", before).
		Where("is_deleted = ? AND status = ?", SoftDeleteStatusActive, UserStatusActive).
		Find(&list).Error
	return list, err
}

// FinalizeAccountDeletion 永久注销：删除绑定类数据，将用户行匿名化并软删除；不删除助手、知识库等业务资源。
func FinalizeAccountDeletion(db *gorm.DB, userID uint, operator string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var u User
		if err := tx.Where("id = ?", userID).First(&u).Error; err != nil {
			return err
		}
		if u.IsDeleted == SoftDeleteStatusDeleted {
			return nil
		}
		if u.AccountDeletionEffectiveAt == nil || u.AccountDeletionEffectiveAt.After(time.Now()) {
			return errors.New("account deletion cooling period not finished")
		}

		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&LoginHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&UserDevice{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("created_by = ?", userID).Delete(&UserCredential{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&GroupMember{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("inviter_id = ? OR invitee_id = ?", userID, userID).Delete(&GroupInvitation{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&AccountLock{}).Where("user_id = ? OR email = ?", userID, strings.ToLower(strings.TrimSpace(u.Email))).
			Update("is_active", false).Error; err != nil {
			return err
		}

		tombstone := fmt.Sprintf("deleted.%d.%d@void.invalid", userID, time.Now().UnixNano())
		userUpdates := map[string]any{
			"email":                         tombstone,
			"password":                      "",
			"github_id":                     "",
			"github_login":                  "",
			"wechat_open_id":                "",
			"wechat_union_id":               "",
			"two_factor_enabled":            false,
			"two_factor_secret":             "",
			"email_verify_token":            "",
			"phone_verify_token":            "",
			"password_reset_token":          "",
			"password_reset_expires":        nil,
			"email_verify_expires":          nil,
			"email_verified":                false,
			"phone_verified":                false,
			"status":                        UserStatusBanned,
			"is_deleted":                    SoftDeleteStatusDeleted,
			"account_deletion_requested_at": nil,
			"account_deletion_effective_at": nil,
			"update_by":                     operator,
		}
		if err := tx.Model(&User{}).Where("id = ?", userID).Updates(userUpdates).Error; err != nil {
			return err
		}
		profUpdates := map[string]any{
			"display_name": "已注销用户",
			"first_name":   "",
			"last_name":    "",
			"phone":        "",
			"avatar":       "",
		}
		_ = EnsureUserProfile(tx, userID)
		if err := tx.Model(&UserProfile{}).Where("user_id = ?", userID).Updates(profUpdates).Error; err != nil {
			return err
		}
		return nil
	})
}
