package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/metrics"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 角色常量
const (
	RoleSuperAdmin = "superadmin" // 超级管理员
	RoleAdmin      = "admin"      // 管理员
	RoleUser       = "user"       // 普通用户
)

type SendEmailVerifyEmail struct {
	Email     string `json:"email"`
	ClientIp  string `json:"clientIp"`
	UserAgent string `json:"userAgent"`
}

type UserBasicInfoUpdate struct {
	FatherCallName string `json:"fatherCallName"`
	MotherCallName string `json:"motherCallName"`
	WifiName       string `json:"wifiName"`
	WifiPassword   string `json:"wifiPassword"`
}

type LoginForm struct {
	Email         string `json:"email" comment:"Email address"`
	Password      string `json:"password,omitempty"`
	Timezone      string `json:"timezone,omitempty"`
	Remember      bool   `json:"remember,omitempty"`
	AuthToken     string `json:"token,omitempty"`
	TwoFactorCode string `json:"twoFactorCode,omitempty"` // 两步验证码
	CaptchaID     string `json:"captchaId,omitempty"`     // 图形验证码ID
	CaptchaCode   string `json:"captchaCode,omitempty"`   // 图形验证码
	CaptchaType   string `json:"captchaType,omitempty"`   // 验证码类型: image/click
	CaptchaData   string `json:"captchaData,omitempty"`   // 点击验证码坐标数据(JSON)
}

