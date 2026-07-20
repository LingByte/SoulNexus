package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voice/voiceprintconfig"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// tenantLogin authenticates a tenant user or a platform admin using the same
// Request body (JSON):
//
//	{
//	  "email":    "alice@example.com",   // required, valid email
//	  "password": "supersecret",         // required
//	  "totpCode": "123456"               // optional, required when TOTP enabled
//	}
//
// Response on success (tenant user branch):
//
//	{
//	  "principal":       "tenant",
//	  "token":           "<jwt>",
//	  "expiresIn":       <seconds>,
//	  "tenant":          { "id":..., "name":..., ... },
//	  "user":            { "id":..., "email":..., ... },
//	  "permissionCodes": ["tenant.user.read", ...]
//	}
//
// Response on success (platform admin branch):
//
//	{
//	  "principal":     "platform",
//	  "token":         "<jwt>",
//	  "expiresIn":     <seconds>,
//	  "platformAdmin": { "id":..., "email":..., ... }
//	}
func (h *Handlers) tenantLogin(c *gin.Context) {
	var req dto.TenantLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}

	loginMethod := loginMethodOrDefault(req.LoginMethod)
	if loginMethod == "phone_code" {
		h.tenantLoginByPhone(c, req)
		return
	}

	clientIP := c.ClientIP()
	geo := utils.LoginGeoFromIP(clientIP)
	email := utils.TrimLower(req.Email)
	if email == "" {
		response.FailI18n(c, i18n.KeyInvalidParams, "email required")
		return
	}
	deviceKey := strings.TrimSpace(req.DeviceID)
	useEmailCode := loginMethod == "email_code"

	if utils.CheckLoginAccountLocked(email) {
		h.recordLoginFailure(c, models.LoginHistoryInput{
			Email: email, ClientIP: clientIP, City: geo.City, Location: geo.Location,
			LoginMethod: loginMethod, DeviceKey: deviceKey,
			FailureReason: "account_locked",
		}, "account_locked")
		response.FailI18n(c, i18n.KeyAuthAccountLocked, nil)
		return
	}

	if strings.TrimSpace(req.TotpCode) == "" {
		if err := captcha.ValidatePayload(req.CaptchaID, req.CaptchaType, req.CaptchaValue); err != nil {
			utils.RecordLoginFailure(email, clientIP)
			reason := "captcha_invalid"
			if err == captcha.ErrPayloadRequired {
				reason = "captcha_required"
				response.FailI18n(c, i18n.KeyValidationCaptchaRequired, nil)
			} else {
				response.FailI18n(c, i18n.KeyValidationCaptchaInvalid, nil)
			}
			h.recordLoginFailure(c, models.LoginHistoryInput{
				Email: email, ClientIP: clientIP, City: geo.City, Location: geo.Location,
				LoginMethod: loginMethod, DeviceKey: deviceKey,
			}, reason)
			return
		}
	}

	if bootstrap.GlobalKeyManager == nil {
		response.FailI18n(c, i18n.KeyAuthJWTNotReady, nil)
		return
	}

	user, err := models.GetActiveTenantUserByEmailGlobal(h.db, email)
	if err == nil {
		tenant, terr := models.GetActiveTenantByID(h.db, user.TenantID)
		if terr != nil {
			response.FailI18n(c, i18n.KeyTenantNotFound, nil)
			return
		}
		if tenant.Status != "" && tenant.Status != "active" {
			response.FailI18n(c, i18n.KeyTenantSuspended, nil)
			return
		}
		if user.Status != constants.TenantUserStatusActive {
			if user.Status == constants.TenantUserStatusPendingDeletion {
				response.FailI18n(c, i18n.KeyAccountDeletionPending, gin.H{"revokePath": "/account/deletion/revoke"})
				return
			}
			response.FailI18n(c, i18n.KeyTenantUserUnavailable, nil)
			return
		}
		if useEmailCode {
			if !verifyEmailCode(emailCodeLogin, email, req.EmailCode) {
				utils.RecordLoginFailure(email, clientIP)
				h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_email_code")
				response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
				return
			}
		} else {
			if strings.TrimSpace(req.Password) == "" {
				utils.RecordLoginFailure(email, clientIP)
				h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "empty_password")
				response.Render(c, utils.ErrEmptyPassword)
				return
			}
			if !access.CheckPassword(user.PasswordHash, req.Password) {
				utils.RecordLoginFailure(email, clientIP)
				h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_credentials")
				response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
				return
			}
		}
		if user.TOTPEnabled && strings.TrimSpace(user.TOTPSecret) != "" {
			code := strings.TrimSpace(req.TotpCode)
			if code == "" {
				response.FailI18n(c, i18n.KeyAuthNeedsTotp, gin.H{"needsTotp": true})
				return
			}
			if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalTenantUser, user.ID, user.TOTPSecret, user.TOTPRecoveryHashes, code) {
				utils.RecordLoginFailure(email, clientIP)
				h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_totp")
				response.FailI18n(c, i18n.KeyAuthInvalidTotp, gin.H{"needsTotp": true})
				return
			}
		}
		_ = models.RecordTenantUserLogin(h.db, user.ID, clientIP, geo.City, geo.Location)
		ds, derr := h.resolveDeviceSession(c, deviceSessionInput{
			PrincipalType:            models.UserDevicePrincipalTenantUser,
			PrincipalID:              user.ID,
			TenantID:                 user.TenantID,
			Email:                    user.Email,
			DisplayName:              user.DisplayName,
			DeviceKey:                req.DeviceID,
			DeviceCode:               req.DeviceVerifyCode,
			VoiceprintAudioBase64:    req.VoiceprintAudioBase64,
			ClientIP:                 clientIP,
			RequireDeviceVerify:      user.RequireDeviceVerify,
			TrustDeviceLoginEnabled:  user.TrustDeviceLoginEnabled,
			TrustDeviceFor7Days:      req.TrustDeviceFor7Days,
			RequireRemoteLoginVerify: user.RequireRemoteLoginVerify,
			KnownLoginCities:         utils.ParseKnownLoginCities(user.KnownLoginCitiesJSON),
		})
		if derr != nil {
			if errors.Is(derr, errInvalidDeviceCode) {
				utils.RecordLoginFailure(email, clientIP)
				h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_device_code")
				response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
				return
			}
			if errors.Is(derr, errInvalidVoiceprintVerify) {
				utils.RecordLoginFailure(email, clientIP)
				h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_voiceprint")
				data := gin.H{"needsDeviceVerify": true, "invalidVoiceprint": true}
				if _, _, vpOK := voiceprintconfig.ResolveEnabled(); vpOK && models.TenantUserHasVoiceprint(user) {
					data["voiceprintAvailable"] = true
				}
				response.FailI18n(c, i18n.KeyAuthInvalidVoiceprint, data)
				return
			}
			response.AbortWithStatusJSON(c, 500, derr)
			return
		}
		if ds.NeedsDeviceVerify || ds.NeedsRemoteVerify {
			data := gin.H{}
			if ds.NeedsDeviceVerify {
				data["needsDeviceVerify"] = true
			}
			if ds.NeedsRemoteVerify {
				data["needsRemoteVerify"] = true
			}
			if _, _, vpOK := voiceprintconfig.ResolveEnabled(); vpOK && models.TenantUserHasVoiceprint(user) {
				data["voiceprintAvailable"] = true
			}
			key := i18n.KeyAuthNeedsDeviceVerify
			if ds.NeedsRemoteVerify && !ds.NeedsDeviceVerify {
				key = i18n.KeyAuthNeedsRemoteVerify
			}
			response.FailI18n(c, key, data)
			return
		}
		token, tokenTTL, terr := signTenantAccessToken(h.db, user, tenant, ds.SessionID, ds.DeviceRecordID)
		if terr != nil {
			response.FailI18n(c, i18n.KeyTenantSignTokenFailed, nil)
			return
		}
		codes, _ := models.ListEffectivePermissionCodesForTenantUser(h.db, user.ID)
		utils.ClearLoginFailures(email)
		models.RecordSuccessfulLoginCity(h.db, user.ID, geo.City)
		h.recordLoginSuccess(c, models.LoginHistoryInput{
			PrincipalType: models.LoginHistoryPrincipalTenantUser,
			PrincipalID:   user.ID,
			TenantID:      user.TenantID,
			Email:         user.Email,
			ClientIP:      clientIP,
			City:          geo.City,
			Location:      geo.Location,
			LoginMethod:   loginMethod,
			DeviceKey:     strings.TrimSpace(req.DeviceID),
		})
		if user.WelcomeNotifiedAt == nil {
			utils.Sig().Emit(constants.SigMailWelcome, nil, constants.MailWelcomePayload{
				PrincipalType: models.UserDevicePrincipalTenantUser,
				UserID:        user.ID,
				Email:         user.Email,
				DisplayName:   user.DisplayName,
				ReceiveEmail:  user.ReceiveEmailNotify,
				ClientIP:      clientIP,
			}, h.db)
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"principal":       "tenant",
			"token":           token,
			"expiresIn":       int(tokenTTL.Seconds()),
			"tenant":          apiresponse.NewTenantResponse(tenant),
			"user":            apiresponse.NewTenantUserResponse(h.db, user),
			"permissionCodes": codes,
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		response.AbortWithStatusJSON(c, 500, err)
		return
	}

	var pendingAdmin models.PlatformAdmin
	if err := h.db.Where("email = ? AND status = ?", email, constants.PlatformAdminStatusPendingDeletion).First(&pendingAdmin).Error; err == nil {
		response.FailI18n(c, i18n.KeyAccountDeletionPending, gin.H{"revokePath": "/account/deletion/revoke"})
		return
	}

	adm, err := models.GetActivePlatformAdminByEmail(h.db, email)
	if err != nil {
		utils.RecordLoginFailure(email, clientIP)
		h.recordLoginFailure(c, models.LoginHistoryInput{
			Email: email, ClientIP: clientIP, City: geo.City, Location: geo.Location,
			LoginMethod: loginMethod, DeviceKey: deviceKey,
		}, "invalid_credentials")
		response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
		return
	}
	if useEmailCode {
		if !verifyEmailCode(emailCodeLogin, email, req.EmailCode) {
			utils.RecordLoginFailure(email, clientIP)
			response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
			return
		}
	} else {
		if strings.TrimSpace(req.Password) == "" {
			utils.RecordLoginFailure(email, clientIP)
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
		if !access.CheckPassword(adm.PasswordHash, req.Password) {
			utils.RecordLoginFailure(email, clientIP)
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
	}
	if adm.TOTPEnabled && strings.TrimSpace(adm.TOTPSecret) != "" {
		code := strings.TrimSpace(req.TotpCode)
		if code == "" {
			response.FailI18n(c, i18n.KeyAuthNeedsTotp, gin.H{"needsTotp": true})
			return
		}
		if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalPlatformAdmin, adm.ID, adm.TOTPSecret, adm.TOTPRecoveryHashes, code) {
			utils.RecordLoginFailure(email, clientIP)
			response.FailI18n(c, i18n.KeyAuthInvalidTotp, gin.H{"needsTotp": true})
			return
		}
	}

	token, err := access.SignPlatformAccessTokenWithKey(access.PlatformPayload{
		AdminID: adm.ID,
		Email:   adm.Email,
		Role:    constants.JWTRolePlatformSuper,
	}, bootstrap.GlobalKeyManager, constants.TenantAccessTokenTTL)
	if err != nil {
		response.FailI18n(c, i18n.KeyTenantSignTokenFailed, nil)
		return
	}

	utils.ClearLoginFailures(email)
	h.recordLoginSuccess(c, models.LoginHistoryInput{
		PrincipalType: models.LoginHistoryPrincipalPlatformAdmin,
		PrincipalID:   adm.ID,
		Email:         adm.Email,
		ClientIP:      clientIP,
		City:          geo.City,
		Location:      geo.Location,
		LoginMethod:   loginMethod,
		DeviceKey:     strings.TrimSpace(req.DeviceID),
	})
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"principal":     "platform",
		"token":         token,
		"expiresIn":     int(constants.TenantAccessTokenTTL.Seconds()),
		"platformAdmin": apiresponse.NewPlatformAdminResponse(adm),
	})
}

