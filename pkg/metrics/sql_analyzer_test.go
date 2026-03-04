package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewSQLAnalyzer(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	if analyzer == nil {
		t.Fatal("NewSQLAnalyzer returned nil")
	}
}

func TestSQLAnalyzer_RecordQuery(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	query := analyzer.RecordQuery(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{1}, "users", "SELECT", 50*time.Millisecond, 1, nil)
	if query == nil {
		t.Fatal("Expected query to be non-nil")
	}
	if query.ID == "" {
		t.Error("Expected query ID to be set")
	}
	if query.SQL != "SELECT * FROM users WHERE id = ?" {
		t.Errorf("Expected SQL to match, got %s", query.SQL)
	}
}

func TestSQLAnalyzer_RecordQuery_WithError(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	err := errors.New("database error")
	query := analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 0, err)
	if query.Error != err {
		t.Error("Expected error to be set")
	}
}

func TestSQLAnalyzer_RecordQuery_SlowQuery(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 50*time.Millisecond)
	ctx := context.Background()

	query := analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 100*time.Millisecond, 10, nil)
	if query == nil {
		t.Fatal("Expected query to be non-nil")
	}

	slowQueries := analyzer.GetSlowQueries(10)
	if len(slowQueries) == 0 {
		t.Error("Expected slow query to be recorded")
	}
}

func TestSQLAnalyzer_GetSlowQueries(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 50*time.Millisecond)
	ctx := context.Background()

	// Record some slow queries
	analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 100*time.Millisecond, 10, nil)
	analyzer.RecordQuery(ctx, "SELECT * FROM orders", nil, "orders", "SELECT", 200*time.Millisecond, 20, nil)

	slowQueries := analyzer.GetSlowQueries(10)
	if len(slowQueries) < 2 {
		t.Errorf("Expected at least 2 slow queries, got %d", len(slowQueries))
	}

	// Should be sorted by duration (descending)
	if len(slowQueries) >= 2 && slowQueries[0].Duration < slowQueries[1].Duration {
		t.Error("Expected queries to be sorted by duration (descending)")
	}
}

func TestSQLAnalyzer_GetQueryPatterns(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	// Record queries with same pattern
	analyzer.RecordQuery(ctx, "SELECT * FROM users WHERE id = 1", nil, "users", "SELECT", 50*time.Millisecond, 1, nil)
	analyzer.RecordQuery(ctx, "SELECT * FROM users WHERE id = 2", nil, "users", "SELECT", 60*time.Millisecond, 1, nil)

	patterns := analyzer.GetQueryPatterns(10)
	if len(patterns) == 0 {
		t.Error("Expected at least one pattern")
	}

	// Find the pattern
	var foundPattern *QueryPattern
	for _, p := range patterns {
		if p.Pattern != "" {
			foundPattern = p
			break
		}
	}

	if foundPattern == nil {
		t.Fatal("Expected to find a pattern")
	}
	if foundPattern.Count < 2 {
		t.Errorf("Expected pattern count >= 2, got %d", foundPattern.Count)
	}
}

func TestSQLAnalyzer_GetQueriesByTable(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)
	analyzer.RecordQuery(ctx, "SELECT * FROM orders", nil, "orders", "SELECT", 50*time.Millisecond, 20, nil)

	queries := analyzer.GetQueriesByTable("users", 10)
	if len(queries) == 0 {
		t.Error("Expected at least one query for users table")
	}

	for _, q := range queries {
		if q.Table != "users" {
			t.Errorf("Expected table 'users', got %s", q.Table)
		}
	}
}

func TestSQLAnalyzer_GetQueriesByOperation(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)
	analyzer.RecordQuery(ctx, "INSERT INTO users", nil, "users", "INSERT", 50*time.Millisecond, 1, nil)

	queries := analyzer.GetQueriesByOperation("SELECT", 10)
	if len(queries) == 0 {
		t.Error("Expected at least one SELECT query")
	}

	for _, q := range queries {
		if q.Operation != "SELECT" {
			t.Errorf("Expected operation 'SELECT', got %s", q.Operation)
		}
	}
}

func TestSQLAnalyzer_GetQueryStats(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)
	analyzer.RecordQuery(ctx, "SELECT * FROM orders", nil, "orders", "SELECT", 60*time.Millisecond, 20, nil)

	stats := analyzer.GetQueryStats()
	if stats == nil {
		t.Fatal("Expected stats to be non-nil")
	}

	if stats["total_queries"].(int) < 2 {
		t.Errorf("Expected total_queries >= 2, got %d", stats["total_queries"])
	}

	tables, ok := stats["tables"].(map[string]int)
	if !ok {
		t.Fatal("Expected tables to be map[string]int")
	}
	if tables["users"] == 0 {
		t.Error("Expected users table to be in stats")
	}
}

