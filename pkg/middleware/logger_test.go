package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerMiddleware_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Capture zap logs
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(LoggerMiddleware(logger))

	// Mock business handler: write 201 status code
	// Use POST instead of GET since LoggerMiddleware filters GET requests
	r.POST("/hello", func(c *gin.Context) {
		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)
		c.String(http.StatusCreated, "created")
	})

	// Construct request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/hello?a=1&b=2", nil)
	req.Header.Set("User-Agent", "UnitTestUA/1.0")
	// For controllable ClientIP, set proxy header (gin.ClientIP prioritizes X-Forwarded-For)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	// Send request
	r.ServeHTTP(w, req)

	// Assert response (ensure logging happens after c.Next())
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "created", w.Body.String())

	// Get one log entry
	entries := recorded.All()
	if !assert.Equal(t, 1, len(entries), "should log exactly one entry") {
		t.FailNow()
	}
	entry := entries[0]
	assert.Equal(t, "Request", entry.Message)

	// Convert fields to map for easier assertion
	fields := map[string]zapcore.Field{}
	for _, f := range entry.Context {
		fields[f.Key] = f
	}

	// Basic field assertions
	if f, ok := fields["status"]; assert.True(t, ok) {
		assert.Equal(t, int64(http.StatusCreated), f.Integer)
	}
	if f, ok := fields["method"]; assert.True(t, ok) {
		assert.Equal(t, "POST", f.String)
	}
	if f, ok := fields["path"]; assert.True(t, ok) {
		assert.Equal(t, "/hello", f.String)
	}
	if f, ok := fields["query"]; assert.True(t, ok) {
		// RawQuery doesn't guarantee order, only check containment
		assert.Contains(t, f.String, "a=1")
		assert.Contains(t, f.String, "b=2")
	}
	if f, ok := fields["ip"]; assert.True(t, ok) {
		assert.Equal(t, "203.0.113.1", f.String)
	}
	if f, ok := fields["user-agent"]; assert.True(t, ok) {
		assert.Equal(t, "UnitTestUA/1.0", f.String)
	}
	// latency is DurationType, unit ns, >0 is sufficient
	if f, ok := fields["latency"]; assert.True(t, ok) {
		assert.Greater(t, f.Integer, int64(0))
		assert.Equal(t, zapcore.DurationType, f.Type)
	}
}

func TestLoggerMiddleware_NoQuery_NoUA_DefaultIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(LoggerMiddleware(logger))
	r.POST("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/ping", nil) // No query/UA/IP headers
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	entries := recorded.All()
	if !assert.Equal(t, 1, len(entries)) {
		t.FailNow()
	}
	fields := map[string]zapcore.Field{}
	for _, f := range entries[0].Context {
		fields[f.Key] = f
	}

	// Basic robustness checks
	if f, ok := fields["path"]; assert.True(t, ok) {
		assert.Equal(t, "/ping", f.String)
	}
	if f, ok := fields["query"]; assert.True(t, ok) {
		assert.Equal(t, "", f.String) // No query
	}
	if f, ok := fields["user-agent"]; assert.True(t, ok) {
		// httptest may provide default UA or empty; just check field exists
		_ = f.String
	}
	// IP might be empty or 127.0.0.1 / ::1, check existence and no crash
	_, ipExists := fields["ip"]
	assert.True(t, ipExists)
	// latency still needs >0
	if f, ok := fields["latency"]; assert.True(t, ok) {
		assert.Greater(t, f.Integer, int64(0))
	}
}
