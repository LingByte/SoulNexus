package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/nlu"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	"github.com/LingByte/SoulNexus/pkg/voice/cloneconfig"
	voiceMetrics "github.com/LingByte/SoulNexus/pkg/voice/metrics"
	"github.com/LingByte/SoulNexus/pkg/voice/voiceprintconfig"
	llmcache "github.com/LingByte/lingllm/cache"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type emailCodePurpose string

const (
	emailCodeLogin          emailCodePurpose = "login"
	emailCodePassword       emailCodePurpose = "password"
	emailCodeChangeEmail    emailCodePurpose = "change_email"
	emailCodeDeletionCancel emailCodePurpose = "deletion_cancel"
	emailCodeDeviceVerify   emailCodePurpose = "device_verify"
	emailCodeTTL                             = 10 * time.Minute
	emailCodeCooldown                        = 60 * time.Second
)

type emailAccount struct {
	UserID   uint
	TenantID uint
	Email    string
}

type sendEmailCodeReq struct {
	Email string `json:"email" binding:"required,email"`
	captcha.CaptchaFields
}

type sendDeviceVerifyEmailCodeReq struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password"`
	EmailCode   string `json:"emailCode"`
	LoginMethod string `json:"loginMethod"`
	DeviceID    string `json:"deviceId"`
	captcha.CaptchaFields
}

type systemInitResp struct {
	SiteName                  string `json:"SITE_NAME"`
	SiteDescription           string `json:"SITE_DESCRIPTION"`
	SiteURL                   string `json:"SITE_URL"`
	SiteLogoURL               string `json:"SITE_LOGO_URL"`
	SiteTermsURL              string `json:"SITE_TERMS_URL"`
	GitHubOAuthEnabled        bool   `json:"githubOAuthEnabled"`
	TenantSelfRegisterEnabled bool   `json:"tenantSelfRegisterEnabled"`
	VoiceCloneProvider        string `json:"VOICE_CLONE_PROVIDER"`
	VoiceCloneLabel           string `json:"VOICE_CLONE_LABEL,omitempty"`
	VoiceprintProvider        string `json:"VOICEPRINT_PROVIDER"`
	VoiceprintLabel           string `json:"VOICEPRINT_LABEL,omitempty"`
	NluEnabled                bool   `json:"nluEnabled"`
	SMSLoginEnabled           bool   `json:"smsLoginEnabled"`
}

// registerSystemRoutes mounts public system utility endpoints:
//   - GET  /system/init
//   - GET  /system/captcha/generate
//   - POST /system/email-codes/login
//   - POST /system/email-codes/login-device-verify
//   - POST /system/email-codes/forgot-password
//   - POST /system/email-codes/account-deletion-revoke
func (h *Handlers) registerSystemRoutes(r *humax.Group) {
	authLimit := middleware.AuthRateLimiter(10, 5*time.Minute, 10)
	captchaLimit := middleware.AuthRateLimiter(60, time.Minute, 20)

	g := r.Group("system")
	g.GET("/init", h.getSystemInit)
	g.GET("/captcha/generate", captchaLimit, h.generateCaptcha)

	email := g.Group("email-codes")
	email.Use(authLimit)
	email.POST("/login", h.sendLoginEmailCode)
	email.POST("/login-device-verify", h.sendLoginDeviceVerifyEmailCode)
	email.POST("/forgot-password", h.sendForgotPasswordEmailCode)
	email.POST("/account-deletion-revoke", h.sendRevokeDeletionEmailCode)

	smsCodes := g.Group("sms-codes")
	smsCodes.Use(authLimit)
	smsCodes.POST("/login", h.sendLoginSMSCode)
}

// registerSystemProtectedRoutes mounts authenticated system utility endpoints:
//   - POST /system/email-codes/me-password
//   - POST /system/email-codes/me-email
func (h *Handlers) registerSystemProtectedRoutes(r *humax.Group) {
	g := r.Group("system/email-codes")
	g.POST("/me-password", h.sendMePasswordEmailCode)
	g.POST("/me-email", h.sendChangeEmailCode)
}

func generateNumericEmailCode() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(b[:])%1_000_000)
}

func emailCodeCacheKey(purpose emailCodePurpose, email string) string {
	return string(purpose) + ":" + utils.TrimLower(email)
}

func emailCodeCooldownCacheKey(purpose emailCodePurpose, email string) string {
	return string(purpose) + ":cooldown:" + utils.TrimLower(email)
}

func emailCodeCacheGet(key string) (string, bool) {
	v, ok := llmcache.Get(context.Background(), key)
	if !ok || v == nil {
		return "", false
	}
	s, _ := v.(string)
	return s, s != ""
}

