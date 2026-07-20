package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// /me — self-service handlers for the currently authenticated principal
// (tenant user OR platform admin). Sibling file tenant_users.go owns
// the admin-facing CRUD on the same table. Split to keep each file
// focused on one concern.
//
// Endpoints in this file:
//   - GET    /me                 → getMe
//   - PUT    /me                 → updateMe
//   - PUT    /me/password        → updateMyPassword
//   - POST   /me/avatar          → uploadMeAvatar
//   - POST   /me/totp/setup      → setupTotp
//   - POST   /me/totp/enable     → enableTotp
//   - POST   /me/totp/disable    → disableTotp
//   - POST   /logout             → logout

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────────────────────────────────────
// /me — Current User Profile & Security
// ──────────────────────────────────────────────────────────────────────────────

type updateMeReq struct {
	DisplayName string `json:"displayName"`
	Phone       string `json:"phone"`
	Username    string `json:"username"`
}

type updateMyPasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
	EmailCode   string `json:"emailCode"`
	Method      string `json:"method"` // "password" (default) | "email_code"
}

type forgotPasswordReq struct {
	Email       string `json:"email" binding:"required,email"`
	EmailCode   string `json:"emailCode" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
	captcha.CaptchaFields
}

// getMe reports the currently authenticated principal.
//
// This endpoint is intentionally shared by tenant users and platform admins
// so the UI can resolve the active role without a second round-trip. The
// response shape differs depending on who is authenticated:
//
//   - Platform admin → { principal: "platform", platformAdmin: {...} }
//   - Tenant user    → { principal: "tenant", user: {...}, tenant: {...}, permissionCodes: [...] }
//
// GET /me — requires a valid tenant user OR platform admin session; no
// path or query parameters. Response is a JSON success envelope.
func (h *Handlers) getMe(c *gin.Context) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		var row models.PlatformAdmin
		if err := h.db.Where("id = ?", aid).First(&row).Error; err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"principal":     "platform",
			"platformAdmin": apiresponse.NewPlatformAdminResponse(row),
		})
		return
	}

	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	var tenant models.Tenant
	if err := h.db.Where("id = ?", u.TenantID).First(&tenant).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyTenantNotFound))
		return
	}
	codes, _ := models.ListEffectivePermissionCodesForTenantUser(h.db, u.ID)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"principal":       "tenant",
		"user":            apiresponse.NewTenantUserResponse(h.db, u),
		"tenant":          apiresponse.NewTenantResponse(tenant),
		"permissionCodes": codes,
	})
}

// updateMe edits the profile fields of the caller.
//
//   - PUT /me — body: { displayName, phone, username }
//
// Request body (tenant user):
//   - displayName (string) — optional display name override.
//   - phone       (string) — optional new phone number; conflict returns 409.
//   - username    (string) — optional new username; conflict returns 409.
//
// Request body (platform admin):
//   - displayName (string) — only the display name is editable via this endpoint.
//
// Response mirrors the current user / platform admin record after the write
// and is gated by the caller's own auth session.
func (h *Handlers) updateMe(c *gin.Context) {
	if middleware.AuthPlatformAdminID(c) > 0 {
		aid := middleware.AuthPlatformAdminID(c)
		var req updateMeReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
			return
		}
		updates := map[string]any{}
		if v := strings.TrimSpace(req.DisplayName); v != "" {
			updates["display_name"] = v
		}
		if len(updates) == 0 {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNoFieldsToUpdate))
			return
		}
		before, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if err := h.db.Model(&models.PlatformAdmin{}).
			Where("id = ?", aid).
			Updates(updates).Error; err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		after, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		h.recordOpChange(c, OpLogEntry{
			Action:   constants.OpActionUpdate,
			Resource: constants.OpResourcePlatformAdmin, ResourceID: aid, ResourceName: before.Email,
			Summary: fmt.Sprintf("Updated profile %s", before.Email), Detail: audit.Redact(req),
		}, before, after)
		response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminResponse(after))
		return
	}

	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	var req updateMeReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if v := strings.TrimSpace(req.DisplayName); v != "" {
		updates["display_name"] = v
	}
	if req.Phone != "" {
		phone := strings.TrimSpace(req.Phone)
		exists, _ := models.CheckTenantUserPhoneExists(h.db, phone, u.ID)
		if exists {
			response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyPhoneExists))
			return
		}
		updates["phone"] = phone
	}
	if req.Username != "" {
		username := strings.TrimSpace(req.Username)
		exists, _ := models.CheckTenantUserUsernameExists(h.db, username, u.ID)
		if exists {
			response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyUsernameExists))
			return
		}
		updates["username"] = username
	}
	if len(updates) == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNoFieldsToUpdate))
		return
	}
	before := u
	if _, err := models.UpdateTenantUser(h.db, u.ID, updates, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: u.TenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Updated profile %s", u.Email), Detail: audit.Redact(req),
	}, before, next)
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, next))
}

// updateMyPassword rotates the caller's password after verifying the old one
// or a one-time email verification code.
//
//   - PUT /me/password — body: { oldPassword, newPassword } or { method: "email_code", emailCode, newPassword }
func (h *Handlers) updateMyPassword(c *gin.Context) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		var req updateMyPasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
			return
		}
		var row models.PlatformAdmin
		if err := h.db.Where("id = ?", aid).First(&row).Error; err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if err := h.verifyPasswordChangeAuth(req, utils.TrimLower(row.Email), row.PasswordHash); err != nil {
			response.Render(c, err)
			return
		}
		hash, err := access.HashPassword(req.NewPassword)
		if err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		before := row
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).
			Update("password_hash", hash).Error; err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		after, _ := models.GetPlatformAdminByID(h.db, aid)
		h.recordOpChange(c, OpLogEntry{
			Action:   constants.OpActionUpdate,
			Resource: constants.OpResourcePlatformAdmin, ResourceID: aid, ResourceName: row.Email,
			Summary: fmt.Sprintf("Changed password %s", row.Email), Detail: audit.Redact(req),
		}, before, after)
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": aid})
		return
	}

	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	var req updateMyPasswordReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if err := h.verifyPasswordChangeAuth(req, utils.TrimLower(u.Email), u.PasswordHash); err != nil {
		response.Render(c, err)
		return
	}
	hash, err := access.HashPassword(req.NewPassword)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	before := u
	if _, err := models.UpdateTenantUser(h.db, u.ID, map[string]any{"password_hash": hash}, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	after, _ := models.GetActiveTenantUserByID(h.db, u.ID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: u.TenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Changed password %s", u.Email), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": u.ID})
}

func (h *Handlers) verifyPasswordChangeAuth(req updateMyPasswordReq, email, passwordHash string) *response.AppError {
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = "password"
	}
	if access.CheckPassword(passwordHash, req.NewPassword) {
		return response.NewI18n(response.CodeValidation, i18n.KeyNewPasswordSameAsOld)
	}
	if method == "email_code" {
		if !verifyEmailCode(emailCodePassword, email, req.EmailCode) {
			return response.NewI18n(response.CodeAuthFailed, i18n.KeyAuthInvalidEmailCode)
		}
		return nil
	}
	if strings.TrimSpace(req.OldPassword) == "" {
		return response.NewI18n(response.CodeAuthFailed, i18n.KeyPasswordWrong)
	}
	if !access.CheckPassword(passwordHash, req.OldPassword) {
		return response.NewI18n(response.CodeAuthFailed, i18n.KeyPasswordWrong)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Avatar & TOTP
// ──────────────────────────────────────────────────────────────────────────────

const maxAvatarBytes = 2 << 20

// uploadMeAvatar accepts a multipart avatar for the signed-in tenant user.
//
//   - POST /me/avatar — multipart/form-data with a single file field ("file").
//
// Form parameters:
//   - file (bytes) — image file; must be JPEG/PNG/GIF/WebP and <= 2 MB.
//
// Platform admins are rejected with 403 (they do not have avatars in this
// release). Uploaded bytes are persisted to the configured object store,
// the previous avatar (if any) is deleted, and the tenant user record is
// patched with the resolved URL.
//
// Response: { avatarUrl, user } where user is the post-update public view.
func (h *Handlers) uploadMeAvatar(c *gin.Context) {
	if middleware.AuthPlatformAdminID(c) > 0 {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyPlatformNoAvatarUpload))
		return
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeySelectImageFile))
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyCannotReadFile))
		return
	}
	defer src.Close()

	body, err := io.ReadAll(io.LimitReader(src, maxAvatarBytes+1))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if len(body) > maxAvatarBytes {
		response.Render(c, response.NewI18n(response.CodeValidation, i18n.KeyImageTooLarge))
		return
	}
	ct := http.DetectContentType(body)
	ext := utils.PickImageExtFromContentType(ct)
	if ext == "" {
		response.Render(c, response.NewI18n(response.CodeValidation, i18n.KeyImageFormatInvalid))
		return
	}

	key := path.Join(
		"avatars",
		"t"+strconv.FormatUint(uint64(u.TenantID), 10),
		fmt.Sprintf("u%d_%d%s", u.ID, time.Now().UnixMilli(), ext),
	)
	st := stores.Default()
	deleteStoredAvatar(st, u.AvatarURL)
	deleteLegacyAvatarKeys(st, u.TenantID, u.ID)
	if err := st.Write(key, bytes.NewReader(body)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	// Avatar download URL resolution order:
	//   1) Backend absolute URL (with STORAGE_PUBLIC_BASE_URL fallback);
	//   2) Falls back to /uploads/<key> via gateway proxy.
	url := ginutil.UploadURL(c, key)
	before := u
	if _, err := models.UpdateTenantUser(h.db, u.ID, map[string]any{"avatar_url": url}, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: u.TenantID, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Updated avatar %s", u.Email),
	}, before, next)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"avatarUrl": url, "user": apiresponse.NewTenantUserResponse(h.db, next)})
}

func objectKeyFromAvatarURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.Index(raw, "?"); i >= 0 {
		raw = raw[:i]
	}
	if idx := strings.Index(raw, "/uploads/"); idx >= 0 {
		return strings.TrimPrefix(raw[idx+len("/uploads/"):], "/")
	}
	if idx := strings.Index(raw, "/avatars/"); idx >= 0 {
		return strings.TrimPrefix(raw[idx+1:], "/")
	}
	return ""
}

func deleteStoredAvatar(st stores.Store, avatarURL string) {
	if st == nil {
		return
	}
	if key := objectKeyFromAvatarURL(avatarURL); key != "" {
		_ = st.Delete(key)
	}
}

// deleteLegacyAvatarKeys removes pre-timestamp avatar paths (u{id}.ext).
func deleteLegacyAvatarKeys(st stores.Store, tenantID, userID uint) {
	if st == nil || tenantID == 0 || userID == 0 {
		return
	}
	base := path.Join("avatars", "t"+strconv.FormatUint(uint64(tenantID), 10), "u"+strconv.FormatUint(uint64(userID), 10))
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		_ = st.Delete(base + ext)
	}
}

type updateSecurityPreferencesReq struct {
	ReceiveEmailNotify       *bool `json:"receiveEmailNotify"`
	RequireDeviceVerify      *bool `json:"requireDeviceVerify"`
	TrustDeviceLoginEnabled  *bool `json:"trustDeviceLoginEnabled"`
	RequireRemoteLoginVerify *bool `json:"requireRemoteLoginVerify"`
	SessionIdleTimeoutHours  *int  `json:"sessionIdleTimeoutHours"`
	SessionMaxLifetimeHours  *int  `json:"sessionMaxLifetimeHours"`
}

func (h *Handlers) updateSecurityPreferences(c *gin.Context) {
	var req updateSecurityPreferencesReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if req.ReceiveEmailNotify == nil && req.RequireDeviceVerify == nil &&
		req.TrustDeviceLoginEnabled == nil && req.RequireRemoteLoginVerify == nil &&
		req.SessionIdleTimeoutHours == nil && req.SessionMaxLifetimeHours == nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNoFieldsToUpdate))
		return
	}
	updates := map[string]any{}
	if req.ReceiveEmailNotify != nil {
		updates["receive_email_notify"] = *req.ReceiveEmailNotify
	}
	if req.RequireDeviceVerify != nil {
		updates["require_device_verify"] = *req.RequireDeviceVerify
	}
	if req.TrustDeviceLoginEnabled != nil {
		updates["trust_device_login_enabled"] = *req.TrustDeviceLoginEnabled
	}
	if req.RequireRemoteLoginVerify != nil {
		updates["require_remote_login_verify"] = *req.RequireRemoteLoginVerify
	}
	if req.SessionIdleTimeoutHours != nil {
		updates["session_idle_timeout_hours"] = utils.SessionIdleTimeout(*req.SessionIdleTimeoutHours)
	}
	if req.SessionMaxLifetimeHours != nil {
		idle := utils.DefaultSessionIdleTimeoutHours
		if req.SessionIdleTimeoutHours != nil {
			idle = utils.SessionIdleTimeout(*req.SessionIdleTimeoutHours)
		}
		updates["session_max_lifetime_hours"] = utils.SessionMaxLifetime(idle, *req.SessionMaxLifetimeHours)
	}
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).Updates(updates).Error; err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		next, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminResponse(next))
		return
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if _, err := models.UpdateTenantUser(h.db, u.ID, updates, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, next))
}

func (h *Handlers) verifyTotpOrRecoveryCode(principalType string, principalID uint, totpSecret, recoveryHashes, code string) bool {
	code = strings.TrimSpace(code)
	if access.ValidateTOTP(code, totpSecret) {
		return true
	}
	updated, ok := access.ConsumeRecoveryCode(code, recoveryHashes)
	if !ok {
		return false
	}
	switch principalType {
	case models.UserDevicePrincipalPlatformAdmin:
		return h.db.Model(&models.PlatformAdmin{}).Where("id = ?", principalID).
			Update("totp_recovery_hashes", updated).Error == nil
	default:
		_, err := models.UpdateTenantUser(h.db, principalID, map[string]any{"totp_recovery_hashes": updated}, "system")
		return err == nil
	}
}

type enableTotpReq struct {
	Secret string `json:"secret" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

type disableTotpReq struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// setupTotp issues a fresh TOTP (secret + URL + QR data URL) for the caller.
//
//   - POST /me/totp/setup — no body.
//
// The caller must NOT already have two-factor enabled; otherwise a 409 Conflict is
// returned so the UI can instruct a disable+re-enable flow. The returned secret
// is meant to be confirmed via /me/totp/enable below.
//
// Response: { secret, url, qrDataUrl } — the qrDataUrl is a data URL
// embeddable in an <img>.
func (h *Handlers) setupTotp(c *gin.Context) {
	issuer := strings.TrimSpace(config.GlobalConfig.Server.Name)
	if issuer == "" {
		issuer = access.DefaultTOTPIssuer
	}
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		var adm models.PlatformAdmin
		if err := h.db.Where("id = ?", aid).First(&adm).Error; err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if adm.TOTPEnabled {
			response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyTotpDisableFirstToRebind))
			return
		}
		account := strings.TrimSpace(adm.Email)
		if account == "" {
			account = "platform:" + strconv.FormatUint(uint64(adm.ID), 10)
		}
		setup, err := access.GenerateTOTPSetup(issuer, account, 0)
		if err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"secret":    setup.Secret,
			"url":       setup.URL,
			"qrDataUrl": setup.QRDataURL,
		})
		return
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if u.TOTPEnabled {
		response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyTotpDisableFirstToRebind))
		return
	}
	account := strings.TrimSpace(u.Email)
	if account == "" {
		account = strconv.FormatUint(uint64(u.ID), 10)
	}
	setup, err := access.GenerateTOTPSetup(issuer, account, 0)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"secret":    setup.Secret,
		"url":       setup.URL,
		"qrDataUrl": setup.QRDataURL,
	})
}

