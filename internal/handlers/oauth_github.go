package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	llmcache "github.com/LingByte/lingllm/cache"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"gorm.io/gorm"
)

const (
	githubOAuthStateTTL  = 10 * time.Minute
	githubOAuthTicketTTL = 5 * time.Minute
)

type githubOAuthState struct {
	Mode          string `json:"mode"` // login | bind
	PrincipalType string `json:"principalType,omitempty"`
	PrincipalID   uint   `json:"principalId,omitempty"`
	TenantID      uint   `json:"tenantId,omitempty"`
	Redirect      string `json:"redirect,omitempty"`
	DeviceKey     string `json:"deviceKey,omitempty"`
}

type githubOAuthTicket struct {
	PrincipalType string `json:"principalType"`
	PrincipalID   uint   `json:"principalId"`
	TenantID      uint   `json:"tenantId,omitempty"`
	DeviceKey     string `json:"deviceKey,omitempty"`
	NeedsTotp     bool   `json:"needsTotp"`
}

type githubUserProfile struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type oauthGitHubExchangeReq struct {
	Ticket   string `json:"ticket" binding:"required"`
	TotpCode string `json:"totpCode"`
}

func (h *Handlers) githubOAuthEnabled() bool {
	return h.githubOAuthConfig() != nil
}

func (h *Handlers) githubOAuthConfig() *oauth2.Config {
	if config.GlobalConfig == nil {
		return nil
	}
	clientID := strings.TrimSpace(config.GlobalConfig.Auth.GitHubClientID)
	clientSecret := strings.TrimSpace(config.GlobalConfig.Auth.GitHubClientSecret)
	if clientID == "" || clientSecret == "" {
		return nil
	}
	redirectURL := strings.TrimSpace(config.GlobalConfig.Auth.GitHubRedirectURL)
	if redirectURL == "" {
		base := strings.TrimRight(strings.TrimSpace(config.GlobalConfig.Server.URL), "/")
		prefix := strings.TrimRight(config.GlobalConfig.Server.APIPrefix, "/")
		if base == "" {
			base = "http://localhost:8082"
		}
		redirectURL = base + prefix + "/oauth/github/callback"
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}
}

func oauthFrontendBaseURL() string {
	if config.GlobalConfig != nil {
		if v := strings.TrimRight(strings.TrimSpace(config.GlobalConfig.Auth.OAuthFrontendBaseURL), "/"); v != "" {
			return v
		}
		if v := strings.TrimRight(strings.TrimSpace(config.GlobalConfig.Server.URL), "/"); v != "" {
			return v
		}
	}
	return "http://localhost:5173"
}

func githubOAuthStateCacheKey(token string) string {
	return "oauth:github:state:" + token
}

func githubOAuthTicketCacheKey(ticket string) string {
	return "oauth:github:ticket:" + ticket
}

func randomOAuthToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *Handlers) storeGitHubOAuthState(st githubOAuthState) string {
	token := randomOAuthToken(24)
	b, _ := json.Marshal(st)
	_ = llmcache.Set(context.Background(), githubOAuthStateCacheKey(token), string(b), githubOAuthStateTTL)
	return token
}

func loadGitHubOAuthState(token string) (githubOAuthState, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return githubOAuthState{}, false
	}
	v, ok := llmcache.Get(context.Background(), githubOAuthStateCacheKey(token))
	if !ok || v == nil {
		return githubOAuthState{}, false
	}
	s, _ := v.(string)
	var st githubOAuthState
	if err := json.Unmarshal([]byte(s), &st); err != nil {
		return githubOAuthState{}, false
	}
	_ = llmcache.Delete(context.Background(), githubOAuthStateCacheKey(token))
	return st, true
}

func (h *Handlers) storeGitHubOAuthTicket(t githubOAuthTicket) string {
	id := randomOAuthToken(24)
	b, _ := json.Marshal(t)
	_ = llmcache.Set(context.Background(), githubOAuthTicketCacheKey(id), string(b), githubOAuthTicketTTL)
	return id
}

func loadGitHubOAuthTicket(id string) (githubOAuthTicket, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return githubOAuthTicket{}, false
	}
	v, ok := llmcache.Get(context.Background(), githubOAuthTicketCacheKey(id))
	if !ok || v == nil {
		return githubOAuthTicket{}, false
	}
	s, _ := v.(string)
	var t githubOAuthTicket
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return githubOAuthTicket{}, false
	}
	return t, true
}

