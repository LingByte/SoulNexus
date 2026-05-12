// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Passkey / WebAuthn 控制器：
//
// 已登录用户管理：
//   GET    /api/me/passkeys                          列出
//   POST   /api/me/passkeys/registration/begin       下发注册 options
//   POST   /api/me/passkeys/registration/finish      完成注册
//   PUT    /api/me/passkeys/:id                      改昵称
//   DELETE /api/me/passkeys/:id                      删除
//
// 无密码登录（discoverable）：
//   POST   /api/auth/passkey/begin                   下发认证 options
//   POST   /api/auth/passkey/finish                  校验后签发 JWT 并写 session

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	webauthnsvc "github.com/LingByte/SoulNexus/pkg/auth/webauthn"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// passkeyServiceFromEnv 从环境变量读 RPID / RPOrigins / RPDisplayName 构造 Service。
//   - WEBAUTHN_RP_ID            站点 effective domain，如 example.com（无 https/端口）
//   - WEBAUTHN_RP_ORIGINS       逗号分隔，如 https://example.com,https://app.example.com
//   - WEBAUTHN_RP_DISPLAY_NAME  弹窗里显示的应用名
func passkeyServiceFromEnv() (*webauthnsvc.Service, error) {
	rpid := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ID"))
	if rpid == "" {
		return nil, errors.New("WEBAUTHN_RP_ID 未配置；请在 env 中设置站点 effective domain（如 example.com）")
	}
	originsRaw := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ORIGINS"))
	if originsRaw == "" {
		return nil, errors.New("WEBAUTHN_RP_ORIGINS 未配置；逗号分隔，如 https://example.com")
	}
	parts := strings.Split(originsRaw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			origins = append(origins, s)
		}
	}
	display := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_DISPLAY_NAME"))
	if display == "" {
		display = "SoulNexus"
	}
	return webauthnsvc.New(webauthnsvc.Config{
		RPID:          rpid,
		RPDisplayName: display,
		Origins:       origins,
	})
}

// passkeyUserAdapter 让 models.User + 已注册 Passkey 列表实现 webauthnsvc.User。
type passkeyUserAdapter struct {
	user        *models.User
	credentials []webauthn.Credential
}

func newPasskeyUserAdapter(db *gorm.DB, user *models.User) (*passkeyUserAdapter, error) {
	if user == nil {
		return nil, errors.New("user nil")
	}
	rows, err := models.ListPasskeysForUser(db, user.ID)
	if err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for i := range rows {
		c, err := webauthnsvc.CredentialFromBytes(rows[i].PublicKey)
		if err != nil {
			continue
		}
		creds = append(creds, *c)
	}
	return &passkeyUserAdapter{user: user, credentials: creds}, nil
}

func (a *passkeyUserAdapter) WebAuthnID() []byte {
	return []byte(strconv.FormatUint(uint64(a.user.ID), 10))
}
func (a *passkeyUserAdapter) WebAuthnName() string {
	if s := strings.TrimSpace(a.user.Email); s != "" {
		return s
	}
	return fmt.Sprintf("user-%d", a.user.ID)
}
func (a *passkeyUserAdapter) WebAuthnDisplayName() string { return a.WebAuthnName() }
func (a *passkeyUserAdapter) WebAuthnIcon() string        { return "" }
func (a *passkeyUserAdapter) WebAuthnCredentials() []webauthn.Credential {
	return a.credentials
}

// emptyPasskeyUser 用于 discoverable login（不知道用户是谁）。
type emptyPasskeyUser struct{}

func (emptyPasskeyUser) WebAuthnID() []byte                         { return []byte{} }
func (emptyPasskeyUser) WebAuthnName() string                       { return "" }
func (emptyPasskeyUser) WebAuthnDisplayName() string                { return "" }
func (emptyPasskeyUser) WebAuthnIcon() string                       { return "" }
func (emptyPasskeyUser) WebAuthnCredentials() []webauthn.Credential { return nil }

// =================== Me handlers ===================

