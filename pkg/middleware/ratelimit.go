package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type limiterStore struct {
	window time.Duration
	store  sync.Map
}

func newLimiterStore(window time.Duration) *limiterStore {
	if window <= 0 {
		window = time.Minute
	}
	ls := &limiterStore{window: window}
	go ls.evictLoop()
	return ls
}

func (ls *limiterStore) evictLoop() {
	ticker := time.NewTicker(ls.window)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-2 * ls.window)
		ls.store.Range(func(k, v any) bool {
			if e, ok := v.(*limiterEntry); ok && e.lastSeen.Before(cutoff) {
				ls.store.Delete(k)
			}
			return true
		})
	}
}

func (ls *limiterStore) allow(key string, rps rate.Limit, burst int) bool {
	if rps <= 0 || burst <= 0 {
		return true
	}
	now := time.Now()
	if v, ok := ls.store.Load(key); ok {
		e := v.(*limiterEntry)
		e.lastSeen = now
		return e.limiter.Allow()
	}
	entry := &limiterEntry{
		limiter:  rate.NewLimiter(rps, burst),
		lastSeen: now,
	}
	actual, _ := ls.store.LoadOrStore(key, entry)
	e := actual.(*limiterEntry)
	e.lastSeen = now
	return e.limiter.Allow()
}

// TrafficControl holds multi-dimension rate limiters.
type TrafficControl struct {
	enabled    bool
	apiPrefix  string
	globalRPS  rate.Limit
	globalBurst int
	globalStore *limiterStore

	ipRPS   rate.Limit
	ipBurst int
	ipStore *limiterStore

	userRPS   rate.Limit
	userBurst int
	userStore *limiterStore

	moduleDefaultRPS   rate.Limit
	moduleDefaultBurst int
	moduleRules        map[string]config.RateLimitRule
	moduleStore        *limiterStore

	apiRules []apiRateRule
	apiStore *limiterStore
}

type apiRateRule struct {
	prefix string
	exact  bool
	rule   config.RateLimitRule
}

var globalTraffic *TrafficControl

// InitTrafficControl builds limiters from middleware config. Safe to call once at startup.
func InitTrafficControl(cfg config.MiddlewareConfig, apiPrefix string) {
	tc := &TrafficControl{
		enabled:    cfg.EnableRateLimit,
		apiPrefix:  strings.TrimSuffix(apiPrefix, "/"),
		globalStore: newLimiterStore(cfg.RateLimit.GlobalWindow),
		ipStore:     newLimiterStore(cfg.RateLimit.IPWindow),
		userStore:   newLimiterStore(cfg.RateLimit.UserWindow),
		moduleStore: newLimiterStore(time.Minute),
		apiStore:    newLimiterStore(time.Minute),
		moduleRules: parseModuleRules(cfg.RateLimit.ModuleRulesJSON),
	}
	rl := cfg.RateLimit
	if rl.GlobalRPS > 0 {
		tc.globalRPS = rate.Limit(float64(rl.GlobalRPS) / rl.GlobalWindow.Seconds())
		tc.globalBurst = rl.GlobalBurst
	}
	if rl.IPRPS > 0 {
		tc.ipRPS = rate.Limit(float64(rl.IPRPS) / rl.IPWindow.Seconds())
		tc.ipBurst = rl.IPBurst
	}
	if rl.UserRPS > 0 {
		tc.userRPS = rate.Limit(float64(rl.UserRPS) / rl.UserWindow.Seconds())
		tc.userBurst = rl.UserBurst
	}
	if rl.ModuleDefaultRPS > 0 {
		tc.moduleDefaultRPS = rate.Limit(float64(rl.ModuleDefaultRPS) / time.Minute.Seconds())
		tc.moduleDefaultBurst = rl.ModuleDefaultBurst
	}
	tc.apiRules = parseAPIRules(cfg.RateLimit.APIRulesJSON)
	globalTraffic = tc
	if cfg.EnableRateLimit {
		if addr := strings.TrimSpace(os.Getenv("REDIS_ADDR")); addr != "" {
			if err := initRedisRateBackend(addr, os.Getenv("REDIS_PASSWORD")); err != nil {
				logger.Warn("REDIS_ADDR set but redis rate-limit backend unavailable; using in-process limiters (single-node)",
					zap.Error(err))
			} else {
				logger.Info("redis rate-limit backend enabled for shared IP limits (other dimensions remain process-local)",
					zap.String("redis_addr", addr))
			}
		} else {
			logger.Info("rate limits are process-local (single-node); set REDIS_ADDR for shared IP limiting across replicas")
		}
	}
}

