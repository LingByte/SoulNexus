package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func setupRateLimiterTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestNewRateLimiter(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "10-S",
		Identifier: "ip",
	}

	rl := NewRateLimiter(cfg, nil)
	assert.NotNil(t, rl)
	assert.NotNil(t, rl.store)
	assert.Equal(t, cfg.Rate, rl.cfg.Rate)
}

func TestNewRateLimiter_WithStore(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "10-S",
		Identifier: "ip",
	}
	store := memory.NewStore()

	rl := NewRateLimiter(cfg, store)
	assert.NotNil(t, rl)
	assert.Equal(t, store, rl.store)
}

func TestRateLimiter_WithStoreFactory(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "10-S",
		Identifier: "ip",
	}
	rl := NewRateLimiter(cfg, nil)

	factory := &PrebuiltStoreFactory{Store: memory.NewStore()}
	rl = rl.WithStoreFactory(factory)

	assert.NotNil(t, rl.storeFactory)
}

func TestRateLimiter_WithObserver(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "10-S",
		Identifier: "ip",
	}
	rl := NewRateLimiter(cfg, nil)

	observer := NewPrometheusObserver()
	rl = rl.WithObserver(observer)

	assert.NotNil(t, rl.observer)
}

func TestRateLimiter_Middleware_Allow(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "100-S",
		Identifier: "ip",
	}
	rl := NewRateLimiter(cfg, memory.NewStore())

	router := setupRateLimiterTestRouter()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Middleware_SkipPath(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "1-S",
		Identifier: "ip",
		SkipPaths:  []string{"/health"},
	}
	rl := NewRateLimiter(cfg, memory.NewStore())

	router := setupRateLimiterTestRouter()
	router.Use(rl.Middleware())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Middleware_WhitelistIP(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:           "1-S",
		Identifier:     "ip",
		WhitelistCIDRs: []string{"192.168.1.0/24"},
	}
	rl := NewRateLimiter(cfg, memory.NewStore())

	router := setupRateLimiterTestRouter()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:8080"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Middleware_BlacklistIP(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:           "100-S",
		Identifier:     "ip",
		BlacklistCIDRs: []string{"192.168.1.0/24"},
	}
	rl := NewRateLimiter(cfg, memory.NewStore())

	router := setupRateLimiterTestRouter()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:8080"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimiter_UpdateConfig(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "10-S",
		Identifier: "ip",
	}
	rl := NewRateLimiter(cfg, memory.NewStore())

	newCfg := RateLimiterConfig{
		Rate:       "20-S",
		Identifier: "user",
	}
	rl.UpdateConfig(newCfg)

	assert.Equal(t, newCfg.Rate, rl.cfg.Rate)
	assert.Equal(t, newCfg.Identifier, rl.cfg.Identifier)
}

func TestNewPrometheusObserver(t *testing.T) {
	obs := NewPrometheusObserver()
	assert.NotNil(t, obs)
	assert.NotNil(t, obs.allow)
	assert.NotNil(t, obs.deny)
}

func TestPrometheusObserver_OnAllow(t *testing.T) {
	obs := NewPrometheusObserver()
	obs.OnAllow("/test", "ip:192.168.1.1")
	// 如果没有panic，测试通过
}

func TestPrometheusObserver_OnDeny(t *testing.T) {
	obs := NewPrometheusObserver()
	obs.OnDeny("/test", "ip:192.168.1.1")
	// 如果没有panic，测试通过
}

func TestSetRateLimiterConfig(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:       "50-S",
		Identifier: "ip",
	}
	SetRateLimiterConfig(cfg)

	retrieved := GetRateLimiterConfig()
	assert.Equal(t, cfg.Rate, retrieved.Rate)
	assert.Equal(t, cfg.Identifier, retrieved.Identifier)
}

func TestSetRateLimiterStore(t *testing.T) {
	store := memory.NewStore()
	SetRateLimiterStore(store)
	// 如果没有panic，测试通过
}

func TestRateLimiterMiddleware(t *testing.T) {
	SetRateLimiterConfig(RateLimiterConfig{
		Rate:       "100-S",
		Identifier: "ip",
	})

	middleware := RateLimiterMiddleware()
	assert.NotNil(t, middleware)

	router := setupRateLimiterTestRouter()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
