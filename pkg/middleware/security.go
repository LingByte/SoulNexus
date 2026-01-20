package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	// CSRF配置
	CSRFSecret    string            `json:"csrf_secret"`
	CSRFTokenName string            `json:"csrf_token_name"`
	CSRFMaxAge    time.Duration     `json:"csrf_max_age"`
	CSRFSecure    bool              `json:"csrf_secure"`
	CSRFHttpOnly  bool              `json:"csrf_http_only"`
	CSRFSameSite  csrf.SameSiteMode `json:"csrf_same_site"`

	// XSS配置
	XSSProtection      bool   `json:"xss_protection"`
	ContentTypeNosniff bool   `json:"content_type_nosniff"`
	XFrameOptions      string `json:"x_frame_options"`

	// 输入验证配置
	MaxRequestSize int64    `json:"max_request_size"`
	AllowedOrigins []string `json:"allowed_origins"`

	// 安全头配置
	HSTSMaxAge     int    `json:"hsts_max_age"`
	ReferrerPolicy string `json:"referrer_policy"`
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		CSRFSecret:         generateRandomKey(32),
		CSRFTokenName:      "csrf_token",
		CSRFMaxAge:         24 * time.Hour,
		CSRFSecure:         true,
		CSRFHttpOnly:       true,
		CSRFSameSite:       csrf.SameSiteDefaultMode,
		XSSProtection:      true,
		ContentTypeNosniff: true,
		XFrameOptions:      "DENY",
		MaxRequestSize:     10 * 1024 * 1024, // 10MB
		HSTSMaxAge:         31536000,         // 1 year
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	}
}

// SecurityMiddleware 安全中间件
func SecurityMiddleware(config *SecurityConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return func(c *gin.Context) {
		// 设置安全头
		setSecurityHeaders(c, config)

		// 检查请求大小
		if c.Request.ContentLength > config.MaxRequestSize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request entity too large",
			})
			return
		}

		// 验证Origin
		if !isOriginAllowed(c, config.AllowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Origin not allowed",
			})
			return
		}

		c.Next()
	}
}

// CSRFMiddleware CSRF保护中间件
func CSRFMiddleware(config *SecurityConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	// 创建CSRF保护器
	protect := csrf.Protect(
		[]byte(config.CSRFSecret),
		csrf.Secure(config.CSRFSecure),
		csrf.HttpOnly(config.CSRFHttpOnly),
		csrf.SameSite(config.CSRFSameSite),
		csrf.MaxAge(int(config.CSRFMaxAge.Seconds())),
		csrf.FieldName(config.CSRFTokenName),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "CSRF token mismatch", http.StatusForbidden)
		})),
	)

	return gin.WrapH(protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 将CSRF token添加到响应头
		token := csrf.Token(r)
		w.Header().Set("X-CSRF-Token", token)
	})))
}

// XSSProtectionMiddleware XSS防护中间件
func XSSProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 清理请求参数中的XSS攻击
		cleanXSSInput(c)
		c.Next()
	}
}

// InputValidationMiddleware 输入验证中间件
func InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证和清理输入
		if err := validateAndSanitizeInput(c); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Invalid input: " + err.Error(),
			})
			return
		}
		c.Next()
	}
}

// setSecurityHeaders 设置安全头
func setSecurityHeaders(c *gin.Context, config *SecurityConfig) {
	// XSS保护
	if config.XSSProtection {
		c.Header("X-XSS-Protection", "1; mode=block")
	}

	// 内容类型嗅探保护
	if config.ContentTypeNosniff {
		c.Header("X-Content-Type-Options", "nosniff")
	}

	// 点击劫持保护
	if config.XFrameOptions != "" {
		c.Header("X-Frame-Options", config.XFrameOptions)
	}

	// HSTS
	if config.HSTSMaxAge > 0 {
		c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", config.HSTSMaxAge))
	}

	// Referrer Policy
	if config.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", config.ReferrerPolicy)
	}

	// 移除服务器信息
	c.Header("Server", "")
	c.Header("X-Powered-By", "")
}

// isOriginAllowed 检查Origin是否被允许
func isOriginAllowed(c *gin.Context, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return true // 如果没有配置，则允许所有
	}

	origin := c.GetHeader("Origin")
	if origin == "" {
		return true // 没有Origin头，可能是同源请求
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed || (strings.HasSuffix(allowed, "*") && strings.HasPrefix(origin, strings.TrimSuffix(allowed, "*"))) {
			return true
		}
	}

	return false
}

// cleanXSSInput 清理XSS输入
func cleanXSSInput(c *gin.Context) {
	// 清理查询参数
	for key, values := range c.Request.URL.Query() {
		for i, value := range values {
			values[i] = html.EscapeString(value)
		}
		c.Request.URL.RawQuery = strings.ReplaceAll(c.Request.URL.RawQuery, key+"="+values[0], key+"="+html.EscapeString(values[0]))
	}

	// 清理表单数据
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		if err := c.Request.ParseForm(); err == nil {
			for key, values := range c.Request.PostForm {
				for i, value := range values {
					c.Request.PostForm[key][i] = html.EscapeString(value)
				}
			}
		}
	}
}

// validateAndSanitizeInput 验证和清理输入
func validateAndSanitizeInput(c *gin.Context) error {
	// 检查SQL注入模式
	sqlInjectionPatterns := []string{
		`(?i)(union\s+select)`,
		`(?i)(drop\s+table)`,
		`(?i)(delete\s+from)`,
		`(?i)(insert\s+into)`,
		`(?i)(update\s+set)`,
		`(?i)(or\s+1\s*=\s*1)`,
		`(?i)(and\s+1\s*=\s*1)`,
		`(?i)(exec\s*\()`,
		`(?i)(script\s*>)`,
		`(?i)(<script)`,
		`(?i)(javascript:)`,
		`(?i)(vbscript:)`,
		`(?i)(onload\s*=)`,
		`(?i)(onerror\s*=)`,
	}

	// 检查所有输入
	checkInput := func(input string) error {
		for _, pattern := range sqlInjectionPatterns {
			if matched, _ := regexp.MatchString(pattern, input); matched {
				return fmt.Errorf("potentially malicious input detected")
			}
		}
		return nil
	}

	// 检查查询参数
	for _, values := range c.Request.URL.Query() {
		for _, value := range values {
			if err := checkInput(value); err != nil {
				return err
			}
		}
	}

	// 检查表单数据
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		if err := c.Request.ParseForm(); err == nil {
			for _, values := range c.Request.PostForm {
				for _, value := range values {
					if err := checkInput(value); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// generateRandomKey 生成随机密钥
func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// SecureCompare 安全比较字符串，防止时序攻击
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SanitizeString 清理字符串，移除危险字符
func SanitizeString(input string) string {
	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	input = re.ReplaceAllString(input, "")

	// HTML转义
	input = html.EscapeString(input)

	// 移除控制字符
	re = regexp.MustCompile(`[\x00-\x1f\x7f]`)
	input = re.ReplaceAllString(input, "")

	return strings.TrimSpace(input)
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}
