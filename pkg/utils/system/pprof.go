package system

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/shirou/gopsutil/v4/cpu"
)

// cpuPProfThreshold 触发自动 CPU profile 的百分比阈值。
// 受 PerformanceMonitorConfig.CPUThreshold 控制，可在运行时通过 listener 调整。
const defaultCPUPProfThreshold = 80.0

// pprofDir 是 pprof 文件落地目录；运行时下若已存在则复用，不存在时尝试创建。
const pprofDir = "./pprof"

// Monitor 定时监控 CPU 使用率，超过阈值则采样一段 CPU profile。
//
// 设计要点：
//   - 每轮 cpu.Percent 失败不再 panic（曾导致整个监控 goroutine 崩溃，
//     连带带走 SafeGo 调用方）；记 warn 然后下一轮重试。
//   - 阈值实时读取 PerformanceMonitorConfig，运维可通过 listener 调整。
//   - 采样窗口固定 10 s，避免被高 CPU 状态持续触发导致磁盘被刷爆。
//   - 文件以时间戳命名，便于排序；调用方需自行轮换（PProfRetention 留给 ops）。
func Monitor() {
	for {
		cfg := GetPerformanceMonitorConfig()
		if !cfg.Enabled {
			time.Sleep(30 * time.Second)
			continue
		}
		threshold := float64(cfg.CPUThreshold)
		if threshold <= 0 {
			threshold = defaultCPUPProfThreshold
		}

		percent, err := cpu.Percent(time.Second, false)
		if err != nil || len(percent) == 0 {
			logger.Warn("cpu monitor sample failed: " + safeErrMsg(err))
			time.Sleep(30 * time.Second)
			continue
		}
		if percent[0] > threshold {
			logger.Warn(fmt.Sprintf("cpu usage too high: %.2f%% (threshold=%.2f%%) — capturing pprof",
				percent[0], threshold))
			capturePProf()
		}
		time.Sleep(30 * time.Second)
	}
}

func capturePProf() {
	if _, err := os.Stat(pprofDir); os.IsNotExist(err) {
		if err := os.Mkdir(pprofDir, os.ModePerm); err != nil {
			logger.Error("create pprof dir failed: " + err.Error())
			return
		}
	}
	path := fmt.Sprintf("%s/cpu-%s.pprof", pprofDir, time.Now().Format("20060102150405"))
	f, err := os.Create(path)
	if err != nil {
		logger.Error("create pprof file failed: " + err.Error())
		return
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		logger.Error("start pprof failed: " + err.Error())
		return
	}
	time.Sleep(10 * time.Second)
	pprof.StopCPUProfile()
	logger.Info("pprof captured: " + path)
}

func safeErrMsg(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