func emailCodeCacheSet(key, value string, ttl time.Duration) {
	_ = llmcache.Set(context.Background(), key, value, ttl)
}

func emailCodeCacheDelete(key string) {
	_ = llmcache.Delete(context.Background(), key)
}

func emailCodeCooldownActive(key string) bool {
	return llmcache.Exists(context.Background(), key)
}

func emailCodeCooldownSet(key string) {
	_ = llmcache.Set(context.Background(), key, "1", emailCodeCooldown)
}

func resolveEmailAccount(db *gorm.DB, email string) (emailAccount, bool) {
	email = utils.TrimLower(email)
	if u, err := models.GetActiveTenantUserByEmailGlobal(db, email); err == nil {
		return emailAccount{UserID: u.ID, TenantID: u.TenantID, Email: email}, true
	}
	if a, err := models.GetActivePlatformAdminByEmail(db, email); err == nil {
		return emailAccount{UserID: a.ID, TenantID: 0, Email: email}, true
	}
	return emailAccount{}, false
}

func verifyEmailCode(purpose emailCodePurpose, email, code string) bool {
	email = utils.TrimLower(email)
	code = strings.TrimSpace(code)
	if email == "" || code == "" {
		return false
	}
	stored, ok := emailCodeCacheGet(emailCodeCacheKey(purpose, email))
	if !ok || stored != code {
		return false
	}
	emailCodeCacheDelete(emailCodeCacheKey(purpose, email))
	return true
}

func (h *Handlers) dispatchEmailVerificationCode(c *gin.Context, purpose emailCodePurpose, acct emailAccount) bool {
	cooldownKey := emailCodeCooldownCacheKey(purpose, acct.Email)
	if emailCodeCooldownActive(cooldownKey) {
		response.FailI18n(c, i18n.KeyAuthEmailCodeCooldown, nil)
		return false
	}

	code := generateNumericEmailCode()
	emailCodeCacheSet(emailCodeCacheKey(purpose, acct.Email), code, emailCodeTTL)
	emailCodeCooldownSet(cooldownKey)

	logger.Info("auth email code dispatch",
		zap.String("purpose", string(purpose)),
		zap.String("email", acct.Email),
		zap.Uint("userId", acct.UserID))

	utils.Sig().Emit(constants.SigMailVerificationCode, nil, constants.MailVerificationCodePayload{
		UserID:   acct.UserID,
		Email:    acct.Email,
		Code:     code,
		Purpose:  string(purpose),
		ClientIP: c.ClientIP(),
	}, h.db)
	return true
}

func (h *Handlers) dispatchDeviceVerifyCode(c *gin.Context, acct emailAccount, deviceKey, displayName string) bool {
	cooldownKey := emailCodeCooldownCacheKey(emailCodeDeviceVerify, acct.Email)
	if emailCodeCooldownActive(cooldownKey) {
		response.FailI18n(c, i18n.KeyAuthEmailCodeCooldown, nil)
		return false
	}
	code := generateNumericEmailCode()
	emailCodeCacheSet(emailCodeCacheKey(emailCodeDeviceVerify, acct.Email), code, emailCodeTTL)
	emailCodeCooldownSet(cooldownKey)
	username := strings.TrimSpace(displayName)
	if username == "" {
		username = acct.Email
	}
	utils.Sig().Emit(constants.SigMailDeviceVerifyCode, nil, constants.MailDeviceVerifyCodePayload{
		UserID:    acct.UserID,
		Email:     acct.Email,
		Username:  username,
		Code:      code,
		DeviceKey: deviceKey,
		ClientIP:  c.ClientIP(),
	}, h.db)
	return true
}

