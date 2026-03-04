package metrics

import (
	"testing"
)

func TestSetGlobalMonitor(t *testing.T) {
	// Save original
	original := GetGlobalMonitor()
	defer SetGlobalMonitor(original)

	// Use config without metrics to avoid Prometheus registration
	config := &MonitorConfig{
		EnableMetrics: false,
		EnableTracing: true,
	}
	monitor := NewMonitor(config)
	SetGlobalMonitor(monitor)

	retrieved := GetGlobalMonitor()
	if retrieved != monitor {
		t.Error("GetGlobalMonitor did not return the set monitor")
	}
}

func TestGetGlobalMonitor(t *testing.T) {
	// Save original monitor
	originalMonitor := GetGlobalMonitor()
	defer SetGlobalMonitor(originalMonitor)

	// Clear global monitor first
	SetGlobalMonitor(nil)

	monitor := GetGlobalMonitor()
	if monitor != nil {
		t.Error("Expected nil monitor initially")
	}

	// Set a monitor (reuse existing if available to avoid Prometheus registration issues)
	var newMonitor *Monitor
	if originalMonitor != nil {
		newMonitor = originalMonitor
	} else {
		// Only create new if we don't have one
		config := &MonitorConfig{
			EnableMetrics:       false, // Disable to avoid Prometheus registration
			EnableTracing:       true,
			EnableSQLAnalysis:   false,
			EnableSystemMonitor: false,
		}
		newMonitor = NewMonitor(config)
	}
	SetGlobalMonitor(newMonitor)

	retrieved := GetGlobalMonitor()
	if retrieved != newMonitor {
		t.Error("GetGlobalMonitor did not return the set monitor")
	}
}

func TestIsGlobalMonitorEnabled(t *testing.T) {
	// Save original monitor
	originalMonitor := GetGlobalMonitor()
	defer SetGlobalMonitor(originalMonitor)

	// Test with nil monitor
	SetGlobalMonitor(nil)
	if IsGlobalMonitorEnabled() {
		t.Error("Expected false when monitor is nil")
	}

	// Test with enabled monitor (avoid creating new Metrics to prevent Prometheus registration issues)
	config := &MonitorConfig{
		EnableMetrics:       false, // Disable metrics to avoid Prometheus registration
		EnableTracing:       true,
		EnableSQLAnalysis:   true,
		EnableSystemMonitor: false,
	}
	monitor := NewMonitor(config)
	SetGlobalMonitor(monitor)

	if !IsGlobalMonitorEnabled() {
		t.Error("Expected true when monitor is enabled")
	}

	// Test with disabled monitor
	config.EnableMetrics = false
	config.EnableTracing = false
	config.EnableSQLAnalysis = false
	config.EnableSystemMonitor = false
	disabledMonitor := NewMonitor(config)
	SetGlobalMonitor(disabledMonitor)

	if IsGlobalMonitorEnabled() {
		t.Error("Expected false when all features are disabled")
	}
}
