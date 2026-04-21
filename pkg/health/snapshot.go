package health

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"gorm.io/gorm"
)

var (
	processStart = time.Now()
	serviceName  = "soulnexus"
)

// ProcessStart reports when this binary began (for uptime).
func ProcessStart() time.Time { return processStart }

// SetServiceName sets the logical service id in ping payloads (e.g. api, auth).
func SetServiceName(name string) {
	if name != "" {
		serviceName = name
	}
}

// ServiceName returns the configured logical service label.
func ServiceName() string { return serviceName }

// BuildSnapshot returns JSON-friendly runtime stats for GET .../system/ping.
func BuildSnapshot(db *gorm.DB) map[string]interface{} {
	out := map[string]interface{}{
		"ping":             "pong",
		"service":          serviceName,
		"uptime_seconds":   time.Since(processStart).Seconds(),
		"uptime_human":     time.Since(processStart).Round(time.Second).String(),
		"go_version":       runtime.Version(),
		"go_max_procs":     runtime.GOMAXPROCS(0),
		"goroutines":       runtime.NumGoroutine(),
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	out["memory_go"] = map[string]interface{}{
		"alloc_mb":  float64(ms.Alloc) / 1e6,
		"sys_mb":    float64(ms.Sys) / 1e6,
		"heap_mb":   float64(ms.HeapAlloc) / 1e6,
		"stack_kb":  float64(ms.StackInuse) / 1024,
		"gc_runs":   ms.NumGC,
		"gc_cpu_ns": ms.GCCPUFraction,
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		out["memory_host"] = map[string]interface{}{
			"total_mb":     float64(vm.Total) / 1e6,
			"available_mb": float64(vm.Available) / 1e6,
			"used_percent": vm.UsedPercent,
		}
	}

	if avg, err := load.Avg(); err == nil {
		out["load_average"] = map[string]float64{
			"1m":  avg.Load1,
			"5m":  avg.Load5,
			"15m": avg.Load15,
		}
	}

	if pct, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(pct) > 0 {
		out["cpu_percent"] = pct[0]
	}

	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			pingErr := sqlDB.Ping()
			st := map[string]interface{}{}
			if pingErr == nil {
				st["status"] = "ok"
			} else {
				st["status"] = "error"
				st["error"] = pingErr.Error()
			}
			out["database"] = st
		}
	}

	return out
}