// registerTenant creates a tenant, default admin role, first admin user, and
// Request body (JSON, bound to tenantRegisterReq):
//
//	{
//	  "companyName":       "Acme Co.",           // required, 2..128 chars
//	  "adminEmail":        "admin@acme.co",      // required, valid email
//	  "adminPassword":     "supersecret",        // required, 8..128 chars
//	  "adminDisplayName":  "Alice",              // optional
//	  "tenantDescription": "Self-service demo",  // optional
//	  "maxUserCount":      5                     // optional
//	}
//
// Response on success:
//
//	{
//	  "principal":       "tenant",
//	  "token":           "<jwt>",
//	  "expiresIn":       <seconds>,
//	  "tenant":          { "id":..., "name":..., ... },
//	  "user":            { "id":..., "email":..., ... },
//	  "permissionCodes": ["tenant.user.read", ...],
//	  "roleCreated":     "tenant_admin",
//	  "roleId":          <uint>
//	}
func (h *Handlers) registerTenant(c *gin.Context) {
	if !utils.GetBoolValue(h.db, constants.KEY_TENANT_SELF_REGISTER) {
		response.Render(c, utils.ErrUserNotAllowSignup)
		return
	}
	var req tenantRegisterReq
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

	hash, err := access.HashPassword(req.AdminPassword)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if !utils.IsEmail(req.AdminEmail) {
		response.FailI18n(c, i18n.KeyTenantInvalidEmail, nil)
		return
	}
	takenMail, mailErr := models.CheckTenantUserEmailExists(h.db, req.AdminEmail, 0)
	if mailErr != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, mailErr)
		return
	}
	if takenMail {
		response.FailI18n(c, i18n.KeyTenantEmailExists, nil)
		return
	}

	tenant, user, role, err := models.ProvisionTenantWithAdmin(h.db, models.TenantProvisionInput{
		CompanyName: req.CompanyName, AdminEmail: req.AdminEmail, AdminDisplayName: req.AdminDisplayName,
		TenantDescription: req.TenantDescription, MaxUserCount: req.MaxUserCount,
	}, hash, "register")
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	geo := utils.LoginGeoFromIP(c.ClientIP())
	_ = models.RecordTenantUserLogin(h.db, user.ID, c.ClientIP(), geo.City, geo.Location)

	ds, derr := h.resolveDeviceSession(c, deviceSessionInput{
		PrincipalType: models.UserDevicePrincipalTenantUser,
		PrincipalID:   user.ID,
		Email:         user.Email,
		DisplayName:   user.DisplayName,
		DeviceKey:     req.DeviceID,
		ClientIP:      c.ClientIP(),
		AutoTrust:     true,
	})
	if derr != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, derr)
		return
	}

	token, tokenTTL, err := signTenantAccessToken(h.db, user, tenant, ds.SessionID, ds.DeviceRecordID)
	if err != nil {
		response.FailI18n(c, i18n.KeyTenantSignTokenFailed, nil)
		return
	}

	h.recordOpChange(c, OpLogEntry{
		TenantID: tenant.ID, Action: constants.OpActionCreate,
		Resource: constants.OpResourceTenant, ResourceID: tenant.ID, ResourceName: tenant.Name,
		Summary: fmt.Sprintf("Self-registered tenant %s", tenant.Name), Detail: audit.Redact(req),
	}, nil, tenant)
	utils.Sig().Emit(constants.SigNotifyTenantProvisioned, nil, constants.NotifyTenantProvisionedPayload{
		TenantID:         tenant.ID,
		TenantName:       tenant.Name,
		AdminUserID:      user.ID,
		AdminEmail:       user.Email,
		AdminDisplayName: user.DisplayName,
		Source:           "register",
		ClientIP:         c.ClientIP(),
	}, h.db)
	pc, _ := models.ListEffectivePermissionCodesForTenantUser(h.db, user.ID)
	models.RecordSuccessfulLoginCity(h.db, user.ID, geo.City)
	if user.WelcomeNotifiedAt == nil {
		utils.Sig().Emit(constants.SigMailWelcome, nil, constants.MailWelcomePayload{
			PrincipalType: models.UserDevicePrincipalTenantUser,
			UserID:        user.ID,
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			ReceiveEmail:  user.ReceiveEmailNotify,
			ClientIP:      c.ClientIP(),
		}, h.db)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"principal":       "tenant",
		"token":           token,
		"expiresIn":       int(tokenTTL.Seconds()),
		"tenant":          apiresponse.NewTenantResponse(tenant),
		"user":            apiresponse.NewTenantUserResponse(h.db, user),
		"permissionCodes": pc,
		"roleCreated":     constants.TenantAdminRoleName,
		"roleId":          role.ID,
	})
}

