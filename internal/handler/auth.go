package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	utilscaptcha "github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type wechatLoginSession struct {
	SceneID   string
	Status    string
	ExpiresAt time.Time
	ScannedAt *time.Time
	OpenID    string
	UserID    uint
	LoginCode string
}

type wechatBindSession struct {
	SessionID string
	Status    string
	ExpiresAt time.Time
	OpenID    string
	UserID    uint
	BindCode  string
	Reason    string
	BoundAt   *time.Time
}

type oidcAuthCode struct {
	Code        string
	UserID      uint
	ClientID    string
	RedirectURI string
	ExpiresAt   time.Time
	Used        bool
}

type githubOAuthState struct {
	Nonce        string
	RedirectURL  string
	OIDCClient   string
	OIDCRedirect string
	OIDCState    string
	BindUserID   uint
	ExpiresAt    time.Time
}

var wechatLoginSessions = struct {
	sync.RWMutex
	items map[string]*wechatLoginSession
}{
	items: make(map[string]*wechatLoginSession),
}

var wechatAccessTokenCache = struct {
	sync.Mutex
	Token     string
	ExpiresAt time.Time
}{}

var oidcAuthCodeStore = struct {
	sync.RWMutex
	items map[string]*oidcAuthCode
}{
	items: make(map[string]*oidcAuthCode),
}

var githubOAuthStateStore = struct {
	sync.RWMutex
	items map[string]*githubOAuthState
}{
	items: make(map[string]*githubOAuthState),
}

var wechatBindSessions = struct {
	sync.RWMutex
	items map[string]*wechatBindSession
}{
	items: make(map[string]*wechatBindSession),
}

func verifyRequestCaptcha(captchaID, captchaCode, captchaData, captchaType string) (bool, error) {
	if utilscaptcha.GlobalManager == nil {
		return false, errors.New("captcha service not initialized")
	}
	if strings.TrimSpace(captchaID) == "" {
		return false, errors.New("captcha is required")
	}
	cType := utilscaptcha.Type(strings.TrimSpace(captchaType))
	if cType == "" {
		cType = utilscaptcha.TypeImage
	}
	switch cType {
	case utilscaptcha.TypeImage:
		if strings.TrimSpace(captchaCode) == "" {
			return false, errors.New("captcha is required")
		}
		return utilscaptcha.GlobalManager.VerifyImage(captchaID, captchaCode)
	case utilscaptcha.TypeClick:
		raw := strings.TrimSpace(captchaData)
		if raw == "" {
			raw = strings.TrimSpace(captchaCode)
		}
		var points []utilscaptcha.Point
		if raw == "" {
			return false, errors.New("click captcha data is required")
		}
		if err := json.Unmarshal([]byte(raw), &points); err != nil {
			return false, errors.New("invalid click captcha data")
		}
		return utilscaptcha.GlobalManager.VerifyClick(captchaID, points)
	default:
		return false, errors.New("unsupported captcha type")
	}
}

type wechatQRCodeCreateResp struct {
	Ticket        string `json:"ticket"`
	ExpireSeconds int    `json:"expire_seconds"`
	URL           string `json:"url"`
	ErrCode       int    `json:"errcode"`
	ErrMsg        string `json:"errmsg"`
}

type wechatAccessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type wechatUserInfoResp struct {
	Subscribe int    `json:"subscribe"`
	OpenID    string `json:"openid"`
	UnionID   string `json:"unionid"`
	Nickname  string `json:"nickname"`
	HeadImg   string `json:"headimgurl"`
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

type wechatMessageXML struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	MsgType      string   `xml:"MsgType"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
	Content      string   `xml:"Content"`
}

type githubOAuthTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

type githubUserResp struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmailResp struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// handleUserSignupPage handle user signup page
func (h *Handlers) handleUserSignupPage(c *gin.Context) {
	ctx := LingEcho.GetRenderPageContext(c)
	ctx["SignupText"] = "Sign Up Now"
	ctx["Site.SignupApi"] = utils.GetValue(h.db, constants.KEY_SITE_SIGNUP_API)
	c.HTML(http.StatusOK, "signup.html", ctx)
}

// handleUserResetPasswordPage handle user reset password page
func (h *Handlers) handleUserResetPasswordPage(c *gin.Context) {
	c.HTML(http.StatusOK, "reset_password.html", LingEcho.GetRenderPageContext(c))
}

// handleUserSigninPage handle user signin page
func (h *Handlers) handleUserSigninPage(c *gin.Context) {
	ctx := LingEcho.GetRenderPageContext(c)
	ctx["SignupText"] = "Sign Up Now"
	if redirectURL := strings.TrimSpace(c.Query("redirecturl")); redirectURL != "" {
		ctx["LoginNext"] = redirectURL
	}
	ctx["OIDCClientID"] = strings.TrimSpace(c.Query("client_id"))
	ctx["OIDCRedirectURI"] = strings.TrimSpace(c.Query("redirect_uri"))
	ctx["OIDCState"] = strings.TrimSpace(c.Query("state"))
	c.HTML(http.StatusOK, "signin.html", ctx)
}

// RenderSigninPage exposes the signin template rendering for external router composition.
func (h *Handlers) RenderSigninPage(c *gin.Context) {
	h.handleUserSigninPage(c)
}

func newWechatSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("wx_%d", time.Now().UnixNano())
	}
	return "wx_" + hex.EncodeToString(buf)
}

func newOIDCCode() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("oidc_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func newGitHubStateNonce() string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("gh_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func cleanExpiredOIDCCodes() {
	now := time.Now()
	oidcAuthCodeStore.Lock()
	defer oidcAuthCodeStore.Unlock()
	for k, v := range oidcAuthCodeStore.items {
		if now.After(v.ExpiresAt) || v.Used {
			delete(oidcAuthCodeStore.items, k)
		}
	}
}

func cleanExpiredGitHubStates() {
	now := time.Now()
	githubOAuthStateStore.Lock()
	defer githubOAuthStateStore.Unlock()
	for k, v := range githubOAuthStateStore.items {
		if now.After(v.ExpiresAt) {
			delete(githubOAuthStateStore.items, k)
		}
	}
}

func cleanExpiredWechatBindSessions() {
	now := time.Now()
	wechatBindSessions.Lock()
	defer wechatBindSessions.Unlock()
	for k, v := range wechatBindSessions.items {
		if now.After(v.ExpiresAt) {
			delete(wechatBindSessions.items, k)
		}
	}
}

func newWechatLoginCode() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	n := binary.BigEndian.Uint32(b) % 1000000
	return fmt.Sprintf("%06d", n)
}

func getWechatMPConfig() (appID, appSecret, token string, err error) {
	appID = strings.TrimSpace(os.Getenv("WECHAT_MP_APP_ID"))
	appSecret = strings.TrimSpace(os.Getenv("WECHAT_MP_SECRET"))
	token = strings.TrimSpace(os.Getenv("WECHAT_MP_TOKEN"))
	if appID == "" || appSecret == "" {
		return "", "", "", errors.New("missing WECHAT_MP_APP_ID or WECHAT_MP_SECRET")
	}
	return appID, appSecret, token, nil
}

func getWechatMPToken() string {
	return strings.TrimSpace(os.Getenv("WECHAT_MP_TOKEN"))
}

func getGitHubOAuthConfig(c *gin.Context) (clientID, clientSecret, redirectURI string, err error) {
	clientID = strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID"))
	clientSecret = strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET"))
	redirectURI = strings.TrimSpace(os.Getenv("GITHUB_OAUTH_REDIRECT_URI"))
	if clientID == "" || clientSecret == "" {
		return "", "", "", errors.New("missing GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET")
	}
	if redirectURI == "" {
		scheme := c.Request.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			if c.Request.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		redirectURI = fmt.Sprintf("%s://%s/api/auth/github/callback", scheme, c.Request.Host)
	}
	return clientID, clientSecret, redirectURI, nil
}

func getWechatOAuthRedirectURI(c *gin.Context) string {
	redirectURI := strings.TrimSpace(os.Getenv("WECHAT_OAUTH_REDIRECT_URI"))
	if redirectURI != "" {
		return redirectURI
	}
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s/api/auth/wechat/oauth/callback", scheme, c.Request.Host)
}

func getRefreshTokenTTL(db *gorm.DB) time.Duration {
	// default 30 days
	ttl := 30 * 24 * time.Hour
	if val := strings.TrimSpace(os.Getenv("REFRESH_TOKEN_EXPIRED")); val != "" {
		if d, err := time.ParseDuration(val); err == nil && d > 0 {
			ttl = d
		}
	}
	if db != nil {
		// Keep access-token TTL behavior consistent with existing config table pattern.
		if val := strings.TrimSpace(utils.GetValue(db, constants.KEY_AUTH_TOKEN_EXPIRED)); val != "" {
			if d, err := time.ParseDuration(val); err == nil && d > 0 && d < ttl {
				ttl = d * 10
			}
		}
	}
	return ttl
}

func buildTokenPair(db *gorm.DB, user *models.User, accessTTL time.Duration) (string, string) {
	if accessTTL <= 0 {
		accessTTL = 24 * time.Hour
	}
	accessToken := models.BuildAuthToken(user, accessTTL, false)
	refreshToken := models.BuildRefreshToken(user, getRefreshTokenTTL(db), false)
	return accessToken, refreshToken
}

func getWechatAccessToken() (string, error) {
	wechatAccessTokenCache.Lock()
	defer wechatAccessTokenCache.Unlock()

	if wechatAccessTokenCache.Token != "" && time.Now().Before(wechatAccessTokenCache.ExpiresAt) {
		return wechatAccessTokenCache.Token, nil
	}

	appID, appSecret, _, err := getWechatMPConfig()
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appID, appSecret)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var tokenResp wechatAccessTokenResp
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.ErrCode != 0 || tokenResp.AccessToken == "" {
		return "", fmt.Errorf("wechat token error: %d %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	exp := tokenResp.ExpiresIn - 300
	if exp < 300 {
		exp = tokenResp.ExpiresIn
	}
	wechatAccessTokenCache.Token = tokenResp.AccessToken
	wechatAccessTokenCache.ExpiresAt = time.Now().Add(time.Duration(exp) * time.Second)
	return tokenResp.AccessToken, nil
}

func cleanExpiredWechatSessions() {
	now := time.Now()
	wechatLoginSessions.Lock()
	defer wechatLoginSessions.Unlock()
	for k, v := range wechatLoginSessions.items {
		if now.After(v.ExpiresAt) {
			delete(wechatLoginSessions.items, k)
		}
	}
}

func verifyWechatSignature(token, signature, timestamp, nonce string) bool {
	items := []string{token, timestamp, nonce}
	sort.Strings(items)
	sum := sha1.Sum([]byte(strings.Join(items, "")))
	return fmt.Sprintf("%x", sum) == signature
}

func calcWechatSignature(token, timestamp, nonce string) string {
	items := []string{token, timestamp, nonce}
	sort.Strings(items)
	sum := sha1.Sum([]byte(strings.Join(items, "")))
	return fmt.Sprintf("%x", sum)
}

func calcWechatMsgSignature(token, timestamp, nonce, encrypted string) string {
	items := []string{token, timestamp, nonce, encrypted}
	sort.Strings(items)
	sum := sha1.Sum([]byte(strings.Join(items, "")))
	return fmt.Sprintf("%x", sum)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty decrypted data")
	}
	padLen := int(data[len(data)-1])
	if padLen < 1 || padLen > aes.BlockSize || padLen > len(data) {
		return nil, errors.New("invalid padding")
	}
	for i := 0; i < padLen; i++ {
		if data[len(data)-1-i] != byte(padLen) {
			return nil, errors.New("invalid padding bytes")
		}
	}
	return data[:len(data)-padLen], nil
}

func decryptWechatXML(encodingAESKey, appID, encrypted string) ([]byte, error) {
	keyData, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, err
	}
	if len(keyData) != 32 {
		return nil, errors.New("invalid EncodingAESKey length")
	}
	cipherText, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}
	if len(cipherText)%aes.BlockSize != 0 {
		return nil, errors.New("invalid encrypted payload size")
	}

	block, err := aes.NewCipher(keyData)
	if err != nil {
		return nil, err
	}
	iv := keyData[:aes.BlockSize]
	plain := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, cipherText)
	plain, err = pkcs7Unpad(plain)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, errors.New("invalid decrypted payload")
	}

	// 16 bytes random + 4 bytes msg length (big endian) + xml + appId
	msgLen := binary.BigEndian.Uint32(plain[16:20])
	if int(20+msgLen) > len(plain) {
		return nil, errors.New("invalid message length")
	}
	xmlMsg := plain[20 : 20+msgLen]
	recvAppID := string(plain[20+msgLen:])
	if appID != "" && recvAppID != appID {
		return nil, fmt.Errorf("appid mismatch, got %s", recvAppID)
	}
	return bytes.TrimSpace(xmlMsg), nil
}

func extractSceneID(eventKey string) string {
	k := strings.TrimSpace(eventKey)
	k = strings.TrimPrefix(k, "qrscene_")
	k = strings.TrimPrefix(k, "login_")
	return k
}

func sendWechatReplyText(c *gin.Context, toUser, fromUser, text string) {
	reply := fmt.Sprintf("<xml><ToUserName><![CDATA[%s]]></ToUserName><FromUserName><![CDATA[%s]]></FromUserName><CreateTime>%d</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[%s]]></Content></xml>",
		toUser, fromUser, time.Now().Unix(), text)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, reply)
}

func (h *Handlers) findOrCreateWechatUser(openID string, info *wechatUserInfoResp) (*models.User, error) {
	if openID == "" {
		return nil, errors.New("empty openid")
	}

	var user models.User
	if err := h.db.Where("wechat_open_id = ?", openID).First(&user).Error; err == nil {
		updates := map[string]any{
			"wechat_union_id": info.UnionID,
		}
		if info.HeadImg != "" {
			updates["avatar"] = info.HeadImg
		}
		if updateErr := h.db.Model(&user).Updates(updates).Error; updateErr != nil {
			logger.Warn("failed to update existing wechat fields, continue login", zap.Error(updateErr))
		}
		return &user, nil
	}

	nickname := strings.TrimSpace(info.Nickname)
	suffix := openID
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	if nickname == "" {
		nickname = fmt.Sprintf("微信用户%s", suffix)
	}
	displayName := fmt.Sprintf("%s_%s", nickname, suffix)
	email := fmt.Sprintf("wechat_%s@temp.local", suffix)
	password := newWechatSessionID()
	created, err := models.CreateUserByEmailWithMeta(h.db, displayName, displayName, email, password, models.UserSourceWechat, models.UserStatusActive)
	if err != nil {
		email = fmt.Sprintf("wechat_%d@temp.local", time.Now().UnixNano())
		created, err = models.CreateUserByEmailWithMeta(h.db, displayName, displayName, email, password, models.UserSourceWechat, models.UserStatusActive)
		if err != nil {
			return nil, err
		}
	}

	updates := map[string]any{
		"wechat_open_id":  openID,
		"wechat_union_id": info.UnionID,
	}
	if info.HeadImg != "" {
		updates["avatar"] = info.HeadImg
	}
	if err = h.db.Model(created).Updates(updates).Error; err != nil {
		// Some old databases may not have wechat_* columns yet.
		// Keep login working even if wechat field persistence fails.
		logger.Warn("failed to persist wechat fields for new user, continue login", zap.Error(err))
	}
	_ = h.db.First(created, created.ID).Error
	return created, nil
}

func (h *Handlers) findOrCreateGitHubUser(githubUser *githubUserResp, githubEmail string) (*models.User, error) {
	if githubUser == nil || githubUser.ID <= 0 {
		return nil, errors.New("invalid github user")
	}
	githubID := strconv.FormatInt(githubUser.ID, 10)
	var user models.User
	if err := h.db.Where("github_id = ?", githubID).First(&user).Error; err == nil {
		updates := map[string]any{
			"github_login": strings.TrimSpace(githubUser.Login),
		}
		if githubUser.AvatarURL != "" {
			updates["avatar"] = githubUser.AvatarURL
		}
		if updateErr := h.db.Model(&user).Updates(updates).Error; updateErr != nil {
			logger.Warn("failed to update github user fields", zap.Error(updateErr))
		}
		return &user, nil
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(githubEmail))
	if normalizedEmail != "" {
		if existing, err := models.GetUserByEmail(h.db, normalizedEmail); err == nil && existing != nil {
			updates := map[string]any{
				"github_id":    githubID,
				"github_login": strings.TrimSpace(githubUser.Login),
			}
			if githubUser.AvatarURL != "" {
				updates["avatar"] = githubUser.AvatarURL
			}
			if updateErr := h.db.Model(existing).Updates(updates).Error; updateErr != nil {
				return nil, updateErr
			}
			_ = h.db.First(existing, existing.ID).Error
			return existing, nil
		}
	}

	displayName := strings.TrimSpace(githubUser.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(githubUser.Login)
	}
	if displayName == "" {
		displayName = "GitHub 用户"
	}
	if normalizedEmail == "" {
		normalizedEmail = fmt.Sprintf("github_%s@temp.local", githubID)
	}

	password := newGitHubStateNonce()
	created, err := models.CreateUserByEmailWithMeta(h.db, displayName, displayName, normalizedEmail, password, models.UserSourceGithub, models.UserStatusActive)
	if err != nil {
		normalizedEmail = fmt.Sprintf("github_%d@temp.local", time.Now().UnixNano())
		created, err = models.CreateUserByEmailWithMeta(h.db, displayName, displayName, normalizedEmail, password, models.UserSourceGithub, models.UserStatusActive)
		if err != nil {
			return nil, err
		}
	}
	updates := map[string]any{
		"github_id":    githubID,
		"github_login": strings.TrimSpace(githubUser.Login),
	}
	if githubUser.AvatarURL != "" {
		updates["avatar"] = githubUser.AvatarURL
	}
	if err = h.db.Model(created).Updates(updates).Error; err != nil {
		logger.Warn("failed to persist github fields for new user", zap.Error(err))
	}
	_ = h.db.First(created, created.ID).Error
	return created, nil
}

func (h *Handlers) finishWechatSessionSuccess(c *gin.Context, session *wechatLoginSession) (gin.H, error) {
	user, err := models.GetUserByUID(h.db, session.UserID)
	if err != nil || user == nil {
		return nil, errors.New("wechat user is not bound")
	}
	models.Login(c, user)
	accessToken, refreshToken := buildTokenPair(h.db, user, 7*24*time.Hour)
	user.AuthToken = accessToken
	return gin.H{
		"status":       "success",
		"token":        user.AuthToken,
		"refreshToken": refreshToken,
		"user":         user,
	}, nil
}

func (h *Handlers) handleWechatLoginCode(c *gin.Context) {
	sceneID := newWechatSessionID()
	loginCode := newWechatLoginCode()
	expiresIn := 600
	if expiresIn <= 0 {
		expiresIn = 600
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	session := &wechatLoginSession{
		SceneID:   sceneID,
		Status:    "pending",
		ExpiresAt: expiresAt,
		LoginCode: loginCode,
	}
	wechatLoginSessions.Lock()
	wechatLoginSessions.items[sceneID] = session
	wechatLoginSessions.Unlock()
	cleanExpiredWechatSessions()
	response.Success(c, "wechat login code generated", gin.H{
		"sessionId":    sceneID,
		"loginCode":    loginCode,
		"expiresAt":    expiresAt.Unix(),
		"expiresInSec": expiresIn,
		"expiresIn":    expiresIn,
		"mode":         "message_push",
	})
}

func (h *Handlers) handleWechatConfigCheck(c *gin.Context) {
	appID := strings.TrimSpace(os.Getenv("WECHAT_MP_APP_ID"))
	appSecret := strings.TrimSpace(os.Getenv("WECHAT_MP_SECRET"))
	token := strings.TrimSpace(os.Getenv("WECHAT_MP_TOKEN"))
	aesKey := strings.TrimSpace(os.Getenv("WECHAT_MP_AES_KEY"))
	tokenFingerprint := ""
	if token != "" {
		sum := sha1.Sum([]byte(token))
		tokenFingerprint = fmt.Sprintf("%x", sum)
		if len(tokenFingerprint) > 12 {
			tokenFingerprint = tokenFingerprint[:12]
		}
	}
	response.Success(c, "wechat config check", gin.H{
		"appId":               appID,
		"appIdLength":         len(appID),
		"appSecretConfigured": appSecret != "",
		"appSecretLength":     len(appSecret),
		"tokenConfigured":     token != "",
		"tokenFingerprint":    tokenFingerprint,
		"aesKeyConfigured":    aesKey != "",
	})
}

func (h *Handlers) handleWechatLoginStatus(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("sessionId"))
	if sessionID == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("sessionId is required"))
		return
	}
	wechatLoginSessions.RLock()
	session, ok := wechatLoginSessions.items[sessionID]
	wechatLoginSessions.RUnlock()
	if !ok {
		response.Success(c, "wechat session status", gin.H{"status": "expired"})
		return
	}
	if time.Now().After(session.ExpiresAt) {
		response.Success(c, "wechat session status", gin.H{"status": "expired"})
		return
	}
	if session.Status != "success" {
		status := session.Status
		if status == "" {
			status = "pending"
		}
		response.Success(c, "wechat session status", gin.H{
			"status":    status,
			"expiresAt": session.ExpiresAt.Unix(),
		})
		return
	}

	data, err := h.finishWechatSessionSuccess(c, session)
	if err != nil {
		response.Success(c, "wechat session status", gin.H{"status": "authorized_but_unbound"})
		return
	}
	wechatLoginSessions.Lock()
	delete(wechatLoginSessions.items, sessionID)
	wechatLoginSessions.Unlock()
	response.Success(c, "wechat session status", data)
}

func (h *Handlers) handleWechatCheckLogin(c *gin.Context) {
	sceneID := strings.TrimSpace(c.Param("sceneId"))
	if sceneID == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("sceneId is required"))
		return
	}
	wechatLoginSessions.RLock()
	session, ok := wechatLoginSessions.items[sceneID]
	wechatLoginSessions.RUnlock()
	if !ok || time.Now().After(session.ExpiresAt) {
		response.Success(c, "wechat check login", gin.H{"status": "expired"})
		return
	}
	if session.Status != "success" {
		status := session.Status
		if status == "" {
			status = "pending"
		}
		response.Success(c, "wechat check login", gin.H{"status": status})
		return
	}
	data, err := h.finishWechatSessionSuccess(c, session)
	if err != nil {
		response.Success(c, "wechat check login", gin.H{"status": "pending"})
		return
	}
	wechatLoginSessions.Lock()
	delete(wechatLoginSessions.items, sceneID)
	wechatLoginSessions.Unlock()
	response.Success(c, "wechat check login", data)
}

func (h *Handlers) handleWechatOAuthCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	sceneID := strings.TrimSpace(c.Query("state"))
	if code == "" || sceneID == "" {
		c.Redirect(http.StatusFound, "/api/auth/login?wechat=failed")
		return
	}

	appID := strings.TrimSpace(os.Getenv("WECHAT_MP_APP_ID"))
	appSecret := strings.TrimSpace(os.Getenv("WECHAT_MP_SECRET"))
	if appID == "" || appSecret == "" {
		c.Redirect(http.StatusFound, "/api/auth/login?wechat=failed")
		return
	}

	tokenURL := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		url.QueryEscape(appID), url.QueryEscape(appSecret), url.QueryEscape(code))
	resp, err := http.Get(tokenURL)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?wechat=failed")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?wechat=failed")
		return
	}
	var oauthResp struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err = json.Unmarshal(body, &oauthResp); err != nil || oauthResp.ErrCode != 0 || oauthResp.OpenID == "" {
		c.Redirect(http.StatusFound, "/api/auth/login?wechat=failed")
		return
	}

	user, userErr := h.findOrCreateWechatUser(oauthResp.OpenID, &wechatUserInfoResp{
		OpenID:  oauthResp.OpenID,
		UnionID: oauthResp.UnionID,
	})
	if userErr != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?wechat=failed")
		return
	}

	wechatLoginSessions.Lock()
	if session, ok := wechatLoginSessions.items[sceneID]; ok {
		session.OpenID = oauthResp.OpenID
		session.UserID = user.ID
		session.Status = "success"
	}
	wechatLoginSessions.Unlock()

	c.Redirect(http.StatusFound, "/api/auth/login?wechat=ok")
}

func (h *Handlers) handleWechatHealth(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func (h *Handlers) handleOIDCAuthorize(c *gin.Context) {
	clientID := strings.TrimSpace(c.Query("client_id"))
	redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
	state := strings.TrimSpace(c.Query("state"))
	if clientID == "" || redirectURI == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("client_id and redirect_uri are required"))
		return
	}
	oauthClient, err := models.GetEnabledOAuthClientByClientID(h.db, clientID)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid oauth client"))
		return
	}
	if !oauthClient.MatchRedirectURI(redirectURI) {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("redirect_uri does not match registered client"))
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		loginURL := fmt.Sprintf("/api/auth/login?client_id=%s&redirect_uri=%s", url.QueryEscape(clientID), url.QueryEscape(redirectURI))
		if state != "" {
			loginURL += "&state=" + url.QueryEscape(state)
		}
		c.Redirect(http.StatusFound, loginURL)
		return
	}

	code := newOIDCCode()
	oidcAuthCodeStore.Lock()
	oidcAuthCodeStore.items[code] = &oidcAuthCode{
		Code:        code,
		UserID:      user.ID,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(2 * time.Minute),
	}
	oidcAuthCodeStore.Unlock()
	cleanExpiredOIDCCodes()

	target, err := url.Parse(redirectURI)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid redirect_uri"))
		return
	}
	q := target.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	target.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, target.String())
}

func (h *Handlers) handleOIDCToken(c *gin.Context) {
	req := readOIDCTokenReq(c)
	if req.GrantType != "authorization_code" || req.Code == "" || req.ClientID == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid oidc token request"))
		return
	}
	h.processOIDCTokenReq(c, req)
}

func (h *Handlers) handleOIDCExchange(c *gin.Context) {
	req := readOIDCTokenReq(c)
	req.GrantType = "authorization_code"
	if req.Code == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("code is required"))
		return
	}
	if configuredClientID := strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")); configuredClientID != "" {
		req.ClientID = configuredClientID
	}
	if req.ClientID == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("client_id is required"))
		return
	}
	req.ClientSecret = strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET"))
	if req.ClientSecret == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("OIDC_CLIENT_SECRET is not configured"))
		return
	}
	h.processOIDCTokenReq(c, req)
}

type oidcTokenReq struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

func readOIDCTokenReq(c *gin.Context) oidcTokenReq {
	var req oidcTokenReq
	_ = c.ShouldBindJSON(&req)
	if req.GrantType == "" {
		req.GrantType = c.PostForm("grant_type")
	}
	if req.Code == "" {
		req.Code = c.PostForm("code")
	}
	if req.ClientID == "" {
		req.ClientID = c.PostForm("client_id")
	}
	if req.ClientSecret == "" {
		req.ClientSecret = c.PostForm("client_secret")
	}
	if req.RedirectURI == "" {
		req.RedirectURI = c.PostForm("redirect_uri")
	}
	req.GrantType = strings.TrimSpace(req.GrantType)
	req.Code = strings.TrimSpace(req.Code)
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.ClientSecret = strings.TrimSpace(req.ClientSecret)
	req.RedirectURI = strings.TrimSpace(req.RedirectURI)
	return req
}

func (h *Handlers) processOIDCTokenReq(c *gin.Context, req oidcTokenReq) {
	oauthClient, err := models.GetEnabledOAuthClientByClientID(h.db, req.ClientID)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid oauth client"))
		return
	}
	if strings.TrimSpace(oauthClient.ClientSecret) != "" {
		if req.ClientSecret == "" {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("client_secret is required"))
			return
		}
		// Use constant-time compare to avoid leaking secret mismatch timing.
		if subtle.ConstantTimeCompare([]byte(req.ClientSecret), []byte(strings.TrimSpace(oauthClient.ClientSecret))) != 1 {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid client_secret"))
			return
		}
	}
	if req.RedirectURI != "" && !oauthClient.MatchRedirectURI(req.RedirectURI) {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("redirect_uri does not match registered client"))
		return
	}

	oidcAuthCodeStore.Lock()
	codeData, ok := oidcAuthCodeStore.items[req.Code]
	if !ok || codeData.Used || time.Now().After(codeData.ExpiresAt) || codeData.ClientID != req.ClientID {
		oidcAuthCodeStore.Unlock()
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid or expired code"))
		return
	}
	if !oauthClient.MatchRedirectURI(codeData.RedirectURI) {
		oidcAuthCodeStore.Unlock()
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("code redirect_uri mismatch"))
		return
	}
	codeData.Used = true
	oidcAuthCodeStore.Unlock()

	user, err := models.GetUserByUID(h.db, codeData.UserID)
	if err != nil || user == nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("user not found"))
		return
	}
	accessToken, refreshToken := buildTokenPair(h.db, user, 24*time.Hour)
	response.Success(c, "oidc token issued", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int64((24 * time.Hour).Seconds()),
		"user":          user,
	})
}

func (h *Handlers) handleRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req)
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		req.RefreshToken = strings.TrimSpace(c.PostForm("refresh_token"))
	}
	if req.RefreshToken == "" {
		req.RefreshToken = strings.TrimSpace(c.GetHeader("X-Refresh-Token"))
	}
	if req.RefreshToken == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("refresh_token is required"))
		return
	}
	user, err := models.DecodeRefreshToken(h.db, req.RefreshToken, false)
	if err != nil || user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("invalid refresh token"))
		return
	}
	if err = models.CheckUserAllowLogin(h.db, user); err != nil {
		response.AbortWithJSONError(c, http.StatusForbidden, err)
		return
	}
	accessToken, refreshToken := buildTokenPair(h.db, user, 24*time.Hour)
	response.Success(c, "token refreshed", gin.H{
		"token":         accessToken,
		"access_token":  accessToken,
		"refreshToken":  refreshToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

func (h *Handlers) handleGitHubLogin(c *gin.Context) {
	clientID, _, redirectURI, err := getGitHubOAuthConfig(c)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	nonce := newGitHubStateNonce()
	stateData := &githubOAuthState{
		Nonce:        nonce,
		RedirectURL:  strings.TrimSpace(c.Query("redirecturl")),
		OIDCClient:   strings.TrimSpace(c.Query("client_id")),
		OIDCRedirect: strings.TrimSpace(c.Query("redirect_uri")),
		OIDCState:    strings.TrimSpace(c.Query("state")),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("bind")), "1") {
		if u := models.CurrentUser(c); u != nil {
			stateData.BindUserID = u.ID
		}
	}
	githubOAuthStateStore.Lock()
	githubOAuthStateStore.items[nonce] = stateData
	githubOAuthStateStore.Unlock()
	cleanExpiredGitHubStates()

	target, _ := url.Parse("https://github.com/login/oauth/authorize")
	q := target.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", "read:user user:email")
	q.Set("state", nonce)
	target.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, target.String())
}

func (h *Handlers) handleGitHubCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}
	githubOAuthStateStore.Lock()
	stateData, ok := githubOAuthStateStore.items[state]
	if ok {
		delete(githubOAuthStateStore.items, state)
	}
	githubOAuthStateStore.Unlock()
	if !ok || time.Now().After(stateData.ExpiresAt) {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}

	clientID, clientSecret, redirectURI, err := getGitHubOAuthConfig(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}

	tokenReqBody := url.Values{}
	tokenReqBody.Set("client_id", clientID)
	tokenReqBody.Set("client_secret", clientSecret)
	tokenReqBody.Set("code", code)
	tokenReqBody.Set("redirect_uri", redirectURI)

	tokenReq, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(tokenReqBody.Encode()))
	tokenReq.Header.Set("Accept", "application/json")
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}
	defer tokenResp.Body.Close()
	tokenRaw, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}
	var tokenData githubOAuthTokenResp
	if err = json.Unmarshal(tokenRaw, &tokenData); err != nil || tokenData.AccessToken == "" || tokenData.Error != "" {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}

	userReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	userReq.Header.Set("Accept", "application/vnd.github+json")
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	userReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}
	defer userResp.Body.Close()
	userRaw, err := io.ReadAll(userResp.Body)
	if err != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}
	var githubUser githubUserResp
	if err = json.Unmarshal(userRaw, &githubUser); err != nil || githubUser.ID <= 0 {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}

	githubEmail := strings.TrimSpace(githubUser.Email)
	if githubEmail == "" {
		emailReq, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Accept", "application/vnd.github+json")
		emailReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
		emailReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		emailResp, reqErr := http.DefaultClient.Do(emailReq)
		if reqErr == nil {
			defer emailResp.Body.Close()
			if emailRaw, readErr := io.ReadAll(emailResp.Body); readErr == nil {
				var emails []githubEmailResp
				if unmarshalErr := json.Unmarshal(emailRaw, &emails); unmarshalErr == nil {
					for _, em := range emails {
						if em.Email != "" && em.Primary && em.Verified {
							githubEmail = strings.TrimSpace(em.Email)
							break
						}
					}
					if githubEmail == "" {
						for _, em := range emails {
							if em.Email != "" && em.Verified {
								githubEmail = strings.TrimSpace(em.Email)
								break
							}
						}
					}
				}
			}
		}
	}

	user, userErr := h.findOrCreateGitHubUser(&githubUser, githubEmail)
	if userErr != nil {
		c.Redirect(http.StatusFound, "/api/auth/login?github=failed")
		return
	}
	if stateData.BindUserID > 0 {
		bindUser, err := models.GetUserByUID(h.db, stateData.BindUserID)
		if err == nil && bindUser != nil {
			newGithubID := strconv.FormatInt(githubUser.ID, 10)
			redirectURL := strings.TrimSpace(stateData.RedirectURL)
			if redirectURL == "" {
				redirectURL = "/profile"
			}
			buildBindRedirect := func(bindStatus string) string {
				target, parseErr := url.Parse(redirectURL)
				if parseErr != nil {
					return "/profile?bind=" + url.QueryEscape(bindStatus)
				}
				q := target.Query()
				q.Set("bind", bindStatus)
				target.RawQuery = q.Encode()
				return target.String()
			}
			// A user can only bind one GitHub account.
			if strings.TrimSpace(bindUser.GithubID) != "" && strings.TrimSpace(bindUser.GithubID) != newGithubID {
				c.Redirect(http.StatusFound, buildBindRedirect("github_already_bound"))
				return
			}
			// A GitHub account can only belong to one local user.
			var existing models.User
			if findErr := h.db.Where("github_id = ?", newGithubID).First(&existing).Error; findErr == nil && existing.ID != bindUser.ID {
				c.Redirect(http.StatusFound, buildBindRedirect("github_bound_other"))
				return
			}
			updates := map[string]any{
				"github_id":    newGithubID,
				"github_login": strings.TrimSpace(githubUser.Login),
			}
			if githubUser.AvatarURL != "" {
				updates["avatar"] = githubUser.AvatarURL
			}
			_ = h.db.Model(bindUser).Updates(updates).Error
			c.Redirect(http.StatusFound, buildBindRedirect("github_ok"))
			return
		}
	}
	models.Login(c, user)
	token, refreshToken := buildTokenPair(h.db, user, 7*24*time.Hour)

	target, _ := url.Parse("/api/auth/login")
	q := target.Query()
	q.Set("github", "ok")
	q.Set("token", token)
	q.Set("refreshToken", refreshToken)
	if stateData.RedirectURL != "" {
		q.Set("redirecturl", stateData.RedirectURL)
	}
	if stateData.OIDCClient != "" {
		q.Set("client_id", stateData.OIDCClient)
	}
	if stateData.OIDCRedirect != "" {
		q.Set("redirect_uri", stateData.OIDCRedirect)
	}
	if stateData.OIDCState != "" {
		q.Set("state", stateData.OIDCState)
	}
	target.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, target.String())
}

func (h *Handlers) handleWechatBindCode(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}
	sessionID := newWechatSessionID()
	bindCode := newWechatLoginCode()
	expiresIn := 600
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	session := &wechatBindSession{
		SessionID: sessionID,
		Status:    "pending",
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		BindCode:  bindCode,
	}
	wechatBindSessions.Lock()
	wechatBindSessions.items[sessionID] = session
	wechatBindSessions.Unlock()
	cleanExpiredWechatBindSessions()
	response.Success(c, "wechat bind code generated", gin.H{
		"sessionId":    sessionID,
		"bindCode":     bindCode,
		"expiresAt":    expiresAt.Unix(),
		"expiresInSec": expiresIn,
	})
}

func (h *Handlers) handleWechatBindStatus(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}
	sessionID := strings.TrimSpace(c.Query("sessionId"))
	if sessionID == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("sessionId is required"))
		return
	}
	wechatBindSessions.RLock()
	session, ok := wechatBindSessions.items[sessionID]
	wechatBindSessions.RUnlock()
	if !ok || time.Now().After(session.ExpiresAt) {
		response.Success(c, "wechat bind status", gin.H{"status": "expired"})
		return
	}
	if session.UserID != user.ID {
		response.AbortWithJSONError(c, http.StatusForbidden, errors.New("invalid bind session owner"))
		return
	}
	payload := gin.H{
		"status":    session.Status,
		"expiresAt": session.ExpiresAt.Unix(),
	}
	if session.Reason != "" {
		payload["reason"] = session.Reason
	}
	if session.Status == "success" {
		wechatBindSessions.Lock()
		delete(wechatBindSessions.items, sessionID)
		wechatBindSessions.Unlock()
	}
	response.Success(c, "wechat bind status", payload)
}

func (h *Handlers) handleWechatLogin(c *gin.Context) {
	c.Redirect(http.StatusFound, "/api/auth/login")
}

func (h *Handlers) handleWechatLoginCallback(c *gin.Context) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")
	token := getWechatMPToken()
	if token == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusBadRequest, "error")
		return
	}
	if !verifyWechatSignature(token, signature, timestamp, nonce) {
		expected := calcWechatSignature(token, timestamp, nonce)
		logger.Warn("wechat verify token failed",
			zap.String("path", c.Request.URL.Path),
			zap.String("signature", signature),
			zap.String("expected", expected),
			zap.String("timestamp", timestamp),
			zap.String("nonce", nonce),
			zap.String("echostr", echostr),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("remoteIP", c.ClientIP()))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusForbidden, "error")
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html.EscapeString(echostr))
}

func (h *Handlers) handleWechatLoginMessage(c *gin.Context) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusOK, "success")
		return
	}

	token := getWechatMPToken()
	if token == "" {
		c.String(http.StatusForbidden, "invalid signature")
		return
	}

	encryptType := strings.ToLower(strings.TrimSpace(c.Query("encrypt_type")))
	if encryptType == "aes" {
		var envelope struct {
			Encrypt string `xml:"Encrypt"`
		}
		if err = xml.Unmarshal(raw, &envelope); err != nil || envelope.Encrypt == "" {
			logger.Warn("wechat aes envelope invalid", zap.Error(err))
			c.String(http.StatusOK, "success")
			return
		}

		msgSignature := c.Query("msg_signature")
		expected := calcWechatMsgSignature(token, timestamp, nonce, envelope.Encrypt)
		if msgSignature == "" || msgSignature != expected {
			logger.Warn("wechat aes signature invalid",
				zap.String("msg_signature", msgSignature),
				zap.String("expected", expected),
				zap.String("timestamp", timestamp),
				zap.String("nonce", nonce),
				zap.String("query", c.Request.URL.RawQuery))
			c.String(http.StatusForbidden, "invalid signature")
			return
		}

		appID := strings.TrimSpace(os.Getenv("WECHAT_MP_APP_ID"))
		aesKey := strings.TrimSpace(os.Getenv("WECHAT_MP_AES_KEY"))
		if aesKey == "" {
			logger.Warn("wechat aes key missing")
			c.String(http.StatusOK, "success")
			return
		}
		raw, err = decryptWechatXML(aesKey, appID, envelope.Encrypt)
		if err != nil {
			logger.Warn("wechat aes decrypt failed", zap.Error(err))
			c.String(http.StatusOK, "success")
			return
		}
	} else {
		if !verifyWechatSignature(token, signature, timestamp, nonce) {
			expected := calcWechatSignature(token, timestamp, nonce)
			logger.Warn("wechat message signature invalid",
				zap.String("path", c.Request.URL.Path),
				zap.String("signature", signature),
				zap.String("expected", expected),
				zap.String("timestamp", timestamp),
				zap.String("nonce", nonce),
				zap.String("query", c.Request.URL.RawQuery),
				zap.String("remoteIP", c.ClientIP()))
			c.String(http.StatusForbidden, "invalid signature")
			return
		}
	}

	var msg wechatMessageXML
	if err = xml.Unmarshal(raw, &msg); err != nil {
		c.String(http.StatusOK, "success")
		return
	}
	if strings.ToLower(msg.MsgType) != "event" {
		// 纯消息推送登录：用户在公众号发送登录码（文本消息）
		if strings.ToLower(msg.MsgType) == "text" {
			inputCode := strings.ToUpper(strings.TrimSpace(msg.Content))
			if inputCode == "" {
				sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "请发送登录码完成登录")
				return
			}

			var bindMatched *wechatBindSession
			wechatBindSessions.Lock()
			for _, s := range wechatBindSessions.items {
				if time.Now().Before(s.ExpiresAt) && strings.EqualFold(s.BindCode, inputCode) {
					bindMatched = s
					break
				}
			}
			if bindMatched != nil {
				bindMatched.OpenID = msg.FromUserName
				// If the OpenID is already bound to another account, reject this bind.
				var existing models.User
				if err := h.db.Where("wechat_open_id = ?", msg.FromUserName).First(&existing).Error; err == nil && existing.ID != bindMatched.UserID {
					bindMatched.Status = "failed"
					bindMatched.Reason = "already_bound"
				} else {
					updates := map[string]any{
						"wechat_open_id": msg.FromUserName,
					}
					if wechatUserInfo, userInfoErr := func() (*wechatUserInfoResp, error) {
						var info wechatUserInfoResp
						accessToken, tokenErr := getWechatAccessToken()
						if tokenErr != nil {
							return &info, tokenErr
						}
						userInfoURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", accessToken, msg.FromUserName)
						userResp, reqErr := http.Get(userInfoURL)
						if reqErr != nil {
							return &info, reqErr
						}
						defer userResp.Body.Close()
						raw, readErr := io.ReadAll(userResp.Body)
						if readErr != nil {
							return &info, readErr
						}
						_ = json.Unmarshal(raw, &info)
						return &info, nil
					}(); userInfoErr == nil && wechatUserInfo != nil && wechatUserInfo.UnionID != "" {
						updates["wechat_union_id"] = wechatUserInfo.UnionID
					}
					if err := h.db.Model(&models.User{}).Where("id = ?", bindMatched.UserID).Updates(updates).Error; err != nil {
						bindMatched.Status = "failed"
						bindMatched.Reason = "bind_failed"
					} else {
						now := time.Now()
						bindMatched.BoundAt = &now
						bindMatched.Status = "success"
					}
				}
			}
			wechatBindSessions.Unlock()
			if bindMatched != nil {
				if bindMatched.Status == "success" {
					sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "微信绑定成功，请返回网页查看")
				} else if bindMatched.Reason == "already_bound" {
					sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "该微信已绑定其他账号，无法重复绑定")
				} else {
					sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "微信绑定失败，请稍后重试")
				}
				return
			}

			var matched *wechatLoginSession
			wechatLoginSessions.Lock()
			// 精确匹配一次性登录码
			for _, s := range wechatLoginSessions.items {
				if time.Now().Before(s.ExpiresAt) && strings.EqualFold(s.LoginCode, inputCode) {
					matched = s
					break
				}
			}
			if matched != nil {
				now := time.Now()
				matched.ScannedAt = &now
				matched.Status = "scanned"
				matched.OpenID = msg.FromUserName
			}
			wechatLoginSessions.Unlock()

			if matched == nil {
				sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "验证码无效或已过期，请刷新页面获取最新验证码")
				return
			}

			var wechatUserInfo wechatUserInfoResp
			if accessToken, tokenErr := getWechatAccessToken(); tokenErr == nil && msg.FromUserName != "" {
				userInfoURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", accessToken, msg.FromUserName)
				if userResp, reqErr := http.Get(userInfoURL); reqErr == nil {
					defer userResp.Body.Close()
					if userRaw, readErr := io.ReadAll(userResp.Body); readErr == nil {
						_ = json.Unmarshal(userRaw, &wechatUserInfo)
					}
				}
			}

			user, userErr := h.findOrCreateWechatUser(msg.FromUserName, &wechatUserInfo)
			if userErr != nil {
				sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "登录处理失败，请稍后重试")
				return
			}

			wechatLoginSessions.Lock()
			if s, ok := wechatLoginSessions.items[matched.SceneID]; ok {
				s.UserID = user.ID
				s.Status = "success"
			}
			wechatLoginSessions.Unlock()
			sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "登录确认成功，请返回网页")
			return
		}
		c.String(http.StatusOK, "success")
		return
	}
	event := strings.ToUpper(strings.TrimSpace(msg.Event))
	if event != "SCAN" && event != "SUBSCRIBE" {
		c.String(http.StatusOK, "success")
		return
	}

	sceneID := extractSceneID(msg.EventKey)
	if sceneID == "" {
		c.String(http.StatusOK, "success")
		return
	}
	wechatLoginSessions.Lock()
	session, ok := wechatLoginSessions.items[sceneID]
	if !ok || time.Now().After(session.ExpiresAt) {
		wechatLoginSessions.Unlock()
		c.String(http.StatusOK, "success")
		return
	}
	now := time.Now()
	session.ScannedAt = &now
	session.Status = "scanned"
	session.OpenID = msg.FromUserName
	wechatLoginSessions.Unlock()

	var wechatUserInfo wechatUserInfoResp
	if accessToken, tokenErr := getWechatAccessToken(); tokenErr == nil && msg.FromUserName != "" {
		userInfoURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", accessToken, msg.FromUserName)
		if userResp, reqErr := http.Get(userInfoURL); reqErr == nil {
			defer userResp.Body.Close()
			if userRaw, readErr := io.ReadAll(userResp.Body); readErr == nil {
				_ = json.Unmarshal(userRaw, &wechatUserInfo)
			}
		}
	}

	var loginUser *models.User
	if bindEmail := strings.TrimSpace(os.Getenv("WECHAT_LOGIN_BIND_EMAIL")); bindEmail != "" {
		if u, findErr := models.GetUserByEmail(h.db, bindEmail); findErr == nil && u != nil {
			loginUser = u
		}
	}
	if loginUser == nil {
		openID := msg.FromUserName
		if wechatUserInfo.OpenID != "" {
			openID = wechatUserInfo.OpenID
		}
		if u, userErr := h.findOrCreateWechatUser(openID, &wechatUserInfo); userErr == nil {
			loginUser = u
		}
	}
	if loginUser != nil {
		wechatLoginSessions.Lock()
		if s, exists := wechatLoginSessions.items[sceneID]; exists {
			s.UserID = loginUser.ID
			s.Status = "success"
		}
		wechatLoginSessions.Unlock()
	}

	if event == "SUBSCRIBE" {
		sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "欢迎关注，正在为您登录...")
		return
	}
	sendWechatReplyText(c, msg.FromUserName, msg.ToUserName, "正在为您登录...")
}

// handleUserLogout handle user logout
func (h *Handlers) handleUserLogout(c *gin.Context) {
	user := models.CurrentUser(c)
	if user != nil {
		models.Logout(c, user)
	} else {
		session := sessions.Default(c)
		session.Delete(constants.UserField)
		_ = session.Save()
	}
	next := c.Query("next")
	if next != "" {
		c.Redirect(http.StatusFound, next)
		return
	}
	response.Success(c, "Logout Success", nil)
}

// handleUserInfo handle user info
func (h *Handlers) handleUserInfo(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	withToken := c.Query("with_token")
	if withToken != "" {
		expired, err := time.ParseDuration(withToken)
		if err == nil {
			if expired >= 24*time.Hour {
				expired = 24 * time.Hour
			}
			user.AuthToken = models.BuildAuthToken(user, expired, false)
		}
	}
	response.Success(c, "success", user)
}

// handleUserSigninByEmail handle user signin by email
func (h *Handlers) handleUserSigninByEmail(c *gin.Context) {
	var form models.EmailOperatorForm
	if err := c.BindJSON(&form); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 1. IP限流检查
	if utils.GlobalLoginSecurityManager != nil {
		if err := utils.GlobalLoginSecurityManager.CheckIPRateLimit(clientIP); err != nil {
			response.AbortWithJSONError(c, http.StatusTooManyRequests, err)
			return
		}
	}

	// 2. 账号锁定检查
	if utils.GlobalLoginSecurityManager != nil {
		checkLockFunc := func(db *gorm.DB, email string, userID uint) (*utils.AccountLockInfo, error) {
			lock, err := models.GetAccountLock(db, email, userID)
			if err != nil {
				return nil, err
			}
			if lock == nil {
				return nil, nil
			}
			return &utils.AccountLockInfo{
				IsLocked: lock.IsLocked(),
				UnlockAt: lock.UnlockAt,
			}, nil
		}
		if err := utils.GlobalLoginSecurityManager.CheckAccountLock(db, form.Email, 0, checkLockFunc); err != nil {
			response.AbortWithJSONError(c, http.StatusForbidden, err)
			return
		}
	}

	// 3. 验证码验证（随机图形/点击）
	if utilscaptcha.GlobalManager != nil {
		valid, err := verifyRequestCaptcha(form.CaptchaID, form.CaptchaCode, form.CaptchaData, form.CaptchaType)
		if err != nil || !valid {
			if utils.GlobalLoginSecurityManager != nil {
				recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
					_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
					return err
				}
				utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, 0, clientIP, recordFunc)
			}
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid captcha"))
			return
		}
	}

	// 检查邮箱是否为空
	if form.Email == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("email is required"))
		return
	}

	// 4. 获取用户
	user, err := models.GetUserByEmail(db, form.Email)
	if err != nil {
		if utils.GlobalLoginSecurityManager != nil {
			recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
				_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
				return err
			}
			utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, 0, clientIP, recordFunc)
		}
		response.Fail(c, "user not exists", errors.New("user not exists"))
		return
	}

	// 5. 校验验证码
	if form.Code == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("verification code is required"))
		return
	}

	// 从缓存中获取验证码
	cachedCode, ok := utils.GlobalCache.Get(form.Email)
	if !ok || cachedCode != form.Code {
		if utils.GlobalLoginSecurityManager != nil {
			recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
				_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
				return err
			}
			utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, user.ID, clientIP, recordFunc)
		}
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid verification code"))
		return
	}

	// 清除已用验证码
	utils.GlobalCache.Remove(form.Email)

	// 6. 检查用户是否允许登录（激活、启用等）
	err = models.CheckUserAllowLogin(db, user)
	if err != nil {
		if utils.GlobalLoginSecurityManager != nil {
			recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
				_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
				return err
			}
			utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, user.ID, clientIP, recordFunc)
		}
		response.AbortWithJSONError(c, http.StatusForbidden, err)
		return
	}

	// 7. 获取IP地理位置
	country, city, location := "Unknown", "Unknown", "Unknown"
	if h.ipLocationService != nil {
		country, city, location, _ = h.ipLocationService.GetLocation(clientIP)
	}

	// 8. 检测异地登录
	isSuspicious := false
	if utils.GlobalLoginSecurityManager != nil {
		getLocationsFunc := func(db *gorm.DB, userID uint, limit int) ([]utils.LoginLocation, error) {
			histories, err := models.GetRecentLoginLocations(db, userID, limit)
			if err != nil {
				return nil, err
			}
			locations := make([]utils.LoginLocation, len(histories))
			for i, h := range histories {
				locations[i] = utils.LoginLocation{
					Country: h.Country,
					City:    h.City,
				}
			}
			return locations, nil
		}
		isSuspicious, _ = utils.GlobalLoginSecurityManager.DetectSuspiciousLogin(db, user.ID, clientIP, location, country, getLocationsFunc)
		if isSuspicious {
			logger.Warn("Suspicious login detected",
				zap.Uint("userID", user.ID),
				zap.String("email", user.Email),
				zap.String("ip", clientIP),
				zap.String("location", location))
		}
	}

	// 9. 解析设备信息
	deviceType, os, browser := utils.ParseUserAgent(userAgent)
	deviceID := utils.GetDeviceID(userAgent, clientIP)

	// 10. 检查设备信任状态
	isTrusted, err := models.CheckDeviceTrust(db, user.ID, deviceID)
	if err != nil {
		logger.Warn("Failed to check device trust", zap.Error(err))
	}

	logger.Info("Device trust check result",
		zap.String("deviceID", deviceID),
		zap.Bool("isTrusted", isTrusted),
		zap.Bool("isSuspicious", isSuspicious),
		zap.Error(err))

	// 如果设备不被信任，要求额外验证或拒绝登录
	// 邮箱验证码登录本身就是一种额外验证，所以对于邮箱登录我们可以更宽松一些
	if !isTrusted {
		logger.Info("Email login from untrusted device, but allowing due to email verification",
			zap.Uint("userID", user.ID),
			zap.String("email", user.Email),
			zap.String("deviceID", deviceID),
			zap.String("ip", clientIP))

		// 记录为可疑但成功的登录
		isSuspicious = true
	}

	// 11. 创建设备记录
	if _, err := models.CreateOrUpdateUserDevice(db, user.ID, deviceID, fmt.Sprintf("%s on %s", browser, os), deviceType, os, browser, userAgent, clientIP, location); err != nil {
		logger.Warn("Failed to create/update user device", zap.Error(err))
	}

	// 12. 记录登录历史
	if err := models.RecordLoginHistory(db, user.ID, form.Email, clientIP, location, country, city, userAgent, deviceID, "email", true, "", isSuspicious); err != nil {
		logger.Warn("Failed to record login history", zap.Error(err))
	}

	// 13. 发送新设备登录警告邮件（异步）
	logger.Info("Checking new device login alert conditions",
		zap.Bool("isTrusted", isTrusted),
		zap.Bool("isSuspicious", isSuspicious),
		zap.String("deviceID", deviceID))

	// 临时强制触发测试
	logger.Info("FORCE TESTING: Sending new device login alert signal regardless of conditions")
	deviceInfo := map[string]interface{}{
		"deviceID":     deviceID,
		"clientIP":     clientIP,
		"location":     location,
		"deviceType":   deviceType,
		"os":           os,
		"browser":      browser,
		"isSuspicious": isSuspicious,
		"loginTime":    time.Now().Format("2006-01-02 15:04:05"),
	}
	utils.Sig().Emit(constants.SigUserNewDeviceLogin, user, deviceInfo, db)

	if !isTrusted || isSuspicious {
		logger.Info("Sending new device login alert signal",
			zap.String("email", user.Email),
			zap.String("deviceID", deviceID),
			zap.Bool("isTrusted", isTrusted),
			zap.Bool("isSuspicious", isSuspicious))

		// 使用信号发送新设备登录警告邮件
		deviceInfo := map[string]interface{}{
			"deviceID":     deviceID,
			"clientIP":     clientIP,
			"location":     location,
			"deviceType":   deviceType,
			"os":           os,
			"browser":      browser,
			"isSuspicious": isSuspicious,
			"loginTime":    time.Now().Format("2006-01-02 15:04:05"),
		}
		utils.Sig().Emit(constants.SigUserNewDeviceLogin, user, deviceInfo, db)
	} else {
		logger.Info("Skipping new device login alert - device is trusted and not suspicious",
			zap.String("email", user.Email),
			zap.String("deviceID", deviceID))
	}

	// 14. 邮箱验证码登录成功后，重置密码登录限制
	// 删除最近的密码登录记录，允许用户重新使用密码登录
	if utils.GlobalLoginSecurityManager != nil {
		// 删除最近7天的密码登录记录，给用户一个重新开始的机会
		if err := db.Where("user_id = ? AND login_type = ? AND created_at > ?",
			user.ID, "password", time.Now().AddDate(0, 0, -7)).
			Delete(&models.LoginHistory{}).Error; err != nil {
			logger.Warn("Failed to reset password login history", zap.Error(err))
		} else {
			logger.Info("Password login history reset after email verification",
				zap.Uint("userID", user.ID),
				zap.String("email", user.Email))
		}
	}

	// 15. 清除失败登录计数
	if utils.GlobalLoginSecurityManager != nil {
		utils.GlobalLoginSecurityManager.ClearFailedLoginCount(form.Email)
	}

	// 设置时区（如果有的话）
	if form.Timezone != "" {
		models.InTimezone(c, form.Timezone)
	}

	// 登录用户，设置 Session
	models.Login(c, user)

	// 检查是否被中止
	if c.IsAborted() {
		return
	}

	// 重新从数据库加载用户信息，确保获取最新的LastLogin等信息
	updatedUser, err := models.GetUserByUID(db, user.ID)
	if err != nil {
		logger.Warn("Failed to reload user after login, using original user object", zap.Error(err))
		updatedUser = user // 如果加载失败，使用原始user对象
	} else {
		user = updatedUser // 使用更新后的用户信息
	}

	// 如果需要 Token，生成 AuthToken
	var refreshToken string
	if form.AuthToken {
		val := utils.GetValue(db, constants.KEY_AUTH_TOKEN_EXPIRED)
		expired, _ := time.ParseDuration(val)
		if expired < 24*time.Hour {
			expired = 24 * time.Hour
		}
		user.AuthToken, refreshToken = buildTokenPair(db, user, expired)
	}

	// 返回登录结果（包含可疑登录警告）
	responseData := gin.H{
		"user":         user,
		"token":        user.AuthToken, // 为了兼容前端，同时返回token字段
		"refreshToken": refreshToken,
	}
	if isSuspicious {
		responseData["suspiciousLogin"] = true
		responseData["message"] = "Login from new location or untrusted device detected. Please verify your identity."
	}

	response.Success(c, "login success", responseData)
}

// handleUserSignin handle user signin
func (h *Handlers) handleUserSigninByPassword(c *gin.Context) {
	var form models.LoginForm
	if err := c.BindJSON(&form); err != nil {
		logger.Error("Failed to bind login form", zap.Error(err))
		response.Fail(c, "login failed", err)
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 1. IP限流检查
	if utils.GlobalLoginSecurityManager != nil {
		if err := utils.GlobalLoginSecurityManager.CheckIPRateLimit(clientIP); err != nil {
			response.Fail(c, "too many login attempts", err)
			return
		}
	}

	// 2. 代理IP检测
	if utils.GlobalLoginSecurityManager != nil {
		isProxy, err := utils.GlobalLoginSecurityManager.CheckProxyIP(clientIP)
		if err != nil {
			logger.Warn("Failed to check proxy IP", zap.String("ip", clientIP), zap.Error(err))
		}
		if isProxy {
			logger.Warn("Login attempt from proxy IP", zap.String("ip", clientIP), zap.String("email", form.Email))
		}
	}

	// 3. 账号锁定检查
	if utils.GlobalLoginSecurityManager != nil {
		checkLockFunc := func(db *gorm.DB, email string, userID uint) (*utils.AccountLockInfo, error) {
			lock, err := models.GetAccountLock(db, email, userID)
			if err != nil {
				return nil, err
			}
			if lock == nil {
				return nil, nil
			}
			return &utils.AccountLockInfo{
				IsLocked: lock.IsLocked(),
				UnlockAt: lock.UnlockAt,
			}, nil
		}
		if err := utils.GlobalLoginSecurityManager.CheckAccountLock(db, form.Email, 0, checkLockFunc); err != nil {
			response.Fail(c, "account is locked", err)
			return
		}
	}

	if form.AuthToken == "" && form.Email == "" {
		logger.Warn("Login attempt without email or token", zap.String("ip", clientIP))
		response.Fail(c, "login failed", errors.New("email is required"))
		return
	}

	if form.Password == "" && form.AuthToken == "" {
		logger.Warn("Login attempt without password or token", zap.String("ip", clientIP), zap.String("email", form.Email))
		response.Fail(c, "login failed", errors.New("empty password"))
		return
	}

	// 4. 获取用户
	var user *models.User
	var err error
	if form.Password != "" {
		user, err = models.GetUserByEmail(db, form.Email)
		if err != nil {
			logger.Warn("Login attempt with non-existent email", zap.String("email", form.Email), zap.String("ip", clientIP), zap.Error(err))
			// 记录失败登录
			if utils.GlobalLoginSecurityManager != nil {
				recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
					_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
					return err
				}
				utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, 0, clientIP, recordFunc)
			}
			response.Fail(c, "用户不存在，请检查邮箱地址", nil)
			return
		}

		// 5. 检查密码登录次数限制（需要邮箱验证）
		if utils.GlobalLoginSecurityManager != nil {
			checkLimitFunc := func(db *gorm.DB, userID uint) (int64, error) {
				// 检查最近是否有邮箱验证码登录
				var recentEmailLogin int64
				err := db.Table("login_histories").
					Where("user_id = ? AND login_type = ? AND success = ? AND created_at > ?",
						userID, "email", true, time.Now().AddDate(0, 0, -7)). // 最近7天
					Count(&recentEmailLogin).Error
				if err != nil {
					return 0, err
				}

				// 如果最近7天内有邮箱登录，则重置密码登录限制
				if recentEmailLogin > 0 {
					return 0, nil // 返回0，表示没有达到限制
				}

				// 否则正常检查密码登录次数
				var count int64
				err = db.Table("login_histories").
					Where("user_id = ? AND login_type = ? AND success = ? AND created_at > ?",
						userID, "password", true, time.Now().AddDate(0, 0, -30)). // 最近30天
					Count(&count).Error
				return count, err
			}
			needsEmailVerification, err := utils.GlobalLoginSecurityManager.CheckPasswordLoginLimit(db, user.ID, form.Email, checkLimitFunc)
			if err != nil {
				logger.Warn("Failed to check password login limit", zap.Error(err))
			}
			if needsEmailVerification {
				// 需要邮箱验证码，但这里先检查密码是否正确
				passwordValid := false
				// 检查是否是加密密码格式（passwordHash:encryptedHash:salt:timestamp）
				if strings.Contains(form.Password, ":") && len(strings.Split(form.Password, ":")) == 4 {
					// 加密密码验证
					passwordValid = models.VerifyEncryptedPassword(form.Password, user.Password)
				} else {
					// 明文密码（向后兼容）
					passwordValid = models.CheckPassword(user, form.Password)
				}

				if !passwordValid {
					logger.Warn("Login failed: incorrect password (email verification required)", zap.String("email", form.Email), zap.Uint("userID", user.ID), zap.String("ip", clientIP))
					if utils.GlobalLoginSecurityManager != nil {
						recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
							_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
							return err
						}
						utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, user.ID, clientIP, recordFunc)
					}
					response.Fail(c, "密码错误，请检查后重试", nil)
					return
				}
				// 密码正确，但需要邮箱验证
				response.Success(c, "Email verification required", gin.H{
					"requiresEmailVerification": true,
					"message":                   "Password login limit reached. Please verify with email code.",
				})
				return
			}
		}

		// 6. 验证码验证（随机图形/点击）
		// 如果已经进入2FA提交阶段(twoFactorCode存在)，跳过验证码二次校验，避免同一验证码重复消费导致失败
		isTwoFactorSubmit := strings.TrimSpace(form.TwoFactorCode) != ""
		if utilscaptcha.GlobalManager != nil && !isTwoFactorSubmit {
			valid, err := verifyRequestCaptcha(form.CaptchaID, form.CaptchaCode, form.CaptchaData, form.CaptchaType)
			if err != nil || !valid {
				logger.Warn("Login failed: invalid captcha", zap.String("email", form.Email), zap.Uint("userID", user.ID), zap.String("ip", clientIP), zap.String("captchaID", form.CaptchaID), zap.Error(err))
				if utils.GlobalLoginSecurityManager != nil {
					recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
						_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
						return err
					}
					utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, user.ID, clientIP, recordFunc)
				}
				response.Fail(c, "验证码错误，请重新输入", nil)
				return
			}
		}

		// 7. 验证密码（支持加密密码和明文密码）
		passwordValid := false
		// 检查是否是加密密码格式（passwordHash:encryptedHash:salt:timestamp）
		if strings.Contains(form.Password, ":") && len(strings.Split(form.Password, ":")) == 4 {
			// 加密密码验证
			logger.Info("Verifying encrypted password",
				zap.String("email", form.Email))
			passwordValid = models.VerifyEncryptedPassword(form.Password, user.Password)
			logger.Info("Encrypted password verification result",
				zap.String("email", form.Email),
				zap.Bool("valid", passwordValid))
		} else {
			// 明文密码（向后兼容）
			passwordValid = models.CheckPassword(user, form.Password)
		}

		if !passwordValid {
			logger.Warn("Login failed: incorrect password", zap.String("email", form.Email), zap.Uint("userID", user.ID), zap.String("ip", clientIP))
			// 记录失败登录
			if utils.GlobalLoginSecurityManager != nil {
				recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
					_, err := models.CreateOrUpdateAccountLock(db, email, userID, ipAddress, failedCount)
					return err
				}
				utils.GlobalLoginSecurityManager.RecordFailedLogin(db, form.Email, user.ID, clientIP, recordFunc)
			}
			response.Fail(c, "密码错误，请检查后重试", nil)
			return
		}
	} else {
		user, err = models.DecodeHashToken(db, form.AuthToken, false)
		if err != nil {
			logger.Warn("Login failed: invalid auth token", zap.String("ip", clientIP), zap.Error(err))
			response.Fail(c, "login failed", err)
			return
		}
	}

	err = models.CheckUserAllowLogin(db, user)
	if err != nil {
		logger.Warn("Login failed: user not allowed to login", zap.String("email", form.Email), zap.Uint("userID", user.ID), zap.String("ip", clientIP), zap.Error(err))
		response.Fail(c, "user no authorization to login", err)
		return
	}

	// 7.5. 两步验证应在真正登录流程前完成
	if user.TwoFactorEnabled {
		code := strings.TrimSpace(form.TwoFactorCode)
		if code == "" {
			response.Success(c, "Two-factor authentication required", gin.H{
				"requiresTwoFactor": true,
				"message":           "Please enter your two-factor authentication code",
			})
			return
		}
		valid := totp.Validate(code, user.TwoFactorSecret)
		if !valid {
			response.Fail(c, "两步验证码错误，请重新输入", errors.New("invalid 2fa code"))
			return
		}
	}

	// 8. 获取IP地理位置
	country, city, location := "Unknown", "Unknown", "Unknown"
	if h.ipLocationService != nil {
		country, city, location, _ = h.ipLocationService.GetLocation(clientIP)
	}

	// 9. 检测异地登录
	isSuspicious := false
	if utils.GlobalLoginSecurityManager != nil {
		getLocationsFunc := func(db *gorm.DB, userID uint, limit int) ([]utils.LoginLocation, error) {
			histories, err := models.GetRecentLoginLocations(db, userID, limit)
			if err != nil {
				return nil, err
			}
			locations := make([]utils.LoginLocation, len(histories))
			for i, h := range histories {
				locations[i] = utils.LoginLocation{
					Country: h.Country,
					City:    h.City,
				}
			}
			return locations, nil
		}
		isSuspicious, _ = utils.GlobalLoginSecurityManager.DetectSuspiciousLogin(db, user.ID, clientIP, location, country, getLocationsFunc)
		if isSuspicious {
			logger.Warn("Suspicious login detected",
				zap.Uint("userID", user.ID),
				zap.String("email", user.Email),
				zap.String("ip", clientIP),
				zap.String("location", location))
		}
	}

	// 10. 解析设备信息
	deviceType, os, browser := utils.ParseUserAgent(userAgent)
	deviceID := utils.GetDeviceID(userAgent, clientIP)

	// 11. 检查设备信任状态
	isTrusted, err := models.CheckDeviceTrust(db, user.ID, deviceID)
	if err != nil {
		logger.Warn("Failed to check device trust", zap.Error(err))
	}
	if !isTrusted {
		// 检查是否是通过有效令牌登录（表示用户已经通过了之前的验证）
		isTokenLogin := form.AuthToken != ""

		if !isTokenLogin {
			// 先创建设备记录（即使是不信任的），这样设备验证时才能更新它
			if _, err := models.CreateOrUpdateUserDevice(db, user.ID, deviceID, fmt.Sprintf("%s on %s", browser, os), deviceType, os, browser, userAgent, clientIP, location); err != nil {
				logger.Warn("Failed to create/update user device before verification", zap.Error(err))
			}

			// 记录可疑登录尝试
			if err := models.RecordLoginHistory(db, user.ID, form.Email, clientIP, location, country, city, userAgent, deviceID, "password", false, "untrusted device", true); err != nil {
				logger.Warn("Failed to record login history for untrusted device", zap.Error(err))
			}

			logger.Warn("Login attempt from untrusted device",
				zap.Uint("userID", user.ID),
				zap.String("email", user.Email),
				zap.String("deviceID", deviceID),
				zap.String("ip", clientIP))

			// 返回需要设备验证的响应
			response.Success(c, "Device verification required", gin.H{
				"requiresDeviceVerification": true,
				"deviceId":                   deviceID,
				"message":                    "This device is not trusted. Please verify this device or use a trusted device to login.",
			})
			return
		} else {
			// 令牌登录时，记录警告但允许继续
			logger.Info("Token login from untrusted device allowed",
				zap.Uint("userID", user.ID),
				zap.String("email", user.Email),
				zap.String("deviceID", deviceID),
				zap.String("ip", clientIP))
		}
	}

	// 12. 创建设备记录
	if _, err := models.CreateOrUpdateUserDevice(db, user.ID, deviceID, fmt.Sprintf("%s on %s", browser, os), deviceType, os, browser, userAgent, clientIP, location); err != nil {
		logger.Warn("Failed to create/update user device", zap.Error(err))
	}

	// 13. 记录登录历史
	if err := models.RecordLoginHistory(db, user.ID, form.Email, clientIP, location, country, city, userAgent, deviceID, "password", true, "", isSuspicious); err != nil {
		logger.Warn("Failed to record login history", zap.Error(err))
	}

	// 14. 发送新设备登录警告邮件
	logger.Info("Checking new device login alert conditions",
		zap.Bool("isTrusted", isTrusted),
		zap.Bool("isSuspicious", isSuspicious),
		zap.String("deviceID", deviceID))

	if !isTrusted || isSuspicious {
		logger.Info("Sending new device login alert signal",
			zap.String("email", user.Email),
			zap.String("deviceID", deviceID),
			zap.Bool("isTrusted", isTrusted),
			zap.Bool("isSuspicious", isSuspicious))
		utils.Sig().Emit(constants.SigUserNewDeviceLogin, user, "", db)
	} else {
		logger.Info("Skipping new device login alert - device is trusted and not suspicious",
			zap.String("email", user.Email),
			zap.String("deviceID", deviceID))
	}

	// 15. 清除失败登录计数
	if utils.GlobalLoginSecurityManager != nil {
		utils.GlobalLoginSecurityManager.ClearFailedLoginCount(form.Email)
	}

	if form.Timezone != "" {
		models.InTimezone(c, form.Timezone)
	}

	// 执行登录操作（设置session等）
	models.Login(c, user)

	// 检查是否被中止（models.Login内部可能出错并中止请求）
	if c.IsAborted() {
		logger.Error("Login failed: models.Login aborted the request", zap.String("email", form.Email), zap.Uint("userID", user.ID), zap.String("ip", clientIP))
		return
	}
	updatedUser, err := models.GetUserByUID(db, user.ID)
	if err != nil {
		logger.Warn("Failed to reload user after login, using original user object", zap.Error(err))
		updatedUser = user // 如果加载失败，使用原始user对象
	} else {
		user = updatedUser // 使用更新后的用户信息
	}

	// 生成认证Token
	val := utils.GetValue(db, constants.KEY_AUTH_TOKEN_EXPIRED) // 7d
	expired, err := time.ParseDuration(val)
	if err != nil {
		logger.Warn("Failed to parse auth token expired duration, using default 7 days", zap.Error(err))
		// 7 days
		expired = 7 * 24 * time.Hour
	}
	accessToken, refreshToken := buildTokenPair(db, user, expired)
	user.AuthToken = accessToken

	// 17. 返回登录结果（包含可疑登录警告）
	responseData := gin.H{
		"user":         user,
		"token":        user.AuthToken, // 为了兼容前端，同时返回token字段
		"refreshToken": refreshToken,
	}
	if isSuspicious {
		responseData["suspiciousLogin"] = true
		responseData["message"] = "Login from new location detected. Please verify your identity."
	}

	logger.Info("Login successful", zap.String("email", form.Email), zap.Uint("userID", user.ID), zap.String("ip", clientIP))
	response.Success(c, "login successful", responseData)
}

// handleUserSignin handle user signin
func (h *Handlers) handleUserSignin(c *gin.Context) {
	var form models.LoginForm
	if err := c.BindJSON(&form); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if form.AuthToken == "" && form.Email == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("email is required"))
		return
	}

	if form.Password == "" && form.AuthToken == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("empty password"))
		return
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)
	var user *models.User
	var err error
	if form.Password != "" {
		user, err = models.GetUserByEmail(db, form.Email)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("user not exists"))
			return
		}
		if !models.CheckPassword(user, form.Password) {
			response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
	} else {
		user, err = models.DecodeHashToken(db, form.AuthToken, false)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusUnauthorized, err)
			return
		}
	}

	err = models.CheckUserAllowLogin(db, user)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusForbidden, err)
		return
	}

	// 检查是否启用了两步验证
	if user.TwoFactorEnabled {
		// 如果提供了两步验证码，验证它
		if form.TwoFactorCode != "" {
			valid := totp.Validate(form.TwoFactorCode, user.TwoFactorSecret)
			if !valid {
				response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("invalid 2fa code"))
				return
			}
		} else {
			// 需要两步验证码
			c.JSON(http.StatusOK, gin.H{
				"code": 200,
				"msg":  "Two-factor authentication required",
				"data": gin.H{
					"requiresTwoFactor": true,
					"message":           "Please enter your two-factor authentication code",
				},
			})
			return
		}
	}

	if form.Timezone != "" {
		models.InTimezone(c, form.Timezone)
	}

	models.Login(c, user)

	if form.Remember {
		val := utils.GetValue(db, constants.KEY_AUTH_TOKEN_EXPIRED) // 7d
		expired, err := time.ParseDuration(val)
		if err != nil {
			// 7 days
			expired = 7 * 24 * time.Hour
		}
		user.AuthToken = models.BuildAuthToken(user, expired, false)
	}
	c.JSON(http.StatusOK, user)
}

// handleUserSignup handle user signup
func (h *Handlers) handleUserSignup(c *gin.Context) {
	var form models.RegisterUserForm
	if err := c.BindJSON(&form); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	clientIP := c.ClientIP()

	// 1. 输入清理和验证
	var err error
	form.Email, err = utils.SanitizeAndValidate(form.Email, "email")
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	form.Password, err = utils.SanitizeAndValidate(form.Password, "password")
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if form.DisplayName != "" {
		form.DisplayName, err = utils.SanitizeAndValidate(form.DisplayName, "displayname")
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	// 2. 智能风控检查
	if utils.GlobalIntelligentRiskControl != nil {
		// 解析行为数据
		var mouseTrack []utils.MouseTrackPoint
		if form.MouseTrack != "" {
			if err := json.Unmarshal([]byte(form.MouseTrack), &mouseTrack); err != nil {
				logger.Warn("Failed to parse mouse track data", zap.Error(err))
			}
		}

		// 准备表单数据用于分析
		formData := map[string]string{
			"email":       form.Email,
			"displayName": form.DisplayName,
			"firstName":   form.FirstName,
			"lastName":    form.LastName,
		}

		// 执行智能风控检查
		if err := utils.GlobalIntelligentRiskControl.CheckRegistrationRisk(
			mouseTrack,
			form.FormFillTime,
			form.KeystrokePattern,
			formData,
		); err != nil {
			if utils.GlobalRegistrationGuard != nil {
				utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "intelligent risk control blocked")
			}
			logger.Warn("Registration blocked by intelligent risk control",
				zap.String("email", form.Email),
				zap.String("ip", clientIP),
				zap.Error(err))
			response.AbortWithJSONError(c, http.StatusForbidden, errors.New("registration blocked due to suspicious behavior"))
			return
		}
	}

	// 3. 验证码验证（随机图形/点击）
	if utilscaptcha.GlobalManager != nil {
		valid, err := verifyRequestCaptcha(form.CaptchaID, form.CaptchaCode, form.CaptchaData, form.CaptchaType)
		if err != nil || !valid {
			if utils.GlobalRegistrationGuard != nil {
				utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "invalid captcha")
			}
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid captcha"))
			return
		}
	}

	// 4. 获取并发注册锁
	lockAcquired, err := utils.AcquireRegistrationLock(form.Email)
	if err != nil || !lockAcquired {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "registration in progress")
		}
		response.AbortWithJSONError(c, http.StatusConflict, errors.New("registration in progress for this email, please try again later"))
		return
	}
	defer utils.ReleaseRegistrationLock(form.Email)

	// 5. 注册防护检查
	if utils.GlobalRegistrationGuard != nil {
		if err := utils.GlobalRegistrationGuard.CheckRegistrationAllowed(clientIP, form.Email, form.Password); err != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, err.Error())
			response.AbortWithJSONError(c, http.StatusTooManyRequests, err)
			return
		}
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)
	if models.IsExistsByEmail(db, form.Email) {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "email already exists")
		}
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("email has exists"))
		return
	}

	// 处理加密密码：如果是加密格式，提取原始密码哈希
	passwordToStore := form.Password
	if strings.Contains(form.Password, ":") && len(strings.Split(form.Password, ":")) == 4 {
		// 加密密码格式：passwordHash:encryptedHash:salt:timestamp
		parts := strings.Split(form.Password, ":")
		passwordHash := parts[0]
		// 提取原始密码的哈希，加上 sha256$ 前缀
		passwordToStore = fmt.Sprintf("sha256$%s", passwordHash)
	}

	user, err := models.CreateUserWithMeta(db, form.Email, passwordToStore, models.NormalizeUserSource(form.Source), models.UserStatusActive)
	if err != nil {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, err.Error())
		}
		logger.Warn("create user failed", zap.Any("email", form.Email), zap.Error(err))
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	// 记录成功注册
	if utils.GlobalRegistrationGuard != nil {
		utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, true, "registration successful")
	}

	vals := utils.StructAsMap(form, []string{
		"DisplayName",
		"FirstName",
		"LastName",
		"Locale",
		"Timezone",
	})

	n := time.Now().Truncate(1 * time.Second)
	vals["LastLogin"] = &n
	vals["LastLoginIP"] = c.ClientIP()

	user.DisplayName = form.DisplayName
	user.FirstName = form.FirstName
	user.LastName = form.LastName
	user.Locale = form.Locale
	user.Timezone = form.Timezone
	user.LastLogin = &n
	user.LastLoginIP = c.ClientIP()

	err = models.UpdateUserFields(db, user, vals)
	if err != nil {
		logger.Warn("update user fields fail id:", zap.Uint("userId", user.ID), zap.Any("vals", vals), zap.Error(err))
	}

	utils.Sig().Emit(constants.SigUserCreate, user, c, db)

	r := gin.H{
		"email": user.Email,
	}

	// Check if user is allowed to login before auto-login
	err = models.CheckUserAllowLogin(db, user)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusForbidden, err)
		return
	}
	models.Login(c, user) //Login now
	c.JSON(http.StatusOK, r)
}

// handleUserSignupByEmail email register email activation
func (h *Handlers) handleUserSignupByEmail(c *gin.Context) {
	var form models.EmailOperatorForm
	if err := c.BindJSON(&form); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	clientIP := c.ClientIP()
	// 1. 输入清理和验证
	var err error
	form.Email, err = utils.SanitizeAndValidate(form.Email, "email")
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	form.Password, err = utils.SanitizeAndValidate(form.Password, "password")
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if form.UserName != "" {
		form.UserName, err = utils.SanitizeAndValidate(form.UserName, "username")
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	if form.DisplayName != "" {
		form.DisplayName, err = utils.SanitizeAndValidate(form.DisplayName, "displayname")
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	// 2. 验证码验证（随机图形/点击）
	if utilscaptcha.GlobalManager != nil {
		valid, err := verifyRequestCaptcha(form.CaptchaID, form.CaptchaCode, form.CaptchaData, form.CaptchaType)
		if err != nil || !valid {
			if utils.GlobalRegistrationGuard != nil {
				utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "invalid captcha")
			}
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid captcha"))
			return
		}
	}

	// 3. 获取并发注册锁
	lockAcquired, err := utils.AcquireRegistrationLock(form.Email)
	if err != nil || !lockAcquired {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "registration in progress")
		}
		response.AbortWithJSONError(c, http.StatusConflict, errors.New("registration in progress for this email, please try again later"))
		return
	}
	defer utils.ReleaseRegistrationLock(form.Email)

	// 4. 注册防护检查
	if utils.GlobalRegistrationGuard != nil {
		if err := utils.GlobalRegistrationGuard.CheckRegistrationAllowed(clientIP, form.Email, form.Password); err != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, err.Error())
			response.AbortWithJSONError(c, http.StatusTooManyRequests, err)
			return
		}
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)
	if models.IsExistsByEmail(db, form.Email) {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "email already exists")
		}
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("email has exists"))
		return
	}
	// 从缓存中获取验证码（假设你使用的是 util.GlobalCache）
	cachedCode, ok := utils.GlobalCache.Get(form.Email)
	if !ok || cachedCode != form.Code {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, "invalid verification code")
		}
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid verification code"))
		return
	}

	// 清除已用验证码
	utils.GlobalCache.Remove(form.Email)

	// 处理加密密码：如果是加密格式，提取原始密码哈希
	passwordToStore := form.Password
	if strings.Contains(form.Password, ":") && len(strings.Split(form.Password, ":")) == 4 {
		// 加密密码格式：passwordHash:encryptedHash:salt:timestamp
		parts := strings.Split(form.Password, ":")
		passwordHash := parts[0]
		passwordToStore = fmt.Sprintf("sha256$%s", passwordHash)
	}

	user, err := models.CreateUserByEmailWithMeta(db, form.UserName, form.DisplayName, form.Email, passwordToStore, models.UserSourceSystem, models.UserStatusActive)
	if err != nil {
		if utils.GlobalRegistrationGuard != nil {
			utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, false, err.Error())
		}
		logger.Warn("create user failed", zap.Any("email", form.Email), zap.Error(err))
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	// 记录成功注册
	if utils.GlobalRegistrationGuard != nil {
		utils.GlobalRegistrationGuard.RecordRegistrationAttempt(clientIP, form.Email, true, "registration successful")
	}
	vals := utils.StructAsMap(form, []string{
		"DisplayName",
		"FirstName",
		"LastName",
		"Locale",
		"Timezone",
	})
	user.Timezone = form.Timezone
	err = models.UpdateUserFields(db, user, vals)
	if err != nil {
		logger.Warn("update user fields fail id:", zap.Uint("userId", user.ID), zap.Any("vals", vals), zap.Error(err))
	}
	utils.Sig().Emit(constants.SigUserCreate, user, db)
	sendHashMail(db, user, constants.SigUserVerifyEmail, constants.KEY_VERIFY_EMAIL_EXPIRED, "180d", c.ClientIP(), c.Request.UserAgent())
	response.Success(c, "signup success", user)
}

// handleUserUpdate Update User Info
func (h *Handlers) handleUserUpdate(c *gin.Context) {
	var req models.UpdateUserRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user := models.CurrentUser(c)
	vals := make(map[string]interface{})

	if req.Email != "" {
		vals["email"] = req.Email
	}
	if req.Phone != "" {
		vals["phone"] = req.Phone
	}
	if req.FirstName != "" {
		vals["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		vals["last_name"] = req.LastName
	}
	if req.DisplayName != "" {
		vals["display_name"] = req.DisplayName
	}
	if req.Locale != "" {
		vals["locale"] = req.Locale
	}
	if req.Timezone != "" {
		vals["timezone"] = req.Timezone
	}
	if req.Gender != "" {
		vals["gender"] = req.Gender
	}
	if req.Extra != "" {
		vals["extra"] = req.Extra
	}
	if req.Avatar != "" {
		vals["avatar"] = req.Avatar
	}
	if req.City != "" {
		vals["city"] = req.City
	}
	if req.Region != "" {
		vals["region"] = req.Region
	}
	operator := fmt.Sprintf("uid:%d", user.ID)
	if user.Email != "" {
		operator = user.Email
	}
	vals["update_by"] = operator

	err := models.UpdateUser(h.db, user, vals)
	if err != nil {
		response.Fail(c, "update user failed", err)
		return
	}

	// 重新获取更新后的用户信息
	updatedUser, err := models.GetUserByUID(h.db, user.ID)
	if err != nil {
		response.Fail(c, "failed to get updated user", err)
		return
	}
	cache.Delete(c, constants.CacheKeyUserByID+strconv.Itoa(int(user.ID)))
	response.Success(c, "update user success", updatedUser)
}

// handleUserUpdate Update User Info
func (h *Handlers) handleUserUpdateBasicInfo(c *gin.Context) {
	var req models.UserBasicInfoUpdate
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}
	user := models.CurrentUser(c)
	vals := make(map[string]interface{})

	if req.WifiName != "" {
		vals["wifiName"] = req.WifiName
	}
	if req.WifiPassword != "" {
		vals["wifiPassword"] = req.WifiPassword
	}
	if req.FatherCallName != "" {
		vals["fatherCallName"] = req.FatherCallName
	}
	if req.MotherCallName != "" {
		vals["motherCallName"] = req.MotherCallName
	}
	operator := fmt.Sprintf("uid:%d", user.ID)
	if user.Email != "" {
		operator = user.Email
	}
	vals["update_by"] = operator
	err := models.UpdateUser(h.db, user, vals)
	if err != nil {
		response.Fail(c, "update user failed", err)
		return
	}
	response.Success(c, "handle update user success", nil)
}

func (h *Handlers) handleUserUpdatePreferences(c *gin.Context) {
	var preferences struct {
		EmailNotifications *bool `json:"emailNotifications"`
		PushNotifications  *bool `json:"pushNotifications"`
	}
	if err := c.ShouldBindJSON(&preferences); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	vals := make(map[string]any)
	if preferences.EmailNotifications != nil {
		vals["email_notifications"] = *preferences.EmailNotifications
	}
	if preferences.PushNotifications != nil {
		vals["push_notifications"] = *preferences.PushNotifications
	}
	if len(vals) == 0 {
		response.Success(c, "No preferences changed", nil)
		return
	}

	user := models.CurrentUser(c)
	operator := fmt.Sprintf("uid:%d", user.ID)
	if user.Email != "" {
		operator = user.Email
	}
	vals["update_by"] = operator
	if err := models.UpdateUser(h.db, user, vals); err != nil {
		response.Fail(c, "update user failed", err)
		return
	}
	response.Success(c, "Update user preferences successfully", nil)
}

// handleChangePassword 修改密码
func (h *Handlers) handleChangePassword(c *gin.Context) {
	// 兼容前端字段：currentPassword/newPassword/confirmPassword
	// 以及旧字段：oldPassword/newPassword
	var form struct {
		OldPassword     string `json:"oldPassword"`
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	// 归一化旧密码字段
	oldPassword := form.OldPassword
	if oldPassword == "" {
		oldPassword = form.CurrentPassword
	}

	// 校验必填与确认密码一致
	if oldPassword == "" {
		response.Fail(c, "Old password is required", errors.New("old password is required"))
		return
	}
	if form.NewPassword == "" {
		response.Fail(c, "New password is required", errors.New("new password is required"))
		return
	}
	if len(form.NewPassword) < 6 {
		response.Fail(c, "New password must be at least 6 characters", errors.New("password too short"))
		return
	}
	if form.ConfirmPassword != "" && form.ConfirmPassword != form.NewPassword {
		response.Fail(c, "Confirm password does not match", errors.New("confirm password mismatch"))
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	if err := models.ChangePassword(h.db, user, oldPassword, form.NewPassword); err != nil {
		response.Fail(c, "Change password failed", err)
		return
	}

	// 修改密码成功后强制下线，要求重新登录
	models.Logout(c, user)
	response.Success(c, "Password changed successfully", map[string]any{"logout": true})
}

// handleChangePasswordByEmail 通过邮箱验证码修改密码
func (h *Handlers) handleChangePasswordByEmail(c *gin.Context) {
	var form struct {
		EmailCode       string `json:"emailCode" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	// 校验必填与确认密码一致
	if form.NewPassword == "" {
		response.Fail(c, "新密码不能为空", errors.New("new password is required"))
		return
	}
	if len(form.NewPassword) < 6 {
		response.Fail(c, "新密码至少需要6个字符", errors.New("password too short"))
		return
	}
	if form.ConfirmPassword != "" && form.ConfirmPassword != form.NewPassword {
		response.Fail(c, "确认密码不匹配", errors.New("confirm password mismatch"))
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未找到", errors.New("user not found"))
		return
	}

	// 验证邮箱验证码
	if form.EmailCode == "" {
		response.Fail(c, "邮箱验证码不能为空", errors.New("email code is required"))
		return
	}

	// 从缓存中获取验证码
	cachedCode, ok := utils.GlobalCache.Get(user.Email)
	if !ok || cachedCode != form.EmailCode {
		response.Fail(c, "邮箱验证码无效或已过期", errors.New("invalid or expired email code"))
		return
	}

	// 清除已用验证码
	utils.GlobalCache.Remove(user.Email)

	// 设置新密码（不验证旧密码）
	err := models.SetPassword(h.db, user, form.NewPassword)
	if err != nil {
		response.Fail(c, "密码修改失败", err)
		return
	}

	// 更新最后密码修改时间
	now := time.Now()
	err = models.UpdateUserFields(h.db, user, map[string]any{
		"LastPasswordChange": &now,
	})
	if err != nil {
		response.Fail(c, "更新密码修改时间失败", err)
		return
	}

	user.LastPasswordChange = &now

	// 修改密码成功后强制下线，要求重新登录
	models.Logout(c, user)
	response.Success(c, "密码修改成功", map[string]any{"logout": true})
}