func (h *Handlers) sendLoginDeviceVerifyEmailCode(c *gin.Context) {
	var req sendDeviceVerifyEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	if err := captcha.ValidatePayload(req.CaptchaID, req.CaptchaType, req.CaptchaValue); err != nil {
		if err == captcha.ErrPayloadRequired {
			response.FailI18n(c, i18n.KeyValidationCaptchaRequired, nil)
		} else {
			response.FailI18n(c, i18n.KeyValidationCaptchaInvalid, nil)
		}
		return
	}

	email := utils.TrimLower(req.Email)
	if _, ok := resolvePendingDeletionAccount(h.db, email); ok {
		response.FailI18n(c, i18n.KeyAccountDeletionPending, gin.H{"revokePath": "/account/deletion/revoke"})
		return
	}

	loginMethod := strings.TrimSpace(req.LoginMethod)
	if loginMethod == "" {
		loginMethod = "password"
	}
	useEmailCode := loginMethod == "email_code"

	displayName := ""
	acct := emailAccount{}

	user, err := models.GetActiveTenantUserByEmailGlobal(h.db, email)
	if err == nil {
		if useEmailCode {
			if !verifyEmailCode(emailCodeLogin, email, req.EmailCode) {
				response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
				return
			}
		} else if strings.TrimSpace(req.Password) == "" || !access.CheckPassword(user.PasswordHash, req.Password) {
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
		displayName = user.DisplayName
		acct = emailAccount{UserID: user.ID, TenantID: user.TenantID, Email: email}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		response.AbortWithStatusJSON(c, 500, err)
		return
	} else {
		adm, aerr := models.GetActivePlatformAdminByEmail(h.db, email)
		if aerr != nil {
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
		if useEmailCode {
			if !verifyEmailCode(emailCodeLogin, email, req.EmailCode) {
				response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
				return
			}
		} else if strings.TrimSpace(req.Password) == "" || !access.CheckPassword(adm.PasswordHash, req.Password) {
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
		displayName = adm.DisplayName
		acct = emailAccount{UserID: adm.ID, TenantID: 0, Email: email}
	}

	deviceKey := strings.TrimSpace(req.DeviceID)
	if !h.dispatchDeviceVerifyCode(c, acct, deviceKey, displayName) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthEmailCodeSent, nil)
}

func (h *Handlers) sendLoginEmailCode(c *gin.Context) {
	var req sendEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	if err := captcha.ValidatePayload(req.CaptchaID, req.CaptchaType, req.CaptchaValue); err != nil {
		if err == captcha.ErrPayloadRequired {
			response.FailI18n(c, i18n.KeyValidationCaptchaRequired, nil)
		} else {
			response.FailI18n(c, i18n.KeyValidationCaptchaInvalid, nil)
		}
		return
	}

	email := utils.TrimLower(req.Email)
	if _, ok := resolvePendingDeletionAccount(h.db, email); ok {
		response.FailI18n(c, i18n.KeyAccountDeletionPending, gin.H{"revokePath": "/account/deletion/revoke"})
		return
	}
	acct, ok := resolveEmailAccount(h.db, email)
	if !ok {
		response.FailI18n(c, i18n.KeyAuthEmailNotRegistered, nil)
		return
	}
	if !h.dispatchEmailVerificationCode(c, emailCodeLogin, acct) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthEmailCodeSent, nil)
}

func (h *Handlers) sendForgotPasswordEmailCode(c *gin.Context) {
	var req sendEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	if err := captcha.ValidatePayload(req.CaptchaID, req.CaptchaType, req.CaptchaValue); err != nil {
		if err == captcha.ErrPayloadRequired {
			response.FailI18n(c, i18n.KeyValidationCaptchaRequired, nil)
		} else {
			response.FailI18n(c, i18n.KeyValidationCaptchaInvalid, nil)
		}
		return
	}

	email := utils.TrimLower(req.Email)
	acct, ok := resolveEmailAccount(h.db, email)
	if !ok {
		response.FailI18n(c, i18n.KeyAuthEmailNotRegistered, nil)
		return
	}
	if !h.dispatchEmailVerificationCode(c, emailCodePassword, acct) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthEmailCodeSent, nil)
}

func (h *Handlers) sendMePasswordEmailCode(c *gin.Context) {
	acct, ok := h.authenticatedEmailAccount(c)
	if !ok {
		return
	}
	if !h.dispatchEmailVerificationCode(c, emailCodePassword, acct) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthEmailCodeSent, nil)
}

func (h *Handlers) authenticatedEmailAccount(c *gin.Context) (emailAccount, bool) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		row, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			response.Render(c, response.Err(response.CodeUnauthorized))
			return emailAccount{}, false
		}
		return emailAccount{
			UserID:   row.ID,
			TenantID: 0,
			Email:    utils.TrimLower(row.Email),
		}, true
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return emailAccount{}, false
	}
	return emailAccount{
		UserID:   u.ID,
		TenantID: u.TenantID,
		Email:    utils.TrimLower(u.Email),
	}, true
}