type EmailOperatorForm struct {
	UserName    string `json:"userName"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email" comment:"Email address"`
	Code        string `json:"code"`
	Password    string `json:"password"`
	AuthToken   bool   `json:"AuthToken,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	CaptchaID   string `json:"captchaId,omitempty"`
	CaptchaCode string `json:"captchaCode,omitempty"`
	CaptchaType string `json:"captchaType,omitempty"`
	CaptchaData string `json:"captchaData,omitempty"`
}

type RegisterUserForm struct {
	Email            string `json:"email" binding:"required"`
	Password         string `json:"password" binding:"required"`
	DisplayName      string `json:"displayName"`
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	Locale           string `json:"locale"`
	Timezone         string `json:"timezone"`
	Source           string `json:"source"`
	CaptchaID        string `json:"captchaId"`
	CaptchaCode      string `json:"captchaCode"`
	CaptchaType      string `json:"captchaType"`
	CaptchaData      string `json:"captchaData"`
	MouseTrack       string `json:"mouseTrack"`
	FormFillTime     int64  `json:"formFillTime"`
	KeystrokePattern string `json:"keystrokePattern"`
}

type ChangePasswordForm struct {
	Password string `json:"password" binding:"required"`
}

type ResetPasswordForm struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordDoneForm struct {
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

type UpdateUserRequest struct {
	Email       string `form:"email" json:"email"`
	Phone       string `form:"phone" json:"phone"`
	FirstName   string `form:"firstName" json:"firstName"`
	LastName    string `form:"lastName" json:"lastName"`
	DisplayName string `form:"displayName" json:"displayName"`
	Locale      string `form:"locale" json:"locale"`
	Timezone    string `form:"timezone" json:"timezone"`
	Gender      string `form:"gender" json:"gender"`
	City        string `form:"city" json:"city"`
	Region      string `form:"region" json:"region"`
	Extra       string `form:"extra" json:"extra"`
	Avatar      string `form:"avatar" json:"avatar"`
}

type User struct {
	BaseModel
	Email                 string     `json:"email" gorm:"size:128;uniqueIndex"`
	WechatOpenID          string     `json:"wechatOpenId,omitempty" gorm:"size:128;index"`
	WechatUnionID         string     `json:"wechatUnionId,omitempty" gorm:"size:128;index"`
	Password              string     `json:"-" gorm:"size:128"`
	Phone                 string     `json:"phone,omitempty" gorm:"size:64;index"`
	FirstName             string     `json:"firstName,omitempty" gorm:"size:128"`
	LastName              string     `json:"lastName,omitempty" gorm:"size:128"`
	DisplayName           string     `json:"displayName,omitempty" gorm:"size:128"`
	IsStaff               bool       `json:"isStaff,omitempty"`
	Enabled               bool       `json:"-"`
	Activated             bool       `json:"-"`
	LastLogin             *time.Time `json:"lastLogin,omitempty"`
	LastLoginIP           string     `json:"-" gorm:"size:128"`
	Source                string     `json:"-" gorm:"size:64;index"`
	Locale                string     `json:"locale,omitempty" gorm:"size:20"`
	Timezone              string     `json:"timezone,omitempty" gorm:"size:200"`
	AuthToken             string     `json:"token,omitempty" gorm:"-"`
	Avatar                string     `json:"avatar,omitempty"`
	Gender                string     `json:"gender,omitempty"`
	City                  string     `json:"city,omitempty"`
	Region                string     `json:"region,omitempty"`
	EmailNotifications    bool       `json:"emailNotifications"`                           // 邮件通知
	PushNotifications     bool       `json:"pushNotifications" gorm:"default:true"`        // 推送通知
	SystemNotifications   bool       `json:"systemNotifications" gorm:"default:true"`      // 系统通知
	AutoCleanUnreadEmails bool       `json:"autoCleanUnreadEmails" gorm:"default:false"`   // 自动清理七天未读邮件
	EmailVerified         bool       `json:"emailVerified" gorm:"default:false"`           // 邮箱已验证
	PhoneVerified         bool       `json:"phoneVerified" gorm:"default:false"`           // 手机已验证
	TwoFactorEnabled      bool       `json:"twoFactorEnabled" gorm:"default:false"`        // 双因素认证
	TwoFactorSecret       string     `json:"-" gorm:"size:128"`                            // 双因素认证密钥
	EmailVerifyToken      string     `json:"-" gorm:"size:128"`                            // 邮箱验证令牌
	PhoneVerifyToken      string     `json:"-" gorm:"size:128"`                            // 手机验证令牌
	PasswordResetToken    string     `json:"-" gorm:"size:128"`                            // 密码重置令牌
	PasswordResetExpires  *time.Time `json:"-"`                                            // 密码重置过期时间
	EmailVerifyExpires    *time.Time `json:"-"`                                            // 邮箱验证过期时间
	LoginCount            int        `json:"loginCount" gorm:"default:0"`                  // 登录次数
	LastPasswordChange    *time.Time `json:"lastPasswordChange,omitempty"`                 // 最后密码修改时间
	ProfileComplete       int        `json:"profileComplete" gorm:"default:0"`             // 资料完整度百分比
	Role                  string     `json:"role,omitempty" gorm:"size:50;default:'user'"` // 用户角色
}

func (u *User) TableName() string {
	return constants.USER_TABLE_NAME
}

// Login Handle-User-Login
func Login(c *gin.Context, user *User) {
	db := c.MustGet(constants.DbField).(*gorm.DB)
	err := SetLastLogin(db, user, c.ClientIP())
	if err != nil {
		logger.Error("user.login", zap.Error(err))
		response.AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	// Increase login count
	err = IncrementLoginCount(db, user)
	if err != nil {
		logger.Error("user.login", zap.Error(err))
		response.AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	// Update profile completeness
	err = UpdateProfileComplete(db, user)
	if err != nil {
		logger.Error("user.login", zap.Error(err))
		response.AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	session := sessions.Default(c)
	session.Set(constants.UserField, user.ID)
	session.Save()
	utils.Sig().Emit(constants.SigUserLogin, user, db)
}

func Logout(c *gin.Context, user *User) {
	c.Set(constants.UserField, nil)
	session := sessions.Default(c)
	session.Delete(constants.UserField)
	session.Save()
	utils.Sig().Emit(constants.SigUserLogout, user, c)
}

func AuthRequired(c *gin.Context) {
	if CurrentUser(c) != nil {
		c.Next()
		return
	}

	// 检查配置是否存在
	if config.GlobalConfig == nil {
		response.AbortWithJSONError(c, http.StatusInternalServerError, errors.New("server configuration not initialized"))
		return
	}

	token := c.GetHeader(config.GlobalConfig.Auth.Header)
	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	token = strings.TrimPrefix(token, constants.AUTHORIZATION_PREFIX)
	user, err := DecodeHashToken(db, token, false)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, err)
		return
	}
	c.Set(constants.UserField, user)
	c.Next()
}

func CurrentUser(c *gin.Context) *User {
	if cachedObj, exists := c.Get(constants.UserField); exists && cachedObj != nil {
		return cachedObj.(*User)
	}
	session := sessions.Default(c)
	userId := session.Get(constants.UserField)
	if userId == nil {
		return nil
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user, err := GetUserByUID(db, userId.(uint))
	if err != nil {
		return nil
	}
	c.Set(constants.UserField, user)
	return user
}

func CheckPassword(user *User, password string) bool {
	if user.Password == "" {
		return false
	}
	return user.Password == HashPassword(password)
}

func SetPassword(db *gorm.DB, user *User, password string) (err error) {
	p := HashPassword(password)
	err = UpdateUserFields(db, user, map[string]any{
		"Password": p,
	})
	if err != nil {
		return
	}
	user.Password = p
	return
}

func HashPassword(password string) string {
	if password == "" {
		return ""
	}
	// 如果已经是哈希格式（sha256$...），直接返回
	if strings.HasPrefix(password, "sha256$") {
		return password
	}
	hashVal := sha256.Sum256([]byte(password))
	return fmt.Sprintf("sha256$%x", hashVal)
}

// VerifyEncryptedPassword 验证加密密码
// 前端发送格式：passwordHash:encryptedHash:salt:timestamp
// passwordHash = SHA256(原始密码) - 用于验证密码正确性
// encryptedHash = SHA256(原始密码 + salt + timestamp) - 用于防重放
// 后端验证：
// 1. passwordHash 与存储的密码哈希匹配（去掉 sha256$ 前缀）
// 2. 时间戳在有效期内（5分钟）
// 3. 验证 salt 是否在缓存中（防重放）
func VerifyEncryptedPassword(encryptedPassword, storedPasswordHash string) bool {
	// 解析加密密码格式：passwordHash:encryptedHash:salt:timestamp
	parts := strings.Split(encryptedPassword, ":")
	if len(parts) != 4 {
		return false
	}

	passwordHash := parts[0]
	encryptedHash := parts[1]
	salt := parts[2]
	timestampStr := parts[3]

	// 验证时间戳（5分钟内有效）
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	// 如果时间戳是毫秒级（13位数字），转换为秒级
	if timestamp > 9999999999 { // 大于10位数字，说明是毫秒时间戳
		timestamp = timestamp / 1000
	}

	now := time.Now().Unix()
	maxAge := int64(300) // 5分钟
	if now-timestamp > maxAge {
		logger.Info(fmt.Sprintf("DEBUG: Timestamp expired. now=%d, timestamp=%d, diff=%d\n",
			now, timestamp, now-timestamp))
		return false
	}

	// 验证 passwordHash 与存储的密码哈希匹配
	storedHash := strings.TrimPrefix(storedPasswordHash, "sha256$")

	if passwordHash != storedHash {
		logger.Info(fmt.Sprintf("DEBUG: Password hash mismatch. Expected: %s, Got: %s\n",
			storedHash, passwordHash))
		return false
	}

	// 验证加密哈希：SHA256(原始密码的SHA256 + salt + timestamp)
	// 注意：这里使用 passwordHash（即 SHA256(原始密码)）而不是原始密码本身
	// 前端使用毫秒时间戳计算hash，所以这里也要使用原始的毫秒时间戳
	originalTimestamp, _ := strconv.ParseInt(timestampStr, 10, 64)
	hashInput := fmt.Sprintf("%s%s%d", passwordHash, salt, originalTimestamp)
	hashVal := sha256.Sum256([]byte(hashInput))
	expectedHash := fmt.Sprintf("%x", hashVal)

	isValid := encryptedHash == expectedHash
	if !isValid {
		fmt.Printf("DEBUG: Hash verification failed. Expected: %s, Got: %s\n",
			expectedHash, encryptedHash)
	}

	return isValid
}

func GetUserByUID(db *gorm.DB, userID uint) (*User, error) {
	var val User
	start := time.Now()
	result := db.Where("id", userID).Where("enabled", true).Take(&val)
	duration := time.Since(start)

	// Record database query metrics (if monitoring system is available)
	if monitor := metrics.GetGlobalMonitor(); monitor != nil {
		monitor.RecordSQLQuery(context.Background(), "SELECT * FROM users WHERE id = ? AND enabled = ?",
			[]interface{}{userID, true}, constants.USER_TABLE_NAME, "SELECT", duration, 1, result.Error)
	}

	if result.Error != nil {
		return nil, result.Error
	}
	return &val, nil
}

func GetUserByEmail(db *gorm.DB, email string) (user *User, err error) {
	var val User
	start := time.Now()
	result := db.Table(constants.USER_TABLE_NAME).Where("email", strings.ToLower(email)).Take(&val)
	duration := time.Since(start)

	// Record database query metrics (if monitoring system is available)
	if monitor := metrics.GetGlobalMonitor(); monitor != nil {
		monitor.RecordSQLQuery(context.Background(), "SELECT * FROM users WHERE email = ?",
			[]interface{}{email}, constants.USER_TABLE_NAME, "SELECT", duration, 1, result.Error)
	}

	if result.Error != nil {
		return nil, result.Error
	}
	return &val, nil
}

func IsExistsByEmail(db *gorm.DB, email string) bool {
	_, err := GetUserByEmail(db, email)
	return err == nil
}

func AuthApiRequired(c *gin.Context) {
	if CurrentUser(c) != nil {
		c.Next()
		return
	}

	apiKey := c.GetHeader(constants.CREDENTIAL_API_KEY)
	apiSecret := c.GetHeader(constants.CREDENTIAL_API_SECRET)
	if apiKey != "" && apiSecret != "" {
		user, err := GetUserByAPIKey(c, apiKey, apiSecret)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusUnauthorized, err)
			return
		}
		c.Set(constants.UserField, user)
		c.Next()
		return
	}

	apiKey = c.Query("apiKey")
	apiSecret = c.Query("apiSecret")
	if apiKey != "" && apiSecret != "" {
		user, err := GetUserByAPIKey(c, apiKey, apiSecret)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusUnauthorized, err)
			return
		}
		c.Set(constants.UserField, user)
		c.Next()
		return
	}

	token := c.GetHeader(config.GlobalConfig.Auth.Header)
	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}

	db := c.MustGet(constants.DbField).(*gorm.DB)
	// split bearer
	token = strings.TrimPrefix(token, "Bearer ")
	user, err := DecodeHashToken(db, token, false)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, err)
		return
	}
	c.Set(constants.UserField, user)
	c.Next()
}

func GetUserByAPIKey(c *gin.Context, apiKey, apiSecret string) (*User, error) {
	db := c.MustGet(constants.DbField).(*gorm.DB)
	var userCredential UserCredential
	err := db.Model(&UserCredential{}).Where("api_key = ? AND api_secret = ?", apiKey, apiSecret).Find(&userCredential).Error
	if err != nil {
		return nil, err
	}
	var user *User
	err = db.Model(&User{}).Where("id = ?", userCredential.UserID).Find(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func CreateUserByEmail(db *gorm.DB, username, display, email, password string) (*User, error) {
	// Properly handle Unicode characters (including Chinese)
	var firstName, lastName string
	if username != "" {
		runes := []rune(username)
		if len(runes) > 0 {
			firstName = string(runes[0]) // First character (rune) as FirstName
		}
		if len(runes) > 1 {
			lastName = string(runes[1:]) // Remaining characters as LastName
		}
	}

	user := User{
		DisplayName:        display,
		FirstName:          firstName,
		LastName:           lastName,
		Email:              email,
		Password:           HashPassword(password),
		Enabled:            true,
		Activated:          false,
		EmailNotifications: true,
		Role:               RoleUser, // Explicitly set default role
	}
	result := db.Create(&user)
	return &user, result.Error
}

func CreateUser(db *gorm.DB, email, password string) (*User, error) {
	user := User{
		Email:     email,
		Password:  HashPassword(password),
		Enabled:   true,
		Activated: false,
		Role:      RoleUser, // Explicitly set default role
	}

	start := time.Now()
	result := db.Create(&user)
	duration := time.Since(start)
	if monitor := metrics.GetGlobalMonitor(); monitor != nil {
		monitor.RecordSQLQuery(context.Background(), "INSERT INTO users (email, password, enabled, activated) VALUES (?, ?, ?, ?)",
			[]interface{}{email, user.Password, true, false}, constants.USER_TABLE_NAME, "INSERT", duration, 1, result.Error)
	}
	return &user, result.Error
}
func UpdateUserFields(db *gorm.DB, user *User, vals map[string]any) error {
	start := time.Now()
	result := db.Model(user).Updates(vals)
	duration := time.Since(start)
	if monitor := metrics.GetGlobalMonitor(); monitor != nil {
		monitor.RecordSQLQuery(context.Background(), "UPDATE users SET ... WHERE id = ?",
			[]interface{}{user.ID}, constants.USER_TABLE_NAME, "UPDATE", duration, 1, result.Error)
	}

	return result.Error
}

func SetLastLogin(db *gorm.DB, user *User, lastIp string) error {
	now := time.Now().Truncate(1 * time.Second)
	vals := map[string]any{
		"LastLoginIP": lastIp,
		"LastLogin":   &now,
	}
	user.LastLogin = &now
	user.LastLoginIP = lastIp

	start := time.Now()
	result := db.Model(user).Updates(vals)
	duration := time.Since(start)
	if monitor := metrics.GetGlobalMonitor(); monitor != nil {
		monitor.RecordSQLQuery(context.Background(), "UPDATE users SET LastLoginIP = ?, LastLogin = ? WHERE id = ?",
			[]interface{}{lastIp, &now, user.ID}, constants.USER_TABLE_NAME, "UPDATE", duration, 1, result.Error)
	}

	return result.Error
}

func EncodeHashToken(user *User, timestamp int64, useLastlogin bool) (hash string) {
	// ts-uid-token
	logintimestamp := "0"
	if useLastlogin && user.LastLogin != nil {
		logintimestamp = fmt.Sprintf("%d", user.LastLogin.Unix())
	}
	t := fmt.Sprintf("%s$%d", user.Email, timestamp)
	hashVal := sha256.Sum256([]byte(logintimestamp + user.Password + t))
	hash = base64.RawStdEncoding.EncodeToString([]byte(t)) + "-" + fmt.Sprintf("%x", hashVal)
	return hash
}

func DecodeHashToken(db *gorm.DB, hash string, useLastLogin bool) (user *User, err error) {
	vals := strings.Split(hash, "-")
	if len(vals) != 2 {
		return nil, errors.New("bad token")
	}
	data, err := base64.RawStdEncoding.DecodeString(vals[0])
	if err != nil {
		return nil, errors.New("bad token")
	}

	vals = strings.Split(string(data), "$")
	if len(vals) != 2 {
		return nil, errors.New("bad token")
	}

	ts, err := strconv.ParseInt(vals[1], 10, 64)
	if err != nil {
		return nil, errors.New("bad token")
	}

	if time.Now().Unix() > ts {
		return nil, errors.New("token expired")
	}

	user, err = GetUserByEmail(db, vals[0])
	if err != nil {
		return nil, errors.New("bad token")
	}
	token := EncodeHashToken(user, ts, useLastLogin)
	if token != hash {
		return nil, errors.New("bad token")
	}
	return user, nil
}

func CheckUserAllowLogin(db *gorm.DB, user *User) error {
	if !user.Enabled {
		return errors.New("user not allow login")
	}

	if utils.GetBoolValue(db, constants.KEY_USER_ACTIVATED) && !user.Activated {
		return errors.New("waiting for activation")
	}

	// Role validation - ensure user has a valid role
	if err := ValidateUserRole(user); err != nil {
		return err
	}

	return nil
}

// ValidateUserRole validates that the user has a valid role
func ValidateUserRole(user *User) error {
	if user.Role == "" {
		return errors.New("user role is not set")
	}

	// Check if role is one of the valid roles
	validRoles := []string{RoleSuperAdmin, RoleAdmin, RoleUser}
	for _, validRole := range validRoles {
		if user.Role == validRole {
			return nil
		}
	}

	return fmt.Errorf("invalid user role: %s", user.Role)
}

func InTimezone(c *gin.Context, timezone string) {
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		return
	}
	c.Set(constants.TzField, tz)

	session := sessions.Default(c)
	session.Set(constants.TzField, timezone)
	session.Save()
}

func BuildAuthToken(user *User, expired time.Duration, useLoginTime bool) string {
	n := time.Now().Add(expired)
	return EncodeHashToken(user, n.Unix(), useLoginTime)
}

func UpdateUser(db *gorm.DB, user *User, vals map[string]any) error {
	return db.Model(user).Updates(vals).Error
}

// ChangePassword 修改密码
func ChangePassword(db *gorm.DB, user *User, oldPassword, newPassword string) error {
	// 验证旧密码
	if !CheckPassword(user, oldPassword) {
		return errors.New("旧密码不正确")
	}

	// 设置新密码
	err := SetPassword(db, user, newPassword)
	if err != nil {
		return err
	}

	// 更新最后密码修改时间
	now := time.Now()
	err = UpdateUserFields(db, user, map[string]any{
		"LastPasswordChange": &now,
	})
	if err != nil {
		return err
	}

	user.LastPasswordChange = &now
	return nil
}

// ResetPassword 重置密码
func ResetPassword(db *gorm.DB, user *User, newPassword string) error {
	// 设置新密码
	err := SetPassword(db, user, newPassword)
	if err != nil {
		return err
	}

	// 清除密码重置令牌
	err = UpdateUserFields(db, user, map[string]any{
		"PasswordResetToken":   "",
		"PasswordResetExpires": nil,
		"LastPasswordChange":   time.Now(),
	})
	if err != nil {
		return err
	}

	user.PasswordResetToken = ""
	user.PasswordResetExpires = nil
	now := time.Now()
	user.LastPasswordChange = &now
	return nil
}

// GeneratePasswordResetToken 生成密码重置令牌
func GeneratePasswordResetToken(db *gorm.DB, user *User) (string, error) {
	token := utils.RandString(32)
	expires := time.Now().Add(24 * time.Hour) // 24小时过期

	err := UpdateUserFields(db, user, map[string]any{
		"PasswordResetToken":   token,
		"PasswordResetExpires": &expires,
	})
	if err != nil {
		return "", err
	}

	user.PasswordResetToken = token
	user.PasswordResetExpires = &expires
	return token, nil
}

// VerifyPasswordResetToken 验证密码重置令牌
func VerifyPasswordResetToken(db *gorm.DB, token string) (*User, error) {
	var user User
	err := db.Where("password_reset_token = ? AND password_reset_expires > ?", token, time.Now()).First(&user).Error
	if err != nil {
		return nil, errors.New("无效或过期的重置令牌")
	}
	return &user, nil
}

// GenerateEmailVerifyToken 生成邮箱验证令牌
func GenerateEmailVerifyToken(db *gorm.DB, user *User) (string, error) {
	token := utils.RandString(32)
	expires := time.Now().Add(24 * time.Hour) // 24小时过期

	err := UpdateUserFields(db, user, map[string]any{
		"EmailVerifyToken":   token,
		"EmailVerifyExpires": &expires,
	})
	if err != nil {
		return "", err
	}

	user.EmailVerifyToken = token
	user.EmailVerifyExpires = &expires
	return token, nil
}

// VerifyEmail 验证邮箱
func VerifyEmail(db *gorm.DB, token string) (*User, error) {
	var user User
	err := db.Where("email_verify_token = ? AND email_verify_expires > ?", token, time.Now()).First(&user).Error
	if err != nil {
		return nil, errors.New("无效或过期的邮箱验证令牌")
	}

	// 更新邮箱验证状态
	err = UpdateUserFields(db, &user, map[string]any{
		"EmailVerified":      true,
		"EmailVerifyToken":   "",
		"EmailVerifyExpires": nil,
	})
	if err != nil {
		return nil, err
	}

	user.EmailVerified = true
	user.EmailVerifyToken = ""
	user.EmailVerifyExpires = nil
	return &user, nil
}

// GeneratePhoneVerifyToken 生成手机验证令牌
func GeneratePhoneVerifyToken(db *gorm.DB, user *User) (string, error) {
	token := utils.RandNumberText(6) // 6位数字验证码
	err := UpdateUserFields(db, user, map[string]any{
		"PhoneVerifyToken": token,
	})
	if err != nil {
		return "", err
	}

	user.PhoneVerifyToken = token
	return token, nil
}

// VerifyPhone 验证手机
func VerifyPhone(db *gorm.DB, user *User, token string) error {
	if user.PhoneVerifyToken != token {
		return errors.New("验证码不正确")
	}

	// 更新手机验证状态
	err := UpdateUserFields(db, user, map[string]any{
		"PhoneVerified":    true,
		"PhoneVerifyToken": "",
	})
	if err != nil {
		return err
	}

	user.PhoneVerified = true
	user.PhoneVerifyToken = ""
	return nil
}

// UpdateNotificationSettings 更新通知设置
func UpdateNotificationSettings(db *gorm.DB, user *User, settings map[string]bool) error {
	vals := make(map[string]any)

	if emailNotif, ok := settings["emailNotifications"]; ok {
		vals["email_notifications"] = emailNotif
	}
	if pushNotif, ok := settings["pushNotifications"]; ok {
		vals["push_notifications"] = pushNotif
	}
	if systemNotif, ok := settings["systemNotifications"]; ok {
		vals["system_notifications"] = systemNotif
	}
	if autoCleanUnreadEmails, ok := settings["autoCleanUnreadEmails"]; ok {
		vals["auto_clean_unread_emails"] = autoCleanUnreadEmails
	}

	if len(vals) == 0 {
		return nil
	}

	err := UpdateUserFields(db, user, vals)
	if err != nil {
		return err
	}

	// 更新用户对象
	if emailNotif, ok := settings["emailNotifications"]; ok {
		user.EmailNotifications = emailNotif
	}
	if pushNotif, ok := settings["pushNotifications"]; ok {
		user.PushNotifications = pushNotif
	}
	if systemNotif, ok := settings["systemNotifications"]; ok {
		user.SystemNotifications = systemNotif
	}
	if autoCleanUnreadEmails, ok := settings["autoCleanUnreadEmails"]; ok {
		user.AutoCleanUnreadEmails = autoCleanUnreadEmails
	}

	return nil
}

// UpdatePreferences 更新用户偏好设置
// 只处理实际使用的字段：timezone 和 locale
func UpdatePreferences(db *gorm.DB, user *User, preferences map[string]string) error {
	vals := make(map[string]any)

	if timezone, ok := preferences["timezone"]; ok {
		vals["timezone"] = timezone
	}
	if locale, ok := preferences["locale"]; ok {
		vals["locale"] = locale
	}

	if len(vals) == 0 {
		return nil
	}

	err := UpdateUserFields(db, user, vals)
	if err != nil {
		return err
	}

	// 更新用户对象
	if timezone, ok := preferences["timezone"]; ok {
		user.Timezone = timezone
	}
	if locale, ok := preferences["locale"]; ok {
		user.Locale = locale
	}

	return nil
}

// CalculateProfileComplete 计算资料完整度
func CalculateProfileComplete(user *User) int {
	complete := 0
	total := 0

	// 基本信息 (40%)
	total += 4
	if user.DisplayName != "" {
		complete++
	}
	if user.FirstName != "" {
		complete++
	}
	if user.LastName != "" {
		complete++
	}
	if user.Avatar != "" {
		complete++
	}

	// 联系方式 (30%)
	total += 3
	if user.Email != "" {
		complete++
	}
	if user.Phone != "" {
		complete++
	}
	if user.EmailVerified {
		complete++
	}

	// 地址信息 (20%)
	total += 2
	if user.City != "" {
		complete++
	}
	if user.Region != "" {
		complete++
	}

	// 偏好设置 (10%)
	total += 1
	if user.Timezone != "" {
		complete++
	}

	percentage := (complete * 100) / total
	if percentage > 100 {
		percentage = 100
	}

	return percentage
}

// UpdateProfileComplete 更新资料完整度
func UpdateProfileComplete(db *gorm.DB, user *User) error {
	complete := CalculateProfileComplete(user)
	err := UpdateUserFields(db, user, map[string]any{
		"ProfileComplete": complete,
	})
	if err != nil {
		return err
	}

	user.ProfileComplete = complete
	return nil
}

// IncrementLoginCount 增加登录次数
func IncrementLoginCount(db *gorm.DB, user *User) error {
	err := UpdateUserFields(db, user, map[string]any{
		"LoginCount": user.LoginCount + 1,
	})
	if err != nil {
		return err
	}

	user.LoginCount++
	return nil
}

// IsAdmin 检查是否为管理员（基于角色）
func (u *User) IsAdmin() bool {
	return u.Role == RoleSuperAdmin || u.Role == RoleAdmin
}

// IsSuperAdmin 检查是否为超级管理员
func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}