// handleGetUserDevices 获取用户的登录设备列表
func (h *Handlers) handleGetUserDevices(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未找到", errors.New("user not found"))
		return
	}

	devices, err := models.GetUserLoginDevices(h.db, user.ID)
	if err != nil {
		response.Fail(c, "获取设备列表失败", err)
		return
	}

	response.Success(c, "获取设备列表成功", gin.H{
		"devices": devices,
	})
}

// handleDeleteUserDevice 删除用户设备
func (h *Handlers) handleDeleteUserDevice(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未找到", errors.New("user not found"))
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		response.Fail(c, "设备ID不能为空", errors.New("deviceId is required"))
		return
	}

	err := models.DeleteUserDevice(h.db, user.ID, deviceID)
	if err != nil {
		response.Fail(c, "删除设备失败", err)
		return
	}

	response.Success(c, "删除设备成功", nil)
}

// handleTrustUserDevice 信任用户设备
func (h *Handlers) handleTrustUserDevice(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未找到", errors.New("user not found"))
		return
	}

	var form struct {
		DeviceID string `json:"deviceId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	err := models.TrustUserDevice(h.db, user.ID, form.DeviceID)
	if err != nil {
		response.Fail(c, "信任设备失败", err)
		return
	}

	response.Success(c, "信任设备成功", nil)
}

// handleUntrustUserDevice 取消信任用户设备
func (h *Handlers) handleUntrustUserDevice(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未找到", errors.New("user not found"))
		return
	}

	var form struct {
		DeviceID string `json:"deviceId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	err := models.UntrustUserDevice(h.db, user.ID, form.DeviceID)
	if err != nil {
		response.Fail(c, "取消信任设备失败", err)
		return
	}

	response.Success(c, "取消信任设备成功", nil)
}

