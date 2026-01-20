package metrics

import (
	"context"
	"testing"
	"time"
)

func TestDefaultMonitorConfig(t *testing.T) {
	config := DefaultMonitorConfig()
	if config == nil {
		t.Fatal("DefaultMonitorConfig returned nil")
	}
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
	if !config.EnableTracing {
		t.Error("Expected EnableTracing to be true")
	}
	if config.MaxSpans <= 0 {
		t.Error("Expected MaxSpans to be positive")
	}
}

func TestNewMonitor(t *testing.T) {
	// Test with nil config (will use default which enables metrics)
	// Note: This will register Prometheus metrics, so we can only do it once per test run
	// We'll skip this part if metrics already exist to avoid Prometheus registration conflicts
	globalMonitor := GetGlobalMonitor()
	if globalMonitor == nil || globalMonitor.GetMetrics() == nil {
		// Only create if no global monitor exists
		monitor := NewMonitor(nil)
		if monitor == nil {
			t.Fatal("NewMonitor returned nil")
		}
		if monitor.GetMetrics() == nil {
			t.Error("Expected metrics to be initialized")
		}
	} else {
		// Use existing global monitor for testing
		if globalMonitor.GetMetrics() == nil {
			t.Error("Expected metrics to be initialized")
		}
	}

	// Test with custom config (disable metrics to avoid Prometheus registration)
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor2 := NewMonitor(config)
	if monitor2.GetMetrics() != nil {
		t.Error("Expected metrics to be nil when disabled")
	}
	if monitor2.GetTracer() == nil {
		t.Error("Expected tracer to be initialized")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       false,
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: true,
		MonitorInterval:     1 * time.Second, // Must be positive
	}
	monitor := NewMonitor(config)

	monitor.Start()
	// Should not panic

	monitor.Stop()
	// Should not panic
}

func TestMonitor_GetMetrics(t *testing.T) {
	// Use existing monitor if available to avoid Prometheus registration
	globalMonitor := GetGlobalMonitor()
	if globalMonitor != nil && globalMonitor.GetMetrics() != nil {
		metrics := globalMonitor.GetMetrics()
		if metrics == nil {
			t.Error("Expected metrics to be non-nil")
		}
		return
	}

	// Otherwise create one with metrics enabled (only once per test run)
	// This test should run first or use a singleton pattern
	// For now, we'll skip if we can't safely create
	t.Skip("Skipping to avoid Prometheus registration conflicts - run TestNewMonitor first")
}

func TestMonitor_GetTracer(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	tracer := monitor.GetTracer()
	if tracer == nil {
		t.Error("Expected tracer to be non-nil")
	}
}

func TestMonitor_GetSQLAnalyzer(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:     false,
		EnableSQLAnalysis: true,
	}
	monitor := NewMonitor(config)
	analyzer := monitor.GetSQLAnalyzer()
	if analyzer == nil {
		t.Error("Expected SQL analyzer to be non-nil")
	}
}

func TestMonitor_GetSystemMonitor(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableSystemMonitor: true,
		MonitorInterval:     1 * time.Second, // Must be positive
	}
	monitor := NewMonitor(config)
	systemMonitor := monitor.GetSystemMonitor()
	if systemMonitor == nil {
		t.Error("Expected system monitor to be non-nil")
	}
}

func TestMonitor_StartSpan(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()

	newCtx, span := monitor.StartSpan(ctx, "test_span")
	if span == nil {
		t.Error("Expected span to be non-nil")
	}
	if newCtx == ctx {
		t.Error("Expected new context to be different")
	}
}

func TestMonitor_EndSpan(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()

	_, span := monitor.StartSpan(ctx, "test_span")
	monitor.EndSpan(span, nil)
	// Should not panic

	// Test with error
	_, span2 := monitor.StartSpan(ctx, "test_span2")
	monitor.EndSpan(span2, context.Canceled)
	// Should not panic
}

func TestMonitor_RecordSQLQuery(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:     false,
		EnableSQLAnalysis: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()

	monitor.RecordSQLQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 100*time.Millisecond, 10, nil)
	// Should not panic
}

func TestMonitor_RecordHTTPRequest(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
	}
	monitor := NewMonitor(config)
	monitor.RecordHTTPRequest("GET", "/test", "200", "handler", 100*time.Millisecond, 100, 200)
	// Should not panic (metrics disabled, so no-op)
}

func TestMonitor_RecordDBQuery(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
	}
	monitor := NewMonitor(config)
	monitor.RecordDBQuery("SELECT", "users", "query", 50*time.Millisecond)
	// Should not panic (metrics disabled, so no-op)
}

