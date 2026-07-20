package system

import (
	voiceMetrics "github.com/LingByte/SoulNexus/pkg/voice/metrics"
)

// SyncPrometheusRuntimeGauges pushes runtime stats into the voice metrics registry.
func SyncPrometheusRuntimeGauges() {
	rt := CollectRuntimeSnapshot()
	host := CollectHostSnapshot()

	set := func(name string, v float64) {
		voiceMetrics.Default.SetGauge(name, "runtime gauge", nil, v)
	}

	set("go_goroutines", float64(rt.Goroutine.NumGoroutine))
	set("go_threads", float64(host.Process.NumThreads))
	set("go_memstats_heap_inuse_bytes", float64(rt.Memory.HeapInuse))
	set("go_memstats_heap_idle_bytes", float64(rt.Memory.HeapIdle))
	set("go_memstats_heap_released_bytes", float64(rt.Memory.HeapReleased))
	set("go_memstats_heap_sys_bytes", float64(rt.Memory.HeapSys))
	set("go_memstats_stack_inuse_bytes", float64(rt.Memory.StackInuse))
	set("go_memstats_stack_sys_bytes", float64(rt.Memory.StackSys))
	set("go_memstats_gc_sys_bytes", float64(rt.Memory.GCSys))
	set("go_memstats_next_gc_bytes", float64(rt.Memory.NextGC))
	set("go_memstats_gc_cpu_fraction", rt.Memory.GCCPUFraction)
	set("go_memstats_alloc_bytes", float64(rt.Memory.Alloc))
	set("go_memstats_sys_bytes", float64(rt.Memory.Sys))
	set("go_gc_duration_seconds_max", rt.GC.RecentPauseMaxMs/1000)
	set("go_gc_duration_seconds_avg", rt.GC.RecentPauseAvgMs/1000)
	set("process_resident_memory_bytes", float64(host.Process.RSS))
	set("process_virtual_memory_bytes", float64(host.Process.VMS))
	set("process_open_fds", float64(host.Process.OpenFDs))
	set("process_max_fds", float64(host.Process.MaxFDs))
	set("process_cpu_seconds_total", host.Process.CPUPercent)
}