// handleVerifyDeviceForLogin 验证设备用于登录（无需认证）
func (h *Handlers) handleVerifyDeviceForLogin(c *gin.Context) {
	var form struct {
		Email      string `json:"email" binding:"required"`
		DeviceID   string `json:"deviceId" binding:"required"`
		VerifyCode string `json:"verifyCode" binding:"required"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 验证邮箱验证码
	cachedCode, ok := utils.GlobalCache.Get(form.Email + ":device_verify")
	if !ok || cachedCode != form.VerifyCode {
		response.Fail(c, "验证码无效或已过期", errors.New("invalid or expired verification code"))
		return
	}

	// 清除验证码
	utils.GlobalCache.Remove(form.Email + ":device_verify")

	// 获取用户
	user, err := models.GetUserByEmail(db, form.Email)
	if err != nil {
		response.Fail(c, "用户不存在", err)
		return
	}

	// 信任设备
	err = models.TrustUserDevice(db, user.ID, form.DeviceID)
	if err != nil {
		response.Fail(c, "信任设备失败", err)
		return
	}

	logger.Info("Device verified and trusted for login",
		zap.Uint("userID", user.ID),
		zap.String("email", user.Email),
		zap.String("deviceID", form.DeviceID))

	response.Success(c, "设备验证成功，现在可以使用此设备登录", nil)
}

// handleSendDeviceVerificationCode 发送设备验证码
func (h *Handlers) handleSendDeviceVerificationCode(c *gin.Context) {
	var form struct {
		Email    string `json:"email" binding:"required"`
		DeviceID string `json:"deviceId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)

	// 验证用户存在
	user, err := models.GetUserByEmail(db, form.Email)
	if err != nil {
		response.Fail(c, "用户不存在", err)
		return
	}

	// 生成验证码
	code := utils.RandNumberText(6)
	cacheKey := form.Email + ":device_verify"
	utils.GlobalCache.Add(cacheKey, code)

	// 发送邮件
	go func() {
		err := notification.NewMailNotificationWithDB(config.GlobalConfig.Services.Mail, db, user.ID).SendDeviceVerificationCode(user.Email, user.DisplayName, code, form.DeviceID)
		if err != nil {
			logger.Error("Failed to send device verification email", zap.Error(err), zap.String("email", user.Email))
		}
	}()

	logger.Info("Device verification code sent",
		zap.Uint("userID", user.ID),
		zap.String("email", user.Email),
		zap.String("deviceID", form.DeviceID))

	response.Success(c, "设备验证码已发送到您的邮箱", nil)
}

// handleResetPassword 重置密码请求
func (h *Handlers) handleResetPassword(c *gin.Context) {
	var form struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user, err := models.GetUserByEmail(h.db, form.Email)
	if err != nil {
		response.Success(c, "If the email exists, a reset link has been sent", nil)
		return
	}

	token, err := models.GeneratePasswordResetToken(h.db, user)
	if err != nil {
		response.Fail(c, "Failed to generate reset token", err)
		return
	}

	// 发射密码重置信号
	utils.Sig().Emit(constants.SigUserResetPassword, user, token, c.ClientIP(), c.Request.UserAgent(), h.db)

	response.Success(c, "If the email exists, a reset link has been sent", nil)
}

// handleResetPasswordConfirm 确认重置密码
func (h *Handlers) handleResetPasswordConfirm(c *gin.Context) {
	var form struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user, err := models.VerifyPasswordResetToken(h.db, form.Token)
	if err != nil {
		response.Fail(c, "Invalid or expired token", err)
		return
	}

	err = models.ResetPassword(h.db, user, form.Password)
	if err != nil {
		response.Fail(c, "Reset password failed", err)
		return
	}

	response.Success(c, "Password reset successfully", nil)
}

// handleVerifyEmail 验证邮箱
func (h *Handlers) handleVerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Fail(c, "Token is required", errors.New("token is required"))
		return
	}

	user, err := models.VerifyEmail(h.db, token)
	if err != nil {
		response.Fail(c, "Invalid or expired token", err)
		return
	}

	response.Success(c, "Email verified successfully", user)
}

