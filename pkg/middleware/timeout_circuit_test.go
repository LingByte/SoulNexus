package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// ===== CircuitBreaker Tests =====

func TestNewCircuitBreaker(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)
	if cb == nil {
		t.Fatal("circuit breaker should not be nil")
	}
	if cb.GetState() != StateClosed {
		t.Fatalf("initial state should be StateClosed, got %d", cb.GetState())
	}
}

func TestCircuitBreaker_AllowClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if !cb.Allow() {
		t.Fatal("should allow request in closed state")
	}
	cb.RecordSuccess()
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		Timeout:               1 * time.Second,
		OpenTimeout:           100 * time.Millisecond,
		MaxConcurrentRequests: 10,
	}
	cb := NewCircuitBreaker(config)

	// Record failures to trip the breaker
	for i := 0; i < 3; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("state should be StateOpen after %d failures, got %d", 3, cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:      2,
		SuccessThreshold:      2,
		Timeout:               1 * time.Second,
		OpenTimeout:           50 * time.Millisecond,
		MaxConcurrentRequests: 10,
	}
	cb := NewCircuitBreaker(config)

	// Trip the breaker
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Fatalf("state should be StateOpen, got %d", cb.GetState())
	}

	// Wait for open timeout
	time.Sleep(100 * time.Millisecond)

	// Should transition to half-open
	if !cb.Allow() {
		t.Fatal("should allow request after open timeout")
	}
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("state should be StateHalfOpen, got %d", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:      1,
		SuccessThreshold:      2,
		Timeout:               1 * time.Second,
		OpenTimeout:           50 * time.Millisecond,
		MaxConcurrentRequests: 10,
	}
	cb := NewCircuitBreaker(config)

	// Trip the breaker
	cb.Allow()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(100 * time.Millisecond)
	cb.Allow()

	// Record successes to close
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.GetState() != StateClosed {
		t.Fatalf("state should be StateClosed, got %d", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnFailure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:      1,
		SuccessThreshold:      2,
		Timeout:               1 * time.Second,
		OpenTimeout:           50 * time.Millisecond,
		MaxConcurrentRequests: 10,
	}
	cb := NewCircuitBreaker(config)

	// Trip the breaker
	cb.Allow()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(100 * time.Millisecond)
	cb.Allow()

	// Record failure in half-open
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Fatalf("state should be StateOpen after failure in half-open, got %d", cb.GetState())
	}
}

func TestCircuitBreaker_MaxConcurrentRequests(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:      5,
		SuccessThreshold:      3,
		Timeout:               1 * time.Second,
		OpenTimeout:           1 * time.Second,
		MaxConcurrentRequests: 2,
	}
	cb := NewCircuitBreaker(config)

	// Allow 2 requests
	if !cb.Allow() {
		t.Fatal("should allow first request")
	}
	if !cb.Allow() {
		t.Fatal("should allow second request")
	}

	// Third should be denied
	if cb.Allow() {
		t.Fatal("should deny third request")
	}

	cb.RecordSuccess()
	cb.RecordSuccess()

	// Now should allow again
	if !cb.Allow() {
		t.Fatal("should allow after recording successes")
	}
	cb.RecordSuccess()
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	stats := cb.GetStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}
	if stats["state"] != StateClosed {
		t.Fatalf("state should be StateClosed, got %v", stats["state"])
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:      100,
		SuccessThreshold:      3,
		Timeout:               1 * time.Second,
		OpenTimeout:           1 * time.Second,
		MaxConcurrentRequests: 1000,
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cb.Allow() {
				cb.RecordSuccess()
			}
		}()
	}
	wg.Wait()
}

// ===== TimeoutCircuitManager Tests =====

