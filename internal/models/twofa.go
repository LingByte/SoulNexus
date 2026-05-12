// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 双因素认证（TOTP + 备用码）：
//
// 设计：
//   - 一个用户一行 TwoFA（Secret + IsEnabled + 失败计数 / 锁定）；
//     若全表迁移期间也保留 User.TwoFactorEnabled / Secret 兼容字段，新流程优先看 TwoFA。
//   - TwoFABackupCode：一行一个备用码，bcrypt 哈希存储。
//   - 失败 5 次锁 5 分钟（与 LingVoice 对齐）。

package models

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	utils "github.com/LingByte/SoulNexus/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	twoFAMaxFailedAttempts = 5
	twoFALockoutSeconds    = 300
	twoFABackupCodeRawLen  = 10 // base32 字符数
)

// TwoFA 用户 2FA 设置（一个用户最多一行）。
type TwoFA struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	UserID         uint           `json:"user_id" gorm:"unique;not null;index"`
	Secret         string         `json:"-" gorm:"type:varchar(255);not null;comment:TOTP 密钥（不返回前端）"`
	IsEnabled      bool           `json:"is_enabled" gorm:"default:false;index"`
	FailedAttempts int            `json:"failed_attempts" gorm:"default:0"`
	LockedUntil    *time.Time     `json:"locked_until,omitempty"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (TwoFA) TableName() string { return "two_fa" }

// TwoFABackupCode 一次性备用码；CodeHash 为 bcrypt 哈希。
type TwoFABackupCode struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	CodeHash  string         `json:"-" gorm:"type:varchar(255);not null;comment:bcrypt 哈希"`
	IsUsed    bool           `json:"is_used" gorm:"default:false;index"`
	UsedAt    *time.Time     `json:"used_at,omitempty"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (TwoFABackupCode) TableName() string { return "two_fa_backup_codes" }