// handleSendEmailVerification 发送邮箱验证邮件
func (h *Handlers) handleSendEmailVerification(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	logger.Info("Email verification request",
		zap.Uint("userId", user.ID),
		zap.String("email", user.Email),
		zap.Bool("emailVerified", user.EmailVerified))

	if user.EmailVerified {
		response.Fail(c, "Email already verified", errors.New("email already verified"))
		return
	}

	token, err := models.GenerateEmailVerifyToken(h.db, user)
	if err != nil {
		logger.Error("Failed to generate verification token", zap.Error(err))
		response.Fail(c, "Failed to generate verification token", err)
		return
	}

	logger.Info("Generated email verification token",
		zap.String("token", token),
		zap.String("email", user.Email))

	// 发送邮箱验证邮件
	utils.Sig().Emit(constants.SigUserVerifyEmail, user, token, c.ClientIP(), c.Request.UserAgent(), h.db)

	logger.Info("Email verification signal emitted",
		zap.String("email", user.Email),
		zap.String("token", token))

	response.Success(c, "Verification email sent", nil)
}

// handleVerifyPhone 验证手机
func (h *Handlers) handleVerifyPhone(c *gin.Context) {
	var form struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	err := models.VerifyPhone(h.db, user, form.Code)
	if err != nil {
		response.Fail(c, "Invalid verification code", err)
		return
	}

	response.Success(c, "Phone verified successfully", nil)
}

