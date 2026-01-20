package utils

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LoginSecurityManager 登录安全管理器
type LoginSecurityManager struct {
	maxFailedAttempts    int           // 最大失败尝试次数（默认7次）
	lockDuration         time.Duration // 锁定时长（默认30分钟）
	maxPasswordLogins    int           // 密码登录最大次数（默认25次）
	ipRateLimitPerMinute int           // IP每分钟登录次数限制
	logger               *zap.Logger
}

// NewLoginSecurityManager 创建登录安全管理器
func NewLoginSecurityManager(logger *zap.Logger) *LoginSecurityManager {
	return &LoginSecurityManager{
		maxFailedAttempts:    7,
		lockDuration:         30 * time.Minute,
		maxPasswordLogins:    25,
		ipRateLimitPerMinute: 7, // 每个IP每分钟最多7次登录尝试
		logger:               logger,
	}
}

// GlobalLoginSecurityManager 全局登录安全管理器
var GlobalLoginSecurityManager *LoginSecurityManager

// InitGlobalLoginSecurityManager 初始化全局登录安全管理器
func InitGlobalLoginSecurityManager(logger *zap.Logger) {
	GlobalLoginSecurityManager = NewLoginSecurityManager(logger)
}

// AccountLockInfo 账号锁定信息
type AccountLockInfo struct {
	IsLocked bool
	UnlockAt time.Time
}

// CheckAccountLockFunc 检查账号锁定的函数类型
type CheckAccountLockFunc func(*gorm.DB, string, uint) (*AccountLockInfo, error)

// CheckAccountLock 检查账号是否被锁定
func (lsm *LoginSecurityManager) CheckAccountLock(db *gorm.DB, email string, userID uint, checkFunc CheckAccountLockFunc) error {
	if checkFunc == nil {
		return nil
	}

	lockInfo, err := checkFunc(db, email, userID)
	if err != nil {
		return err
	}

	if lockInfo != nil && lockInfo.IsLocked {
		remainingTime := time.Until(lockInfo.UnlockAt)
		lsm.logger.Warn("Account is locked",
			zap.String("email", email),
			zap.Uint("userID", userID),
			zap.Time("unlockAt", lockInfo.UnlockAt),
			zap.Duration("remainingTime", remainingTime))

		return fmt.Errorf("account is locked due to too many failed login attempts. Please try again after %d minutes", int(remainingTime.Minutes())+1)
	}

	return nil
}

// RecordFailedLoginFunc 记录失败登录的函数类型
type RecordFailedLoginFunc func(*gorm.DB, string, uint, string, int) error

// RecordFailedLogin 记录失败登录
func (lsm *LoginSecurityManager) RecordFailedLogin(db *gorm.DB, email string, userID uint, ipAddress string, recordFunc RecordFailedLoginFunc) error {
	// 获取当前失败次数
	key := fmt.Sprintf("login:failed:%s", email)
	var failedCount int
	if GlobalCache != nil {
		if val, ok := GlobalCache.Get(key); ok {
			if c, ok := val.(int); ok {
				failedCount = c
			}
		}
	}

	failedCount++
	if GlobalCache != nil {
		GlobalCache.Add(key, failedCount)
	}

	// 如果达到最大失败次数，锁定账号
	if failedCount >= lsm.maxFailedAttempts {
		if recordFunc != nil {
			err := recordFunc(db, email, userID, ipAddress, failedCount)
			if err != nil {
				lsm.logger.Error("Failed to create account lock", zap.Error(err))
			} else {
				lsm.logger.Warn("Account locked due to too many failed attempts",
					zap.String("email", email),
					zap.Uint("userID", userID),
					zap.Int("failedAttempts", failedCount))
			}
		}
	}

	return nil
}

// ClearFailedLoginCount 清除失败登录计数（登录成功时调用）
func (lsm *LoginSecurityManager) ClearFailedLoginCount(email string) {
	key := fmt.Sprintf("login:failed:%s", email)
	if GlobalCache != nil {
		GlobalCache.Remove(key)
	}
}

// CheckIPRateLimit 检查IP登录限流
func (lsm *LoginSecurityManager) CheckIPRateLimit(ip string) error {
	if GlobalCache == nil {
		return nil
	}

	key := fmt.Sprintf("login:ip:%s", ip)
	var count int
	if val, ok := GlobalCache.Get(key); ok {
		if c, ok := val.(int); ok {
			count = c
		}
	}

	if count >= lsm.ipRateLimitPerMinute {
		lsm.logger.Warn("IP login rate limit exceeded",
			zap.String("ip", ip),
			zap.Int("count", count))
		return errors.New("too many login attempts from this IP, please try again later")
	}

	// 增加计数
	count++
	GlobalCache.Add(key, count)

	return nil
}

