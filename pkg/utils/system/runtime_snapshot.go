package system

import (
	"runtime"
	"sync/atomic"
	"time"
)

// RuntimeMemorySnapshot is the full runtime.MemStats export for ops dashboards.
type RuntimeMemorySnapshot struct {
	Alloc         uint64  `json:"alloc"`
	TotalAlloc    uint64  `json:"totalAlloc"`
	Sys           uint64  `json:"sys"`
	Lookups       uint64  `json:"lookups"`
	Mallocs       uint64  `json:"mallocs"`
	Frees         uint64  `json:"frees"`
	HeapAlloc     uint64  `json:"heapAlloc"`
	HeapSys       uint64  `json:"heapSys"`
	HeapIdle      uint64  `json:"heapIdle"`
	HeapInuse     uint64  `json:"heapInuse"`
	HeapReleased  uint64  `json:"heapReleased"`
	HeapObjects   uint64  `json:"heapObjects"`
	StackInuse    uint64  `json:"stackInuse"`
	StackSys      uint64  `json:"stackSys"`
	MSpanInuse    uint64  `json:"mSpanInuse"`
	MSpanSys      uint64  `json:"mSpanSys"`
	MCacheInuse   uint64  `json:"mCacheInuse"`
	MCacheSys     uint64  `json:"mCacheSys"`
	BuckHashSys   uint64  `json:"buckHashSys"`
	GCSys         uint64  `json:"gcSys"`
	OtherSys      uint64  `json:"otherSys"`
	NextGC        uint64  `json:"nextGC"`
	LastGC        uint64  `json:"lastGC"`
	PauseTotalNs  uint64  `json:"pauseTotalNs"`
	NumGC         uint32  `json:"numGC"`
	NumForcedGC   uint32  `json:"numForcedGC"`
	GCCPUFraction float64 `json:"gcCpuFraction"`
	EnableGC      bool    `json:"enableGC"`
}

// GCSnapshot summarizes recent GC STW pauses.
type GCSnapshot struct {
	NumGC              uint32  `json:"numGC"`
	NumForcedGC        uint32  `json:"numForcedGC"`
	PauseTotalNs       uint64  `json:"pauseTotalNs"`
	PauseTotalMs       float64 `json:"pauseTotalMs"`
	RecentPauseAvgMs   float64 `json:"recentPauseAvgMs"`
	RecentPauseMaxMs   float64 `json:"recentPauseMaxMs"`
	RecentPauseSamples int     `json:"recentPauseSamples"`
	GCCPUFraction      float64 `json:"gcCpuFraction"`
	NextGC             uint64  `json:"nextGC"`
	LastGCUnixNano     int64   `json:"lastGCUnixNano"`
	GCPerMinute        float64 `json:"gcPerMinute"`
}

// GoroutineSnapshot captures goroutine / thread runtime counters.
type GoroutineSnapshot struct {
	NumGoroutine   int            `json:"numGoroutine"`
	NumGoroutineMax int           `json:"numGoroutineMax"`
	NumThread      int            `json:"numThread"`
	NumCgoCall     int64          `json:"numCgoCall"`
	ByState        map[string]int `json:"byState"`
	LeakSuspect    bool           `json:"leakSuspect"`
}

// RuntimeSnapshot bundles Go runtime telemetry.
type RuntimeSnapshot struct {
	GoVersion string                `json:"goVersion"`
	Memory    RuntimeMemorySnapshot `json:"memory"`
	GC        GCSnapshot            `json:"gc"`
	Goroutine GoroutineSnapshot     `json:"goroutine"`
}

var (
	goroutineHighWater int64
	gcCountAtStart     uint32
	startedAt          = time.Now()
	lastGCNum          uint32
	lastGCAt           time.Time
)

func init() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	gcCountAtStart = m.NumGC
	lastGCNum = m.NumGC
	lastGCAt = time.Now()
}

// CollectRuntimeSnapshot reads MemStats and goroutine breakdown.
func CollectRuntimeSnapshot() RuntimeSnapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	n := int64(runtime.NumGoroutine())
	for {
		old := atomic.LoadInt64(&goroutineHighWater)
		if n <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&goroutineHighWater, old, n) {
			break
		}
	}

	gcSnap := buildGCSnapshot(&mem)
	gr := buildGoroutineSnapshot(n)

	return RuntimeSnapshot{
		GoVersion: runtime.Version(),
		Memory:    memStatsToSnapshot(&mem),
		GC:        gcSnap,
		Goroutine: gr,
	}
}

func memStatsToSnapshot(m *runtime.MemStats) RuntimeMemorySnapshot {
	return RuntimeMemorySnapshot{
		Alloc: m.Alloc, TotalAlloc: m.TotalAlloc, Sys: m.Sys,
		Lookups: m.Lookups, Mallocs: m.Mallocs, Frees: m.Frees,
		HeapAlloc: m.HeapAlloc, HeapSys: m.HeapSys, HeapIdle: m.HeapIdle,
		HeapInuse: m.HeapInuse, HeapReleased: m.HeapReleased, HeapObjects: m.HeapObjects,
		StackInuse: m.StackInuse, StackSys: m.StackSys,
		MSpanInuse: m.MSpanInuse, MSpanSys: m.MSpanSys,
		MCacheInuse: m.MCacheInuse, MCacheSys: m.MCacheSys,
		BuckHashSys: m.BuckHashSys, GCSys: m.GCSys, OtherSys: m.OtherSys,
		NextGC: m.NextGC, LastGC: m.LastGC, PauseTotalNs: m.PauseTotalNs,
		NumGC: m.NumGC, NumForcedGC: m.NumForcedGC, GCCPUFraction: m.GCCPUFraction,
		EnableGC: m.EnableGC,
	}
}

func buildGCSnapshot(m *runtime.MemStats) GCSnapshot {
	snap := GCSnapshot{
		NumGC: m.NumGC, NumForcedGC: m.NumForcedGC,
		PauseTotalNs: m.PauseTotalNs,
		PauseTotalMs: float64(m.PauseTotalNs) / 1e6,
		GCCPUFraction: m.GCCPUFraction, NextGC: m.NextGC,
	}
	if m.LastGC > 0 {
		snap.LastGCUnixNano = int64(m.LastGC)
	}

	n := int(m.NumGC)
	if n > 256 {
		n = 256
	}
	var sum, max uint64
	for i := 0; i < n; i++ {
		idx := (int(m.NumGC) - 1 - i) & 255
		p := m.PauseNs[idx]
		if p == 0 {
			continue
		}
		sum += p
		if p > max {
			max = p
		}
		snap.RecentPauseSamples++
	}
	if snap.RecentPauseSamples > 0 {
		snap.RecentPauseAvgMs = float64(sum) / float64(snap.RecentPauseSamples) / 1e6
	}
	snap.RecentPauseMaxMs = float64(max) / 1e6

	if m.NumGC > lastGCNum {
		elapsed := time.Since(lastGCAt).Minutes()
		if elapsed > 0 {
			snap.GCPerMinute = float64(m.NumGC-lastGCNum) / elapsed
		}
		lastGCNum = m.NumGC
		lastGCAt = time.Now()
	}

	return snap
}