// handleGetSalt 获取随机盐（用于密码加密）
func (h *Handlers) handleGetSalt(c *gin.Context) {
	// 生成随机盐（32字符）
	salt := utils.GenerateRandomString(32)
	timestamp := time.Now().Unix()
	expiresIn := int64(300) // 5分钟有效期

	// 将盐和时间戳存储到缓存中，用于验证
	key := fmt.Sprintf("password_salt:%s", salt)
	if utils.GlobalCache != nil {
		utils.GlobalCache.Add(key, timestamp)
	}

	response.Success(c, "success", gin.H{
		"salt":      salt,
		"timestamp": timestamp,
		"expiresIn": expiresIn,
	})
}

// handleSendPhoneVerification 发送手机验证码
func (h *Handlers) handleSendPhoneVerification(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	if user.Phone == "" {
		response.Fail(c, "Phone number not set", errors.New("phone number not set"))
		return
	}

	if user.PhoneVerified {
		response.Fail(c, "Phone already verified", errors.New("phone already verified"))
		return
	}

	token, err := models.GeneratePhoneVerifyToken(h.db, user)
	if err != nil {
		response.Fail(c, "Failed to generate verification code", err)
		return
	}

	// 这里可以集成短信服务发送验证码
	// 目前只是记录日志
	logger.Info("Phone verification code", zap.String("phone", user.Phone), zap.String("code", token))

	response.Success(c, "Verification code sent", nil)
}