// forgotPassword resets the password for a tenant user or platform admin using a
// verified email verification code. Tries platform admins first, then tenant users.
//
// Request body (JSON):
//
//	{
//	  "email":       "user@example.com",  // required, valid email
//	  "newPassword": "newsecret123",      // required
//	  "emailCode":   "123456",            // required, email verification code
//	  "captchaId":   "...",               // optional, enabled when no totpCode
//	  "captchaType": "...",               // optional
//	  "captchaValue": "..."               // optional
//	}
func (h *Handlers) forgotPassword(c *gin.Context) {
	var req forgotPasswordReq
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
	if !verifyEmailCode(emailCodePassword, email, req.EmailCode) {
		response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
		return
	}
	hash, err := access.HashPassword(req.NewPassword)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := models.GetActivePlatformAdminByEmail(h.db, email); err == nil {
		if err := h.db.Model(&models.PlatformAdmin{}).Where("email = ?", email).
			Update("password_hash", hash).Error; err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		response.SuccessI18n(c, i18n.KeyPasswordChanged, gin.H{"email": email})
		return
	}
	user, err := models.GetActiveTenantUserByEmailGlobal(h.db, email)
	if err != nil {
		response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
		return
	}
	if _, err := models.UpdateTenantUser(h.db, user.ID, map[string]any{"password_hash": hash}, "forgot-password"); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyPasswordChanged, gin.H{"email": email})
}

