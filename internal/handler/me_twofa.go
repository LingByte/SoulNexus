// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 用户自助 2FA 恢复路径：备用码生成/校验、锁定状态查询。
// 与既有 /api/auth/* 上的 TOTP setup/enable/disable 配合：
//   - 启用 TOTP 之后，前端引导用户调用 POST /api/me/twofa/backup-codes/regenerate 生成 8 个备用码（仅一次明文返回）。
//   - 登录侧若 TOTP 校验失败，可弹出"使用备用码"流程：POST /api/me/twofa/backup-codes/use { code }。
//   - 失败 5 次锁 5 分钟由 models.IncrementTwoFAFailedAttempts/ResetTwoFAFailedAttempts 管理。

package handlers

import (
	"errors"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// twoFAHelperGetOrSync 取 TwoFA 行；若不存在但 User 上有 secret，迁移到 TwoFA 表。
func twoFAHelperGetOrSync(h *Handlers, user *models.User) (*models.TwoFA, error) {
	t, err := models.GetTwoFAByUserID(h.db, user.ID)
	if err != nil {
		return nil, err
	}
	if t != nil {
		return t, nil
	}
	if user.TwoFactorSecret == "" {
		return nil, nil
	}
	row, err := models.UpsertTwoFASecret(h.db, user.ID, user.TwoFactorSecret)
	if err != nil {
		return nil, err
	}
	if user.TwoFactorEnabled {
		_ = models.EnableTwoFA(h.db, user.ID)
		row.IsEnabled = true
	}
	return row, nil
}

// handleMeTwoFAStatus GET /api/me/twofa/status — 含锁定与剩余备用码数量。
func (h *Handlers) handleMeTwoFAStatus(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	t, err := twoFAHelperGetOrSync(h, user)
	if err != nil {
		response.Fail(c, "load twofa failed", err)
		return
	}
	out := gin.H{
		"enabled":           user.TwoFactorEnabled || (t != nil && t.IsEnabled),
		"has_secret":        user.TwoFactorSecret != "" || (t != nil && t.Secret != ""),
		"locked":            t != nil && t.IsTwoFALocked(),
		"failed_attempts":   0,
		"backup_codes_left": int64(0),
	}
	if t != nil {
		out["failed_attempts"] = t.FailedAttempts
		if t.LockedUntil != nil {
			out["locked_until"] = t.LockedUntil
		}
		if cnt, err := models.CountUnusedBackupCodes(h.db, user.ID); err == nil {
			out["backup_codes_left"] = cnt
		}
	}
	response.Success(c, "twofa status", out)
}

// handleMeTwoFABackupCodesRegenerate POST /api/me/twofa/backup-codes/regenerate
//
// 要求当前 TOTP code 校验通过后，生成 8 个新的备用码（旧的全部失效）。
// 返回明文一次（"XXXXX-XXXXX"）；DB 仅存 bcrypt hash。
func (h *Handlers) handleMeTwoFABackupCodesRegenerate(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	t, err := twoFAHelperGetOrSync(h, user)
	if err != nil || t == nil || !t.IsEnabled {
		response.Fail(c, "two-factor not enabled", errors.New("two-factor not enabled"))
		return
	}
	ok, err := models.ValidateTOTPAndUpdateUsage(h.db, t, req.Code)
	if !ok {
		msg := "verification failed"
		if err != nil {
			msg = err.Error()
		}
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New(msg))
		return
	}
	codes, err := models.GenerateBackupCodes(h.db, user.ID, 8)
	if err != nil {
		response.Fail(c, "generate backup codes failed", err)
		return
	}
	response.Success(c, "backup codes generated", gin.H{
		"codes":   codes,
		"warning": "请立即妥善保存；离开页面后将无法再次查看明文。每码仅可使用一次。",
	})
}

// handleMeTwoFABackupCodeUse POST /api/me/twofa/backup-codes/use
//
// 用户在 TOTP 设备丢失时使用备用码登录的辅助接口（要求当前会话已通过密码校验）。
// 校验通过后：
//   1. 标记该备用码为已使用；
//   2. 重置 TwoFA.FailedAttempts；
//   3. 不会自动停用 2FA（用户应在登录后立即在设置页重新绑定 TOTP）。
func (h *Handlers) handleMeTwoFABackupCodeUse(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	t, err := twoFAHelperGetOrSync(h, user)
	if err != nil || t == nil {
		response.Fail(c, "two-factor not enabled", errors.New("two-factor not enabled"))
		return
	}
	ok, err := models.ValidateBackupCodeAndConsume(h.db, t, req.Code)
	if !ok {
		msg := "backup code rejected"
		if err != nil {
			msg = err.Error()
		}
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New(msg))
		return
	}
	left, _ := models.CountUnusedBackupCodes(h.db, user.ID)
	response.Success(c, "backup code accepted", gin.H{"remaining": left})
}

// handleMeTwoFAReset POST /api/me/twofa/reset
//
// 紧急重置：用户提供有效备用码并确认后，禁用 2FA（清空 secret 与备用码）。
// 用户在登录后才可调用；备用码消费一次即生效。
func (h *Handlers) handleMeTwoFAReset(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	var req struct {
		BackupCode string `json:"backup_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	t, err := twoFAHelperGetOrSync(h, user)
	if err != nil || t == nil {
		response.Fail(c, "two-factor not enabled", errors.New("two-factor not enabled"))
		return
	}
	ok, err := models.ValidateBackupCodeAndConsume(h.db, t, req.BackupCode)
	if !ok {
		msg := "backup code rejected"
		if err != nil {
			msg = err.Error()
		}
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New(msg))
		return
	}
	if err := models.DisableTwoFA(h.db, user.ID); err != nil {
		response.Fail(c, "disable twofa failed", err)
		return
	}
	// 同步 User 兼容字段。
	_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"two_factor_enabled": false,
		"two_factor_secret":  "",
	}).Error
	response.Success(c, "twofa disabled", nil)
}
