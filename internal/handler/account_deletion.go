package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// loginBlockedByAccountDeletion 冷静期内阻止签发新的登录态（密码 / 邮箱登录等）。已响应则返回 true。
func (h *Handlers) loginBlockedByAccountDeletion(c *gin.Context, db *gorm.DB, user *models.User) bool {
	if user == nil {
		return false
	}
	var fresh models.User
	if err := db.Where("id = ? AND is_deleted = ?", user.ID, models.SoftDeleteStatusActive).First(&fresh).Error; err != nil {
		return false
	}
	if !models.AccountDeletionPending(&fresh) {
		return false
	}
	effective := ""
	if fresh.AccountDeletionEffectiveAt != nil {
		effective = fresh.AccountDeletionEffectiveAt.UTC().Format(time.RFC3339)
	}
	response.Success(c, "账号处于注销冷静期，暂无法登录 SoulNexus。请使用「撤销注销」完成验证后恢复账号，或等待冷静期结束后账号将被永久注销。", gin.H{
		"accountDeletionPending":     true,
		"accountDeletionEffectiveAt": effective,
		"email":                      fresh.Email,
	})
	return true
}

func verifyUserLoginPassword(user *models.User, password string) bool {
	if user == nil || password == "" {
		return false
	}
	if strings.Contains(password, ":") && len(strings.Split(password, ":")) == 4 {
		return models.VerifyEncryptedPassword(password, user.Password)
	}
	return models.CheckPassword(user, password)
}

func (h *Handlers) accountDeletionRiskFlags(c *gin.Context, db *gorm.DB, user *models.User) (accountLocked, remoteLoginRisk, recentSuspicious bool, err error) {
	lock, err := models.GetAccountLock(db, user.Email, user.ID)
	if err != nil {
		return false, false, false, err
	}
	if lock != nil && lock.IsLocked() {
		accountLocked = true
	}

	clientIP := c.ClientIP()
	country, _, location := "Unknown", "Unknown", "Unknown"
	if h.ipLocationService != nil {
		country, _, location, _ = h.ipLocationService.GetLocation(clientIP)
	}
	if utils.GlobalLoginSecurityManager != nil {
		getLocationsFunc := func(db *gorm.DB, userID uint, limit int) ([]utils.LoginLocation, error) {
			histories, e := models.GetRecentLoginLocations(db, userID, limit)
			if e != nil {
				return nil, e
			}
			locations := make([]utils.LoginLocation, len(histories))
			for i, hi := range histories {
				locations[i] = utils.LoginLocation{Country: hi.Country, City: hi.City}
			}
			return locations, nil
		}
		remoteLoginRisk, _ = utils.GlobalLoginSecurityManager.DetectSuspiciousLogin(db, user.ID, clientIP, location, country, getLocationsFunc)
	}

	recentSuspicious, err = models.HasRecentSuspiciousLogins(db, user.ID, 14*24*time.Hour)
	if err != nil {
		return false, false, false, err
	}
	return accountLocked, remoteLoginRisk, recentSuspicious, nil
}

// handleAccountDeletionEligibility 注销前置：状态、风控、第三方绑定说明（文案由前端展示时可结合 warnings）。
func (h *Handlers) handleAccountDeletionEligibility(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", errors.New("unauthorized"))
		return
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	var fresh models.User
	if err := db.Where("id = ? AND is_deleted = ?", user.ID, models.SoftDeleteStatusActive).First(&fresh).Error; err != nil {
		response.Fail(c, "用户不存在", err)
		return
	}

	locked, remote, suspicious, err := h.accountDeletionRiskFlags(c, db, &fresh)
	if err != nil {
		response.Fail(c, "风险检测失败", err)
		return
	}
	reasons := models.AccountDeletionEligibilityReasons(db, &fresh, locked, remote, suspicious)
	gh, wx := models.ThirdPartyBindings(&fresh)

	warnings := []string{
		"注销完成后数据永久不可恢复。",
		"账号内权益、配额与订阅关联将清零或解除。",
		"注销后无法通过原邮箱或第三方登录找回该账号。",
	}

	response.Success(c, "ok", gin.H{
		"eligible":                   len(reasons) == 0,
		"reasons":                    reasons,
		"githubBound":                gh,
		"wechatBound":                wx,
		"accountLocked":              locked,
		"remoteLoginRisk":            remote,
		"recentSuspiciousLogins":     suspicious,
		"warnings":                   warnings,
		"cooldownHours":              int(models.DefaultAccountDeletionCooldown().Hours()),
		"deletionPending":            models.AccountDeletionPending(&fresh),
		"accountDeletionEffectiveAt": fresh.AccountDeletionEffectiveAt,
		"accountDeletionRequestedAt": fresh.AccountDeletionRequestedAt,
	})
}