// enableTotp binds a TOTP secret for the caller after validating one code.
//
//   - POST /me/totp/enable — body: { secret, code }
//
// Request body:
//   - secret (string) — required; the TOTP base32 secret returned by /setup.
//   - code   (string) — required; 6-digit code generated by the authenticator app.
//
// Fails with 401 on code mismatch and 409 if two-factor is already enabled.
// Response on success is the updated platform admin / tenant user public
// view so the UI can reflect the new state.
func (h *Handlers) enableTotp(c *gin.Context) {
	var req enableTotpReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if !access.ValidateTOTP(req.Code, req.Secret) {
		response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyAuthInvalidEmailCode))
		return
	}
	secret := strings.TrimSpace(req.Secret)
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		var adm models.PlatformAdmin
		if err := h.db.Where("id = ?", aid).First(&adm).Error; err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if adm.TOTPEnabled {
			response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyTotpAlreadyOn))
			return
		}
		recoveryCodes, rerr := access.GenerateRecoveryCodes(access.DefaultRecoveryCodeCount)
		if rerr != nil {
			ginutil.WriteInternalError(c, rerr)
			return
		}
		recoveryHashes, rerr := access.HashRecoveryCodes(recoveryCodes)
		if rerr != nil {
			ginutil.WriteInternalError(c, rerr)
			return
		}
		before := adm
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).Updates(map[string]any{
			"totp_secret":          secret,
			"totp_enabled":         true,
			"totp_recovery_hashes": recoveryHashes,
		}).Error; err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		next, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		h.recordOpChange(c, OpLogEntry{
			Action:   constants.OpActionEnable,
			Resource: constants.OpResourcePlatformAdmin, ResourceID: aid, ResourceName: adm.Email,
			Summary: fmt.Sprintf("Enabled two-factor authentication %s", adm.Email), Detail: audit.Redact(req),
		}, before, next)
		response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminTotpEnabledResponse(next, recoveryCodes))
		return
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if u.TOTPEnabled {
		response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyTotpAlreadyOn))
		return
	}
	recoveryCodes, rerr := access.GenerateRecoveryCodes(access.DefaultRecoveryCodeCount)
	if rerr != nil {
		ginutil.WriteInternalError(c, rerr)
		return
	}
	recoveryHashes, rerr := access.HashRecoveryCodes(recoveryCodes)
	if rerr != nil {
		ginutil.WriteInternalError(c, rerr)
		return
	}
	before := u
	if _, err := models.UpdateTenantUser(h.db, u.ID, map[string]any{
		"totp_secret":          secret,
		"totp_enabled":         true,
		"totp_recovery_hashes": recoveryHashes,
	}, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: u.TenantID, Action: constants.OpActionEnable,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Enabled two-factor authentication %s", u.Email), Detail: audit.Redact(req),
	}, before, next)
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserTotpEnabledResponse(h.db, next, recoveryCodes))
}

