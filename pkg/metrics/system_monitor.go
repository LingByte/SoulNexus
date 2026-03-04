package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// SystemStats 系统统计信息
type SystemStats struct {
	Timestamp     time.Time              `json:"timestamp"`
	CPU           CPUStats               `json:"cpu"`
	Memory        MemoryStats            `json:"memory"`
	Disk          DiskStats              `json:"disk"`
	Network       NetworkStats           `json:"network"`
	Process       ProcessStats           `json:"process"`
	Runtime       RuntimeStats           `json:"runtime"`
	Host          HostStats              `json:"host"`
	CustomMetrics map[string]interface{} `json:"custom_metrics,omitempty"`
}

// CPUStats CPU统计信息
type CPUStats struct {
	UsagePercent    float64   `json:"usage_percent"`
	UsagePercentAll []float64 `json:"usage_percent_all"`
	Count           int       `json:"count"`
	CountLogical    int       `json:"count_logical"`
	Frequency       float64   `json:"frequency"`
	Temperature     float64   `json:"temperature"`
	LoadAvg         []float64 `json:"load_avg"`
}

// MemoryStats 内存统计信息
type MemoryStats struct {
	Total        uint64  `json:"total"`
	Available    uint64  `json:"available"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
	SwapTotal    uint64  `json:"swap_total"`
	SwapUsed     uint64  `json:"swap_used"`
	SwapFree     uint64  `json:"swap_free"`
}

// DiskStats 磁盘统计信息
type DiskStats struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
	ReadBytes    uint64  `json:"read_bytes"`
	WriteBytes   uint64  `json:"write_bytes"`
	ReadCount    uint64  `json:"read_count"`
	WriteCount   uint64  `json:"write_count"`
	ReadTime     uint64  `json:"read_time"`
	WriteTime    uint64  `json:"write_time"`
}

// NetworkStats 网络统计信息
type NetworkStats struct {
	BytesSent   uint64                    `json:"bytes_sent"`
	BytesRecv   uint64                    `json:"bytes_recv"`
	PacketsSent uint64                    `json:"packets_sent"`
	PacketsRecv uint64                    `json:"packets_recv"`
	ErrIn       uint64                    `json:"err_in"`
	ErrOut      uint64                    `json:"err_out"`
	DropIn      uint64                    `json:"drop_in"`
	DropOut     uint64                    `json:"drop_out"`
	Connections int                       `json:"connections"`
	Interfaces  map[string]InterfaceStats `json:"interfaces"`
}

// InterfaceStats 网络接口统计
type InterfaceStats struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrIn       uint64 `json:"err_in"`
	ErrOut      uint64 `json:"err_out"`
	DropIn      uint64 `json:"drop_in"`
	DropOut     uint64 `json:"drop_out"`
	IsUp        bool   `json:"is_up"`
}

// ProcessStats 进程统计信息
type ProcessStats struct {
	PID           int32   `json:"pid"`
	PPID          int32   `json:"ppid"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	MemoryRSS     uint64  `json:"memory_rss"`
	MemoryVMS     uint64  `json:"memory_vms"`
	NumThreads    int32   `json:"num_threads"`
	NumFDs        int32   `json:"num_fds"`
	CreateTime    int64   `json:"create_time"`
	Uptime        float64 `json:"uptime"`
}

