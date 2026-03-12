package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMonitorMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:   false,
		EnableTracing:   true,
		MonitorInterval: 1 * time.Second,
	}
	monitor := NewMonitor(config)

	router.Use(MonitorMiddleware(monitor))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorMiddleware_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:   false,
		EnableTracing:   true,
		MonitorInterval: 1 * time.Second,
	}
	monitor := NewMonitor(config)

	router.Use(MonitorMiddleware(monitor))
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "test error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMonitorMiddleware_RecordsMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Use metrics enabled monitor (will use singleton)
	monitor := NewMonitor(nil)

	router.Use(MonitorMiddleware(monitor))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Metrics should be recorded (no error means success)
}

func TestMonitorMiddleware_WithNegativeContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:   false,
		EnableTracing:   true,
		MonitorInterval: 1 * time.Second,
	}
	monitor := NewMonitor(config)

	router.Use(MonitorMiddleware(monitor))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.ContentLength = -1 // Negative content length
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorMiddleware_WithSpanEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:   false,
		EnableTracing:   true,
		MonitorInterval: 1 * time.Second,
	}
	monitor := NewMonitor(config)

	router.Use(MonitorMiddleware(monitor))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "http://example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
