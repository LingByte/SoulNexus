package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() (*gin.Engine, *MonitorAPI) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       true,
		EnableSQLAnalysis:   true,
		EnableSystemMonitor: true,
		MonitorInterval:     100 * time.Millisecond, // Shorter interval for tests
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)

	return router, api
}

func TestNewMonitorAPI(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)

	assert.NotNil(t, api)
	assert.Equal(t, monitor, api.monitor)
}

func TestMonitorAPI_GetOverview(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/overview", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSystemStats(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSystemStats_NoMonitor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       false,
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: false, // Disable system monitor
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSystemStats_WithSince(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system?since=1000", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSystemStats_WithLimit(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system?limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetLatestSystemStats(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system/latest", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSlowQueries(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/slow", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSlowQueries_WithPagination(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/slow?page=1&limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetQueryPatterns(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/patterns", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSQLStats(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetSQLStats_NoAnalyzer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       false,
		EnableSQLAnalysis:   false, // Disable SQL analysis
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetQueriesByTable(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/table/users", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetQueriesByOperation(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/operation/SELECT", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetTraces(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/traces", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetTraces_WithFilters(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	// Create some spans first
	ctx := context.Background()
	_, span1 := api.monitor.StartSpan(ctx, "test_span")
	api.monitor.EndSpan(span1, nil) // Mark as OK

	_, span2 := api.monitor.StartSpan(ctx, "other_span")
	api.monitor.EndSpan(span2, context.Canceled) // Mark as ERROR

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/traces?status=OK&name=test&page=1&limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetTraces_NoTracer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       false, // Disable tracing
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/traces", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetTraceDetail(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	// First create a trace
	ctx := context.Background()
	_, span := api.monitor.StartSpan(ctx, "test")
	traceID := span.TraceID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/traces/"+traceID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetTraceDetail_NoTracer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       false, // Disable tracing
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/traces/test_trace_id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetMetrics(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetMetrics_NoMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &MonitorConfig{
		EnableMetrics:       false, // Disable metrics
		EnableTracing:       false,
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)
	api := NewMonitorAPI(monitor)
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetPrometheusMetrics(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/metrics/prometheus", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestMonitorAPI_RegisterRoutes(t *testing.T) {
	router, api := setupTestRouter()
	apiGroup := router.Group("/api")
	api.RegisterRoutes(apiGroup)

	// Test that routes are registered by making requests
	routes := []string{
		"/api/overview",
		"/api/system",
		"/api/system/latest",
		"/api/sql/slow",
		"/api/sql/patterns",
		"/api/sql/stats",
		"/api/sql/table/users",
		"/api/sql/operation/SELECT",
		"/api/traces",
		"/api/metrics",
		"/api/metrics/prometheus",
		"/api/ui",
		"/api/ui.json",
		"/api/metric",
	}

	for _, route := range routes {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", route, nil)
		router.ServeHTTP(w, req)
		// Routes should exist (may return 200 or other status, but not 404)
		if w.Code == http.StatusNotFound {
			t.Errorf("Route %s not found", route)
		}
	}
}

func TestMonitorAPI_GetSlowQueries_EdgeCases(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	// Test with invalid page/limit
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/slow?page=0&limit=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with large page number
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/sql/slow?page=999&limit=10", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetQueryPatterns_EdgeCases(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	// Test with invalid page/limit
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sql/patterns?page=0&limit=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitorAPI_GetTraces_EdgeCases(t *testing.T) {
	router, api := setupTestRouter()
	api.RegisterRoutes(router.Group("/api"))

	// Test with invalid page/limit
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/traces?page=0&limit=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with ERROR status filter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/traces?status=ERROR", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with name prefix filter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/traces?name=test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