// handleUpdateNotificationSettings 更新通知设置
func (h *Handlers) handleUpdateNotificationSettings(c *gin.Context) {
	var settings map[string]bool

	if err := c.ShouldBindJSON(&settings); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	err := models.UpdateNotificationSettings(h.db, user, settings)
	if err != nil {
		response.Fail(c, "Update notification settings failed", err)
		return
	}

	response.Success(c, "Notification settings updated successfully", nil)
}

// handleUpdateUserPreferences 更新用户偏好设置
func (h *Handlers) handleUpdateUserPreferences(c *gin.Context) {
	var preferences map[string]string

	if err := c.ShouldBindJSON(&preferences); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	err := models.UpdatePreferences(h.db, user, preferences)
	if err != nil {
		response.Fail(c, "Update preferences failed", err)
		return
	}

	// 更新资料完整度
	err = models.UpdateProfileComplete(h.db, user)
	if err != nil {
		logger.Warn("Failed to update profile complete", zap.Error(err))
	}

	response.Success(c, "Preferences updated successfully", nil)
}

// handleGetUserStats 获取用户统计信息
func (h *Handlers) handleGetUserStats(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	// 更新资料完整度
	err := models.UpdateProfileComplete(h.db, user)
	if err != nil {
		logger.Warn("Failed to update profile complete", zap.Error(err))
	}

	stats := map[string]interface{}{
		"loginCount":         user.LoginCount,
		"profileComplete":    user.ProfileComplete,
		"emailVerified":      user.EmailVerified,
		"phoneVerified":      user.PhoneVerified,
		"twoFactorEnabled":   user.TwoFactorEnabled,
		"lastLogin":          user.LastLogin,
		"lastPasswordChange": user.LastPasswordChange,
		"createdAt":          user.CreatedAt,
	}

	response.Success(c, "User stats retrieved successfully", stats)
}