func deleteGitHubOAuthTicket(id string) {
	_ = llmcache.Delete(context.Background(), githubOAuthTicketCacheKey(strings.TrimSpace(id)))
}

func fetchGitHubUser(ctx context.Context, accessToken string) (githubUserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return githubUserProfile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubUserProfile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return githubUserProfile{}, fmt.Errorf("github user api: %s", strings.TrimSpace(string(body)))
	}
	var profile githubUserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return githubUserProfile{}, err
	}
	if profile.ID <= 0 {
		return githubUserProfile{}, fmt.Errorf("github user id missing")
	}
	return profile, nil
}

func githubIDString(id int64) string {
	return fmt.Sprintf("%d", id)
}

func (h *Handlers) redirectOAuthFrontend(c *gin.Context, path string, query url.Values) {
	base := oauthFrontendBaseURL()
	if query == nil {
		query = url.Values{}
	}
	target := base + path
	if enc := query.Encode(); enc != "" {
		target += "?" + enc
	}
	c.Redirect(http.StatusFound, target)
}

// startGitHubOAuthLogin redirects the browser to GitHub authorization.
func (h *Handlers) startGitHubOAuthLogin(c *gin.Context) {
	cfg := h.githubOAuthConfig()
	if cfg == nil {
		response.FailI18n(c, i18n.KeyServiceUnavailable, nil)
		return
	}
	st := h.storeGitHubOAuthState(githubOAuthState{
		Mode:      "login",
		Redirect:  strings.TrimSpace(c.Query("redirect")),
		DeviceKey: strings.TrimSpace(c.Query("deviceId")),
	})
	c.Redirect(http.StatusFound, cfg.AuthCodeURL(st, oauth2.AccessTypeOnline))
}

// startGitHubOAuthBind returns an authorize URL for the signed-in user to bind GitHub.
func (h *Handlers) startGitHubOAuthBind(c *gin.Context) {
	cfg := h.githubOAuthConfig()
	if cfg == nil {
		response.FailI18n(c, i18n.KeyServiceUnavailable, nil)
		return
	}
	st := githubOAuthState{Mode: "bind", Redirect: "/profile?tab=security"}
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		st.PrincipalType = models.UserDevicePrincipalPlatformAdmin
		st.PrincipalID = aid
	} else {
		st.PrincipalType = models.UserDevicePrincipalTenantUser
		st.PrincipalID = middleware.AuthUserID(c)
		st.TenantID = middleware.AuthTenantID(c)
	}
	state := h.storeGitHubOAuthState(st)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"authorizeUrl": cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)})
}

// githubOAuthCallback handles GitHub redirect and forwards the user to the SPA with a ticket.
func (h *Handlers) githubOAuthCallback(c *gin.Context) {
	cfg := h.githubOAuthConfig()
	if cfg == nil {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {"unavailable"}})
		return
	}
	if errMsg := strings.TrimSpace(c.Query("error")); errMsg != "" {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {errMsg}})
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	stateToken := strings.TrimSpace(c.Query("state"))
	if code == "" || stateToken == "" {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {"invalid_callback"}})
		return
	}
	st, ok := loadGitHubOAuthState(stateToken)
	if !ok {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {"state_expired"}})
		return
	}
	tok, err := cfg.Exchange(c.Request.Context(), code)
	if err != nil {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {"exchange_failed"}})
		return
	}
	profile, err := fetchGitHubUser(c.Request.Context(), tok.AccessToken)
	if err != nil {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {"profile_failed"}})
		return
	}
	ghID := githubIDString(profile.ID)
	ghLogin := strings.TrimSpace(profile.Login)

	if st.Mode == "bind" {
		if err := h.bindGitHubAccount(st, ghID, ghLogin); err != nil {
			h.redirectOAuthFrontend(c, "/profile", url.Values{"tab": {"security"}, "oauth_error": {"bind_failed"}})
			return
		}
		h.redirectOAuthFrontend(c, "/profile", url.Values{"tab": {"security"}, "oauth": {"bound"}})
		return
	}

	ticket, needsTotp, err := h.prepareGitHubLoginTicket(c, ghID, st.DeviceKey)
	if err != nil {
		h.redirectOAuthFrontend(c, "/login", url.Values{"oauth_error": {"not_linked"}})
		return
	}
	q := url.Values{"oauth_ticket": {ticket}}
	if needsTotp {
		q.Set("oauth_needs_totp", "1")
	}
	h.redirectOAuthFrontend(c, "/login", q)
}