// logout acknowledges the caller's sign-out intent and deactivates the current
// device session so it can no longer be used for authentication.
//
// Requires a valid access token (Authorization header).
//
// Response: { "data": { "loggedOut": true } }
func (h *Handlers) logout(c *gin.Context) {
	if sid := middleware.AuthSessionID(c); sid != "" {
		if did := middleware.AuthDeviceRecordID(c); did > 0 {
			_ = models.DeactivateUserDeviceSession(h.db, did, sid)
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"loggedOut": true})
}

type accountDeletionStatusResp struct {
	Pending           bool       `json:"pending"`
	RequestedAt       *time.Time `json:"requestedAt,omitempty"`
	ScheduledDeleteAt *time.Time `json:"scheduledDeleteAt,omitempty"`
	CoolingDays       int        `json:"coolingDays"`
}

// accountDeletionStatus builds the deletion-status response payload from a
// DeletionRequestedAt timestamp. When nil the account is not pending deletion;
// otherwise includes the scheduled deletion time (current + cooling period).
func accountDeletionStatus(requestedAt *time.Time) accountDeletionStatusResp {
	if requestedAt == nil {
		return accountDeletionStatusResp{Pending: false, CoolingDays: int(constants.AccountDeletionCoolingPeriod / (24 * time.Hour))}
	}
	scheduled := requestedAt.Add(constants.AccountDeletionCoolingPeriod)
	return accountDeletionStatusResp{
		Pending:           true,
		RequestedAt:       requestedAt,
		ScheduledDeleteAt: &scheduled,
		CoolingDays:       int(constants.AccountDeletionCoolingPeriod / (24 * time.Hour)),
	}
}

