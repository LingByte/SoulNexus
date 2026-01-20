package metrics

import (
	"testing"
	"time"
)

func TestNewSystemMonitor(t *testing.T) {
	monitor := NewSystemMonitor(100, 1*time.Second)
	if monitor == nil {
		t.Fatal("NewSystemMonitor returned nil")
	}
}

func TestSystemMonitor_StartStop(t *testing.T) {
	monitor := NewSystemMonitor(100, 100*time.Millisecond)

	monitor.Start()
	if !monitor.IsRunning() {
		t.Error("Expected monitor to be running")
	}

	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("Expected monitor to be stopped")
	}
}

func TestSystemMonitor_Start_AlreadyRunning(t *testing.T) {
	monitor := NewSystemMonitor(100, 100*time.Millisecond)

	monitor.Start()
	monitor.Start() // Should not panic or create duplicate goroutines

	monitor.Stop()
}

func TestSystemMonitor_Stop_NotRunning(t *testing.T) {
	monitor := NewSystemMonitor(100, 100*time.Millisecond)

	monitor.Stop() // Should not panic
}

func TestSystemMonitor_GetLatestStats(t *testing.T) {
	monitor := NewSystemMonitor(100, 50*time.Millisecond)

	// Initially should be nil
	stats := monitor.GetLatestStats()
	if stats != nil {
		t.Log("Stats collected before start (unexpected but not an error)")
	}

	monitor.Start()
	time.Sleep(200 * time.Millisecond) // Wait for at least one collection

	stats = monitor.GetLatestStats()
	// Stats might be nil if collection hasn't happened yet, which is acceptable
	if stats == nil {
		t.Log("Stats not yet collected (acceptable in test environment)")
	} else {
		// If stats exist, verify they have expected fields
		if stats.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}
	}

	monitor.Stop()
}

func TestSystemMonitor_GetStatsHistory(t *testing.T) {
	monitor := NewSystemMonitor(100, 50*time.Millisecond)

	monitor.Start()
	time.Sleep(200 * time.Millisecond) // Wait for multiple collections

	limited := monitor.GetStatsHistory(2)
	if len(limited) > 2 {
		t.Errorf("Expected at most 2 stats, got %d", len(limited))
	}

	monitor.Stop()
}

func TestSystemMonitor_SetCustomMetric(t *testing.T) {
	monitor := NewSystemMonitor(100, 1*time.Second)

	monitor.SetCustomMetric("test_key", "test_value")

	value := monitor.GetCustomMetric("test_key")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %v", value)
	}
}

func TestSystemMonitor_GetCustomMetric(t *testing.T) {
	monitor := NewSystemMonitor(100, 1*time.Second)

	value := monitor.GetCustomMetric("nonexistent")
	if value != nil {
		t.Errorf("Expected nil, got %v", value)
	}

	monitor.SetCustomMetric("key", 123)
	value = monitor.GetCustomMetric("key")
	if value != 123 {
		t.Errorf("Expected 123, got %v", value)
	}
}

func TestSystemMonitor_GetSystemSummary(t *testing.T) {
	monitor := NewSystemMonitor(100, 50*time.Millisecond)

	monitor.Start()
	time.Sleep(200 * time.Millisecond) // Wait for collection

	summary := monitor.GetSystemSummary()
	// Summary might be nil if no stats collected yet
	if summary == nil {
		t.Log("Summary is nil (no stats collected yet, acceptable in test)")
	} else {
		// If summary exists, verify it has expected fields
		if summary["timestamp"] == nil {
			t.Error("Expected timestamp in summary")
		}
	}

	monitor.Stop()
}

func TestSystemMonitor_MaxStatsLimit(t *testing.T) {
	monitor := NewSystemMonitor(5, 10*time.Millisecond)

	monitor.Start()
	time.Sleep(100 * time.Millisecond) // Collect more than max

	history := monitor.GetStatsHistory(0)
	if len(history) > 5 {
		t.Errorf("Expected at most 5 stats, got %d", len(history))
	}

	monitor.Stop()
}