func (h *Handlers) generateCaptcha(c *gin.Context) {
	rawType := strings.TrimSpace(c.Query("type"))
	var (
		result *captcha.Result
		err    error
	)
	switch rawType {
	case string(captcha.TypeRandom):
		result, err = captcha.EnsureGlobalManager().GenerateRandom()
	case string(captcha.TypeImage), string(captcha.TypeClick), string(captcha.TypeSlider):
		result, err = captcha.EnsureGlobalManager().Generate(captcha.Type(rawType))
	default:
		response.FailI18n(c, i18n.KeyInvalidParams, gin.H{"error": "unsupported captcha type"})
		return
	}
	if err != nil {
		response.AbortWithStatusJSON(c, 500, err)
		return
	}
	if result.Type == captcha.TypeImage {
		delete(result.Data, "code")
	}
	response.SuccessI18n(c, i18n.KeySuccess, result)
}

// JWKSHandler serves the public JSON Web Key Set used by external systems
// to verify access tokens issued by this service.
func (h *Handlers) JWKSHandler(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	if bootstrap.GlobalKeyManager == nil {
		response.FailI18n(c, i18n.KeyServiceUnavailable, nil)
		return
	}
	jwksJSON, err := bootstrap.GlobalKeyManager.GetJWKSJSON()
	if err != nil {
		response.FailI18n(c, i18n.KeyServiceUnavailable, nil)
		return
	}
	c.Data(200, "application/json", []byte(jwksJSON))
}

func (h *Handlers) getSystemInit(c *gin.Context) {
	db := h.db

	name := utils.GetValue(db, constants.KEY_SITE_NAME)
	if name == "" {
		name = "SoulNexus"
	}
	desc := utils.GetValue(db, constants.KEY_SITE_DESCRIPTION)
	if desc == "" {
		desc = "SoulNexus Description"
	}
	url := utils.GetValue(db, constants.KEY_SITE_URL)
	logoRaw := utils.GetValue(db, constants.KEY_SITE_LOGO)
	if logoRaw == "" {
		logoRaw = constants.DefaultSiteLogoPath
	}
	terms := utils.GetValue(db, constants.KEY_SITE_TERMS_URL)
	githubEnabled := false
	if config.GlobalConfig != nil {
		githubEnabled = strings.TrimSpace(config.GlobalConfig.Auth.GitHubClientID) != "" &&
			strings.TrimSpace(config.GlobalConfig.Auth.GitHubClientSecret) != ""
	}
	voiceCloneProvider := ""
	voiceCloneLabel := ""
	if slug, prov, ok := cloneconfig.ResolveEnabled(); ok {
		voiceCloneProvider = slug
		voiceCloneLabel = cloneconfig.ProviderLabel(prov)
	}
	voiceprintProvider := ""
	voiceprintLabel := ""
	if slug, prov, ok := voiceprintconfig.ResolveEnabled(); ok {
		voiceprintProvider = slug
		voiceprintLabel = voiceprintconfig.ProviderLabel(prov)
	}
	smsLoginEnabled := false
	if _, err := listeners.EnabledSMSChannels(db); err == nil {
		smsLoginEnabled = true
	}
	response.SuccessI18n(c, i18n.KeySuccess, systemInitResp{
		SiteName:                  name,
		SiteDescription:           desc,
		SiteURL:                   url,
		SiteLogoURL:               logoRaw,
		SiteTermsURL:              terms,
		GitHubOAuthEnabled:        githubEnabled,
		TenantSelfRegisterEnabled: utils.GetBoolValue(db, constants.KEY_TENANT_SELF_REGISTER),
		VoiceCloneProvider:        voiceCloneProvider,
		VoiceCloneLabel:           voiceCloneLabel,
		VoiceprintProvider:        voiceprintProvider,
		VoiceprintLabel:           voiceprintLabel,
		NluEnabled:                nlu.DeployEnabled(),
		SMSLoginEnabled:           smsLoginEnabled,
	})
}

// systemStartedAt 由 init 记录，可用于在响应里给运维一个 uptime 数字，
// 比让前端去算服务启动时间方便。
var systemStartedAt = time.Now()

