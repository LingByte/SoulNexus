package svcmodels

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"time"

	auth "github.com/LingByte/SoulNexus/internal/models/auth"
	"gorm.io/gorm"
)

// AdminListCredentials returns paginated credentials for admin management.
func AdminListCredentials(db *gorm.DB, page, pageSize int, search, status string, userID uint) ([]auth.UserCredential, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	query := db.Model(&auth.UserCredential{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR api_key LIKE ? OR llm_provider LIKE ?", like, like, like)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	if userID > 0 {
		query = query.Where("created_by = ?", userID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var creds []auth.UserCredential
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&creds).Error; err != nil {
		return nil, 0, err
	}
	return creds, total, nil
}

// GetCredentialByID loads a credential by primary key.
func GetCredentialByID(db *gorm.DB, id uint) (*auth.UserCredential, error) {
	var cred auth.UserCredential
	if err := db.First(&cred, id).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

// AdminUpdateCredentialStatus updates credential status and optional quota fields.
func AdminUpdateCredentialStatus(db *gorm.DB, id uint, status, bannedReason string, expiresAt *string, tokenQuota, requestQuota *int64, useNativeQuota, unlimitedQuota *bool) (*auth.UserCredential, error) {
	var cred auth.UserCredential
	if err := db.First(&cred, id).Error; err != nil {
		return nil, err
	}
	statusVal := auth.CredentialStatus(strings.TrimSpace(status))
	switch statusVal {
	case auth.CredentialStatusActive, auth.CredentialStatusBanned, auth.CredentialStatusSuspended:
	default:
		return nil, gorm.ErrInvalidData
	}
	updateVals := map[string]any{"status": statusVal}
	switch statusVal {
	case auth.CredentialStatusActive:
		updateVals["banned_at"] = nil
		updateVals["banned_reason"] = ""
		updateVals["banned_by"] = nil
	case auth.CredentialStatusBanned:
		now := time.Now()
		updateVals["banned_at"] = &now
		updateVals["banned_reason"] = bannedReason
	}
	if expiresAt != nil {
		raw := strings.TrimSpace(*expiresAt)
		if raw == "" {
			updateVals["expires_at"] = nil
		} else {
			var parsed time.Time
			var parseErr error
			if strings.Contains(raw, "T") {
				parsed, parseErr = time.Parse(time.RFC3339, raw)
			} else {
				parsed, parseErr = time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
			}
			if parseErr != nil {
				return nil, parseErr
			}
			updateVals["expires_at"] = &parsed
		}
	}
	if tokenQuota != nil {
		if *tokenQuota < 0 {
			return nil, gorm.ErrInvalidData
		}
		updateVals["token_quota"] = *tokenQuota
	}
	if requestQuota != nil {
		if *requestQuota < 0 {
			return nil, gorm.ErrInvalidData
		}
		updateVals["request_quota"] = *requestQuota
	}
	if useNativeQuota != nil {
		updateVals["use_native_quota"] = *useNativeQuota
	}
	if unlimitedQuota != nil {
		updateVals["unlimited_quota"] = *unlimitedQuota
	}
	if err := db.Model(&cred).Updates(updateVals).Error; err != nil {
		return nil, err
	}
	if err := db.First(&cred, cred.ID).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

// AdminDeleteCredential removes a credential by id (admin).
func AdminDeleteCredential(db *gorm.DB, id uint) error {
	return db.Delete(&auth.UserCredential{}, id).Error
}