// RuntimeStats Go运行时统计信息
type RuntimeStats struct {
	Goroutines   int      `json:"goroutines"`
	Threads      int      `json:"threads"`
	HeapAlloc    uint64   `json:"heap_alloc"`
	HeapSys      uint64   `json:"heap_sys"`
	HeapIdle     uint64   `json:"heap_idle"`
	HeapInuse    uint64   `json:"heap_inuse"`
	HeapReleased uint64   `json:"heap_released"`
	HeapObjects  uint64   `json:"heap_objects"`
	StackInuse   uint64   `json:"stack_inuse"`
	StackSys     uint64   `json:"stack_sys"`
	MSpanInuse   uint64   `json:"mspan_inuse"`
	MSpanSys     uint64   `json:"mspan_sys"`
	MCacheInuse  uint64   `json:"mcache_inuse"`
	MCacheSys    uint64   `json:"mcache_sys"`
	BuckHashSys  uint64   `json:"buck_hash_sys"`
	GCSys        uint64   `json:"gc_sys"`
	OtherSys     uint64   `json:"other_sys"`
	NextGC       uint64   `json:"next_gc"`
	LastGC       uint64   `json:"last_gc"`
	PauseTotalNs uint64   `json:"pause_total_ns"`
	PauseNs      []uint64 `json:"pause_ns"`
	NumGC        uint32   `json:"num_gc"`
}

