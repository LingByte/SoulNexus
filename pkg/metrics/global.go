package metrics

import (
	"sync"
)

var (
	globalMonitor *Monitor
	mu            sync.RWMutex
)

// SetGlobalMonitor 设置全局监控器实例
func SetGlobalMonitor(monitor *Monitor) {
	mu.Lock()
	defer mu.Unlock()
	globalMonitor = monitor
}

// GetGlobalMonitor 获取全局监控器实例
func GetGlobalMonitor() *Monitor {
	mu.RLock()
	defer mu.RUnlock()
	return globalMonitor
}

// IsGlobalMonitorEnabled 检查全局监控器是否启用
func IsGlobalMonitorEnabled() bool {
	monitor := GetGlobalMonitor()
	return monitor != nil && monitor.IsEnabled()
}