// normalizeBackupCode 规范化用户输入：去空白、转大写、去 dash。
func normalizeBackupCode(code string) string {
	s := strings.ToUpper(strings.TrimSpace(code))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// isValidBackupCodeInput 校验长度 / 字符集。
func isValidBackupCodeInput(code string) bool {
	n := normalizeBackupCode(code)
	if len(n) < 8 || len(n) > 32 {
		return false
	}
	for _, r := range n {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// GenerateTwoFATOTPSetupMaterial 生成 TOTP secret + URL + 二维码（包装 utils）。
func GenerateTwoFATOTPSetupMaterial(issuer string, user *User) (*utils.TOTPSetup, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}
	if strings.TrimSpace(issuer) == "" {
		issuer = utils.DefaultTOTPIssuer
	}
	account := strings.TrimSpace(user.Email)
	if account == "" {
		account = fmt.Sprintf("user-%d", user.ID)
	}
	return utils.GenerateTOTPSetup(issuer, account, utils.DefaultTOTPSecretSize)
}

// GetTwoFAByUserID 读单条；找不到返回 (nil, nil)。
func GetTwoFAByUserID(db *gorm.DB, userID uint) (*TwoFA, error) {
	if db == nil || userID == 0 {
		return nil, errors.New("db or userID nil")
	}
	var t TwoFA
	err := db.Where("user_id = ?", userID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// IsTwoFAEnabled 是否已启用。
func IsTwoFAEnabled(db *gorm.DB, userID uint) bool {
	t, err := GetTwoFAByUserID(db, userID)
	return err == nil && t != nil && t.IsEnabled
}

// UpsertTwoFASecret 写入 / 更新 secret（未启用前的"绑定中"阶段）。
func UpsertTwoFASecret(db *gorm.DB, userID uint, secret string) (*TwoFA, error) {
	if db == nil || userID == 0 || strings.TrimSpace(secret) == "" {
		return nil, errors.New("invalid args")
	}
	existing, err := GetTwoFAByUserID(db, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		row := &TwoFA{UserID: userID, Secret: secret, IsEnabled: false}
		if err := db.Create(row).Error; err != nil {
			return nil, err
		}
		return row, nil
	}
	existing.Secret = secret
	existing.IsEnabled = false
	existing.FailedAttempts = 0
	existing.LockedUntil = nil
	if err := db.Save(existing).Error; err != nil {
		return nil, err
	}
	return existing, nil
}

// EnableTwoFA 启用：清空失败计数与锁定。
func EnableTwoFA(db *gorm.DB, userID uint) error {
	t, err := GetTwoFAByUserID(db, userID)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("two_fa not bound; call UpsertTwoFASecret first")
	}
	t.IsEnabled = true
	t.FailedAttempts = 0
	t.LockedUntil = nil
	return db.Save(t).Error
}

// DisableTwoFA 关闭并清除备用码。
func DisableTwoFA(db *gorm.DB, userID uint) error {
	t, err := GetTwoFAByUserID(db, userID)
	if err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	t.IsEnabled = false
	t.FailedAttempts = 0
	t.LockedUntil = nil
	if err := db.Save(t).Error; err != nil {
		return err
	}
	return db.Where("user_id = ?", userID).Delete(&TwoFABackupCode{}).Error
}

// IsTwoFALocked 是否处于锁定期。
func (t *TwoFA) IsTwoFALocked() bool {
	if t == nil || t.LockedUntil == nil {
		return false
	}
	return t.LockedUntil.After(time.Now())
}

// IncrementTwoFAFailedAttempts 增加失败计数；达阈值时设置锁定期。
func IncrementTwoFAFailedAttempts(db *gorm.DB, t *TwoFA) error {
	if db == nil || t == nil {
		return errors.New("nil")
	}
	t.FailedAttempts++
	if t.FailedAttempts >= twoFAMaxFailedAttempts {
		until := time.Now().Add(time.Duration(twoFALockoutSeconds) * time.Second)
		t.LockedUntil = &until
	}
	return db.Save(t).Error
}

// ResetTwoFAFailedAttempts 校验通过后清零。
func ResetTwoFAFailedAttempts(db *gorm.DB, t *TwoFA) error {
	if db == nil || t == nil {
		return errors.New("nil")
	}
	t.FailedAttempts = 0
	t.LockedUntil = nil
	return db.Save(t).Error
}

// ValidateTOTPAndUpdateUsage 校验当前 TOTP；锁定期内直接拒绝；成功后清零。
func ValidateTOTPAndUpdateUsage(db *gorm.DB, t *TwoFA, code string) (bool, error) {
	if t == nil {
		return false, errors.New("two_fa nil")
	}
	if t.IsTwoFALocked() {
		return false, fmt.Errorf("账户已被锁定，请在 %s 后重试", t.LockedUntil.Format("2006-01-02 15:04:05"))
	}
	if !utils.ValidateTOTP(strings.TrimSpace(code), t.Secret) {
		_ = IncrementTwoFAFailedAttempts(db, t)
		return false, errors.New("验证码不正确")
	}
	_ = ResetTwoFAFailedAttempts(db, t)
	return true, nil
}

// =============== Backup Codes ===============

// GenerateBackupCodes 生成 n 个备用码（明文返回给用户一次，DB 仅存哈希）。
func GenerateBackupCodes(db *gorm.DB, userID uint, n int) ([]string, error) {
	if db == nil || userID == 0 {
		return nil, errors.New("invalid args")
	}
	if n <= 0 {
		n = 8
	}
	plain := make([]string, 0, n)
	rows := make([]TwoFABackupCode, 0, n)
	for i := 0; i < n; i++ {
		raw, err := newBackupCodeRaw()
		if err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		plain = append(plain, formatBackupCodeForDisplay(raw))
		rows = append(rows, TwoFABackupCode{UserID: userID, CodeHash: string(hash)})
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&TwoFABackupCode{}).Error; err != nil {
			return err
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return plain, nil
}

// CountUnusedBackupCodes 未用条数。
func CountUnusedBackupCodes(db *gorm.DB, userID uint) (int64, error) {
	var n int64
	err := db.Model(&TwoFABackupCode{}).Where("user_id = ? AND is_used = ?", userID, false).Count(&n).Error
	return n, err
}

// ValidateBackupCodeAndConsume 校验备用码；通过则标记 IsUsed。
// 锁定期内直接拒绝。
func ValidateBackupCodeAndConsume(db *gorm.DB, t *TwoFA, code string) (bool, error) {
	if db == nil || t == nil {
		return false, errors.New("nil")
	}
	if t.IsTwoFALocked() {
		return false, fmt.Errorf("账户已被锁定，请在 %s 后重试", t.LockedUntil.Format("2006-01-02 15:04:05"))
	}
	if !isValidBackupCodeInput(code) {
		_ = IncrementTwoFAFailedAttempts(db, t)
		return false, errors.New("备用码格式不正确")
	}
	norm := normalizeBackupCode(code)
	var rows []TwoFABackupCode
	if err := db.Where("user_id = ? AND is_used = ?", t.UserID, false).Find(&rows).Error; err != nil {
		return false, err
	}
	for i := range rows {
		if bcrypt.CompareHashAndPassword([]byte(rows[i].CodeHash), []byte(norm)) == nil {
			now := time.Now()
			rows[i].IsUsed = true
			rows[i].UsedAt = &now
			if err := db.Save(&rows[i]).Error; err != nil {
				return false, err
			}
			_ = ResetTwoFAFailedAttempts(db, t)
			return true, nil
		}
	}
	_ = IncrementTwoFAFailedAttempts(db, t)
	return false, errors.New("备用码无效或已使用")
}

func newBackupCodeRaw() (string, error) {
	// 6 bytes -> 10 base32 chars (without padding)
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	s := strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "=")
	if len(s) > twoFABackupCodeRawLen {
		s = s[:twoFABackupCodeRawLen]
	}
	return strings.ToUpper(s), nil
}

func formatBackupCodeForDisplay(raw string) string {
	if len(raw) <= 5 {
		return raw
	}
	return raw[:5] + "-" + raw[5:]
}