func (h *Handlers) bindGitHubAccount(st githubOAuthState, ghID, ghLogin string) error {
	if st.PrincipalID == 0 {
		return fmt.Errorf("missing principal")
	}
	if otherUser, err := models.GetActiveTenantUserByGitHubID(h.db, ghID); err == nil && otherUser.ID != st.PrincipalID {
		return fmt.Errorf("github already bound")
	}
	if otherAdmin, err := models.GetActivePlatformAdminByGitHubID(h.db, ghID); err == nil && otherAdmin.ID != st.PrincipalID {
		return fmt.Errorf("github already bound")
	}
	updates := map[string]any{
		"github_id":    ghID,
		"github_login": ghLogin,
	}
	switch st.PrincipalType {
	case models.UserDevicePrincipalPlatformAdmin:
		return h.db.Model(&models.PlatformAdmin{}).Where("id = ?", st.PrincipalID).Updates(updates).Error
	default:
		updates["source"] = constants.TenantUserSourceGitHub
		_, err := models.UpdateTenantUser(h.db, st.PrincipalID, updates, "oauth-github")
		return err
	}
}

func (h *Handlers) prepareGitHubLoginTicket(c *gin.Context, ghID, deviceKey string) (string, bool, error) {
	user, err := models.GetActiveTenantUserByGitHubID(h.db, ghID)
	if err == nil {
		tenant, terr := models.GetActiveTenantByID(h.db, user.TenantID)
		if terr != nil {
			return "", false, terr
		}
		if tenant.Status != "" && tenant.Status != constants.TenantStatusActive {
			return "", false, fmt.Errorf("tenant suspended")
		}
		needsTotp := user.TOTPEnabled && strings.TrimSpace(user.TOTPSecret) != ""
		ticket := h.storeGitHubOAuthTicket(githubOAuthTicket{
			PrincipalType: models.UserDevicePrincipalTenantUser,
			PrincipalID:   user.ID,
			TenantID:      user.TenantID,
			DeviceKey:     deviceKey,
			NeedsTotp:     needsTotp,
		})
		return ticket, needsTotp, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}
	adm, err := models.GetActivePlatformAdminByGitHubID(h.db, ghID)
	if err != nil {
		return "", false, err
	}
	needsTotp := adm.TOTPEnabled && strings.TrimSpace(adm.TOTPSecret) != ""
	ticket := h.storeGitHubOAuthTicket(githubOAuthTicket{
		PrincipalType: models.UserDevicePrincipalPlatformAdmin,
		PrincipalID:   adm.ID,
		NeedsTotp:     needsTotp,
	})
	return ticket, needsTotp, nil
}