func (h *Handlers) handleAccountDeletionSendEmailCode(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", errors.New("unauthorized"))
		return
	}
	cacheKey := user.Email + ":account_deletion"
	if _, ok := utils.GlobalCache.Get(cacheKey); ok {
		response.Fail(c, "验证码发送过于频繁，请稍后再试", errors.New("rate limited"))
		return
	}
	code := utils.RandNumberText(6)
	utils.GlobalCache.Add(cacheKey, code)

	go func() {
		mailer := notification.NewMailer(h.db, 0, user.ID, "")
		if err := mailer.SendVerificationCode(user.Email, code); err != nil {
			logger.Error("account deletion email code failed", zap.Error(err), zap.Uint("userID", user.ID))
		}
	}()

	response.Success(c, "验证码已发送，请在有效期内完成验证", nil)
}

// handleAccountDeletionRequest 密码 + 邮箱验证码 + 风险与解绑校验通过后，进入三天冷静期。
func (h *Handlers) handleAccountDeletionRequest(c *gin.Context) {
	var form struct {
		Password                string `json:"password" binding:"required"`
		EmailCode               string `json:"emailCode" binding:"required"`
		AcknowledgeConsequences bool   `json:"acknowledgeConsequences"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "参数无效", err)
		return
	}
	if !form.AcknowledgeConsequences {
		response.Fail(c, "请确认已了解注销后果（acknowledgeConsequences 为 true）", errors.New("acknowledgement required"))
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", errors.New("unauthorized"))
		return
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	var fresh models.User
	if err := db.Where("id = ? AND is_deleted = ?", user.ID, models.SoftDeleteStatusActive).First(&fresh).Error; err != nil {
		response.Fail(c, "用户不存在", err)
		return
	}

	if models.AccountDeletionPending(&fresh) {
		response.Fail(c, "已处于注销冷静期", errors.New("already pending"))
		return
	}

	cacheKey := fresh.Email + ":account_deletion"
	cached, ok := utils.GlobalCache.Get(cacheKey)
	if !ok || cached != form.EmailCode {
		response.Fail(c, "邮箱验证码无效或已过期", errors.New("invalid email code"))
		return
	}
	utils.GlobalCache.Remove(cacheKey)

	if !verifyUserLoginPassword(&fresh, form.Password) {
		response.Fail(c, "密码错误", errors.New("invalid password"))
		return
	}

	locked, remote, suspicious, err := h.accountDeletionRiskFlags(c, db, &fresh)
	if err != nil {
		response.Fail(c, "风险检测失败", err)
		return
	}
	reasons := models.AccountDeletionEligibilityReasons(db, &fresh, locked, remote, suspicious)
	if len(reasons) > 0 {
		response.Fail(c, "当前不满足注销条件", gin.H{"reasons": reasons})
		return
	}

	operator := strings.TrimSpace(fresh.Email)
	if operator == "" {
		operator = "self"
	}
	if err := models.ScheduleAccountDeletion(db, fresh.ID, operator); err != nil {
		response.Fail(c, "申请注销失败", err)
		return
	}

	_ = db.Where("id = ?", fresh.ID).First(&fresh).Error
	response.Success(c, "已进入注销冷静期，期间可随时撤回", gin.H{
		"accountDeletionRequestedAt": fresh.AccountDeletionRequestedAt,
		"accountDeletionEffectiveAt": fresh.AccountDeletionEffectiveAt,
	})
}

func (h *Handlers) handleAccountDeletionCancel(c *gin.Context) {
	var form struct {
		EmailCode string `json:"emailCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "参数无效", err)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未登录", errors.New("unauthorized"))
		return
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	var fresh models.User
	if err := db.Where("id = ? AND is_deleted = ?", user.ID, models.SoftDeleteStatusActive).First(&fresh).Error; err != nil {
		response.Fail(c, "用户不存在", err)
		return
	}
	if !models.AccountDeletionPending(&fresh) {
		response.Fail(c, "当前没有进行中的注销申请", errors.New("not pending"))
		return
	}

	cacheKey := fresh.Email + ":account_deletion_cancel"
	cached, ok := utils.GlobalCache.Get(cacheKey)
	if !ok || cached != form.EmailCode {
		response.Fail(c, "邮箱验证码无效或已过期", errors.New("invalid email code"))
		return
	}
	utils.GlobalCache.Remove(cacheKey)

	operator := strings.TrimSpace(fresh.Email)
	if operator == "" {
		operator = "self"
	}
	if err := models.CancelAccountDeletion(db, fresh.ID, operator); err != nil {
		response.Fail(c, "撤回失败", err)
		return
	}
	response.Success(c, "已撤回注销申请，账号恢复正常", nil)
}

