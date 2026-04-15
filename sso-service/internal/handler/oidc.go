package handler

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/sso-service/internal/config"
	"github.com/LingByte/SoulNexus/sso-service/internal/models"
	"github.com/LingByte/SoulNexus/sso-service/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OIDCHandler struct {
	cfg        config.Config
	db         *gorm.DB
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	signingKey *models.SigningKey
}

const (
	modeToken     = "token"
	modeSession   = "session"
	sessionCookie = "sso_session"
)

func NewOIDCHandler(cfg config.Config, db *gorm.DB, signingKey *models.SigningKey) (*OIDCHandler, error) {
	privateKey, err := security.ParseRSAPrivateKeyFromPEM(signingKey.PrivatePEM)
	if err != nil {
		return nil, err
	}
	publicKey, err := security.ParseRSAPublicKeyFromPEM(signingKey.PublicPEM)
	if err != nil {
		return nil, err
	}
	return &OIDCHandler{
		cfg:        cfg,
		db:         db,
		privateKey: privateKey,
		publicKey:  publicKey,
		signingKey: signingKey,
	}, nil
}

func (h *OIDCHandler) Discovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                h.cfg.Issuer,
		"authorization_endpoint":                h.cfg.Issuer + "/oauth/authorize",
		"token_endpoint":                        h.cfg.Issuer + "/oauth/token",
		"userinfo_endpoint":                     h.cfg.Issuer + "/oauth/userinfo",
		"jwks_uri":                              h.cfg.Issuer + "/oauth/jwks",
		"revocation_endpoint":                   h.cfg.Issuer + "/oauth/revoke",
		"introspection_endpoint":                h.cfg.Issuer + "/oauth/introspect",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"claims_supported":                      []string{"sub", "email", "name"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	})
}

func (h *OIDCHandler) Authorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")
	scope := c.Query("scope")
	responseType := c.Query("response_type")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.DefaultQuery("code_challenge_method", "S256")
	principal, ok := h.authenticateBySession(c)
	if !ok {
		return
	}
	userID := principal.UserID

	if responseType != "code" || clientID == "" || redirectURI == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var client models.OAuthClient
	if err := h.db.First(&client, "id = ?", clientID).Error; err != nil || client.RedirectURI != redirectURI {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ? AND status = ?", userID, "active").Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_user"})
		return
	}

	code := uuid.NewString()
	authCode := models.AuthorizationCode{
		Code:                code,
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           time.Now().Add(h.cfg.AuthorizationCodeTTL),
	}
	if err := h.db.Create(&authCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// For backend-driven flows we return JSON; browser redirects can be added later.
	c.JSON(http.StatusOK, gin.H{
		"code":  code,
		"state": state,
	})
}