// handleUploadAvatar 处理用户头像上传
func (h *Handlers) handleUploadAvatar(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Fail(c, "Failed to get uploaded file", err)
		return
	}
	defer file.Close()

	// 验证文件类型
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	// 从文件头获取Content-Type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// 如果header中没有Content-Type，尝试从文件扩展名判断
		fileExt := strings.ToLower(filepath.Ext(header.Filename))
		extToType := map[string]string{
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".png":  "image/png",
			".gif":  "image/gif",
			".webp": "image/webp",
		}
		if mappedType, exists := extToType[fileExt]; exists {
			contentType = mappedType
		}
	}

	if !allowedTypes[contentType] {
		response.Fail(c, "Invalid file type", errors.New("only jpeg, jpg, png, gif, webp files are allowed"))
		return
	}

	// 验证文件大小 (最大5MB)
	maxSize := int64(5 * 1024 * 1024)
	if header.Size > maxSize {
		response.Fail(c, "File too large", errors.New("file size must be less than 5MB"))
		return
	}

	// 生成文件名
	fileExt := getFileExtension(header.Filename)
	fileName := fmt.Sprintf("avatars/%d_%d%s", user.ID, time.Now().Unix(), fileExt)

	reader, err := config.GlobalStore.UploadFromReader(&lingstorage.UploadFromReaderRequest{
		Reader:   file,
		Bucket:   config.GlobalConfig.Services.Storage.Bucket,
		Filename: fileName,
		Key:      fileName,
	})
	if err != nil {
		response.Fail(c, "Failed to upload avatar", err)
		return
	}
	// 更新用户头像URL
	avatarURL := reader.URL

	// 保存相对路径用于返回
	avatarRelativePath := avatarURL

	// 如果是相对路径，转换为完整URL用于数据库存储
	if strings.HasPrefix(avatarURL, "/") {
		// 获取请求的Host和Scheme
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		if host == "" {
			host = "localhost:7072" // 默认host
		}
		avatarURL = fmt.Sprintf("%s://%s%s", scheme, host, avatarURL)
	}

	err = models.UpdateUser(h.db, user, map[string]any{
		"avatar": avatarURL,
	})
	if err != nil {
		// 如果数据库更新失败，删除已上传的文件
		//store.Delete(fileName)
		response.Fail(c, "Failed to update user avatar", err)
		return
	}

	// 更新用户对象
	user.Avatar = avatarURL

	// 更新资料完整度
	err = models.UpdateProfileComplete(h.db, user)
	if err != nil {
		logger.Warn("Failed to update profile complete", zap.Error(err))
	}

	// 返回相对路径，方便反向代理
	response.Success(c, "Avatar uploaded successfully", gin.H{
		"avatar": avatarRelativePath,
	})
}

// getFileExtension 获取文件扩展名
func getFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ".jpg" // 默认扩展名
	}
	return ext
}

// isDefaultAvatar 检查是否为默认头像
func isDefaultAvatar(avatarURL string) bool {
	// 检查是否包含默认头像的标识
	return strings.Contains(avatarURL, "default") ||
		strings.Contains(avatarURL, "placeholder") ||
		strings.Contains(avatarURL, "gravatar")
}

func sendHashMail(db *gorm.DB, user *models.User, signame, expireKey, defaultExpired, clientIp, useragent string) {
	d, err := time.ParseDuration(utils.GetValue(db, expireKey))
	if err != nil {
		d, _ = time.ParseDuration(defaultExpired)
	}
	n := time.Now().Add(d)
	hash := models.EncodeHashToken(user, n.Unix(), true)

	// 发送信号，让监听器处理邮件发送
	utils.Sig().Emit(signame, user, hash, clientIp, useragent, db)
}

// handleSendEmailCode Send Email Code
func (h *Handlers) handleSendEmailCode(context *gin.Context) {
	var req models.SendEmailVerifyEmail
	if err := context.BindJSON(&req); err != nil {
		response.AbortWithJSONError(context, http.StatusBadRequest, err)
		return
	}
	req.UserAgent = context.Request.UserAgent()
	req.ClientIp = context.ClientIP()
	text := utils.RandNumberText(6)
	utils.GlobalCache.Add(req.Email, text)
	go func() {
		// Use IP address for tracking since no user context
		mailNotif := notification.NewMailNotificationWithIP(config.GlobalConfig.Services.Mail, h.db, req.ClientIp)
		err := mailNotif.SendVerificationCode(req.Email, text)
		if err != nil {
			response.AbortWithJSONError(context, http.StatusBadRequest, err)
			return
		}
	}()
	response.Success(context, "success", "Send Email Successful, Must be verified within the valid time [5 minutes]")
	return
}

// handleTwoFactorSetup 设置两步验证
func (h *Handlers) handleTwoFactorSetup(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	// 如果已经启用两步验证，返回错误
	if user.TwoFactorEnabled {
		response.Fail(c, "Two-factor authentication is already enabled", errors.New("two-factor already enabled"))
		return
	}

	// 生成新的密钥
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "LingEcho",
		AccountName: user.Email,
		SecretSize:  32,
	})
	if err != nil {
		response.Fail(c, "Failed to generate two-factor secret", err)
		return
	}

	// 保存密钥到数据库（不启用）
	err = models.UpdateUser(h.db, user, map[string]interface{}{
		"two_factor_secret": key.Secret(),
	})
	if err != nil {
		response.Fail(c, "Failed to save two-factor secret", err)
		return
	}

	// 生成QR码
	qrCode, err := qrcode.New(key.URL(), qrcode.Medium)
	if err != nil {
		response.Fail(c, "Failed to generate QR code", err)
		return
	}

	// 将QR码转换为PNG图片的base64编码
	png, err := qrCode.PNG(256)
	if err != nil {
		response.Fail(c, "Failed to generate QR code image", err)
		return
	}

	// 转换为base64字符串
	qrCodeBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	response.Success(c, "Two-factor setup initiated", gin.H{
		"secret": key.Secret(),
		"qrCode": qrCodeBase64,
		"url":    key.URL(),
	})
}

// handleTwoFactorEnable 启用两步验证
func (h *Handlers) handleTwoFactorEnable(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	// 验证TOTP代码
	valid := totp.Validate(req.Code, user.TwoFactorSecret)
	if !valid {
		response.Fail(c, "Invalid verification code", errors.New("invalid code"))
		return
	}

	// 启用两步验证
	err := models.UpdateUser(h.db, user, map[string]interface{}{
		"two_factor_enabled": true,
	})
	if err != nil {
		response.Fail(c, "Failed to enable two-factor authentication", err)
		return
	}

	response.Success(c, "Two-factor authentication enabled successfully", nil)
}

// handleTwoFactorDisable 禁用两步验证
func (h *Handlers) handleTwoFactorDisable(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	// 验证TOTP代码
	valid := totp.Validate(req.Code, user.TwoFactorSecret)
	if !valid {
		response.Fail(c, "Invalid verification code", errors.New("invalid code"))
		return
	}

	// 禁用两步验证并清除密钥
	err := models.UpdateUser(h.db, user, map[string]interface{}{
		"two_factor_enabled": false,
		"two_factor_secret":  "",
	})
	if err != nil {
		response.Fail(c, "Failed to disable two-factor authentication", err)
		return
	}

	response.Success(c, "Two-factor authentication disabled successfully", nil)
}

// handleTwoFactorStatus 获取两步验证状态
func (h *Handlers) handleTwoFactorStatus(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	response.Success(c, "Two-factor status retrieved", gin.H{
		"enabled":   user.TwoFactorEnabled,
		"hasSecret": user.TwoFactorSecret != "",
	})
}

// handleGetCaptcha 获取图形验证码
func (h *Handlers) handleGetCaptcha(c *gin.Context) {
	if utilscaptcha.GlobalManager == nil {
		response.Fail(c, "Captcha service not available", errors.New("captcha service not initialized"))
		return
	}
	captchaType := utilscaptcha.TypeImage
	if time.Now().UnixNano()%2 == 0 {
		captchaType = utilscaptcha.TypeClick
	}
	capt, err := utilscaptcha.GlobalManager.Generate(captchaType)
	if err != nil {
		response.Fail(c, "Failed to generate captcha", err)
		return
	}
	image, _ := capt.Data["image"]
	response.Success(c, "Captcha generated", gin.H{
		"id":        capt.ID,
		"type":      capt.Type,
		"image":     image,
		"count":     capt.Data["count"],
		"tolerance": capt.Data["tolerance"],
		"words":     capt.Data["words"],
	})
}

// handleVerifyCaptcha 验证图形验证码
func (h *Handlers) handleVerifyCaptcha(c *gin.Context) {
	var req struct {
		ID          string               `json:"id" binding:"required"`
		Type        string               `json:"type"`
		Code        string               `json:"code"`
		CaptchaData string               `json:"captchaData"`
		Positions   []utilscaptcha.Point `json:"positions"`
	}

	if err := c.BindJSON(&req); err != nil {
		response.Fail(c, "Invalid request", err)
		return
	}

	if utilscaptcha.GlobalManager == nil {
		response.Fail(c, "Captcha service not available", errors.New("captcha service not initialized"))
		return
	}
	captchaType := utilscaptcha.Type(strings.TrimSpace(req.Type))
	if captchaType == "" {
		captchaType = utilscaptcha.TypeImage
	}
	var verifyData interface{}
	switch captchaType {
	case utilscaptcha.TypeClick:
		verifyData = req.Positions
		if len(req.Positions) == 0 && strings.TrimSpace(req.Code) != "" {
			var points []utilscaptcha.Point
			if err := json.Unmarshal([]byte(req.Code), &points); err == nil {
				verifyData = points
			}
		}
		if len(req.Positions) == 0 && strings.TrimSpace(req.CaptchaData) != "" {
			var points []utilscaptcha.Point
			if err := json.Unmarshal([]byte(req.CaptchaData), &points); err == nil {
				verifyData = points
			}
		}
	case utilscaptcha.TypeImage:
		verifyData = req.Code
	default:
		response.Fail(c, "Invalid captcha type", errors.New("unsupported captcha type"))
		return
	}
	valid, err := utilscaptcha.GlobalManager.Verify(captchaType, req.ID, verifyData)
	if err != nil {
		response.Fail(c, "Failed to verify captcha", err)
		return
	}

	if valid {
		response.Success(c, "Captcha verified", gin.H{"valid": true})
	} else {
		response.Fail(c, "Invalid captcha code", errors.New("invalid captcha code"))
	}
}

// handleGetUserActivity 获取用户活动记录
func (h *Handlers) handleGetUserActivity(c *gin.Context) {
	user, exists := c.Get(constants.UserField)
	if !exists {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}

	// 获取查询参数
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")
	action := c.Query("action") // 可选：按操作类型筛选

	// 转换分页参数
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 || limitInt > 100 {
		limitInt = 20
	}

	// 计算偏移量
	offset := (pageInt - 1) * limitInt

	// 构建查询
	query := h.db.Model(&middleware.OperationLog{}).Where("user_id = ?", user.(*models.User).ID)

	// 按操作类型筛选
	if action != "" {
		query = query.Where("action = ?", action)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "Failed to count activities", err)
		return
	}

	// 获取活动记录
	var activities []middleware.OperationLog
	if err := query.Order("created_at DESC").Limit(limitInt).Offset(offset).Find(&activities).Error; err != nil {
		response.Fail(c, "Failed to get activities", err)
		return
	}

	// 格式化响应数据
	activityList := make([]gin.H, 0) // 初始化为空切片，确保JSON序列化为[]
	if len(activities) > 0 {
		activityList = make([]gin.H, 0, len(activities)) // 预分配容量
		for _, activity := range activities {
			activityList = append(activityList, gin.H{
				"id":        activity.ID,
				"action":    activity.Action,
				"target":    activity.Target,
				"details":   activity.Details,
				"ipAddress": activity.IPAddress,
				"userAgent": activity.UserAgent,
				"device":    activity.Device,
				"browser":   activity.Browser,
				"os":        activity.OperatingSystem,
				"location":  activity.Location,
				"createdAt": activity.CreatedAt,
			})
		}
	}

	response.Success(c, "Activities retrieved", gin.H{
		"activities": activityList,
		"pagination": gin.H{
			"page":       pageInt,
			"limit":      limitInt,
			"total":      total,
			"totalPages": (total + int64(limitInt) - 1) / int64(limitInt),
		},
	})
}
