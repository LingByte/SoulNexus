package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const sessionActivityTouchInterval = 5 * time.Minute

func validateBearerDeviceSession(c *gin.Context) bool {
	sid := AuthSessionID(c)
	did := AuthDeviceRecordID(c)
	if sid == "" || did == 0 {
		return true
	}
	dbIface, exists := c.Get(constants.DbField)
	db, ok := dbIface.(*gorm.DB)
	if !exists || !ok || db == nil {
		return true
	}
	active, err := models.IsUserDeviceSessionActive(db, did, sid)
	if err != nil || !active {
		return false
	}
	row, err := models.GetUserDeviceByID(db, did)
	if err != nil {
		return false
	}
	idleH, maxH := sessionTimeoutHours(c, db)
	now := time.Now()
	if row.SessionIssuedAt != nil && now.After(row.SessionIssuedAt.Add(time.Duration(maxH)*time.Hour)) {
		_ = models.DeactivateUserDeviceSession(db, did, sid)
		return false
	}
	lastActivity := row.SessionLastActivityAt
	if lastActivity == nil {
		lastActivity = row.SessionIssuedAt
	}
	if lastActivity != nil && now.After(lastActivity.Add(time.Duration(idleH)*time.Hour)) {
		_ = models.DeactivateUserDeviceSession(db, did, sid)
		return false
	}
	if lastActivity == nil || now.Sub(*lastActivity) >= sessionActivityTouchInterval {
		_ = models.TouchUserDeviceSessionActivity(db, did, sid, now)
	}
	return true
}

func sessionTimeoutHours(c *gin.Context, db *gorm.DB) (idle, max int) {
	idle = utils.DefaultSessionIdleTimeoutHours
	max = utils.DefaultSessionMaxLifetimeHours
	if uid := AuthUserID(c); uid > 0 {
		var u models.TenantUser
		if err := db.Select("session_idle_timeout_hours", "session_max_lifetime_hours").First(&u, uid).Error; err == nil {
			return utils.SessionIdleTimeout(u.SessionIdleTimeoutHours), utils.SessionMaxLifetime(u.SessionIdleTimeoutHours, u.SessionMaxLifetimeHours)
		}
	}
	if aid := AuthPlatformAdminID(c); aid > 0 {
		var a models.PlatformAdmin
		if err := db.Select("session_idle_timeout_hours", "session_max_lifetime_hours").First(&a, aid).Error; err == nil {
			return utils.SessionIdleTimeout(a.SessionIdleTimeoutHours), utils.SessionMaxLifetime(a.SessionIdleTimeoutHours, a.SessionMaxLifetimeHours)
		}
	}
	return idle, max
}

func abortSessionRevoked(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code": 401,
		"msg":  i18n.TGin(c, i18n.KeyAuthSessionRevoked),
		"data": gin.H{"sessionRevoked": true},
	})
}
