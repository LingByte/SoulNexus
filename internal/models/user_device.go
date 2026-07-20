package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	UserDevicePrincipalTenantUser    = "tenant_user"
	UserDevicePrincipalPlatformAdmin = "platform_admin"
)

// UserDevice tracks trusted devices and active login sessions per principal.
type UserDevice struct {
	common.BaseModel
	PrincipalType         string     `json:"principalType" gorm:"size:32;not null;uniqueIndex:idx_ud_principal_device,priority:1;index:idx_ud_principal,priority:1"`
	PrincipalID           uint       `json:"principalId" gorm:"not null;uniqueIndex:idx_ud_principal_device,priority:2;index:idx_ud_principal,priority:2"`
	DeviceKey             string     `json:"deviceKey" gorm:"size:64;not null;uniqueIndex:idx_ud_principal_device,priority:3;comment:客户端持久化 deviceId"`
	Category              string     `json:"category" gorm:"size:16;not null;comment:mobile|desktop|tablet"`
	LimitCategory         string     `json:"limitCategory" gorm:"size:16;not null;comment:mobile|desktop 用于并发登录限制"`
	DisplayName           string     `json:"displayName" gorm:"size:128"`
	UserAgent             string     `json:"userAgent,omitempty" gorm:"size:512"`
	LastIP                string     `json:"lastIp,omitempty" gorm:"size:64"`
	IsTrusted             bool       `json:"isTrusted" gorm:"not null;default:false"`
	TrustedAt             *time.Time `json:"trustedAt,omitempty"`
	TrustedUntil          *time.Time `json:"trustedUntil,omitempty" gorm:"comment:7天免验证到期"`
	LastLoginCity         string     `json:"lastLoginCity,omitempty" gorm:"size:64;comment:最近登录城市"`
	SessionID             string     `json:"sessionId,omitempty" gorm:"size:64;index"`
	SessionActive         bool       `json:"sessionActive" gorm:"not null;default:false;index"`
	SessionIssuedAt       *time.Time `json:"sessionIssuedAt,omitempty" gorm:"comment:会话签发时间"`
	SessionLastActivityAt *time.Time `json:"sessionLastActivityAt,omitempty" gorm:"comment:会话最后活动时间"`
	LastLoginAt           *time.Time `json:"lastLoginAt,omitempty"`
}

func (UserDevice) TableName() string { return "user_devices" }

// DeviceWithinTrustWindow reports whether the device is still within its trust period.
func DeviceWithinTrustWindow(row UserDevice, now time.Time) bool {
	return row.TrustedUntil != nil && row.TrustedUntil.After(now)
}

type UserDevicePublic struct {
	ID            string     `json:"id"`
	DeviceKey     string     `json:"deviceKey"`
	Category      string     `json:"category"`
	LimitCategory string     `json:"limitCategory"`
	DisplayName   string     `json:"displayName"`
	LastIP        string     `json:"lastIp,omitempty"`
	LastLoginCity string     `json:"lastLoginCity,omitempty"`
	IsTrusted     bool       `json:"isTrusted"`
	SessionActive bool       `json:"sessionActive"`
	LastLoginAt   *time.Time `json:"lastLoginAt,omitempty"`
	TrustedAt     *time.Time `json:"trustedAt,omitempty"`
	IsCurrent     bool       `json:"isCurrent"`
}

func UserDevicePublicRow(row UserDevice, currentDeviceKey string) UserDevicePublic {
	now := time.Now()
	trusted := row.IsTrusted && DeviceWithinTrustWindow(row, now)
	return UserDevicePublic{
		ID:            strconv.FormatUint(uint64(row.ID), 10),
		DeviceKey:     row.DeviceKey,
		Category:      row.Category,
		LimitCategory: row.LimitCategory,
		DisplayName:   row.DisplayName,
		LastIP:        row.LastIP,
		LastLoginCity: row.LastLoginCity,
		IsTrusted:     trusted,
		SessionActive: row.SessionActive,
		LastLoginAt:   row.LastLoginAt,
		TrustedAt:     row.TrustedAt,
		IsCurrent:     currentDeviceKey != "" && row.DeviceKey == currentDeviceKey,
	}
}

func GetUserDeviceByKey(db *gorm.DB, principalType string, principalID uint, deviceKey string) (UserDevice, error) {
	var row UserDevice
	err := db.Where("principal_type = ? AND principal_id = ? AND device_key = ?",
		strings.TrimSpace(principalType), principalID, strings.TrimSpace(deviceKey)).First(&row).Error
	return row, err
}

// GetUserDeviceByKeyUnscoped finds a device row including soft-deleted records.
func GetUserDeviceByKeyUnscoped(db *gorm.DB, principalType string, principalID uint, deviceKey string) (UserDevice, error) {
	var row UserDevice
	err := db.Unscoped().Where("principal_type = ? AND principal_id = ? AND device_key = ?",
		strings.TrimSpace(principalType), principalID, strings.TrimSpace(deviceKey)).First(&row).Error
	return row, err
}

