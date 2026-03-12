package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DeviceErrorLog 设备错误日志表
type DeviceErrorLog struct {
	BaseModel
	DeviceID   string         `json:"deviceId" gorm:"size:64;index;not null"` // 设备ID (MAC地址)
	MacAddress string         `json:"macAddress" gorm:"size:64;index"`        // MAC地址
	ErrorType  string         `json:"errorType" gorm:"size:64;index"`         // 错误类型
	ErrorLevel string         `json:"errorLevel" gorm:"size:16;index"`        // 错误级别 (INFO, WARN, ERROR, FATAL)
	ErrorCode  string         `json:"errorCode" gorm:"size:32"`               // 错误代码
	ErrorMsg   string         `json:"errorMsg" gorm:"type:text"`              // 错误消息
	StackTrace string         `json:"stackTrace" gorm:"type:text"`            // 堆栈跟踪
	Context    datatypes.JSON `json:"context" gorm:"type:json"`               // 错误上下文（JSON格式）
	Resolved   bool           `json:"resolved" gorm:"default:false;index"`    // 是否已解决
	ResolvedAt time.Time      `json:"resolvedAt,omitempty"`                   // 解决时间
	ResolvedBy string         `json:"resolvedBy" gorm:"size:128"`             // 解决人
}

func (DeviceErrorLog) TableName() string {
	return constants.DEVICE_ERROR_LOG_TABLE_NAME
}

// LogDeviceError 记录设备错误
func LogDeviceError(db *gorm.DB, deviceID, macAddress, errorType, errorLevel, errorCode, errorMsg, stackTrace, context string) error {
	// 将context字符串转换为有效的JSON
	var contextJSON datatypes.JSON
	if context != "" {
		// 尝试将字符串作为JSON对象包装
		contextData := map[string]interface{}{
			"message": context,
		}
		if jsonBytes, err := json.Marshal(contextData); err == nil {
			contextJSON = jsonBytes
		} else {
			// 如果转换失败，创建一个包含错误信息的JSON
			fallbackData := map[string]interface{}{
				"message": context,
				"error":   "failed to parse context",
			}
			if jsonBytes, err := json.Marshal(fallbackData); err == nil {
				contextJSON = jsonBytes
			}
		}
	} else {
		// 如果context为空，创建一个空的JSON对象
		contextJSON = datatypes.JSON(`{}`)
	}

	errorLog := DeviceErrorLog{
		DeviceID:   deviceID,
		MacAddress: macAddress,
		ErrorType:  errorType,
		ErrorLevel: errorLevel,
		ErrorCode:  errorCode,
		ErrorMsg:   errorMsg,
		StackTrace: stackTrace,
		Context:    contextJSON,
	}

	now := time.Now()
	db.Model(&Device{}).Where("mac_address = ?", macAddress).Updates(map[string]interface{}{
		"error_count":   gorm.Expr("error_count + 1"),
		"last_error":    errorMsg,
		"last_error_at": &now,
	})
	return db.Create(&errorLog).Error
}

// GetDeviceErrorLogs 获取设备错误日志列表
func GetDeviceErrorLogs(db *gorm.DB, macAddress string, limit, offset int) ([]DeviceErrorLog, int64, error) {
	var logs []DeviceErrorLog
	var total int64

	query := db.Where("mac_address = ?", macAddress)
	query.Model(&DeviceErrorLog{}).Count(&total)

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

// GetDeviceErrorLogsByType 按错误类型获取设备错误日志
func GetDeviceErrorLogsByType(db *gorm.DB, macAddress, errorType string, limit, offset int) ([]DeviceErrorLog, int64, error) {
	var logs []DeviceErrorLog
	var total int64

	query := db.Where("mac_address = ? AND error_type = ?", macAddress, errorType)
	query.Model(&DeviceErrorLog{}).Count(&total)

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

// GetDeviceErrorLogsByLevel 按错误级别获取设备错误日志
func GetDeviceErrorLogsByLevel(db *gorm.DB, macAddress, errorLevel string, limit, offset int) ([]DeviceErrorLog, int64, error) {
	var logs []DeviceErrorLog
	var total int64

	query := db.Where("mac_address = ? AND error_level = ?", macAddress, errorLevel)
	query.Model(&DeviceErrorLog{}).Count(&total)

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

// ResolveDeviceError 标记设备错误为已解决
func ResolveDeviceError(db *gorm.DB, errorID uint, resolvedBy string) error {
	now := time.Now()
	return db.Model(&DeviceErrorLog{}).Where("id = ?", errorID).Updates(map[string]interface{}{
		"resolved":    true,
		"resolved_at": now,
		"resolved_by": resolvedBy,
	}).Error
}

// GetUnresolvedErrorCount 获取未解决的错误数量
func GetUnresolvedErrorCount(db *gorm.DB, macAddress string) (int64, error) {
	var count int64
	err := db.Model(&DeviceErrorLog{}).Where("mac_address = ? AND resolved = ?", macAddress, false).Count(&count).Error
	return count, err
}