func TestMonitor_RecordCacheHit(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
	}
	monitor := NewMonitor(config)
	monitor.RecordCacheHit("redis", "get")
	// Should not panic (metrics disabled, so no-op)
}

func TestMonitor_RecordCacheMiss(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
	}
	monitor := NewMonitor(config)
	monitor.RecordCacheMiss("redis", "get")
	// Should not panic (metrics disabled, so no-op)
}

func TestMonitor_SetSystemMetric(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
	}
	monitor := NewMonitor(config)
	monitor.SetSystemMetric("test_metric", "category", 100.0)
	// Should not panic (metrics disabled, so no-op)
}

func TestMonitor_GetSystemSummary(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableSystemMonitor: true,
		MonitorInterval:     1 * time.Second, // Must be positive
	}
	monitor := NewMonitor(config)
	summary := monitor.GetSystemSummary()
	if summary == nil {
		t.Error("Expected summary to be non-nil")
	}
	if summary["timestamp"] == nil {
		t.Error("Expected timestamp in summary")
	}
}

func TestMonitor_GetSlowQueries(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:     false,
		EnableSQLAnalysis: true,
	}
	monitor := NewMonitor(config)
	queries := monitor.GetSlowQueries(10)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}
}

func TestMonitor_GetQueryPatterns(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:     false,
		EnableSQLAnalysis: true,
	}
	monitor := NewMonitor(config)
	patterns := monitor.GetQueryPatterns(10)
	if patterns == nil {
		t.Error("Expected patterns to be non-nil")
	}
}

func TestMonitor_GetTraceSpans(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()
	_, span := monitor.StartSpan(ctx, "test")
	traceID := span.TraceID

	spans := monitor.GetTraceSpans(traceID)
	if len(spans) == 0 {
		t.Error("Expected at least one span")
	}
}

func TestMonitor_GetSystemStats(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableSystemMonitor: true,
		MonitorInterval:     1 * time.Second, // Must be positive
	}
	monitor := NewMonitor(config)
	stats := monitor.GetSystemStats(10)
	if stats == nil {
		t.Error("Expected stats to be non-nil")
	}
}

func TestMonitor_GetLatestSystemStats(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableSystemMonitor: true,
		MonitorInterval:     1 * time.Second, // Must be positive
	}
	monitor := NewMonitor(config)
	stats := monitor.GetLatestSystemStats()
	// Can be nil if no stats collected yet
	_ = stats
}

func TestMonitor_IsEnabled(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       true,
		EnableTracing:       false,
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)
	if !monitor.IsEnabled() {
		t.Error("Expected monitor to be enabled")
	}

	config.EnableMetrics = false
	disabledMonitor := NewMonitor(config)
	if disabledMonitor.IsEnabled() {
		t.Error("Expected monitor to be disabled")
	}
}

func TestMonitor_GetConfig(t *testing.T) {
	config := DefaultMonitorConfig()
	monitor := NewMonitor(config)
	retrieved := monitor.GetConfig()
	if retrieved != config {
		t.Error("Expected config to match")
	}
}

func TestMonitor_GetQueriesByTable(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:     false,
		EnableSQLAnalysis: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()
	monitor.RecordSQLQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 100*time.Millisecond, 10, nil)

	queries := monitor.GetQueriesByTable("users", 10)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}
}

func TestMonitor_GetQueriesByOperation(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:     false,
		EnableSQLAnalysis: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()
	monitor.RecordSQLQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 100*time.Millisecond, 10, nil)

	queries := monitor.GetQueriesByOperation("SELECT", 10)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}
}

func TestMonitor_WithDisabledComponents(t *testing.T) {
	config := &MonitorConfig{
		EnableMetrics:       false,
		EnableTracing:       false,
		EnableSQLAnalysis:   false,
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)

	if monitor.GetMetrics() != nil {
		t.Error("Expected metrics to be nil")
	}
	if monitor.GetTracer() != nil {
		t.Error("Expected tracer to be nil")
	}
	if monitor.GetSQLAnalyzer() != nil {
		t.Error("Expected SQL analyzer to be nil")
	}
	if monitor.GetSystemMonitor() != nil {
		t.Error("Expected system monitor to be nil")
	}

	// These should not panic
	monitor.RecordHTTPRequest("GET", "/test", "200", "handler", time.Second, 100, 200)
	monitor.RecordDBQuery("SELECT", "users", "query", time.Millisecond*100)
	monitor.RecordCacheHit("redis", "get")
	_, span := monitor.StartSpan(context.Background(), "test")
	monitor.EndSpan(span, nil)
}