func ListUserDevices(db *gorm.DB, principalType string, principalID uint) ([]UserDevice, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var rows []UserDevice
	err := db.Where("principal_type = ? AND principal_id = ?", principalType, principalID).
		Order("session_active DESC, last_login_at DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

func IsUserDeviceSessionActive(db *gorm.DB, deviceRecordID uint, sessionID string) (bool, error) {
	if deviceRecordID == 0 || strings.TrimSpace(sessionID) == "" {
		return true, nil
	}
	var row UserDevice
	if err := db.Select("session_active", "session_id").First(&row, deviceRecordID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return row.SessionActive && row.SessionID == sessionID, nil
}

func DeactivateDesktopSessions(db *gorm.DB, principalType string, principalID uint, exceptDeviceID uint) error {
	q := db.Model(&UserDevice{}).
		Where("principal_type = ? AND principal_id = ? AND limit_category = ? AND session_active = ?",
			principalType, principalID, "desktop", true)
	if exceptDeviceID > 0 {
		q = q.Where("id <> ?", exceptDeviceID)
	}
	return q.Update("session_active", false).Error
}

// ListUserDevicesPage lists devices with pagination.
func ListUserDevicesPage(db *gorm.DB, principalType string, principalID uint, page, size int) ([]UserDevice, int64, error) {
	if db == nil {
		return nil, 0, errors.New("nil db")
	}
	q := db.Model(&UserDevice{}).
		Where("principal_type = ? AND principal_id = ?", principalType, principalID)
	return utils.FindPage[UserDevice](q, page, size, "session_active DESC, last_login_at DESC, id DESC", utils.MaxPageSize100)
}

// TrustUserDevice marks a device as trusted for TrustDeviceLoginDays.
func TrustUserDevice(db *gorm.DB, principalType string, principalID, deviceID uint, operator string) error {
	now := time.Now()
	until := now.Add(time.Duration(utils.TrustDeviceLoginDays) * 24 * time.Hour)
	updates := map[string]any{
		"is_trusted":    true,
		"trusted_at":    now,
		"trusted_until": until,
	}
	if op := strings.TrimSpace(operator); op != "" {
		updates["update_by"] = op
	}
	res := db.Model(&UserDevice{}).
		Where("id = ? AND principal_type = ? AND principal_id = ?", deviceID, principalType, principalID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func RevokeUserDevice(db *gorm.DB, principalType string, principalID, deviceID uint, operator string) error {
	updates := map[string]any{
		"is_trusted":               false,
		"session_active":           false,
		"trusted_at":               nil,
		"trusted_until":            nil,
		"session_id":               "",
		"session_issued_at":        nil,
		"session_last_activity_at": nil,
	}
	if op := strings.TrimSpace(operator); op != "" {
		updates["update_by"] = op
	}
	res := db.Model(&UserDevice{}).
		Where("id = ? AND principal_type = ? AND principal_id = ?", deviceID, principalType, principalID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteUserDevice removes a device record. Current device cannot be deleted.
func DeleteUserDevice(db *gorm.DB, principalType string, principalID, deviceID uint, currentDeviceKey, operator string) error {
	if strings.TrimSpace(currentDeviceKey) != "" {
		var row UserDevice
		if err := db.Select("device_key").Where("id = ? AND principal_type = ? AND principal_id = ?",
			deviceID, principalType, principalID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			return err
		}
		if row.DeviceKey == strings.TrimSpace(currentDeviceKey) {
			return errors.New("cannot delete current device")
		}
	}
	if op := strings.TrimSpace(operator); op != "" {
		_ = db.Model(&UserDevice{}).
			Where("id = ? AND principal_type = ? AND principal_id = ?", deviceID, principalType, principalID).
			Update("update_by", op).Error
	}
	res := db.Unscoped().Where("id = ? AND principal_type = ? AND principal_id = ?", deviceID, principalType, principalID).
		Delete(&UserDevice{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func RevokeAllUserDevices(db *gorm.DB, principalType string, principalID uint) error {
	return db.Model(&UserDevice{}).
		Where("principal_type = ? AND principal_id = ?", principalType, principalID).
		Updates(map[string]any{
			"session_active":           false,
			"session_id":               "",
			"session_issued_at":        nil,
			"session_last_activity_at": nil,
		}).Error
}

func GetUserDeviceByID(db *gorm.DB, id uint) (UserDevice, error) {
	var row UserDevice
	err := db.First(&row, id).Error
	return row, err
}

func TouchUserDeviceSessionActivity(db *gorm.DB, deviceRecordID uint, sessionID string, at time.Time) error {
	if deviceRecordID == 0 || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return db.Model(&UserDevice{}).
		Where("id = ? AND session_id = ? AND session_active = ?", deviceRecordID, sessionID, true).
		Update("session_last_activity_at", at).Error
}

func DeactivateUserDeviceSession(db *gorm.DB, deviceRecordID uint, sessionID string) error {
	if deviceRecordID == 0 {
		return nil
	}
	q := db.Model(&UserDevice{}).Where("id = ?", deviceRecordID)
	if strings.TrimSpace(sessionID) != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	return q.Updates(map[string]any{
		"session_active":           false,
		"session_id":               "",
		"session_issued_at":        nil,
		"session_last_activity_at": nil,
	}).Error
}