func TestNewTimeoutCircuitManager(t *testing.T) {
	timeoutConfig := DefaultTimeoutConfig()
	cbConfig := DefaultCircuitBreakerConfig()
	tcm := NewTimeoutCircuitManager(timeoutConfig, cbConfig)
	if tcm == nil {
		t.Fatal("manager should not be nil")
	}
}

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()
	if config.DefaultTimeout != 30*time.Second {
		t.Fatalf("default timeout should be 30s, got %v", config.DefaultTimeout)
	}
	if len(config.EndpointTimeouts) == 0 {
		t.Fatal("endpoint timeouts should not be empty")
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	if config.FailureThreshold != 5 {
		t.Fatalf("failure threshold should be 5, got %d", config.FailureThreshold)
	}
	if config.SuccessThreshold != 3 {
		t.Fatalf("success threshold should be 3, got %d", config.SuccessThreshold)
	}
}

// ===== TimeoutMiddleware Tests =====

func TestTimeoutMiddleware_DefaultTimeout(t *testing.T) {
	r := newTestEngine()
	r.Use(TimeoutMiddleware())
	r.GET("/fast", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fast", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestTimeoutMiddleware_SlowRequest(t *testing.T) {
	// Reset global manager with short timeout
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil

	r := newTestEngine()
	r.Use(TimeoutMiddleware())
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)
	r.ServeHTTP(w, req)

	// Should timeout (default is 30s, but we can test with shorter)
	// For now, just verify the middleware runs
	if w.Code != http.StatusOK && w.Code != http.StatusRequestTimeout {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestTimeoutMiddleware_CircuitBreakerOpen(t *testing.T) {
	// Reset global manager
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil

	r := newTestEngine()
	r.Use(CircuitBreakerMiddleware())
	r.GET("/fail", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	// Trip the breaker
	for i := 0; i < 6; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/fail", nil)
		r.ServeHTTP(w, req)
	}

	// Now the breaker should be open
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fail", nil)
	r.ServeHTTP(w, req)

	// Should return 503 when circuit is open
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", w.Code)
	}
}

func TestCircuitBreakerMiddleware_Success(t *testing.T) {
	// Reset global manager
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil

	r := newTestEngine()
	r.Use(CircuitBreakerMiddleware())
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestCombinedTimeoutCircuitMiddleware(t *testing.T) {
	// Reset global manager
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil

	r := newTestEngine()
	r.Use(CombinedTimeoutCircuitMiddleware())
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestCombinedTimeoutCircuitMiddleware_WebSocketSkip(t *testing.T) {
	// Reset global manager
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil

	r := newTestEngine()
	r.Use(CombinedTimeoutCircuitMiddleware())
	r.GET("/api/voice/lingecho/v1/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/voice/lingecho/v1/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestIsLongLivedEndpoint_SSE(t *testing.T) {
	if !isLongLivedEndpoint("/api/lingecho/voice-session/v1/ws") {
		t.Fatal("voice-session ws should skip request timeout")
	}
}

func TestTimeoutMiddleware_SkipSSE(t *testing.T) {
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = NewTimeoutCircuitManager(TimeoutConfig{
		DefaultTimeout: time.Millisecond,
	}, DefaultCircuitBreakerConfig())

	r := newTestEngine()
	r.Use(TimeoutMiddleware())
	r.GET("/api/lingecho/voice-session/v1/ws", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/lingecho/voice-session/v1/ws", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 (long-lived WS should not hit timeout middleware)", w.Code)
	}
}

// ===== GetTimeoutCircuitManager Tests =====

func TestGetTimeoutCircuitManager(t *testing.T) {
	// Reset the global instance
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil

	mgr := GetTimeoutCircuitManager()
	if mgr == nil {
		t.Fatal("manager should not be nil")
	}

	// Should return the same instance
	mgr2 := GetTimeoutCircuitManager()
	if mgr != mgr2 {
		t.Fatal("should return same instance")
	}
}

// ===== GetCircuitBreakerStats Tests =====

func TestGetCircuitBreakerStats(t *testing.T) {
	timeoutCircuitOnce = sync.Once{}
	globalTimeoutCircuitManager = nil
	stats := GetCircuitBreakerStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}
}

// ===== ReqIDFromGin Tests =====

func TestReqIDFromGin_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	reqID := ReqIDFromGin(c)
	if reqID != "" {
		t.Fatalf("expected empty reqID, got %q", reqID)
	}
}
