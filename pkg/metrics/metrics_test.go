package metrics

import (
	"testing"
	"time"
)

func getTestMetrics() *Metrics {
	// NewMetrics now uses sync.Once internally, so we can call it multiple times safely
	return NewMetrics()
}

func TestNewMetrics(t *testing.T) {
	m := getTestMetrics()
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	m := getTestMetrics()
	m.RecordHTTPRequest("GET", "/api/test", "200", "handler", 100*time.Millisecond, 1024, 2048)
	// No error means success
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	m := getTestMetrics()
	m.RecordDBQuery("SELECT", "users", "query", 50*time.Millisecond)
	// No error means success
}

func TestMetrics_RecordDBConnection(t *testing.T) {
	m := getTestMetrics()
	m.RecordDBConnection("testdb", "open")
	m.RecordDBConnection("testdb", "close")
	// No error means success
}

func TestMetrics_SetDBConnectionsActive(t *testing.T) {
	m := getTestMetrics()
	m.SetDBConnectionsActive("testdb", "active", 10)
	m.SetDBConnectionsActive("testdb", "idle", 5)
	// No error means success
}

func TestMetrics_RecordCacheHit(t *testing.T) {
	m := getTestMetrics()
	m.RecordCacheHit("redis", "get")
	// No error means success
}

func TestMetrics_RecordCacheMiss(t *testing.T) {
	m := getTestMetrics()
	m.RecordCacheMiss("redis", "get")
	// No error means success
}

func TestMetrics_SetCacheSize(t *testing.T) {
	m := getTestMetrics()
	m.SetCacheSize("redis", 1000)
	m.SetCacheSize("memcache", 500)
	// No error means success
}

func TestMetrics_RecordBusinessOperation(t *testing.T) {
	m := getTestMetrics()
	m.RecordBusinessOperation("create_user", "success", "premium")
	m.RecordBusinessOperation("create_user", "failed", "free")
	// No error means success
}

func TestMetrics_SetBusinessMetric(t *testing.T) {
	m := getTestMetrics()
	m.SetBusinessMetric("active_users", "daily", 1000.0)
	m.SetBusinessMetric("revenue", "monthly", 50000.0)
	// No error means success
}

func TestMetrics_RecordBusinessDuration(t *testing.T) {
	m := getTestMetrics()
	m.RecordBusinessDuration("process_order", "ecommerce", 200*time.Millisecond)
	// No error means success
}

func TestMetrics_SetSystemMemoryUsage(t *testing.T) {
	m := getTestMetrics()
	m.SetSystemMemoryUsage("heap", 1024*1024*100)
	m.SetSystemMemoryUsage("stack", 1024*1024*50)
	// No error means success
}

func TestMetrics_SetSystemCPUUsage(t *testing.T) {
	m := getTestMetrics()
	m.SetSystemCPUUsage("core0", 25.5)
	m.SetSystemCPUUsage("core1", 30.2)
	// No error means success
}

func TestMetrics_SetSystemGoroutines(t *testing.T) {
	m := getTestMetrics()
	m.SetSystemGoroutines(100)
	m.SetSystemGoroutines(200)
	// No error means success
}

func TestMetrics_GetCacheHitRate(t *testing.T) {
	m := getTestMetrics()
	rate := m.GetCacheHitRate("redis", "get")
	if rate != 0.0 {
		t.Errorf("Expected 0.0, got %f", rate)
	}
}

func TestMetrics_Reset(t *testing.T) {
	m := getTestMetrics()

	// Record some metrics
	m.RecordHTTPRequest("GET", "/test", "200", "handler", time.Second, 100, 200)
	m.RecordDBQuery("SELECT", "users", "query", time.Millisecond*100)
	m.SetCacheSize("redis", 100)

	// Reset should not panic
	m.Reset()
}