// handleMeListPasskeys GET /api/me/passkeys
func (h *Handlers) handleMeListPasskeys(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	rows, err := models.ListPasskeysForUser(h.db, user.ID)
	if err != nil {
		response.Fail(c, "list passkeys failed", err)
		return
	}
	response.Success(c, "passkeys fetched", gin.H{"items": rows})
}

// handleMePasskeyRegistrationBegin POST /api/me/passkeys/registration/begin
func (h *Handlers) handleMePasskeyRegistrationBegin(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	svc, err := passkeyServiceFromEnv()
	if err != nil {
		response.Fail(c, "passkey not configured", err)
		return
	}
	adapter, err := newPasskeyUserAdapter(h.db, user)
	if err != nil {
		response.Fail(c, "load user passkeys failed", err)
		return
	}
	options, sessionData, err := svc.BeginRegistration(adapter)
	if err != nil {
		response.Fail(c, "begin registration failed", err)
		return
	}
	chid := uuid.NewString()
	if err := models.SavePasskeyChallenge(h.db, &models.PasskeyChallenge{
		ID:          chid,
		UserID:      user.ID,
		Type:        "registration",
		SessionData: sessionData,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}); err != nil {
		response.Fail(c, "save challenge failed", err)
		return
	}
	var optsAny any
	_ = json.Unmarshal(options, &optsAny)
	response.Success(c, "options generated", gin.H{
		"session_id": chid,
		"public_key": optsAny,
	})
}

// handleMePasskeyRegistrationFinish POST /api/me/passkeys/registration/finish
type passkeyRegistrationFinishReq struct {
	SessionID   string          `json:"session_id" binding:"required"`
	Nickname    string          `json:"nickname"`
	Attestation json.RawMessage `json:"attestation" binding:"required"`
}

func (h *Handlers) handleMePasskeyRegistrationFinish(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	var req passkeyRegistrationFinishReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	ch, err := models.LoadPasskeyChallenge(h.db, req.SessionID)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("invalid session: %w", err))
		return
	}
	if ch.Type != "registration" || ch.UserID != user.ID {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("session/user mismatch"))
		return
	}
	svc, err := passkeyServiceFromEnv()
	if err != nil {
		response.Fail(c, "passkey not configured", err)
		return
	}
	adapter, err := newPasskeyUserAdapter(h.db, user)
	if err != nil {
		response.Fail(c, "load user passkeys failed", err)
		return
	}
	cred, err := svc.FinishRegistration(adapter, ch.SessionData, req.Attestation)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("finish registration: %w", err))
		return
	}
	credBlob, err := webauthnsvc.CredentialToBytes(cred)
	if err != nil {
		response.Fail(c, "marshal credential failed", err)
		return
	}
	row := &models.Passkey{
		UserID:          user.ID,
		CredentialID:    base64.RawURLEncoding.EncodeToString(cred.ID),
		PublicKey:       credBlob,
		AAGUID:          cred.Authenticator.AAGUID,
		SignCount:       cred.Authenticator.SignCount,
		Transports:      strings.Join(transportsToStrings(cred.Transport), ","),
		AttestationType: cred.AttestationType,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
		UserPresent:     cred.Flags.UserPresent,
		UserVerified:    cred.Flags.UserVerified,
		Nickname:        strings.TrimSpace(req.Nickname),
	}
	if err := h.db.Create(row).Error; err != nil {
		response.Fail(c, "save passkey failed", err)
		return
	}
	response.Success(c, "passkey registered", gin.H{"passkey": row})
}

