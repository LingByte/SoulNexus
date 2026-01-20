package utils

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RegistrationGuard 注册防护服务
type RegistrationGuard struct {
	// IP限流配置
	maxRegistrationsPerIP  int           // 每个IP在时间窗口内允许的最大注册次数
	ipRateLimitWindow      time.Duration // IP限流时间窗口
	maxFailedAttemptsPerIP int           // 每个IP允许的最大失败尝试次数
	failedAttemptWindow    time.Duration // 失败尝试时间窗口

	// 邮箱验证配置
	emailDomainBlacklist   []string // 邮箱域名黑名单
	disposableEmailDomains []string // 临时邮箱域名列表

	// 密码强度配置
	minPasswordLength  int  // 最小密码长度
	requireUppercase   bool // 是否需要大写字母
	requireLowercase   bool // 是否需要小写字母
	requireNumber      bool // 是否需要数字
	requireSpecialChar bool // 是否需要特殊字符

	// IP黑名单配置
	ipBlacklist []*net.IPNet // IP黑名单（CIDR格式）

	// 日志记录
	logger *zap.Logger
}

// RegistrationAttempt 注册尝试记录
type RegistrationAttempt struct {
	IP        string
	Email     string
	Timestamp time.Time
	Success   bool
	Reason    string
}

// NewRegistrationGuard 创建注册防护服务实例
func NewRegistrationGuard(logger *zap.Logger) *RegistrationGuard {
	rg := &RegistrationGuard{
		maxRegistrationsPerIP:  3, // 每个IP每小时最多3次注册
		ipRateLimitWindow:      1 * time.Hour,
		maxFailedAttemptsPerIP: 5, // 每个IP每小时最多5次失败尝试
		failedAttemptWindow:    1 * time.Hour,
		emailDomainBlacklist:   []string{}, // 可以配置黑名单域名
		disposableEmailDomains: getDefaultDisposableEmailDomains(),
		minPasswordLength:      8,
		requireUppercase:       false,
		requireLowercase:       true,
		requireNumber:          false,
		requireSpecialChar:     false,
		ipBlacklist:            []*net.IPNet{},
		logger:                 logger,
	}

	// 初始化IP黑名单（可以加载已知的恶意IP段）
	rg.initIPBlacklist()

	return rg
}

// initIPBlacklist 初始化IP黑名单
func (rg *RegistrationGuard) initIPBlacklist() {
	// 可以从配置文件或数据库加载黑名单IP段
	// 这里提供一些示例恶意IP段（实际使用时应该从配置加载）
	blacklistCIDRs := []string{
		// 示例：可以添加已知的恶意IP段
		// "1.2.3.0/24",
		// "5.6.7.0/24",
	}

	for _, cidr := range blacklistCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			rg.ipBlacklist = append(rg.ipBlacklist, ipnet)
		}
	}
}

// AddIPToBlacklist 动态添加IP到黑名单
func (rg *RegistrationGuard) AddIPToBlacklist(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %v", err)
	}
	rg.ipBlacklist = append(rg.ipBlacklist, ipnet)
	rg.logger.Info("IP added to blacklist", zap.String("cidr", cidr))
	return nil
}

// GlobalRegistrationGuard 全局注册防护服务实例
var GlobalRegistrationGuard *RegistrationGuard

// InitGlobalRegistrationGuard 初始化全局注册防护服务
func InitGlobalRegistrationGuard(logger *zap.Logger) {
	GlobalRegistrationGuard = NewRegistrationGuard(logger)
}

// CheckIPRateLimit 检查IP注册限流
func (rg *RegistrationGuard) CheckIPRateLimit(ip string) error {
	if GlobalCache == nil {
		// 如果缓存未初始化，跳过限流检查
		return nil
	}

	// 获取IP的注册次数
	key := fmt.Sprintf("reg:ip:%s", ip)
	var count int
	if val, ok := GlobalCache.Get(key); ok {
		if c, ok := val.(int); ok {
			count = c
		}
	}

	// 检查是否超过限制
	if count >= rg.maxRegistrationsPerIP {
		rg.logger.Warn("IP registration rate limit exceeded",
			zap.String("ip", ip),
			zap.Int("count", count),
			zap.Duration("window", rg.ipRateLimitWindow))
		return errors.New("registration rate limit exceeded for this IP, please try again later")
	}

	return nil
}

// RecordRegistrationAttempt 记录注册尝试
func (rg *RegistrationGuard) RecordRegistrationAttempt(ip string, email string, success bool, reason string) {
	if GlobalCache == nil {
		// 如果缓存未初始化，只记录日志
		if success {
			rg.logger.Info("Registration attempt recorded (cache not initialized)",
				zap.String("ip", ip),
				zap.String("email", maskEmail(email)),
				zap.Bool("success", success))
		} else {
			rg.logger.Warn("Failed registration attempt (cache not initialized)",
				zap.String("ip", ip),
				zap.String("email", maskEmail(email)),
				zap.String("reason", reason))
		}
		return
	}

	// 记录成功注册
	if success {
		key := fmt.Sprintf("reg:ip:%s", ip)
		var count int
		if val, ok := GlobalCache.Get(key); ok {
			if c, ok := val.(int); ok {
				count = c
			}
		}
		count++
		GlobalCache.Add(key, count)
		rg.logger.Info("Registration attempt recorded",
			zap.String("ip", ip),
			zap.String("email", maskEmail(email)),
			zap.Bool("success", success))
	} else {
		// 记录失败尝试
		failedKey := fmt.Sprintf("reg:failed:ip:%s", ip)
		var failedCount int
		if val, ok := GlobalCache.Get(failedKey); ok {
			if c, ok := val.(int); ok {
				failedCount = c
			}
		}
		failedCount++
		GlobalCache.Add(failedKey, failedCount)

		rg.logger.Warn("Failed registration attempt",
			zap.String("ip", ip),
			zap.String("email", maskEmail(email)),
			zap.String("reason", reason))
	}
}