func parseModuleRules(raw string) map[string]config.RateLimitRule {
	out := map[string]config.RateLimitRule{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseAPIRules(raw string) []apiRateRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var m map[string]config.RateLimitRule
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	var rules []apiRateRule
	for path, rule := range m {
		path = strings.TrimSpace(path)
		if path == "" || rule.RPS <= 0 {
			continue
		}
		if rule.Window <= 0 {
			rule.Window = time.Minute
		}
		if rule.Burst <= 0 {
			rule.Burst = rule.RPS
		}
		rules = append(rules, apiRateRule{
			prefix: path,
			exact:  !strings.HasSuffix(path, "/"),
			rule:   rule,
		})
	}
	return rules
}

func (tc *TrafficControl) checkDimension(name, key string, rps rate.Limit, burst int, store *limiterStore, window time.Duration) (bool, string) {
	if rps <= 0 || burst <= 0 || key == "" {
		return true, ""
	}
	if store == nil {
		store = newLimiterStore(window)
	}
	if !store.allow(key, rps, burst) {
		return false, name
	}
	return true, ""
}

func (tc *TrafficControl) allow(c *gin.Context, includeUser bool) (bool, string) {
	if tc == nil || !tc.enabled {
		return true, ""
	}
	path := c.Request.URL.Path
	if shouldSkipTrafficControl(path) {
		return true, ""
	}

	if ok, dim := tc.checkDimension("global", "global", tc.globalRPS, tc.globalBurst, tc.globalStore, time.Minute); !ok {
		return false, dim
	}

	ip := clientIP(c)
	if redisRateEnabled() {
		limit := tc.ipBurst
		if limit <= 0 {
			limit = int(tc.ipRPS)
		}
		if limit > 0 && !redisAllow("ip:"+ip, limit, time.Minute) {
			return false, "ip"
		}
	} else if ok, dim := tc.checkDimension("ip", ip, tc.ipRPS, tc.ipBurst, tc.ipStore, time.Minute); !ok {
		return false, dim
	}

	if mod := detectModule(path, tc.apiPrefix); mod != "" {
		rule, custom := tc.moduleRules[mod]
		rps, burst := tc.moduleDefaultRPS, tc.moduleDefaultBurst
		window := time.Minute
		if custom && rule.RPS > 0 {
			if rule.Window <= 0 {
				rule.Window = time.Minute
			}
			rps = rate.Limit(float64(rule.RPS) / rule.Window.Seconds())
			burst = rule.Burst
			window = rule.Window
		}
		if ok, dim := tc.checkDimension("module", mod, rps, burst, tc.moduleStore, window); !ok {
			return false, dim
		}
	}

	for _, ar := range tc.apiRules {
		matched := ar.exact && path == ar.prefix
		if !ar.exact && strings.HasPrefix(path, ar.prefix) {
			matched = true
		}
		if !matched {
			continue
		}
		rule := ar.rule
		rps := rate.Limit(float64(rule.RPS) / rule.Window.Seconds())
		if ok, dim := tc.checkDimension("api", ar.prefix, rps, rule.Burst, tc.apiStore, rule.Window); !ok {
			return false, dim
		}
		break
	}

	if includeUser {
		if key := rateLimitUserKey(c); key != "" {
			if ok, dim := tc.checkDimension("user", key, tc.userRPS, tc.userBurst, tc.userStore, time.Minute); !ok {
				return false, dim
			}
		}
	}
	return true, ""
}

// BaseRateLimit applies global / IP / module / API dimensions (before auth).
func BaseRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalTraffic == nil {
			c.Next()
			return
		}
		ok, dim := globalTraffic.allow(c, false)
		if !ok {
			abortRateLimited(c, dim)
			return
		}
		c.Next()
	}
}