// getAccountDeletionStatus returns the current deletion request status for the
// authenticated caller (platform admin or tenant user).
//
// Requires a valid access token.
//
// Response: { "data": { "pending": bool, "coolingDays": int, ... } }
func (h *Handlers) getAccountDeletionStatus(c *gin.Context) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		row, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, accountDeletionStatus(row.DeletionRequestedAt))
		return
	}
	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, accountDeletionStatus(u.DeletionRequestedAt))
}

// requestAccountDeletion initiates account deletion for the authenticated caller.
// The account enters a pending-deletion cooling period (soft delete). It can be
// cancelled within the cooling period before the final hard delete.
//
// Supports two authentication methods:
//   - "password" (default): requires the current password
//   - "email_code": requires an email verification code
//
// Requires a valid access token.
//
// Response: { "data": { "pending": true, "coolingDays": ..., ... } }
func (h *Handlers) requestAccountDeletion(c *gin.Context) {
	var req dto.AccountDeletionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	now := time.Now()
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		row, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if row.DeletionRequestedAt != nil {
			response.FailI18n(c, i18n.KeyAccountDeletionAlreadyPending, nil)
			return
		}
		method := strings.TrimSpace(req.Method)
		if method == "" {
			method = "password"
		}
		if method == "email_code" {
			if !verifyEmailCode(emailCodePassword, utils.TrimLower(row.Email), req.EmailCode) {
				response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyAuthInvalidEmailCode))
				return
			}
		} else {
			if !access.CheckPassword(row.PasswordHash, req.Password) {
				response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyPasswordWrong))
				return
			}
		}
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).Updates(map[string]any{
			"status":                constants.PlatformAdminStatusPendingDeletion,
			"deletion_requested_at": now,
		}).Error; err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		response.SuccessI18n(c, i18n.KeyAccountDeletionRequested, accountDeletionStatus(&now))
		return
	}

	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if u.DeletionRequestedAt != nil {
		response.FailI18n(c, i18n.KeyAccountDeletionAlreadyPending, nil)
		return
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = "password"
	}
	if method == "email_code" {
		if !verifyEmailCode(emailCodePassword, utils.TrimLower(u.Email), req.EmailCode) {
			response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyAuthInvalidEmailCode))
			return
		}
	} else {
		if !access.CheckPassword(u.PasswordHash, req.Password) {
			response.Render(c, response.NewI18n(response.CodeAuthFailed, i18n.KeyPasswordWrong))
			return
		}
	}
	if _, err := models.UpdateTenantUser(h.db, u.ID, map[string]any{
		"status":                constants.TenantUserStatusPendingDeletion,
		"deletion_requested_at": now,
	}, middleware.AuditOperator(c)); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyAccountDeletionRequested, accountDeletionStatus(&now))
}

// resolvePendingDeletionAccount looks up an account that is currently in the
// pending-deletion state by email. Returns the account info and true when found.
// Searches tenant users first, then platform admins.
func resolvePendingDeletionAccount(db *gorm.DB, email string) (emailAccount, bool) {
	email = utils.TrimLower(email)
	var u models.TenantUser
	if err := db.Where("email = ? AND status = ?", email, constants.TenantUserStatusPendingDeletion).First(&u).Error; err == nil {
		return emailAccount{UserID: u.ID, TenantID: u.TenantID, Email: email}, true
	}
	var a models.PlatformAdmin
	if err := db.Where("email = ? AND status = ?", email, constants.PlatformAdminStatusPendingDeletion).First(&a).Error; err == nil {
		return emailAccount{UserID: a.ID, TenantID: 0, Email: email}, true
	}
	return emailAccount{}, false
}