// HostStats 主机统计信息
type HostStats struct {
	Hostname     string `json:"hostname"`
	Uptime       uint64 `json:"uptime"`
	BootTime     uint64 `json:"boot_time"`
	Platform     string `json:"platform"`
	Family       string `json:"family"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	Users        []User `json:"users"`
}

// User 用户信息
type User struct {
	User     string `json:"user"`
	Terminal string `json:"terminal"`
	Host     string `json:"host"`
	Started  int    `json:"started"`
}

// SystemMonitor 系统监控器
type SystemMonitor struct {
	stats         []*SystemStats
	mu            sync.RWMutex
	maxStats      int
	interval      time.Duration
	stopChan      chan struct{}
	isRunning     bool
	customMetrics map[string]interface{}
}

// NewSystemMonitor 创建系统监控器
func NewSystemMonitor(maxStats int, interval time.Duration) *SystemMonitor {
	return &SystemMonitor{
		stats:         make([]*SystemStats, 0),
		maxStats:      maxStats,
		interval:      interval,
		stopChan:      make(chan struct{}),
		isRunning:     false,
		customMetrics: make(map[string]interface{}),
	}
}

// Start 启动监控
func (sm *SystemMonitor) Start() {
	if sm.isRunning {
		return
	}

	sm.isRunning = true
	go sm.monitorLoop()
}

// Stop 停止监控
func (sm *SystemMonitor) Stop() {
	if !sm.isRunning {
		return
	}

	sm.isRunning = false
	close(sm.stopChan)
}

// monitorLoop 监控循环
func (sm *SystemMonitor) monitorLoop() {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.collectStats()
		case <-sm.stopChan:
			return
		}
	}
}

// collectStats 收集统计信息
func (sm *SystemMonitor) collectStats() {
	stats := &SystemStats{
		Timestamp:     time.Now(),
		CustomMetrics: make(map[string]interface{}),
	}

	// 收集CPU信息
	sm.collectCPUStats(stats)

	// 收集内存信息
	sm.collectMemoryStats(stats)

	// 收集磁盘信息
	sm.collectDiskStats(stats)

	// 收集网络信息
	sm.collectNetworkStats(stats)

	// 收集进程信息
	sm.collectProcessStats(stats)

	// 收集运行时信息
	sm.collectRuntimeStats(stats)

	// 收集主机信息
	sm.collectHostStats(stats)

	// 复制自定义指标
	for k, v := range sm.customMetrics {
		stats.CustomMetrics[k] = v
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 添加新统计信息
	sm.stats = append(sm.stats, stats)

	// 保持统计信息在合理范围内
	if len(sm.stats) > sm.maxStats {
		sm.stats = sm.stats[1:]
	}
}

// collectCPUStats 收集CPU统计信息
func (sm *SystemMonitor) collectCPUStats(stats *SystemStats) {
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		stats.CPU.UsagePercent = cpuPercent[0]
	}

	if cpuPercentAll, err := cpu.Percent(0, true); err == nil {
		stats.CPU.UsagePercentAll = cpuPercentAll
	}

	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		stats.CPU.Count = len(cpuInfo)
		stats.CPU.CountLogical = runtime.NumCPU()
		stats.CPU.Frequency = cpuInfo[0].Mhz
	}
	stats.CPU.LoadAvg = []float64{0, 0, 0}
}

// collectMemoryStats 收集内存统计信息
func (sm *SystemMonitor) collectMemoryStats(stats *SystemStats) {
	if vmstat, err := mem.VirtualMemory(); err == nil {
		stats.Memory.Total = vmstat.Total
		stats.Memory.Available = vmstat.Available
		stats.Memory.Used = vmstat.Used
		stats.Memory.Free = vmstat.Free
		stats.Memory.UsagePercent = vmstat.UsedPercent
	}

	if swapStat, err := mem.SwapMemory(); err == nil {
		stats.Memory.SwapTotal = swapStat.Total
		stats.Memory.SwapUsed = swapStat.Used
		stats.Memory.SwapFree = swapStat.Free
	}
}

// collectDiskStats 收集磁盘统计信息
func (sm *SystemMonitor) collectDiskStats(stats *SystemStats) {
	if diskStat, err := disk.Usage("/"); err == nil {
		stats.Disk.Total = diskStat.Total
		stats.Disk.Used = diskStat.Used
		stats.Disk.Free = diskStat.Free
		stats.Disk.UsagePercent = diskStat.UsedPercent
	}

	if ioCounters, err := disk.IOCounters(); err == nil {
		for _, io := range ioCounters {
			stats.Disk.ReadBytes += io.ReadBytes
			stats.Disk.WriteBytes += io.WriteBytes
			stats.Disk.ReadCount += io.ReadCount
			stats.Disk.WriteCount += io.WriteCount
			stats.Disk.ReadTime += io.ReadTime
			stats.Disk.WriteTime += io.WriteTime
		}
	}
}

// collectNetworkStats 收集网络统计信息
func (sm *SystemMonitor) collectNetworkStats(stats *SystemStats) {
	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		stats.Network.BytesSent = netIO[0].BytesSent
		stats.Network.BytesRecv = netIO[0].BytesRecv
		stats.Network.PacketsSent = netIO[0].PacketsSent
		stats.Network.PacketsRecv = netIO[0].PacketsRecv
	}

	// 收集各接口统计
	if netIO, err := net.IOCounters(true); err == nil {
		stats.Network.Interfaces = make(map[string]InterfaceStats)
		for _, io := range netIO {
			stats.Network.Interfaces[io.Name] = InterfaceStats{
				Name:        io.Name,
				BytesSent:   io.BytesSent,
				BytesRecv:   io.BytesRecv,
				PacketsSent: io.PacketsSent,
				PacketsRecv: io.PacketsRecv,
			}
		}
	}
}

// collectProcessStats 收集进程统计信息
func (sm *SystemMonitor) collectProcessStats(stats *SystemStats) {
	if p, err := process.NewProcess(int32(runtime.NumCPU())); err == nil {
		if name, err := p.Name(); err == nil {
			stats.Process.Name = name
		}
		stats.Process.Status = "unknown"
		if cpuPercent, err := p.CPUPercent(); err == nil {
			stats.Process.CPUPercent = cpuPercent
		}
		if memoryPercent, err := p.MemoryPercent(); err == nil {
			stats.Process.MemoryPercent = memoryPercent
		}
		if memoryInfo, err := p.MemoryInfo(); err == nil {
			stats.Process.MemoryRSS = memoryInfo.RSS
			stats.Process.MemoryVMS = memoryInfo.VMS
		}
		if numThreads, err := p.NumThreads(); err == nil {
			stats.Process.NumThreads = numThreads
		}
		if createTime, err := p.CreateTime(); err == nil {
			stats.Process.CreateTime = createTime
			stats.Process.Uptime = float64(time.Now().Unix()-createTime/1000) / 3600 // 小时
		}
	}
}

// collectRuntimeStats 收集运行时统计信息
func (sm *SystemMonitor) collectRuntimeStats(stats *SystemStats) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats.Runtime.Goroutines = runtime.NumGoroutine()
	stats.Runtime.Threads = runtime.GOMAXPROCS(0)
	stats.Runtime.HeapAlloc = m.HeapAlloc
	stats.Runtime.HeapSys = m.HeapSys
	stats.Runtime.HeapIdle = m.HeapIdle
	stats.Runtime.HeapInuse = m.HeapInuse
	stats.Runtime.HeapReleased = m.HeapReleased
	stats.Runtime.HeapObjects = m.HeapObjects
	stats.Runtime.StackInuse = m.StackInuse
	stats.Runtime.StackSys = m.StackSys
	stats.Runtime.MSpanInuse = m.MSpanInuse
	stats.Runtime.MSpanSys = m.MSpanSys
	stats.Runtime.MCacheInuse = m.MCacheInuse
	stats.Runtime.MCacheSys = m.MCacheSys
	stats.Runtime.BuckHashSys = m.BuckHashSys
	stats.Runtime.GCSys = m.GCSys
	stats.Runtime.OtherSys = m.OtherSys
	stats.Runtime.NextGC = m.NextGC
	stats.Runtime.LastGC = m.LastGC
	stats.Runtime.PauseTotalNs = m.PauseTotalNs
	stats.Runtime.PauseNs = []uint64{}
	stats.Runtime.NumGC = m.NumGC
}

// collectHostStats 收集主机统计信息
func (sm *SystemMonitor) collectHostStats(stats *SystemStats) {
	if hostInfo, err := host.Info(); err == nil {
		stats.Host.Hostname = hostInfo.Hostname
		stats.Host.Uptime = hostInfo.Uptime
		stats.Host.BootTime = hostInfo.BootTime
		stats.Host.Platform = hostInfo.Platform
	}

	if users, err := host.Users(); err == nil {
		stats.Host.Users = make([]User, len(users))
		for i, user := range users {
			stats.Host.Users[i] = User{
				User:     user.User,
				Terminal: user.Terminal,
				Host:     user.Host,
				Started:  user.Started,
			}
		}
	}
}

// GetLatestStats 获取最新统计信息
func (sm *SystemMonitor) GetLatestStats() *SystemStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.stats) == 0 {
		return nil
	}
	return sm.stats[len(sm.stats)-1]
}

// GetStatsHistory 获取统计历史
func (sm *SystemMonitor) GetStatsHistory(limit int) []*SystemStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if limit <= 0 || limit > len(sm.stats) {
		limit = len(sm.stats)
	}

	start := len(sm.stats) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*SystemStats, limit)
	copy(result, sm.stats[start:])
	return result
}

// SetCustomMetric 设置自定义指标
func (sm *SystemMonitor) SetCustomMetric(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.customMetrics[key] = value
}

// GetCustomMetric 获取自定义指标
func (sm *SystemMonitor) GetCustomMetric(key string) interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.customMetrics[key]
}

// GetSystemSummary 获取系统摘要
func (sm *SystemMonitor) GetSystemSummary() map[string]interface{} {
	latest := sm.GetLatestStats()
	if latest == nil {
		return nil
	}

	summary := map[string]interface{}{
		"timestamp":      latest.Timestamp,
		"cpu_usage":      latest.CPU.UsagePercent,
		"memory_usage":   latest.Memory.UsagePercent,
		"disk_usage":     latest.Disk.UsagePercent,
		"goroutines":     latest.Runtime.Goroutines,
		"heap_alloc":     latest.Runtime.HeapAlloc,
		"uptime":         latest.Host.Uptime,
		"custom_metrics": latest.CustomMetrics,
		"hostname":       latest.Host.Hostname,
		"platform":       latest.Host.Platform,
		"architecture":   runtime.GOARCH,
	}

	return summary
}

// IsRunning 检查是否正在运行
func (sm *SystemMonitor) IsRunning() bool {
	return sm.isRunning
}