// handleAccountDeletionSendCancelCode 未登录也可调用：向邮箱发送用于撤销注销的验证码。
func (h *Handlers) handleAccountDeletionSendCancelCode(c *gin.Context) {
	var form struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(form.Email))
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user, err := models.GetUserByEmail(db, email)
	if err != nil || user == nil {
		response.Success(c, "若该邮箱存在冷静期内的注销申请，将发送验证码", nil)
		return
	}
	var fresh models.User
	if err := db.Where("id = ? AND is_deleted = ?", user.ID, models.SoftDeleteStatusActive).First(&fresh).Error; err != nil {
		response.Success(c, "若该邮箱存在冷静期内的注销申请，将发送验证码", nil)
		return
	}
	if !models.AccountDeletionPending(&fresh) {
		response.Success(c, "若该邮箱存在冷静期内的注销申请，将发送验证码", nil)
		return
	}
	cacheKey := fresh.Email + ":account_deletion_cancel"
	if _, ok := utils.GlobalCache.Get(cacheKey); ok {
		response.Fail(c, "验证码发送过于频繁，请稍后再试", errors.New("rate limited"))
		return
	}
	code := utils.RandNumberText(6)
	utils.GlobalCache.Add(cacheKey, code)
	go func() {
		mailer := notification.NewMailer(h.db, 0, fresh.ID, "")
		if err := mailer.SendVerificationCode(fresh.Email, code); err != nil {
			logger.Error("account deletion cancel code email failed", zap.Error(err), zap.Uint("userID", fresh.ID))
		}
	}()
	response.Success(c, "若该邮箱存在冷静期内的注销申请，将发送验证码", nil)
}

// handleAccountDeletionCancelByEmail 未登录：邮箱 + 密码 + 验证码撤销注销。
func (h *Handlers) handleAccountDeletionCancelByEmail(c *gin.Context) {
	var form struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required"`
		EmailCode string `json:"emailCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(form.Email))
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user, err := models.GetUserByEmail(db, email)
	if err != nil || user == nil {
		response.Fail(c, "邮箱或验证码不正确", errors.New("invalid"))
		return
	}
	var fresh models.User
	if err := db.Where("id = ? AND is_deleted = ?", user.ID, models.SoftDeleteStatusActive).First(&fresh).Error; err != nil {
		response.Fail(c, "邮箱或验证码不正确", errors.New("invalid"))
		return
	}
	if !models.AccountDeletionPending(&fresh) {
		response.Fail(c, "当前没有进行中的注销申请", errors.New("not pending"))
		return
	}
	cacheKey := fresh.Email + ":account_deletion_cancel"
	cached, ok := utils.GlobalCache.Get(cacheKey)
	if !ok || cached != form.EmailCode {
		response.Fail(c, "邮箱验证码无效或已过期", errors.New("invalid email code"))
		return
	}
	utils.GlobalCache.Remove(cacheKey)
	if !verifyUserLoginPassword(&fresh, form.Password) {
		response.Fail(c, "密码错误", errors.New("invalid password"))
		return
	}
	operator := strings.TrimSpace(fresh.Email)
	if operator == "" {
		operator = "self"
	}
	if err := models.CancelAccountDeletion(db, fresh.ID, operator); err != nil {
		response.Fail(c, "撤回失败", err)
		return
	}
	response.Success(c, "已撤回注销申请，账号恢复正常", nil)
}

func (h *Handlers) handleUnbindGitHub(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"github_id":    "",
		"github_login": "",
		"update_by":    strings.TrimSpace(user.Email),
	}).Error; err != nil {
		response.Fail(c, "解绑失败", err)
		return
	}
	response.Success(c, "GitHub 已解绑", nil)
}

func (h *Handlers) handleUnbindWechat(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"wechat_open_id":  "",
		"wechat_union_id": "",
		"update_by":       strings.TrimSpace(user.Email),
	}).Error; err != nil {
		response.Fail(c, "解绑失败", err)
		return
	}
	response.Success(c, "微信已解绑", nil)
}
