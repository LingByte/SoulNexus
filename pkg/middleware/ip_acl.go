package middleware

import (
	"net/http"
	"sync"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/gin-gonic/gin"
)

var (
	globalIPBlocked IPList
	globalIPAllowed IPList
	globalIPACLOnce sync.Once
	globalIPACLOn   bool
)

// InitIPACL loads global HTTP IP ACL from config. No-op when disabled.
func InitIPACL(cfg config.IPACLConfig) {
	globalIPACLOnce.Do(func() {
		globalIPACLOn = cfg.Enable
		globalIPBlocked = ParseIPList(cfg.BlockedIPs)
		globalIPAllowed = ParseIPList(cfg.AllowedIPs)
	})
}

// GlobalIPACL enforces HTTP_IP_ACL_* when HTTP_IP_ACL_ENABLE=true.
// Reserved: disabled by default; enable via env when needed.
func GlobalIPACL() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !globalIPACLOn {
			c.Next()
			return
		}
		if shouldSkipTrafficControl(c.Request.URL.Path) {
			c.Next()
			return
		}
		ip := clientIP(c)
		if globalIPBlocked.Contains(ip) {
			abortIPDenied(c, "blocked")
			return
		}
		if !globalIPAllowed.Empty() && !globalIPAllowed.Contains(ip) {
			abortIPDenied(c, "global_allowlist")
			return
		}
		c.Next()
	}
}

// RouteIPACL applies a per-route IP allow/block overlay (checked at route group).
func RouteIPACL(rule IPACLRule) gin.HandlerFunc {
	blocked := ParseIPList(rule.Blocked)
	allowed := ParseIPList(rule.Allowed)
	if blocked.Empty() && allowed.Empty() {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if shouldSkipTrafficControl(c.Request.URL.Path) {
			c.Next()
			return
		}
		ip := clientIP(c)
		if blocked.Contains(ip) {
			abortIPDenied(c, "route_blocked")
			return
		}
		if !allowed.Empty() && !allowed.Contains(ip) {
			abortIPDenied(c, "route_allowlist")
			return
		}
		c.Next()
	}
}

func abortIPDenied(c *gin.Context, reason string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"code": 403, "msg": "ip access denied", "data": gin.H{"reason": reason},
	})
}

// IPACLStats returns current global IP ACL config for ops dashboards.
func IPACLStats() gin.H {
	return gin.H{
		"enabled":      globalIPACLOn,
		"blockedCount": len(globalIPBlocked.IPs) + len(globalIPBlocked.Nets),
		"allowedCount": len(globalIPAllowed.IPs) + len(globalIPAllowed.Nets),
		"allowAny":     globalIPAllowed.Any,
	}
}