// cancelDeletionForAccount restores a pending-deletion account back to active
// status, clearing the DeletionRequestedAt timestamp. Handles both platform
// admins (TenantID == 0) and tenant users.
func cancelDeletionForAccount(db *gorm.DB, acct emailAccount, updateBy string) error {
	if acct.TenantID == 0 {
		return db.Model(&models.PlatformAdmin{}).
			Where("id = ? AND status = ?", acct.UserID, constants.PlatformAdminStatusPendingDeletion).
			Updates(map[string]any{
				"status":                constants.PlatformAdminStatusActive,
				"deletion_requested_at": nil,
			}).Error
	}
	_, err := models.UpdateTenantUser(db, acct.UserID, map[string]any{
		"status":                constants.TenantUserStatusActive,
		"deletion_requested_at": nil,
	}, updateBy)
	return err
}

// sendRevokeDeletionEmailCode dispatches an email verification code to the
// address of a pending-deletion account so the owner can cancel the deletion
// via the public revoke endpoint.
//
// Request body: { "email": "user@example.com" }
func (h *Handlers) sendRevokeDeletionEmailCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	acct, ok := resolvePendingDeletionAccount(h.db, req.Email)
	if !ok {
		response.FailI18n(c, i18n.KeyAccountDeletionNotPending, nil)
		return
	}
	if !h.dispatchEmailVerificationCode(c, emailCodeDeletionCancel, acct) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthEmailCodeSent, nil)
}

