package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/gin-gonic/gin"
)

func TestParseIPList_Contains(t *testing.T) {
	list := ParseIPList("127.0.0.1,10.0.0.0/8,*")
	if !list.Contains("127.0.0.1") {
		t.Fatal("expected loopback")
	}
	if !list.Contains("10.1.2.3") {
		t.Fatal("expected cidr match")
	}
	if !list.Contains("8.8.8.8") {
		t.Fatal("expected any")
	}
}

func TestIPACLDecision_BlockWins(t *testing.T) {
	blocked := ParseIPList("203.0.113.1")
	allowed := ParseIPList("*")
	ok, reason := IPACLDecision(blocked, allowed, IPList{}, IPList{}, "203.0.113.1")
	if ok || reason != "blocked" {
		t.Fatalf("got ok=%v reason=%q", ok, reason)
	}
}

func TestBaseRateLimit_Global(t *testing.T) {
	gin.SetMode(gin.TestMode)
	InitTrafficControl(config.MiddlewareConfig{
		EnableRateLimit: true,
		RateLimit: config.RateLimiterConfig{
			GlobalRPS: 1, GlobalBurst: 1, GlobalWindow: time.Minute,
		},
	}, "/api")

	r := gin.New()
	r.Use(BaseRateLimit())
	r.GET("/api/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first=%d", w1.Code)
	}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second=%d want 429", w2.Code)
	}
}

func TestDetectModule(t *testing.T) {
	if got := detectModule("/api/knowledge/docs", "/api"); got != "knowledge" {
		t.Fatalf("got %q", got)
	}
}
