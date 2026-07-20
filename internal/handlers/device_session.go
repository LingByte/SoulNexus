package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var errInvalidDeviceCode = errors.New("invalid device verification code")
var errInvalidVoiceprintVerify = errors.New("invalid voiceprint verification")

type deviceSessionInput struct {
	PrincipalType            string
	PrincipalID              uint
	TenantID                 uint
	Email                    string
	DisplayName              string
	DeviceKey                string
	DeviceCode               string
	VoiceprintAudioBase64    string
	UserAgent                string
	ClientIP                 string
	AutoTrust                bool
	RequireDeviceVerify      bool
	TrustDeviceLoginEnabled  bool
	TrustDeviceFor7Days      bool
	RequireRemoteLoginVerify bool
	KnownLoginCities         []string
}

type deviceSessionResult struct {
	NeedsDeviceVerify bool
	NeedsRemoteVerify bool
	DeviceRecordID    uint
	SessionID         string
}

func (h *Handlers) resolveDeviceSession(c *gin.Context, in deviceSessionInput) (deviceSessionResult, error) {
	deviceKey := strings.TrimSpace(in.DeviceKey)
	if deviceKey == "" {
		deviceKey = uuid.NewString()
	}
	ua := strings.TrimSpace(in.UserAgent)
	if ua == "" {
		ua = c.GetHeader("User-Agent")
	}
	category := utils.CategoryFromUserAgent(ua)
	limitCategory := utils.LoginLimitCategory(ua)
	displayName := utils.DisplayNameFromUserAgent(ua)
	now := time.Now()
	currentCity := utils.LoginCityFromIP(in.ClientIP)

	row, err := models.GetUserDeviceByKeyUnscoped(h.db, in.PrincipalType, in.PrincipalID, deviceKey)
	isNewDevice := errors.Is(err, gorm.ErrRecordNotFound)
	wasDeleted := false
	if err == nil && row.DeletedAt.Valid {
		wasDeleted = true
		isNewDevice = false
	}
	if err != nil && !isNewDevice {
		return deviceSessionResult{}, err
	}

	withinTrust := in.TrustDeviceLoginEnabled && models.DeviceWithinTrustWindow(row, now)
	needsDeviceVerify := !in.AutoTrust && in.RequireDeviceVerify && !withinTrust
	needsRemoteVerify := !in.AutoTrust && utils.NeedsRemoteLoginVerify(in.RequireRemoteLoginVerify, in.KnownLoginCities, currentCity)
	needsVerify := needsDeviceVerify || needsRemoteVerify

	if needsVerify {
		if strings.TrimSpace(in.DeviceCode) == "" {
			if audio, err := decodeLoginVoiceprintAudio(in.VoiceprintAudioBase64); err == nil && len(audio) > 0 {
				if in.TenantID > 0 && in.PrincipalID > 0 {
					ok, verr := h.verifyUserVoiceprintAudio(in.TenantID, in.PrincipalID, audio, 0)
					if verr != nil {
						return deviceSessionResult{}, errInvalidVoiceprintVerify
					}
					if !ok {
						return deviceSessionResult{}, errInvalidVoiceprintVerify
					}
				} else {
					return deviceSessionResult{}, errInvalidVoiceprintVerify
				}
			} else {
				return deviceSessionResult{NeedsDeviceVerify: needsDeviceVerify, NeedsRemoteVerify: needsRemoteVerify}, nil
			}
		} else if !verifyEmailCode(emailCodeDeviceVerify, in.Email, in.DeviceCode) {
			return deviceSessionResult{}, errInvalidDeviceCode
		}
	}

	trustedNow := in.TrustDeviceLoginEnabled && in.TrustDeviceFor7Days && needsVerify

	if isNewDevice {
		row = models.UserDevice{
			PrincipalType: in.PrincipalType,
			PrincipalID:   in.PrincipalID,
			DeviceKey:     deviceKey,
			Category:      category,
			LimitCategory: limitCategory,
			DisplayName:   displayName,
			UserAgent:     ua,
			LastIP:        in.ClientIP,
			LastLoginCity: currentCity,
			IsTrusted:     trustedNow,
		}
		row.SetCreateInfo(deviceAuditOperator(in))
		if trustedNow {
			row.TrustedAt = &now
			until := now.Add(time.Duration(utils.TrustDeviceLoginDays) * 24 * time.Hour)
			row.TrustedUntil = &until
		}
	} else {
		row.Category = category
		row.LimitCategory = limitCategory
		row.DisplayName = displayName
		row.UserAgent = ua
		row.LastIP = in.ClientIP
		row.LastLoginCity = currentCity
		row.SetUpdateInfo(deviceAuditOperator(in))
		if trustedNow {
			row.IsTrusted = true
			row.TrustedAt = &now
			until := now.Add(time.Duration(utils.TrustDeviceLoginDays) * 24 * time.Hour)
			row.TrustedUntil = &until
		}
	}

	if !in.TrustDeviceLoginEnabled {
		row.TrustedUntil = nil
	}

	sessionID := uuid.NewString()
	row.SessionID = sessionID
	row.SessionActive = true
	row.LastLoginAt = &now
	row.SessionIssuedAt = &now
	row.SessionLastActivityAt = &now
	row.LastLoginCity = currentCity

	if limitCategory == utils.CategoryDesktop {
		_ = models.DeactivateDesktopSessions(h.db, in.PrincipalType, in.PrincipalID, 0)
	}

	if isNewDevice {
		if err := h.db.Create(&row).Error; err != nil {
			return deviceSessionResult{}, err
		}
	} else {
		if wasDeleted {
			row.DeletedAt = gorm.DeletedAt{}
		}
		db := h.db
		if wasDeleted {
			db = db.Unscoped()
		}
		if err := db.Save(&row).Error; err != nil {
			return deviceSessionResult{}, err
		}
	}

	if limitCategory == utils.CategoryDesktop {
		_ = models.DeactivateDesktopSessions(h.db, in.PrincipalType, in.PrincipalID, row.ID)
		_ = h.db.Model(&models.UserDevice{}).Where("id = ?", row.ID).Updates(map[string]any{
			"session_id":               sessionID,
			"session_active":           true,
			"session_issued_at":        now,
			"session_last_activity_at": now,
		}).Error
	}

	if isNewDevice || needsVerify || wasDeleted {
		username := strings.TrimSpace(in.DisplayName)
		if username == "" {
			username = in.Email
		}
		location := currentCity
		if location == "" {
			_, _, fullLoc, _ := utils.GetIPLocation(in.ClientIP)
			location = strings.TrimSpace(fullLoc)
		}
		if location == "" {
			location = in.ClientIP
		}
		utils.Sig().Emit(constants.SigMailNewDeviceLogin, nil, constants.MailNewDeviceLoginPayload{
			UserID:       in.PrincipalID,
			Email:        in.Email,
			Username:     username,
			LoginTime:    timeutil.FormatLocaleDateTime(now),
			ClientIP:     in.ClientIP,
			Location:     location,
			DeviceType:   category,
			IsSuspicious: needsRemoteVerify,
		}, h.db)
	}

	return deviceSessionResult{
		DeviceRecordID: row.ID,
		SessionID:      sessionID,
	}, nil
}

func deviceAuditOperator(in deviceSessionInput) string {
	if email := strings.TrimSpace(in.Email); email != "" {
		return email
	}
	return "system:device"
}
