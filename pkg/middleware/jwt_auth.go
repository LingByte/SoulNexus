package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	"github.com/gin-gonic/gin"
)

const (
	CtxAuthUserID         = "auth.userId"
	CtxAuthTenantID       = "auth.tenantId"
	CtxAuthTenantSlug     = "auth.tenantSlug"
	CtxAuthEmail          = "auth.email"
	CtxAuthRole           = "auth.role"
	CtxAuthSessionID      = "auth.sessionId"
	CtxAuthDeviceRecordID = "auth.deviceRecordId"
)

// TryAttachTenantJWT parses Authorization: Bearer JWT (or ?token= for WebSocket) and sets tenant auth context.
func TryAttachTenantJWT(c *gin.Context) bool {
	token := bearerTokenFromRequest(c)
	if token == "" {
		return false
	}
	// API keys also use Bearer / ?token=; do not treat them as JWTs.
	if models.IsAPIKeyToken(token) {
		return false
	}
	km := access.JWTKeyManager()
	if km == nil {
		return false
	}
	p, err := access.ParseAccessTokenWithKey(token, km)
	if err != nil || p.TenantID == 0 {
		return false
	}
	c.Set(CtxAuthUserID, p.UserID)
	c.Set(CtxAuthTenantID, p.TenantID)
	c.Set(CtxAuthTenantSlug, p.TenantSlug)
	c.Set(CtxAuthEmail, p.Email)
	c.Set(CtxAuthRole, p.Role)
	c.Set(CtxAuthSessionID, p.SessionID)
	c.Set(CtxAuthDeviceRecordID, p.DeviceRecordID)
	return true
}

func bearerTokenFromRequest(c *gin.Context) string {
	if c == nil {
		return ""
	}
	authz := strings.TrimSpace(c.GetHeader("Authorization"))
	if authz != "" && strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		if t := strings.TrimSpace(authz[len("Bearer "):]); t != "" {
			return t
		}
	}
	if t := strings.TrimSpace(c.Query("token")); t != "" {
		return t
	}
	if t := strings.TrimSpace(c.Query("access_token")); t != "" {
		return t
	}
	return ""
}

// RequireTenantAuth verifies Bearer JWT only (no AK/SK).
func RequireTenantAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := strings.TrimSpace(c.GetHeader("Authorization"))
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			response.Render(c, utils.ErrTokenRequired)
			return
		}
		if !TryAttachTenantJWT(c) {
			system.IncJWTVerifyFail()
			response.Render(c, utils.ErrInvalidToken)
			return
		}
		c.Next()
	}
}

func AuthTenantID(c *gin.Context) uint {
	if v, ok := c.Get(CtxAuthTenantID); ok {
		if n, ok := v.(uint); ok {
			return n
		}
	}
	return 0
}

func AuthUserID(c *gin.Context) uint {
	if v, ok := c.Get(CtxAuthUserID); ok {
		if n, ok := v.(uint); ok {
			return n
		}
	}
	return 0
}

// CurrentTenantID is an alias for AuthTenantID for use in handlers
// where middleware has already guaranteed a non-zero tenant context.
func CurrentTenantID(c *gin.Context) uint { return AuthTenantID(c) }

// RequireTenantInboxUser allows authenticated tenant users to access personal inbox APIs.
func RequireTenantInboxUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthUserID(c) > 0 && AuthTenantID(c) > 0 {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": i18n.TGin(c, i18n.KeyForbidden), "data": nil})
	}
}

// RequireHumanJWTUser rejects requests that lack a human user ID (i.e. AKSK-only callers).
// Use on routes where only logged-in humans should operate (e.g. credential management).
func RequireHumanJWTUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthUserID(c) == 0 || AuthTenantID(c) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": i18n.TGin(c, i18n.KeyForbidden), "data": nil})
			return
		}
		c.Next()
	}
}

func AuthEmail(c *gin.Context) string {
	if v, ok := c.Get(CtxAuthEmail); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func AuthSessionID(c *gin.Context) string {
	if v, ok := c.Get(CtxAuthSessionID); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func AuthDeviceRecordID(c *gin.Context) uint {
	if v, ok := c.Get(CtxAuthDeviceRecordID); ok {
		if n, ok := v.(uint); ok {
			return n
		}
	}
	return 0
}

// AuditOperator returns email, user id string, or "system" for create_by / update_by fields.
func AuditOperator(c *gin.Context) string {
	if s := strings.TrimSpace(AuthEmail(c)); s != "" {
		return s
	}
	if uid := AuthUserID(c); uid > 0 {
		return strconv.FormatUint(uint64(uid), 10)
	}
	return "system"
}