// UserRateLimit applies the user dimension after JWT/AKSK attach.
func UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalTraffic == nil || !globalTraffic.enabled {
			c.Next()
			return
		}
		key := rateLimitUserKey(c)
		if key == "" {
			c.Next()
			return
		}
		ok, dim := globalTraffic.checkDimension("user", key, globalTraffic.userRPS, globalTraffic.userBurst, globalTraffic.userStore, time.Minute)
		if !ok {
			abortRateLimited(c, dim)
			return
		}
		c.Next()
	}
}

// RouteRateLimit applies a per-route rate limit overlay at the route group.
func RouteRateLimit(rule config.RateLimitRule) gin.HandlerFunc {
	if rule.RPS <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	if rule.Window <= 0 {
		rule.Window = time.Minute
	}
	if rule.Burst <= 0 {
		rule.Burst = rule.RPS
	}
	rps := rate.Limit(float64(rule.RPS) / rule.Window.Seconds())
	return func(c *gin.Context) {
		if globalTraffic == nil || !globalTraffic.enabled {
			c.Next()
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		routeKey := "route:" + path
		ok, dim := globalTraffic.checkDimension("route", routeKey, rps, rule.Burst, globalTraffic.apiStore, rule.Window)
		if !ok {
			abortRateLimited(c, dim)
			return
		}
		c.Next()
	}
}

// RouteTraffic bundles optional per-route rate + IP ACL overlays.
type RouteTraffic struct {
	Rate  *config.RateLimitRule
	IPACL *IPACLRule
}

func RouteTrafficControl(opts RouteTraffic) gin.HandlerFunc {
	var handlers []gin.HandlerFunc
	if opts.IPACL != nil {
		handlers = append(handlers, RouteIPACL(*opts.IPACL))
	}
	if opts.Rate != nil && opts.Rate.RPS > 0 {
		handlers = append(handlers, RouteRateLimit(*opts.Rate))
	}
	if len(handlers) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		for _, h := range handlers {
			h(c)
			if c.IsAborted() {
				return
			}
		}
	}
}

func abortRateLimited(c *gin.Context, dimension string) {
	if logger.Lg != nil {
		logger.Lg.Warn("rate limit triggered",
			zap.String("dimension", dimension),
			zap.String("ip", clientIP(c)),
			zap.String("path", c.Request.URL.Path),
		)
	}
	c.Header("Retry-After", "60")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code": 429,
		"msg":  i18n.TGin(c, i18n.KeyRateLimited),
		"data": gin.H{"dimension": dimension},
	})
}

func rateLimitUserKey(c *gin.Context) string {
	if uid := AuthUserID(c); uid > 0 {
		tid := AuthTenantID(c)
		return strings.Join([]string{"u", itoa64(uint64(tid)), itoa64(uint64(uid))}, ":")
	}
	if aid := AuthPlatformAdminID(c); aid > 0 {
		return "pa:" + itoa64(uint64(aid))
	}
	if cid := AuthCredentialID(c); cid > 0 {
		return "cred:" + itoa64(uint64(cid))
	}
	return ""
}

func detectModule(path, apiPrefix string) string {
	path = strings.TrimPrefix(path, apiPrefix)
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return ""
	}
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		ip = c.Request.RemoteAddr
	}
	return ip
}

func shouldSkipTrafficControl(path string) bool {
	switch path {
	case "/metrics", "/health", "/healthz", "/ready", "/readyz", "/livez":
		return true
	}
	if strings.HasPrefix(path, "/.well-known/") {
		return true
	}
	if strings.HasPrefix(path, "/openapi") || strings.HasPrefix(path, "/schemas/") {
		return true
	}
	if strings.Contains(path, "/docs") {
		return true
	}
	return false
}

func itoa64(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// RateLimitStats exposes enabled state and rule counts for ops.
func RateLimitStats() gin.H {
	if globalTraffic == nil {
		return gin.H{"enabled": false}
	}
	return gin.H{
		"enabled":         globalTraffic.enabled,
		"moduleRules":     len(globalTraffic.moduleRules),
		"apiRules":        len(globalTraffic.apiRules),
		"globalRPS":       globalTraffic.globalRPS,
		"userRPS":         globalTraffic.userRPS,
		"ipRPS":           globalTraffic.ipRPS,
		"moduleDefaultRPS": globalTraffic.moduleDefaultRPS,
	}
}