func (h *OIDCHandler) Token(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	switch grantType {
	case "authorization_code":
		h.tokenByAuthorizationCode(c)
	case "refresh_token":
		h.tokenByRefreshToken(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
	}
}

func (h *OIDCHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		ClientID string `json:"client_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "email = ? AND status = ?", req.Email, "active").Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}
	if !security.VerifyPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	clientID := req.ClientID
	if clientID == "" {
		clientID = "portal-web"
	}

	sessionID := uuid.NewString()
	session := models.UserSession{
		ID:        sessionID,
		UserID:    user.ID,
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(h.cfg.RefreshTokenTTL),
	}
	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.SetCookie(sessionCookie, sessionID, int(h.cfg.RefreshTokenTTL.Seconds()), "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	h.writeAudit(c, user.ID, clientID, "login", modeSession)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func (h *OIDCHandler) Logout(c *gin.Context) {
	principal, ok := h.authenticateBySession(c)
	if !ok {
		return
	}
	if principal.SessionID != "" {
		now := time.Now()
		_ = h.db.Model(&models.UserSession{}).Where("id = ? AND revoked_at IS NULL", principal.SessionID).Update("revoked_at", &now).Error
		c.SetCookie(sessionCookie, "", -1, "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	}
	h.writeAudit(c, principal.UserID, principal.ClientID, "logout", modeSession)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *OIDCHandler) tokenByAuthorizationCode(c *gin.Context) {
	code := c.PostForm("code")
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	redirectURI := c.PostForm("redirect_uri")
	codeVerifier := c.PostForm("code_verifier")

	var client models.OAuthClient
	if err := h.db.First(&client, "id = ?", clientID).Error; err != nil || client.Secret != clientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	var authCode models.AuthorizationCode
	err := h.db.First(&authCode, "code = ? AND client_id = ? AND consumed = ?", code, clientID, false).Error
	if err != nil || authCode.RedirectURI != redirectURI || time.Now().After(authCode.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}

	if authCode.CodeChallenge != "" {
		if authCode.CodeChallengeMethod == "S256" {
			if security.S256CodeChallenge(codeVerifier) != authCode.CodeChallenge {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
				return
			}
		} else if codeVerifier != authCode.CodeChallenge {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
			return
		}
	}

	accessToken, accessJTI, refreshRaw, err := h.issueTokenPair(authCode.UserID, clientID, authCode.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.AuthorizationCode{}).Where("code = ?", authCode.Code).Update("consumed", true).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.cfg.AccessTokenTTL.Seconds()),
		"refresh_token": refreshRaw,
		"scope":         authCode.Scope,
		"jti":           accessJTI,
	})
	h.writeAudit(c, authCode.UserID, clientID, "token_authorization_code", modeToken)
}

func (h *OIDCHandler) tokenByRefreshToken(c *gin.Context) {
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	refreshRaw := c.PostForm("refresh_token")

	var client models.OAuthClient
	if err := h.db.First(&client, "id = ?", clientID).Error; err != nil || client.Secret != clientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	refreshHash := security.HashToken(refreshRaw)
	var refresh models.RefreshToken
	if err := h.db.First(&refresh, "token_hash = ? AND client_id = ?", refreshHash, clientID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}
	if refresh.RevokedAt != nil || time.Now().After(refresh.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}

	accessToken, _, err := security.BuildAccessToken(
		h.privateKey, h.signingKey.KID, h.cfg.Issuer, h.cfg.Audience, refresh.UserID, refresh.Scope, h.cfg.AccessTokenTTL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(h.cfg.AccessTokenTTL.Seconds()),
		"scope":        refresh.Scope,
	})
	h.writeAudit(c, refresh.UserID, clientID, "token_refresh", modeToken)
}

func (h *OIDCHandler) UserInfo(c *gin.Context) {
	principal, ok := h.authenticateByToken(c)
	if !ok {
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", principal.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func (h *OIDCHandler) Revoke(c *gin.Context) {
	token := c.PostForm("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	hash := security.HashToken(token)
	now := time.Now()
	_ = h.db.Model(&models.RefreshToken{}).Where("token_hash = ? AND revoked_at IS NULL", hash).Update("revoked_at", &now).Error
	c.JSON(http.StatusOK, gin.H{})
}

func (h *OIDCHandler) Introspect(c *gin.Context) {
	principal, ok := h.authenticateByToken(c)
	if !ok {
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ? AND status = ?", principal.UserID, "active").Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	claims := principal.Claims
	c.JSON(http.StatusOK, gin.H{
		"active":    true,
		"sub":       claims.Subject,
		"scope":     claims.Scope,
		"iss":       claims.Issuer,
		"aud":       claims.Audience,
		"exp":       claims.ExpiresAt.Time.Unix(),
		"iat":       claims.IssuedAt.Time.Unix(),
		"token_use": "access_token",
	})
}

func (h *OIDCHandler) JWKS(c *gin.Context) {
	n := base64.RawURLEncoding.EncodeToString(h.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(h.publicKey.E)).Bytes())
	c.JSON(http.StatusOK, gin.H{
		"keys": []gin.H{
			{
				"kty": "RSA",
				"kid": h.signingKey.KID,
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	})
}

func (h *OIDCHandler) parseBearerClaims(c *gin.Context) (*security.Claims, bool) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return nil, false
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return nil, false
	}

	token, err := jwtParse(tokenString, h.publicKey)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return nil, false
	}

	claims, ok := token.Claims.(*security.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return nil, false
	}
	if claims.Issuer != h.cfg.Issuer {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return nil, false
	}
	return claims, true
}

type principal struct {
	UserID    string
	ClientID  string
	SessionID string
	Claims    *security.Claims
}

func (h *OIDCHandler) authenticateByToken(c *gin.Context) (*principal, bool) {
	claims, ok := h.parseBearerClaims(c)
	if !ok {
		return nil, false
	}
	return &principal{
		UserID: claims.Subject,
		Claims: claims,
	}, true
}

func (h *OIDCHandler) authenticateBySession(c *gin.Context) (*principal, bool) {
	sessionID, err := c.Cookie(sessionCookie)
	if err != nil || strings.TrimSpace(sessionID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
		return nil, false
	}
	var session models.UserSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
		return nil, false
	}
	if session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
		return nil, false
	}
	return &principal{
		UserID:    session.UserID,
		ClientID:  session.ClientID,
		SessionID: session.ID,
	}, true
}

func (h *OIDCHandler) issueTokenPair(userID, clientID, scope string) (string, string, string, error) {
	accessToken, accessJTI, err := security.BuildAccessToken(
		h.privateKey, h.signingKey.KID, h.cfg.Issuer, h.cfg.Audience, userID, scope, h.cfg.AccessTokenTTL,
	)
	if err != nil {
		return "", "", "", err
	}
	refreshRaw := uuid.NewString() + "." + uuid.NewString()
	refreshHash := security.HashToken(refreshRaw)
	refresh := models.RefreshToken{
		TokenHash: refreshHash,
		ClientID:  clientID,
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(h.cfg.RefreshTokenTTL),
	}
	if err := h.db.Create(&refresh).Error; err != nil {
		return "", "", "", err
	}
	return accessToken, accessJTI, refreshRaw, nil
}

func (h *OIDCHandler) writeAudit(c *gin.Context, userID, clientID, action, mode string) {
	_ = h.db.Create(&models.AuditLog{
		ID:        uuid.NewString(),
		UserID:    userID,
		ClientID:  clientID,
		Action:    action,
		AuthMode:  mode,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}).Error
}

func jwtParse(token string, publicKey *rsa.PublicKey) (*jwt.Token, error) {
	return jwt.ParseWithClaims(token, &security.Claims{}, func(parsed *jwt.Token) (interface{}, error) {
		if _, ok := parsed.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return publicKey, nil
	})
}
