package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
)

// UserDevice 用户设备表
type UserDevice struct {
	BaseModel
	UserID     uint      `gorm:"index;not null" json:"userId"`
	DeviceID   string    `gorm:"size:128;index;not null" json:"deviceId"`
	DeviceName string    `gorm:"size:128" json:"deviceName"`
	DeviceType string    `gorm:"size:64" json:"deviceType"`
	OS         string    `gorm:"size:64" json:"os"`
	Browser    string    `gorm:"size:64" json:"browser"`
	UserAgent  string    `gorm:"type:text" json:"userAgent"`
	IPAddress  string    `gorm:"size:128;index" json:"ipAddress"`
	Location   string    `gorm:"size:256" json:"location"`
	IsTrusted  bool      `gorm:"default:false" json:"isTrusted"`
	IsActive   bool      `gorm:"default:true" json:"isActive"`
	LastUsedAt time.Time `gorm:"index" json:"lastUsedAt"`
}

func (UserDevice) TableName() string {
	return constants.USER_DEVICE_TABLE_NAME
}

// LoginHistory 登录历史记录表（用于异地登录检测）
type LoginHistory struct {
	BaseModel
	UserID        uint   `gorm:"index;not null" json:"userId"`
	Email         string `gorm:"size:128;index" json:"email"`
	IPAddress     string `gorm:"size:128;index" json:"ipAddress"`
	Location      string `gorm:"size:256" json:"location"`
	Country       string `gorm:"size:64" json:"country"`
	City          string `gorm:"size:128" json:"city"`
	UserAgent     string `gorm:"type:text" json:"userAgent"`
	DeviceID      string `gorm:"size:128;index" json:"deviceId"`
	LoginType     string `gorm:"size:32" json:"loginType"`
	Success       bool   `gorm:"index" json:"success"`
	FailureReason string `gorm:"size:256" json:"failureReason"`
	IsSuspicious  bool   `gorm:"default:false;index" json:"isSuspicious"`
}

func (LoginHistory) TableName() string {
	return constants.LOGIN_HISTORY_TABLE_NAME
}

// AccountLock 账号锁定记录
type AccountLock struct {
	BaseModel
	UserID         uint      `gorm:"index;not null" json:"userId"`
	Email          string    `gorm:"size:128;index;not null" json:"email"` // 邮箱（用于未登录时的锁定）
	IPAddress      string    `gorm:"size:128;index" json:"ipAddress"`      // 锁定IP
	LockedAt       time.Time `gorm:"index" json:"lockedAt"`                // 锁定时间
	UnlockAt       time.Time `gorm:"index" json:"unlockAt"`                // 解锁时间
	Reason         string    `gorm:"size:256" json:"reason"`               // 锁定原因
	FailedAttempts int       `gorm:"default:0" json:"failedAttempts"`      // 失败次数
	IsActive       bool      `gorm:"default:true;index" json:"isActive"`   // 是否激活
}

func (AccountLock) TableName() string {
	return constants.ACCOUNT_LOCK_TABLE_NAME
}

// IsLocked 检查账号是否被锁定
func (al *AccountLock) IsLocked() bool {
	if !al.IsActive {
		return false
	}
	return time.Now().Before(al.UnlockAt)
}

// CreateOrUpdateAccountLock 创建或更新账号锁定记录
func CreateOrUpdateAccountLock(db *gorm.DB, email string, userID uint, ipAddress string, failedAttempts int) (*AccountLock, error) {
	var lock AccountLock
	query := db.Where("email = ? AND is_active = ?", email, true)
	if userID > 0 {
		query = query.Or("user_id = ? AND is_active = ?", userID, true)
	}
	err := query.First(&lock).Error
	lockTime := 30 * time.Minute // 锁定30分钟
	if err == gorm.ErrRecordNotFound {
		lock = AccountLock{
			Email:          email,
			UserID:         userID,
			IPAddress:      ipAddress,
			LockedAt:       time.Now(),
			UnlockAt:       time.Now().Add(lockTime),
			FailedAttempts: failedAttempts,
			Reason:         "Too many failed login attempts",
			IsActive:       true,
		}
		err = db.Create(&lock).Error
	} else if err == nil {
		// 更新现有锁定记录
		lock.FailedAttempts = failedAttempts
		lock.UnlockAt = time.Now().Add(lockTime)
		lock.IPAddress = ipAddress
		lock.UpdatedAt = time.Now()
		err = db.Save(&lock).Error
	}

	return &lock, err
}

