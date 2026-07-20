package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// CPUSnapshot breaks down CPU usage.
type CPUSnapshot struct {
	UsagePercent  float64 `json:"usagePercent"`
	UserPercent   float64 `json:"userPercent"`
	SystemPercent float64 `json:"systemPercent"`
	IdlePercent   float64 `json:"idlePercent"`
	IowaitPercent float64 `json:"iowaitPercent"`
	NumCPU        int     `json:"numCpu"`
}

// LoadSnapshot is 1/5/15 minute load average.
type LoadSnapshot struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// HostMemorySnapshot is OS-level memory including swap.
type HostMemorySnapshot struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
	SwapTotal   uint64  `json:"swapTotal"`
	SwapUsed    uint64  `json:"swapUsed"`
	SwapPercent float64 `json:"swapPercent"`
}

// DiskPathUsage is usage for one mounted path.
type DiskPathUsage struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	InodesTotal uint64  `json:"inodesTotal"`
	InodesFree  uint64  `json:"inodesFree"`
	InodesUsed  uint64  `json:"inodesUsed"`
}

// NetSnapshot is aggregate network IO since boot.
type NetSnapshot struct {
	BytesSent   uint64 `json:"bytesSent"`
	BytesRecv   uint64 `json:"bytesRecv"`
	PacketsSent uint64 `json:"packetsSent"`
	PacketsRecv uint64 `json:"packetsRecv"`
	DropIn      uint64 `json:"dropIn"`
	DropOut     uint64 `json:"dropOut"`
	ErrIn       uint64 `json:"errIn"`
	ErrOut      uint64 `json:"errOut"`
}

// ProcessSnapshot is this process RSS/VMS/CPU/fds.
type ProcessSnapshot struct {
	PID            int32   `json:"pid"`
	RSS            uint64  `json:"rss"`
	VMS            uint64  `json:"vms"`
	CPUPercent     float64 `json:"cpuPercent"`
	OpenFDs        int32   `json:"openFds"`
	MaxFDs         uint64  `json:"maxFds"`
	NumThreads     int32   `json:"numThreads"`
	CreateTimeUnix int64   `json:"createTimeUnix"`
}

// HostSnapshot aggregates host + process telemetry.
type HostSnapshot struct {
	CPU     CPUSnapshot        `json:"cpu"`
	Load    LoadSnapshot       `json:"load"`
	Memory  HostMemorySnapshot `json:"memory"`
	Disks   []DiskPathUsage    `json:"disks"`
	Network NetSnapshot        `json:"network"`
	Process ProcessSnapshot    `json:"process"`
}

// CollectHostSnapshot samples gopsutil host/process metrics.
func CollectHostSnapshot() HostSnapshot {
	out := HostSnapshot{CPU: CPUSnapshot{NumCPU: runtime.NumCPU()}}

	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		out.CPU.UsagePercent = percents[0]
	}
	if times, err := cpu.Times(false); err == nil && len(times) > 0 {
		t := times[0]
		total := t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
		if total > 0 {
			out.CPU.UserPercent = t.User / total * 100
			out.CPU.SystemPercent = t.System / total * 100
			out.CPU.IdlePercent = t.Idle / total * 100
			out.CPU.IowaitPercent = t.Iowait / total * 100
		}
	}
	if avg, err := load.Avg(); err == nil && avg != nil {
		out.Load = LoadSnapshot{Load1: avg.Load1, Load5: avg.Load5, Load15: avg.Load15}
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		out.Memory = HostMemorySnapshot{
			Total: vm.Total, Available: vm.Available, Used: vm.Used,
			UsedPercent: vm.UsedPercent,
		}
	}
	if sm, err := mem.SwapMemory(); err == nil {
		out.Memory.SwapTotal = sm.Total
		out.Memory.SwapUsed = sm.Used
		out.Memory.SwapPercent = sm.UsedPercent
	}

	out.Disks = collectDiskPaths()
	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		c := counters[0]
		out.Network = NetSnapshot{
			BytesSent: c.BytesSent, BytesRecv: c.BytesRecv,
			PacketsSent: c.PacketsSent, PacketsRecv: c.PacketsRecv,
			DropIn: c.Dropin, DropOut: c.Dropout,
			ErrIn: c.Errin, ErrOut: c.Errout,
		}
	}

	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		ps := ProcessSnapshot{PID: int32(os.Getpid())}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			ps.RSS = mi.RSS
			ps.VMS = mi.VMS
		}
		if cp, err := p.CPUPercent(); err == nil {
			ps.CPUPercent = cp
		}
		if fds, err := p.NumFDs(); err == nil {
			ps.OpenFDs = fds
		}
		if th, err := p.NumThreads(); err == nil {
			ps.NumThreads = th
		}
		if ct, err := p.CreateTime(); err == nil {
			ps.CreateTimeUnix = ct / 1000
		}
		ps.MaxFDs = processMaxFDs()
		out.Process = ps
	}

	return out
}

func processSelfThreads() (int32, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, err
	}
	return p.NumThreads()
}

func collectDiskPaths() []DiskPathUsage {
	paths := []string{
		config.GlobalConfig.Log.Filename,
		config.GlobalConfig.JWT.KeyFile,
	}
	uploadDir := utils.GetEnv(pkgconst.ENV_UPLOAD_DIR)
	if uploadDir == "" {
		uploadDir = pkgconst.DefaultUploadDir
	}
	paths = append(paths, uploadDir)
	if strings.EqualFold(config.GlobalConfig.Database.Driver, pkgconst.DBDriverSQLite) {
		paths = append(paths, config.GlobalConfig.Database.DSN)
	}

	seen := map[string]struct{}{}
	var out []DiskPathUsage
	for _, p := range paths {
		p = diskMonitorPath(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		usage, err := disk.Usage(p)
		if err != nil {
			continue
		}
		out = append(out, DiskPathUsage{
			Path: p, Total: usage.Total, Used: usage.Used, Free: usage.Free,
			UsedPercent: usage.UsedPercent,
			InodesTotal: usage.InodesTotal, InodesFree: usage.InodesFree, InodesUsed: usage.InodesUsed,
		})
	}
	return out
}

func diskMonitorPath(p string) string {
	p = filepath.Clean(strings.TrimSpace(p))
	if p == "" || p == "." {
		return ""
	}
	info, err := os.Stat(p)
	if err == nil && !info.IsDir() {
		return filepath.Dir(p)
	}
	return p
}