// exchangeGitHubOAuthTicket completes GitHub login using a one-time ticket (and optional TOTP).
func (h *Handlers) exchangeGitHubOAuthTicket(c *gin.Context) {
	var req oauthGitHubExchangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	ticket, ok := loadGitHubOAuthTicket(req.Ticket)
	if !ok {
		response.FailI18n(c, i18n.KeyAuthInvalidToken, nil)
		return
	}
	if ticket.NeedsTotp {
		code := strings.TrimSpace(req.TotpCode)
		if code == "" {
			response.FailI18n(c, i18n.KeyAuthNeedsTotp, gin.H{"needsTotp": true})
			return
		}
	}
	if bootstrap.GlobalKeyManager == nil {
		response.FailI18n(c, i18n.KeyAuthJWTNotReady, nil)
		return
	}

	switch ticket.PrincipalType {
	case models.UserDevicePrincipalPlatformAdmin:
		var adm models.PlatformAdmin
		if err := h.db.Where("id = ?", ticket.PrincipalID).First(&adm).Error; err != nil {
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
		if ticket.NeedsTotp {
			if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalPlatformAdmin, adm.ID, adm.TOTPSecret, adm.TOTPRecoveryHashes, req.TotpCode) {
				response.FailI18n(c, i18n.KeyAuthInvalidTotp, gin.H{"needsTotp": true})
				return
			}
		}
		deleteGitHubOAuthTicket(req.Ticket)
		token, err := access.SignPlatformAccessTokenWithKey(access.PlatformPayload{
			AdminID: adm.ID,
			Email:   adm.Email,
			Role:    constants.JWTRolePlatformSuper,
		}, bootstrap.GlobalKeyManager, constants.TenantAccessTokenTTL)
		if err != nil {
			response.FailI18n(c, i18n.KeyTenantSignTokenFailed, nil)
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"principal":     "platform",
			"token":         token,
			"expiresIn":     int(constants.TenantAccessTokenTTL.Seconds()),
			"platformAdmin": apiresponse.NewPlatformAdminResponse(adm),
		})
		return
	default:
		user, err := models.GetActiveTenantUserByID(h.db, ticket.PrincipalID)
		if err != nil {
			response.FailI18n(c, i18n.KeyAuthInvalidCredentials, nil)
			return
		}
		tenant, terr := models.GetActiveTenantByID(h.db, user.TenantID)
		if terr != nil {
			response.FailI18n(c, i18n.KeyTenantNotFound, nil)
			return
		}
		if ticket.NeedsTotp {
			if !h.verifyTotpOrRecoveryCode(models.UserDevicePrincipalTenantUser, user.ID, user.TOTPSecret, user.TOTPRecoveryHashes, req.TotpCode) {
				response.FailI18n(c, i18n.KeyAuthInvalidTotp, gin.H{"needsTotp": true})
				return
			}
		}
		geo := utils.LoginGeoFromIP(c.ClientIP())
		_ = models.RecordTenantUserLogin(h.db, user.ID, c.ClientIP(), geo.City, geo.Location)
		ds, derr := h.resolveDeviceSession(c, deviceSessionInput{
			PrincipalType:            models.UserDevicePrincipalTenantUser,
			PrincipalID:              user.ID,
			Email:                    user.Email,
			DisplayName:              user.DisplayName,
			DeviceKey:                ticket.DeviceKey,
			ClientIP:                 c.ClientIP(),
			RequireDeviceVerify:      user.RequireDeviceVerify,
			TrustDeviceLoginEnabled:  user.TrustDeviceLoginEnabled,
			RequireRemoteLoginVerify: user.RequireRemoteLoginVerify,
			KnownLoginCities:         utils.ParseKnownLoginCities(user.KnownLoginCitiesJSON),
		})
		if derr != nil {
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
			key := i18n.KeyAuthNeedsDeviceVerify
			if ds.NeedsRemoteVerify && !ds.NeedsDeviceVerify {
				key = i18n.KeyAuthNeedsRemoteVerify
			}
			response.FailI18n(c, key, data)
			return
		}
		deleteGitHubOAuthTicket(req.Ticket)
		token, tokenTTL, terr := signTenantAccessToken(h.db, user, tenant, ds.SessionID, ds.DeviceRecordID)
		if terr != nil {
			response.FailI18n(c, i18n.KeyTenantSignTokenFailed, nil)
			return
		}
		codes, _ := models.ListEffectivePermissionCodesForTenantUser(h.db, user.ID)
		models.RecordSuccessfulLoginCity(h.db, user.ID, geo.City)
		h.recordLoginSuccess(c, models.LoginHistoryInput{
			PrincipalType: models.LoginHistoryPrincipalTenantUser,
			PrincipalID:   user.ID,
			TenantID:      user.TenantID,
			Email:         user.Email,
			ClientIP:      c.ClientIP(),
			City:          geo.City,
			Location:      geo.Location,
			LoginMethod:   "oauth_github",
			DeviceKey:     strings.TrimSpace(ticket.DeviceKey),
		})
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
			"permissionCodes": codes,
		})
	}
}

// unbindGitHubOAuth removes the GitHub binding for the signed-in user.
func (h *Handlers) unbindGitHubOAuth(c *gin.Context) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		if err := h.db.Model(&models.PlatformAdmin{}).Where("id = ?", aid).
			Updates(map[string]any{"github_id": "", "github_login": ""}).Error; err != nil {
			response.AbortWithStatusJSON(c, 500, err)
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"unbound": true})
		return
	}
	uid := middleware.AuthUserID(c)
	if uid == 0 {
		response.FailI18n(c, i18n.KeyUnauthorized, nil)
		return
	}
	_, err := models.UpdateTenantUser(h.db, uid, map[string]any{
		"github_id":    "",
		"github_login": "",
	}, "oauth-github")
	if err != nil {
		response.AbortWithStatusJSON(c, 500, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"unbound": true})
}