// disableTotp removes the caller's TOTP binding. Requires both the current
// password and one valid 6-digit code.
//
//   - POST /me/totp/disable — body: { password, code }
//
// Request body:
//   - password (string) — required; re-confirmation of the caller's password.
//   - code     (string) — required; current authenticator app code.
//
// Returns 401 on credential mismatch and 409 when TOTP is not currently
// enabled. Success response is the updated caller record.
func (h *Handlers) disableTotp(c *gin.Context) {
	var req disableTotpReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		var adm models.PlatformAdmin
		if err := h.db.Where("id = ?", aid).First(&adm).Error; err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if !adm.TOTPEnabled || strings.TrimSpace(adm.TOTPSecret) == "" {
			response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyTotpNotOn))
			return
		}
		if !access.CheckPassword(adm.PasswordHash, req.Password) {
			response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyPasswordWrong))
			return
		}
		if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalPlatformAdmin, aid, adm.TOTPSecret, adm.TOTPRecoveryHashes, req.Code) {
			response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyAuthInvalidEmailCode))
			return
		}
		before := adm
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).Updates(map[string]any{
			"totp_secret":          "",
			"totp_enabled":         false,
			"totp_recovery_hashes": "",
		}).Error; err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		next, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			ginutil.WriteInternalError(c, err)
			return
		}
		h.recordOpChange(c, OpLogEntry{
			Action:   constants.OpActionDisable,
			Resource: constants.OpResourcePlatformAdmin, ResourceID: aid, ResourceName: adm.Email,
			Summary: fmt.Sprintf("Disabled two-factor authentication %s", adm.Email), Detail: audit.Redact(req),
		}, before, next)
		response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewPlatformAdminResponse(next))
		return
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if !u.TOTPEnabled || strings.TrimSpace(u.TOTPSecret) == "" {
		response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyTotpNotOn))
		return
	}
	if !access.CheckPassword(u.PasswordHash, req.Password) {
		response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyPasswordWrong))
		return
	}
	if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalTenantUser, u.ID, u.TOTPSecret, u.TOTPRecoveryHashes, req.Code) {
		response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyAuthInvalidEmailCode))
		return
	}
	before := u
	if _, err := models.UpdateTenantUser(h.db, u.ID, map[string]any{
		"totp_secret":          "",
		"totp_enabled":         false,
		"totp_recovery_hashes": "",
	}, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: u.TenantID, Action: constants.OpActionDisable,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Disabled two-factor authentication %s", u.Email), Detail: audit.Redact(req),
	}, before, next)
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, next))
}