// revokeAccountDeletionPublic cancels a pending deletion via email verification
// code (public endpoint, no auth token required). The email must match an account
// currently in pending-deletion status.
//
// Request body: { "email": "user@example.com", "emailCode": "123456" }
func (h *Handlers) revokeAccountDeletionPublic(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		EmailCode string `json:"emailCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	email := utils.TrimLower(req.Email)
	if !verifyEmailCode(emailCodeDeletionCancel, email, req.EmailCode) {
		response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
		return
	}
	acct, ok := resolvePendingDeletionAccount(h.db, email)
	if !ok {
		response.FailI18n(c, i18n.KeyAccountDeletionNotPending, nil)
		return
	}
	if err := cancelDeletionForAccount(h.db, acct, "deletion_revoke"); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyAccountDeletionCancelled, accountDeletionStatus(nil))
}

// cancelAccountDeletion cancels the pending deletion for the authenticated
// caller (platform admin or tenant user), restoring the account to active status.
//
// Requires a valid access token. The account must currently be in pending-deletion status.
//
// Response: { "data": { "pending": false, ... } }
func (h *Handlers) cancelAccountDeletion(c *gin.Context) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		row, err := models.GetPlatformAdminByID(h.db, aid)
		if err != nil {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if row.DeletionRequestedAt == nil {
			response.FailI18n(c, i18n.KeyAccountDeletionNotPending, nil)
			return
		}
		if err := cancelDeletionForAccount(h.db, emailAccount{UserID: row.ID, TenantID: 0, Email: utils.TrimLower(row.Email)}, middleware.AuditOperator(c)); err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		response.SuccessI18n(c, i18n.KeyAccountDeletionCancelled, accountDeletionStatus(nil))
		return
	}

	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if u.DeletionRequestedAt == nil {
		response.FailI18n(c, i18n.KeyAccountDeletionNotPending, nil)
		return
	}
	if err := cancelDeletionForAccount(h.db, emailAccount{UserID: u.ID, TenantID: u.TenantID, Email: utils.TrimLower(u.Email)}, middleware.AuditOperator(c)); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyAccountDeletionCancelled, accountDeletionStatus(nil))
}

func loginMethodOrDefault(raw string) string {
	if m := strings.TrimSpace(raw); m != "" {
		return m
	}
	return "password"
}

func (h *Handlers) tenantLoginByPhone(c *gin.Context, req dto.TenantLoginReq) {
	clientIP := c.ClientIP()
	geo := utils.LoginGeoFromIP(clientIP)
	phone := utils.NormalizePhone(req.Phone)
	deviceKey := strings.TrimSpace(req.DeviceID)
	loginMethod := "phone_code"
	lockKey := "phone:" + phone

	if phone == "" {
		response.FailI18n(c, i18n.KeyInvalidParams, "phone required")
		return
	}
	if utils.CheckLoginAccountLocked(lockKey) {
		h.recordLoginFailure(c, models.LoginHistoryInput{
			Email: phone, ClientIP: clientIP, City: geo.City, Location: geo.Location,
			LoginMethod: loginMethod, DeviceKey: deviceKey, FailureReason: "account_locked",
		}, "account_locked")
		response.FailI18n(c, i18n.KeyAuthAccountLocked, nil)
		return
	}
	if strings.TrimSpace(req.TotpCode) == "" {
		if err := captcha.ValidatePayload(req.CaptchaID, req.CaptchaType, req.CaptchaValue); err != nil {
			utils.RecordLoginFailure(lockKey, clientIP)
			if err == captcha.ErrPayloadRequired {
				response.FailI18n(c, i18n.KeyValidationCaptchaRequired, nil)
			} else {
				response.FailI18n(c, i18n.KeyValidationCaptchaInvalid, nil)
			}
			return
		}
	}
	if bootstrap.GlobalKeyManager == nil {
		response.FailI18n(c, i18n.KeyAuthJWTNotReady, nil)
		return
	}

	user, err := models.GetActiveTenantUserByPhoneGlobal(h.db, phone)
	if err != nil {
		utils.RecordLoginFailure(lockKey, clientIP)
		h.recordLoginFailure(c, models.LoginHistoryInput{
			Email: phone, ClientIP: clientIP, City: geo.City, Location: geo.Location,
			LoginMethod: loginMethod, DeviceKey: deviceKey,
		}, "invalid_credentials")
		response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
		return
	}
	tenant, terr := models.GetActiveTenantByID(h.db, user.TenantID)
	if terr != nil {
		response.FailI18n(c, i18n.KeyTenantNotFound, nil)
		return
	}
	if tenant.Status != "" && tenant.Status != "active" {
		response.FailI18n(c, i18n.KeyTenantSuspended, nil)
		return
	}
	if user.Status != constants.TenantUserStatusActive {
		if user.Status == constants.TenantUserStatusPendingDeletion {
			response.FailI18n(c, i18n.KeyAccountDeletionPending, gin.H{"revokePath": "/account/deletion/revoke"})
			return
		}
		response.FailI18n(c, i18n.KeyTenantUserUnavailable, nil)
		return
	}
	if !verifySMSCode(emailCodeLogin, phone, req.SMSCode) {
		utils.RecordLoginFailure(lockKey, clientIP)
		h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_sms_code")
		response.FailI18n(c, i18n.KeyAuthInvalidSMSCode, nil)
		return
	}
	if user.TOTPEnabled && strings.TrimSpace(user.TOTPSecret) != "" {
		code := strings.TrimSpace(req.TotpCode)
		if code == "" {
			response.FailI18n(c, i18n.KeyAuthNeedsTotp, gin.H{"needsTotp": true})
			return
		}
		if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalTenantUser, user.ID, user.TOTPSecret, user.TOTPRecoveryHashes, code) {
			utils.RecordLoginFailure(lockKey, clientIP)
			h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_totp")
			response.FailI18n(c, i18n.KeyAuthInvalidTotp, gin.H{"needsTotp": true})
			return
		}
	}
	_ = models.RecordTenantUserLogin(h.db, user.ID, clientIP, geo.City, geo.Location)
	ds, derr := h.resolveDeviceSession(c, deviceSessionInput{
		PrincipalType:            models.UserDevicePrincipalTenantUser,
		PrincipalID:              user.ID,
		TenantID:                 user.TenantID,
		Email:                    user.Email,
		DisplayName:              user.DisplayName,
		DeviceKey:                req.DeviceID,
		DeviceCode:               req.DeviceVerifyCode,
		VoiceprintAudioBase64:    req.VoiceprintAudioBase64,
		ClientIP:                 clientIP,
		RequireDeviceVerify:      user.RequireDeviceVerify,
		TrustDeviceLoginEnabled:  user.TrustDeviceLoginEnabled,
		TrustDeviceFor7Days:      req.TrustDeviceFor7Days,
		RequireRemoteLoginVerify: user.RequireRemoteLoginVerify,
		KnownLoginCities:         utils.ParseKnownLoginCities(user.KnownLoginCitiesJSON),
	})
	if derr != nil {
		if errors.Is(derr, errInvalidDeviceCode) {
			utils.RecordLoginFailure(lockKey, clientIP)
			h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_device_code")
			response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
			return
		}
		if errors.Is(derr, errInvalidVoiceprintVerify) {
			utils.RecordLoginFailure(lockKey, clientIP)
			h.recordTenantLoginFailure(c, user, loginMethod, deviceKey, clientIP, geo, "invalid_voiceprint")
			data := gin.H{"needsDeviceVerify": true, "invalidVoiceprint": true}
			if _, _, vpOK := voiceprintconfig.ResolveEnabled(); vpOK && models.TenantUserHasVoiceprint(user) {
				data["voiceprintAvailable"] = true
			}
			response.FailI18n(c, i18n.KeyAuthInvalidVoiceprint, data)
			return
		}
		response.AbortWithStatusJSON(c, 500, derr)
		return
	}
	if ds.NeedsDeviceVerify || ds.NeedsRemoteVerify {
		data := gin.H{}
		if ds.NeedsDeviceVerify {
			data["needsDeviceVerify"] = true
		}
		if ds.NeedsRemoteVerify {
			data["needsRemoteVerify"] = true
		}
		if _, _, vpOK := voiceprintconfig.ResolveEnabled(); vpOK && models.TenantUserHasVoiceprint(user) {
			data["voiceprintAvailable"] = true
		}
		key := i18n.KeyAuthNeedsDeviceVerify
		if ds.NeedsRemoteVerify && !ds.NeedsDeviceVerify {
			key = i18n.KeyAuthNeedsRemoteVerify
		}
		response.FailI18n(c, key, data)
		return
	}
	token, tokenTTL, terr := signTenantAccessToken(h.db, user, tenant, ds.SessionID, ds.DeviceRecordID)
	if terr != nil {
		response.FailI18n(c, i18n.KeyTenantSignTokenFailed, nil)
		return
	}
	codes, _ := models.ListEffectivePermissionCodesForTenantUser(h.db, user.ID)
	utils.ClearLoginFailures(lockKey)
	models.RecordSuccessfulLoginCity(h.db, user.ID, geo.City)
	h.recordLoginSuccess(c, models.LoginHistoryInput{
		PrincipalType: models.LoginHistoryPrincipalTenantUser,
		PrincipalID:   user.ID,
		TenantID:      user.TenantID,
		Email:         user.Email,
		ClientIP:      clientIP,
		City:          geo.City,
		Location:      geo.Location,
		LoginMethod:   loginMethod,
		DeviceKey:     deviceKey,
	})
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"principal":       "tenant",
		"token":           token,
		"expiresIn":       int(tokenTTL.Seconds()),
		"tenant":          apiresponse.NewTenantResponse(tenant),
		"user":            apiresponse.NewTenantUserResponse(h.db, user),
		"permissionCodes": codes,
	})
}

func (h *Handlers) recordTenantLoginFailure(c *gin.Context, user models.TenantUser, loginMethod, deviceKey, clientIP string, geo utils.LoginGeo, reason string) {
	h.recordLoginFailure(c, models.LoginHistoryInput{
		PrincipalType: models.LoginHistoryPrincipalTenantUser,
		PrincipalID:   user.ID,
		TenantID:      user.TenantID,
		Email:         user.Email,
		ClientIP:      clientIP,
		City:          geo.City,
		Location:      geo.Location,
		LoginMethod:   loginMethod,
		DeviceKey:     deviceKey,
	}, reason)
}

func (h *Handlers) sendChangeEmailCode(c *gin.Context) {
	var req struct {
		NewEmail string `json:"newEmail" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	newEmail := utils.TrimLower(req.NewEmail)
	if _, ok := resolveEmailAccount(h.db, newEmail); ok {
		response.Render(c, utils.ErrEmailExists)
		return
	}
	acct, ok := h.authenticatedEmailAccount(c)
	if !ok {
		return
	}
	if newEmail == acct.Email {
		response.Render(c, utils.ErrSameEmail)
		return
	}
	pending := emailAccount{UserID: acct.UserID, TenantID: 0, Email: newEmail}
	if !h.dispatchEmailVerificationCode(c, emailCodeChangeEmail, pending) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthEmailCodeSent, nil)
}

