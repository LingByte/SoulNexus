package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, h *OIDCHandler) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	r.POST("/auth/login", h.Login)
	r.POST("/auth/logout", h.Logout)

	r.GET("/.well-known/openid-configuration", h.Discovery)
	r.GET("/oauth/authorize", h.Authorize)
	r.POST("/oauth/token", h.Token)
	r.GET("/oauth/jwks", h.JWKS)
	r.GET("/oauth/userinfo", h.UserInfo)
	r.POST("/oauth/revoke", h.Revoke)
	r.POST("/oauth/introspect", h.Introspect)
}