// CheckProxyIP 检查是否为代理IP（简单检测）
func (lsm *LoginSecurityManager) CheckProxyIP(ip string) (bool, error) {
	// 检查是否为已知的代理IP段
	// 这里可以集成第三方API或使用IP数据库
	// 简单实现：检查是否为私有IP（可能是内网代理）
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false, errors.New("invalid IP address")
	}

	// 检查是否为私有IP（可能是内网代理）
	if parsedIP.IsPrivate() || parsedIP.IsLoopback() {
		// 允许私有IP，但记录日志
		lsm.logger.Info("Login from private IP", zap.String("ip", ip))
		return false, nil
	}

	// 可以在这里添加更多代理检测逻辑
	// 例如：检查IP是否在已知代理列表中

	return false, nil
}

// LoginLocation 登录位置信息
type LoginLocation struct {
	Country string
	City    string
}

// GetRecentLoginLocationsFunc 获取最近登录位置的函数类型
type GetRecentLoginLocationsFunc func(*gorm.DB, uint, int) ([]LoginLocation, error)

// DetectSuspiciousLogin 检测可疑登录（异地登录）
func (lsm *LoginSecurityManager) DetectSuspiciousLogin(db *gorm.DB, userID uint, currentIP, currentLocation, currentCountry string, getLocationsFunc GetRecentLoginLocationsFunc) (bool, error) {
	if userID == 0 {
		return false, nil // 新用户无法检测
	}

	if getLocationsFunc == nil {
		return false, nil
	}

	// 获取最近的登录位置
	recentLogins, err := getLocationsFunc(db, userID, 5)
	if err != nil {
		return false, err
	}

	if len(recentLogins) == 0 {
		return false, nil // 首次登录，不视为可疑
	}

	// 检查是否与最近登录位置不同
	for _, login := range recentLogins {
		// 如果国家不同，视为可疑
		if login.Country != "" && currentCountry != "" && login.Country != currentCountry {
			lsm.logger.Warn("Suspicious login detected: different country",
				zap.Uint("userID", userID),
				zap.String("previousCountry", login.Country),
				zap.String("currentCountry", currentCountry),
				zap.String("currentIP", currentIP))
			return true, nil
		}
	}

	return false, nil
}

// CheckPasswordLoginLimitFunc 检查密码登录次数限制的函数类型
type CheckPasswordLoginLimitFunc func(*gorm.DB, uint) (int64, error)

// CheckPasswordLoginLimit 检查密码登录次数限制
func (lsm *LoginSecurityManager) CheckPasswordLoginLimit(db *gorm.DB, userID uint, email string, checkFunc CheckPasswordLoginLimitFunc) (bool, error) {
	if userID == 0 {
		return false, nil
	}

	if checkFunc == nil {
		return false, nil
	}

	// 获取用户密码登录次数（从登录历史中统计）
	count, err := checkFunc(db, userID)
	if err != nil {
		return false, err
	}

	// 如果超过限制，需要邮箱验证
	if int(count) >= lsm.maxPasswordLogins {
		lsm.logger.Info("Password login limit reached, email verification required",
			zap.Uint("userID", userID),
			zap.String("email", email),
			zap.Int64("passwordLoginCount", count))
		return true, nil // 需要邮箱验证
	}

	return false, nil
}

// GetDeviceID 从User-Agent生成设备ID
func GetDeviceID(userAgent, ipAddress string) string {
	// 使用User-Agent和IP的哈希值作为设备ID
	// 这是一个简化的实现，实际可以使用更复杂的设备指纹算法
	data := fmt.Sprintf("%s:%s", userAgent, ipAddress)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16]) // 使用前16字节，生成32字符的十六进制字符串
}

// ParseUserAgent 解析User-Agent获取设备信息
func ParseUserAgent(userAgent string) (deviceType, os, browser string) {
	ua := strings.ToLower(userAgent)

	// 检测设备类型
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		deviceType = "mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		deviceType = "tablet"
	} else {
		deviceType = "desktop"
	}

	// 检测操作系统
	if strings.Contains(ua, "windows") {
		os = "Windows"
	} else if strings.Contains(ua, "mac") || strings.Contains(ua, "darwin") {
		os = "macOS"
	} else if strings.Contains(ua, "linux") {
		os = "Linux"
	} else if strings.Contains(ua, "android") {
		os = "Android"
	} else if strings.Contains(ua, "ios") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		os = "iOS"
	} else {
		os = "Unknown"
	}

	// 检测浏览器
	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg") {
		browser = "Chrome"
	} else if strings.Contains(ua, "firefox") {
		browser = "Firefox"
	} else if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		browser = "Safari"
	} else if strings.Contains(ua, "edg") {
		browser = "Edge"
	} else if strings.Contains(ua, "opera") {
		browser = "Opera"
	} else {
		browser = "Unknown"
	}

	return deviceType, os, browser
}