// GetAccountLock 获取账号锁定记录
func GetAccountLock(db *gorm.DB, email string, userID uint) (*AccountLock, error) {
	var lock AccountLock

	query := db.Where("is_active = ?", true)
	if email != "" {
		query = query.Where("email = ?", email)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.First(&lock).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &lock, err
}

// UnlockAccount 解锁账号
func UnlockAccount(db *gorm.DB, email string, userID uint) error {
	query := db.Model(&AccountLock{}).Where("is_active = ?", true)
	if email != "" {
		query = query.Where("email = ?", email)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	return query.Update("is_active", false).Error
}

// RecordLoginHistory 记录登录历史
func RecordLoginHistory(db *gorm.DB, userID uint, email, ipAddress, location, country, city, userAgent, deviceID, loginType string, success bool, failureReason string, isSuspicious bool) error {
	history := LoginHistory{
		UserID:        userID,
		Email:         email,
		IPAddress:     ipAddress,
		Location:      location,
		Country:       country,
		City:          city,
		UserAgent:     userAgent,
		DeviceID:      deviceID,
		LoginType:     loginType,
		Success:       success,
		FailureReason: failureReason,
		IsSuspicious:  isSuspicious,
	}

	return db.Create(&history).Error
}

// GetRecentLoginLocations 获取最近的登录位置（用于异地登录检测）
func GetRecentLoginLocations(db *gorm.DB, userID uint, limit int) ([]LoginHistory, error) {
	var histories []LoginHistory
	err := db.Where("user_id = ? AND success = ?", userID, true).
		Order("created_at DESC").
		Limit(limit).
		Find(&histories).Error
	return histories, err
}

// CreateOrUpdateUserDevice 创建或更新用户设备
func CreateOrUpdateUserDevice(db *gorm.DB, userID uint, deviceID, deviceName, deviceType, os, browser, userAgent, ipAddress, location string) (*UserDevice, error) {
	var device UserDevice
	err := db.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&device).Error
	if err == gorm.ErrRecordNotFound {
		device = UserDevice{
			UserID:     userID,
			DeviceID:   deviceID,
			DeviceName: deviceName,
			DeviceType: deviceType,
			OS:         os,
			Browser:    browser,
			UserAgent:  userAgent,
			IPAddress:  ipAddress,
			Location:   location,
			IsTrusted:  false,
			IsActive:   true,
			LastUsedAt: time.Now(),
		}
		err = db.Create(&device).Error
	} else if err == nil {
		device.DeviceName = deviceName
		device.OS = os
		device.Browser = browser
		device.UserAgent = userAgent
		device.IPAddress = ipAddress
		device.Location = location
		device.LastUsedAt = time.Now()
		device.UpdatedAt = time.Now()
		err = db.Save(&device).Error
	}

	return &device, err
}

// DeleteUserDevice 删除用户设备
func DeleteUserDevice(db *gorm.DB, userID uint, deviceID string) error {
	return db.Model(&UserDevice{}).Where("user_id = ? AND device_id = ?", userID, deviceID).
		Update("is_active", false).Error
}

// TrustUserDevice 信任设备
func TrustUserDevice(db *gorm.DB, userID uint, deviceID string) error {
	// 先检查设备是否存在
	device, err := GetUserDevice(db, userID, deviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 如果设备不存在，先创建一个基本记录（这种情况理论上不应该发生，但为了健壮性）
	if device == nil {
		device = &UserDevice{
			UserID:     userID,
			DeviceID:   deviceID,
			DeviceName: "Unknown Device",
			DeviceType: "unknown",
			OS:         "unknown",
			Browser:    "unknown",
			IsTrusted:  true, // 直接设置为信任
			IsActive:   true,
			LastUsedAt: time.Now(),
		}
		if err := db.Create(device).Error; err != nil {
			return err
		}
		return nil
	}

	// 如果设备存在，更新信任状态
	return db.Model(&UserDevice{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Update("is_trusted", true).Error
}

// UntrustUserDevice 取消信任设备
func UntrustUserDevice(db *gorm.DB, userID uint, deviceID string) error {
	return db.Model(&UserDevice{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Update("is_trusted", false).Error
}

// GetUserDevice 获取用户的特定设备
func GetUserDevice(db *gorm.DB, userID uint, deviceID string) (*UserDevice, error) {
	var device UserDevice
	err := db.Where("user_id = ? AND device_id = ? AND is_active = ?", userID, deviceID, true).
		First(&device).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &device, err
}

// CheckDeviceTrust 检查设备是否被信任
func CheckDeviceTrust(db *gorm.DB, userID uint, deviceID string) (bool, error) {
	device, err := GetUserDevice(db, userID, deviceID)
	if err != nil {
		return false, err
	}
	if device == nil {
		return false, nil
	}
	return device.IsTrusted, nil
}

// GetUserLoginDevices 获取用户的登录设备列表
func GetUserLoginDevices(db *gorm.DB, userID uint) ([]UserDevice, error) {
	var devices []UserDevice
	err := db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("last_used_at DESC").
		Find(&devices).Error
	return devices, err
}