// getSystemStatus returns a process-wide resource snapshot for the
// platform-admin dashboard. Access is restricted by the caller's route
// (this handler is wired behind RequirePlatformAdmin), so tenant users
// never see it.
//
//   - GET /system/status — no path/query parameters.
//
// The response is assembled from in-process monitors (no DB I/O):
//   - pkg/utils/system.GetSystemStatus()          → CPU/memory/disk usage
//   - pkg/utils/system.GetDiskSpaceInfo()         → disk free/used bytes
//   - pkg/utils/system.GetDiskCacheStats()        → disk cache hit counters
//   - pkg/utils/system.GetPerformanceMonitorConfig() → threshold config
//   - runtime.ReadMemStats                        → live heap / GC metrics
//   - uptimeSeconds / goroutines / go version
//
// Response is a success envelope with these fields merged under
// "resource", "disk", "runtimeMemory", "performanceMonitor" and
// "diskCache" for easier consumption by the ops dashboard.
func (h *Handlers) getSystemStatus(c *gin.Context) {
	st := system.GetSystemStatus()
	cfg := system.GetPerformanceMonitorConfig()
	disk := system.GetDiskSpaceInfo()
	cacheStats := system.GetDiskCacheStats()

	refresh := strings.EqualFold(strings.TrimSpace(c.Query("refresh")), "true") ||
		c.Query("refresh") == "1"
	var pf utils.Snapshot
	if refresh {
		addr := config.GlobalConfig.Server.Addr
		if addr == "" {
			addr = pkgconst.DefaultServerAddr
		}
		pf = utils.Run(c.Request.Context(), utils.Options{
			DB:         h.db,
			HTTPAddr:   addr,
			CheckPorts: false,
		})
	} else {
		pf = utils.GetSnapshot()
	}

	dependencies := h.systemDependencyChecks(c.Request.Context())

	runtimeSnap := system.CollectRuntimeSnapshot()
	hostSnap := system.CollectHostSnapshot()

	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"uptimeSeconds": int64(time.Since(systemStartedAt).Seconds()),
		"startedAt":     systemStartedAt.Format(time.RFC3339),
		"goroutines":    runtimeSnap.Goroutine.NumGoroutine,
		"go":            runtimeSnap.GoVersion,
		"server": gin.H{
			"name":       config.GlobalConfig.Server.Name,
			"addr":       config.GlobalConfig.Server.Addr,
			"mode":       config.GlobalConfig.Server.Mode,
			"sslEnabled": config.GlobalConfig.Server.SSLEnabled,
			"apiPrefix":  config.GlobalConfig.Server.APIPrefix,
		},
		"preflight":    pf,
		"dependencies": dependencies,
		"runtime":      runtimeSnap,
		"host":         hostSnap,
		"ops":          system.GetOpsCounters(),
		"business":     h.collectBusinessMetrics(h.db),
		"prometheus":   voiceMetrics.Default.Snapshot(),
		"resource": gin.H{
			"cpuUsage":    st.CPUUsage,
			"memoryUsage": st.MemoryUsage,
			"diskUsage":   st.DiskUsage,
		},
		"disk": gin.H{
			"total":       disk.Total,
			"free":        disk.Free,
			"used":        disk.Used,
			"usedPercent": disk.UsedPercent,
		},
		"runtimeMemory":   runtimeSnap.Memory,
		"gc":              runtimeSnap.GC,
		"goroutineDetail": runtimeSnap.Goroutine,
		"performanceMonitor": gin.H{
			"enabled":         cfg.Enabled,
			"cpuThreshold":    cfg.CPUThreshold,
			"memoryThreshold": cfg.MemoryThreshold,
			"diskThreshold":   cfg.DiskThreshold,
		},
		"diskCache": gin.H{
			"activeFiles":     cacheStats.ActiveDiskFiles,
			"diskUsageBytes":  cacheStats.CurrentDiskUsageBytes,
			"diskHits":        cacheStats.DiskCacheHits,
			"memoryHits":      cacheStats.MemoryCacheHits,
			"diskMaxBytes":    cacheStats.DiskCacheMaxBytes,
			"diskThreshold":   cacheStats.DiskCacheThresholdBytes,
			"memoryBuffers":   cacheStats.ActiveMemoryBuffers,
			"memoryUsageByte": cacheStats.CurrentMemoryUsageBytes,
		},
	})
}

func (h *Handlers) systemDependencyChecks(ctx context.Context) []gin.H {
	var out []gin.H
	if h.kb != nil {
		start := time.Now()
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := h.kb.Ping(pingCtx)
		cancel()
		latencyMs := time.Since(start).Milliseconds()
		if err != nil {
			system.IncDependencyFail()
			out = append(out, gin.H{
				"id": "knowledge.vector", "category": "dependencies", "level": "error",
				"message": "知识库向量后端不可达", "detail": err.Error(), "latencyMs": latencyMs,
			})
		} else {
			out = append(out, gin.H{
				"id": "knowledge.vector", "category": "dependencies", "level": "ok",
				"message": "知识库向量后端连通正常", "latencyMs": latencyMs,
			})
		}
	} else {
		out = append(out, gin.H{
			"id": "knowledge.vector", "category": "dependencies", "level": "warn",
			"message": "知识库服务未初始化（检查 EMBED/VECTOR 配置）", "latencyMs": 0,
		})
	}
	return out
}