// changeMyEmail updates the authenticated caller's email address after
// verifying the email code sent to the new address.
//
// Request body: { "newEmail": "new@example.com", "emailCode": "123456" }
// Requires a valid access token.
//
// Response: updated user/admin profile.
func (h *Handlers) changeMyEmail(c *gin.Context) {
	var req dto.ChangeEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	newEmail := utils.TrimLower(req.NewEmail)
	if _, ok := resolveEmailAccount(h.db, newEmail); ok {
		response.Render(c, utils.ErrEmailExists)
		return
	}
	if !verifyEmailCode(emailCodeChangeEmail, newEmail, req.EmailCode) {
		response.FailI18n(c, i18n.KeyAuthInvalidEmailCode, nil)
		return
	}

	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).Update("email", newEmail).Error; err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		row, _ := models.GetPlatformAdminByID(h.db, aid)
		response.SuccessI18n(c, i18n.KeyAuthEmailChanged, apiresponse.NewPlatformAdminResponse(row))
		return
	}

	u, err := models.GetAuthenticatedTenantUser(h.db, middleware.AuthUserID(c), middleware.AuthTenantID(c))
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if _, err := models.UpdateTenantUser(h.db, u.ID, map[string]any{"email": newEmail}, middleware.AuditOperator(c)); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	after, _ := models.GetActiveTenantUserByID(h.db, u.ID)
	response.SuccessI18n(c, i18n.KeyAuthEmailChanged, apiresponse.NewTenantUserResponse(h.db, after))
}