// CheckFailedAttempts 检查失败尝试次数
func (rg *RegistrationGuard) CheckFailedAttempts(ip string) error {
	if GlobalCache == nil {
		// 如果缓存未初始化，跳过检查
		return nil
	}

	failedKey := fmt.Sprintf("reg:failed:ip:%s", ip)
	var failedCount int
	if val, ok := GlobalCache.Get(failedKey); ok {
		if c, ok := val.(int); ok {
			failedCount = c
		}
	}

	if failedCount >= rg.maxFailedAttemptsPerIP {
		rg.logger.Warn("Too many failed registration attempts",
			zap.String("ip", ip),
			zap.Int("failed_count", failedCount))
		return errors.New("too many failed registration attempts, please try again later")
	}

	return nil
}

// ValidateEmail 验证邮箱格式和域名
func (rg *RegistrationGuard) ValidateEmail(email string) error {
	// 基本格式验证
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	// 提取域名
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email format")
	}
	domain := strings.ToLower(parts[1])

	// 检查黑名单域名
	for _, blacklisted := range rg.emailDomainBlacklist {
		if domain == strings.ToLower(blacklisted) {
			return errors.New("email domain is not allowed")
		}
	}

	// 检查临时邮箱域名
	for _, disposable := range rg.disposableEmailDomains {
		if domain == disposable {
			return errors.New("disposable email addresses are not allowed")
		}
	}

	return nil
}

// ValidatePassword 验证密码强度
func (rg *RegistrationGuard) ValidatePassword(password string) error {
	if len(password) < rg.minPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", rg.minPasswordLength)
	}

	if rg.requireUppercase {
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
		if !hasUpper {
			return errors.New("password must contain at least one uppercase letter")
		}
	}

	if rg.requireLowercase {
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
		if !hasLower {
			return errors.New("password must contain at least one lowercase letter")
		}
	}

	if rg.requireNumber {
		hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
		if !hasNumber {
			return errors.New("password must contain at least one number")
		}
	}

	if rg.requireSpecialChar {
		hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
		if !hasSpecial {
			return errors.New("password must contain at least one special character")
		}
	}

	return nil
}

// ValidateIP 验证IP地址是否在黑名单中
func (rg *RegistrationGuard) ValidateIP(ip string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return errors.New("invalid IP address")
	}

	// 检查IP是否在黑名单中
	for _, ipnet := range rg.ipBlacklist {
		if ipnet.Contains(parsedIP) {
			rg.logger.Warn("Registration attempt from blacklisted IP",
				zap.String("ip", ip),
				zap.String("cidr", ipnet.String()))
			return errors.New("IP address is blocked")
		}
	}

	// 检查是否是私有IP（可选：生产环境可以禁止私有IP注册）
	// 这里允许私有IP，但记录日志
	if parsedIP.IsLoopback() || parsedIP.IsPrivate() || parsedIP.IsLinkLocalUnicast() {
		rg.logger.Info("Registration attempt from private/local IP",
			zap.String("ip", ip))
		// 生产环境可以取消下面的注释来禁止私有IP
		// return errors.New("private IP addresses are not allowed for registration")
	}

	return nil
}

// CheckRegistrationAllowed 综合检查是否允许注册
func (rg *RegistrationGuard) CheckRegistrationAllowed(ip string, email string, password string) error {
	// 1. 验证IP
	if err := rg.ValidateIP(ip); err != nil {
		return err
	}

	// 2. 检查失败尝试次数
	if err := rg.CheckFailedAttempts(ip); err != nil {
		return err
	}

	// 3. 检查IP限流
	if err := rg.CheckIPRateLimit(ip); err != nil {
		return err
	}

	// 4. 验证邮箱
	if err := rg.ValidateEmail(email); err != nil {
		rg.RecordRegistrationAttempt(ip, email, false, err.Error())
		return err
	}

	// 5. 验证密码
	if err := rg.ValidatePassword(password); err != nil {
		rg.RecordRegistrationAttempt(ip, email, false, err.Error())
		return err
	}

	return nil
}

// maskEmail 掩码邮箱地址用于日志记录
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}

	username := parts[0]
	domain := parts[1]

	// 掩码用户名（保留前2个字符）
	if len(username) <= 2 {
		username = "***"
	} else {
		username = username[:2] + "***"
	}

	return username + "@" + domain
}

// getDefaultDisposableEmailDomains 获取默认的临时邮箱域名列表
func getDefaultDisposableEmailDomains() []string {
	return []string{
		"10minutemail.com",
		"guerrillamail.com",
		"mailinator.com",
		"tempmail.com",
		"throwaway.email",
		"yopmail.com",
		"temp-mail.org",
		"getnada.com",
		"maildrop.cc",
		"mohmal.com",
		"fakeinbox.com",
		"trashmail.com",
		"meltmail.com",
		"mintemail.com",
		"sharklasers.com",
		"spamgourmet.com",
		"throwawaymail.com",
		"tmpmail.org",
		"getairmail.com",
		"mytemp.email",
	}
}
