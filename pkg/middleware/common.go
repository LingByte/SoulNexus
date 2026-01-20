package middleware

import (
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
)

// CorsMiddleware 跨域处理中间件
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 设置 CORS 头
		if origin != "" {
			// 允许具体的 Origin
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin") // 避免缓存污染
		} else {
			// 如果没有Origin头，允许所有来源（开发环境）
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true") // 允许携带 Cookie
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin, X-API-KEY, X-API-SECRET, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		// 继续处理请求，确保 CORS 头在所有响应中都存在
		c.Next()

		// 确保响应中也包含 CORS 头（处理重定向等情况）
		if c.Writer.Header().Get("Access-Control-Allow-Origin") == "" {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}
	}
}

func WithMemSession(secret string) gin.HandlerFunc {
	store := memstore.NewStore([]byte(secret))
	store.Options(sessions.Options{Path: "/", MaxAge: 0})
	return sessions.Sessions(GetCarrotSessionField(), store)
}

func WithCookieSession(secret string, maxAge int) gin.HandlerFunc {
	store := cookie.NewStore([]byte(secret))
	store.Options(sessions.Options{Path: "/", MaxAge: maxAge})
	return sessions.Sessions(GetCarrotSessionField(), store)
}

func GetCarrotSessionField() string {
	v := utils.GetEnv(constants.ENV_SESSION_FIELD)
	if v == "" {
		return "lingecho"
	}
	return v
}

// SecurityMiddlewareChain 安全中间件链
func SecurityMiddlewareChain() []gin.HandlerFunc {
	config := DefaultSecurityConfig()

	return []gin.HandlerFunc{
		// 1. 基础安全头
		SecurityMiddleware(config),

		// 2. XSS防护
		XSSProtectionMiddleware(),

		// 3. 输入验证
		InputValidationMiddleware(),

		// 4. CSRF保护（仅对状态改变的操作）
		CSRFMiddleware(config),
	}
}

// ApplySecurityMiddleware 应用安全中间件到路由组
func ApplySecurityMiddleware(r *gin.RouterGroup) {
	middlewares := SecurityMiddlewareChain()
	for _, middleware := range middlewares {
		r.Use(middleware)
	}
}
