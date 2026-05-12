// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Passkey / FIDO2 / WebAuthn 数据持久化。
//
// 表设计：
//   - Passkey：注册成功后的凭证（CredentialID + 公钥 + sign_count + transports + AAGUID）。
//     CredentialID 是 WebAuthn 协议的 raw bytes，存 base64url。
//   - PasskeyChallenge：注册/登录开始时下发的 challenge，临时存储以避免内存丢失（本地多副本部署需要持久化）。
//
// 使用：
//   - 注册：POST /api/me/passkeys/registration/begin → 返回 publicKey options
//          POST /api/me/passkeys/registration/finish → 校验 attestation 并写入 Passkey
//   - 认证：POST /api/auth/passkey/begin → 返回 assertion options（discoverable login，不需要 username）
//          POST /api/auth/passkey/finish → 校验 assertion 后签发 JWT
//
// 校验由 pkg/auth/webauthn 包包装 go-webauthn 库完成。

package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Passkey 一条已注册的 WebAuthn 凭证。
type Passkey struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          uint           `json:"user_id" gorm:"index;not null"`
	CredentialID    string         `json:"credential_id" gorm:"size:512;uniqueIndex;not null;comment:base64url 编码的 raw credential id"`
	PublicKey       []byte         `json:"-" gorm:"type:blob;not null;comment:CBOR 编码的公钥"`
	AAGUID          []byte         `json:"-" gorm:"type:blob;comment:authenticator AAGUID"`
	SignCount       uint32         `json:"sign_count" gorm:"default:0;comment:重放保护计数器"`
	Transports      string         `json:"transports" gorm:"size:128;comment:逗号分隔，usb|nfc|ble|internal|hybrid"`
	AttestationType string         `json:"attestation_type" gorm:"size:32;comment:none|packed|fido-u2f 等"`
	UserPresent     bool           `json:"user_present" gorm:"default:true"`
	UserVerified    bool           `json:"user_verified" gorm:"default:true"`
	BackupEligible  bool           `json:"backup_eligible" gorm:"default:false;comment:可被云同步的密钥"`
	BackupState     bool           `json:"backup_state" gorm:"default:false"`
	Nickname        string         `json:"nickname" gorm:"size:128;comment:用户起的别名（如 'iCloud 钥匙串'）"`
	LastUsedAt      *time.Time     `json:"last_used_at,omitempty"`
	LastUsedIP      string         `json:"last_used_ip,omitempty" gorm:"size:64"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName GORM 表名。
func (Passkey) TableName() string { return "passkeys" }

// PasskeyChallenge 临时下发给客户端的注册 / 登录 challenge。
//
// 用 SessionID（uuid）作为主键；UserID = 0 表示无密码登录态（discoverable）。
type PasskeyChallenge struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserID        uint      `json:"user_id" gorm:"index;default:0"`
	Type          string    `json:"type" gorm:"size:16;not null;index;comment:registration|login"`
	SessionData   []byte    `json:"-" gorm:"type:blob;not null;comment:webauthn.SessionData JSON"`
	ChallengeB64U string    `json:"challenge_b64u" gorm:"size:128;index;comment:base64url challenge"`
	ExpiresAt     time.Time `json:"expires_at" gorm:"index;not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName GORM 表名。
func (PasskeyChallenge) TableName() string { return "passkey_challenges" }

// IsExpired 是否已过期。
func (p *PasskeyChallenge) IsExpired() bool {
	return p == nil || time.Now().After(p.ExpiresAt)
}

// ListPasskeysForUser 列出用户已注册的 Passkey（不返回公钥与 AAGUID）。
func ListPasskeysForUser(db *gorm.DB, userID uint) ([]Passkey, error) {
	if db == nil || userID == 0 {
		return nil, errors.New("invalid args")
	}
	var rows []Passkey
	err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// FindPasskeyByCredentialID 按 raw credential id（base64url）定位。
func FindPasskeyByCredentialID(db *gorm.DB, credentialID string) (*Passkey, error) {
	id := strings.TrimSpace(credentialID)
	if id == "" {
		return nil, errors.New("empty credential id")
	}
	var row Passkey
	if err := db.Where("credential_id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// DeletePasskey 按 owner 检验后删除。
func DeletePasskey(db *gorm.DB, userID uint, id uint) error {
	if db == nil || userID == 0 || id == 0 {
		return errors.New("invalid args")
	}
	res := db.Where("user_id = ? AND id = ?", userID, id).Delete(&Passkey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("passkey not found")
	}
	return nil
}

// PrunePasskeyChallenges 周期性清理过期 challenge（建议挂到 task）。
func PrunePasskeyChallenges(db *gorm.DB) (int64, error) {
	res := db.Where("expires_at < ?", time.Now()).Delete(&PasskeyChallenge{})
	return res.RowsAffected, res.Error
}

// SavePasskeyChallenge 写入；id 由调用方提供（webauthn session id / uuid）。
func SavePasskeyChallenge(db *gorm.DB, ch *PasskeyChallenge) error {
	if db == nil || ch == nil || strings.TrimSpace(ch.ID) == "" {
		return errors.New("invalid args")
	}
	if ch.ExpiresAt.IsZero() {
		ch.ExpiresAt = time.Now().Add(5 * time.Minute)
	}
	return db.Create(ch).Error
}

// LoadPasskeyChallenge 读取并自动删除（防重放）。
func LoadPasskeyChallenge(db *gorm.DB, id string) (*PasskeyChallenge, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("empty id")
	}
	var row PasskeyChallenge
	if err := db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	_ = db.Where("id = ?", id).Delete(&PasskeyChallenge{}).Error
	if row.IsExpired() {
		return nil, errors.New("challenge expired")
	}
	return &row, nil
}