// handleMeUpdatePasskey PUT /api/me/passkeys/:id
func (h *Handlers) handleMeUpdatePasskey(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	res := h.db.Model(&models.Passkey{}).
		Where("id = ? AND user_id = ?", id, user.ID).
		Update("nickname", strings.TrimSpace(req.Nickname))
	if res.Error != nil {
		response.Fail(c, "update passkey failed", res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.AbortWithJSONError(c, http.StatusNotFound, errors.New("passkey not found"))
		return
	}
	response.Success(c, "passkey updated", nil)
}

// handleMeDeletePasskey DELETE /api/me/passkeys/:id
func (h *Handlers) handleMeDeletePasskey(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := models.DeletePasskey(h.db, user.ID, uint(id)); err != nil {
		response.Fail(c, "delete passkey failed", err)
		return
	}
	response.Success(c, "passkey deleted", nil)
}

// =================== Discoverable login ===================

// handleAuthPasskeyBegin POST /api/auth/passkey/begin
func (h *Handlers) handleAuthPasskeyBegin(c *gin.Context) {
	svc, err := passkeyServiceFromEnv()
	if err != nil {
		response.Fail(c, "passkey not configured", err)
		return
	}
	options, sessionData, err := svc.BeginLogin(emptyPasskeyUser{})
	if err != nil {
		response.Fail(c, "begin login failed", err)
		return
	}
	chid := uuid.NewString()
	if err := models.SavePasskeyChallenge(h.db, &models.PasskeyChallenge{
		ID:          chid,
		UserID:      0,
		Type:        "login",
		SessionData: sessionData,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}); err != nil {
		response.Fail(c, "save challenge failed", err)
		return
	}
	var optsAny any
	_ = json.Unmarshal(options, &optsAny)
	response.Success(c, "options generated", gin.H{
		"session_id": chid,
		"public_key": optsAny,
	})
}

// handleAuthPasskeyFinish POST /api/auth/passkey/finish
type passkeyLoginFinishReq struct {
	SessionID string          `json:"session_id" binding:"required"`
	Assertion json.RawMessage `json:"assertion" binding:"required"`
}

func (h *Handlers) handleAuthPasskeyFinish(c *gin.Context) {
	var req passkeyLoginFinishReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	ch, err := models.LoadPasskeyChallenge(h.db, req.SessionID)
	if err != nil || ch.Type != "login" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid session"))
		return
	}

	// 解析 assertion 拿到 credentialId 来定位用户。
	var probe struct {
		ID    string `json:"id"`
		RawID string `json:"rawId"`
	}
	if err := json.Unmarshal(req.Assertion, &probe); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("parse assertion: %w", err))
		return
	}
	credID := strings.TrimSpace(probe.ID)
	if credID == "" {
		credID = strings.TrimSpace(probe.RawID)
	}
	if credID == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("missing credential id"))
		return
	}
	row, err := models.FindPasskeyByCredentialID(h.db, credID)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("passkey not registered"))
		return
	}
	user, err := models.GetUserByID(h.db, row.UserID)
	if err != nil || user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("user not found"))
		return
	}
	svc, err := passkeyServiceFromEnv()
	if err != nil {
		response.Fail(c, "passkey not configured", err)
		return
	}
	adapter, err := newPasskeyUserAdapter(h.db, user)
	if err != nil {
		response.Fail(c, "load user passkeys failed", err)
		return
	}
	cred, err := svc.FinishLogin(adapter, ch.SessionData, req.Assertion)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, fmt.Errorf("login: %w", err))
		return
	}
	// 更新 sign_count + last used。
	credBlob, _ := webauthnsvc.CredentialToBytes(cred)
	now := time.Now()
	_ = h.db.Model(&models.Passkey{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"public_key":   credBlob,
		"sign_count":   cred.Authenticator.SignCount,
		"last_used_at": &now,
		"last_used_ip": c.ClientIP(),
	}).Error

	// 写 session 并签 JWT（与密码登录路径一致）。
	models.Login(c, user)
	if c.IsAborted() {
		return
	}
	expired := authTokenTTLFromDB(h.db, 7*24*time.Hour)
	accessToken, refreshToken, err := buildTokenPair(h.db, user, expired)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}
	user.AuthToken = accessToken
	response.Success(c, "passkey login ok", gin.H{
		"user":         user,
		"token":        accessToken,
		"refreshToken": refreshToken,
	})
}

// transportsToStrings 把 protocol.AuthenticatorTransport 转为字符串列表。
func transportsToStrings(ts []protocol.AuthenticatorTransport) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		s := strings.TrimSpace(string(t))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