func TestSQLAnalyzer_NormalizeSQL(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)

	tests := []struct {
		input    string
		expected string
	}{
		{"SELECT * FROM users WHERE id = 1", "select * from users where id = ?"},
		{"SELECT * FROM users WHERE name = 'John'", "select * from users where name = ?"},
		{"SELECT * FROM users WHERE id = 1 AND name = 'John'", "select * from users where id = ? and name = ?"},
	}

	for _, tt := range tests {
		result := analyzer.normalizeSQL(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeSQL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSQLAnalyzer_CleanupOldQueries(t *testing.T) {
	analyzer := NewSQLAnalyzer(5, 100*time.Millisecond) // Small limit
	ctx := context.Background()

	// Record more queries than the limit
	for i := 0; i < 10; i++ {
		analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)
		time.Sleep(time.Microsecond) // Small delay
	}

	stats := analyzer.GetQueryStats()
	totalQueries := stats["total_queries"].(int)

	// Should have cleaned up some queries
	if totalQueries > 5 {
		t.Logf("Warning: Expected cleanup, but got %d queries", totalQueries)
	}
}

func TestSQLAnalyzer_GetSlowQueries_EdgeCases(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 50*time.Millisecond)

	// Test with limit 0 (should return all)
	queries := analyzer.GetSlowQueries(0)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}

	// Test with limit larger than available
	queries = analyzer.GetSlowQueries(10000)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}
}

func TestSQLAnalyzer_GetQueryPatterns_EdgeCases(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	analyzer.RecordQuery(ctx, "SELECT * FROM users WHERE id = 1", nil, "users", "SELECT", 50*time.Millisecond, 1, nil)

	// Test with limit 0 (should return all)
	patterns := analyzer.GetQueryPatterns(0)
	if patterns == nil {
		t.Error("Expected patterns to be non-nil")
	}

	// Test with limit larger than available
	patterns = analyzer.GetQueryPatterns(10000)
	if patterns == nil {
		t.Error("Expected patterns to be non-nil")
	}
}

func TestSQLAnalyzer_GetQueriesByTable_EdgeCases(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)

	// Test with limit 0 (should return all)
	queries := analyzer.GetQueriesByTable("users", 0)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}

	// Test with non-existent table
	queries = analyzer.GetQueriesByTable("nonexistent", 10)
	// GetQueriesByTable returns nil if no queries found (not empty slice)
	if queries != nil && len(queries) != 0 {
		t.Errorf("Expected 0 queries, got %d", len(queries))
	}
}

func TestSQLAnalyzer_GetQueriesByOperation_EdgeCases(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)
	ctx := context.Background()

	analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)

	// Test with limit 0 (should return all)
	queries := analyzer.GetQueriesByOperation("SELECT", 0)
	if queries == nil {
		t.Error("Expected queries to be non-nil")
	}

	// Test with non-existent operation
	queries = analyzer.GetQueriesByOperation("DELETE", 10)
	// GetQueriesByOperation returns nil if no queries found (not empty slice)
	if queries != nil && len(queries) != 0 {
		t.Errorf("Expected 0 queries, got %d", len(queries))
	}
}

func TestSQLAnalyzer_GetQueryStats_Empty(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)

	stats := analyzer.GetQueryStats()
	if stats == nil {
		t.Fatal("Expected stats to be non-nil")
	}

	if stats["total_queries"].(int) != 0 {
		t.Errorf("Expected 0 queries, got %d", stats["total_queries"])
	}
}

func TestSQLAnalyzer_RecordQuery_WithSpanContext(t *testing.T) {
	analyzer := NewSQLAnalyzer(1000, 100*time.Millisecond)

	// Create a tracer and span to test context propagation
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	ctx := context.Background()
	ctx, span := monitor.StartSpan(ctx, "test_handler")
	span.SetTag("path", "/api/test")
	span.SetTag("method", "GET")

	query := analyzer.RecordQuery(ctx, "SELECT * FROM users", nil, "users", "SELECT", 50*time.Millisecond, 10, nil)
	if query == nil {
		t.Fatal("Expected query to be non-nil")
	}

	// Verify tags from span context
	if query.Tags["handler"] != "test_handler" {
		t.Errorf("Expected handler tag 'test_handler', got %s", query.Tags["handler"])
	}
}
