package audutil

// jitter snap 窗口的运行时配置入口。原本在两处硬编码 80ms：
//
//   - pkg/voice/recorder/recorder.go::placePCMTrackBytes（立体声 recorder）
//   - pkg/utils/audutil/wall_pcm.go::placeWallPCMTrack（SOA1→WAV 时间线）
//
// 抽到这里统一通过 RECORDING_JITTER_SNAP_MS 配置，让上线时可以临时调整：
//
//   - 调试某些极端网络抖动（≥80ms 但仍然"同一句话"）可临时放宽至 120ms；
//   - 录音回放仪器化测试场景需要更短窗口（如 30ms）以暴露调度抖动；
//
// 放在 pkg/utils 而不是 pkg/voice/recorder：utils 是项目里通用配置/环境变量的
// 集散地（已有 GetEnv 等），其他包都依赖它；recorder 反向依赖 utils 没有循环风险。

import (
	"strconv"

	"github.com/LingByte/SoulNexus/pkg/utils/common"
)

// RecordingJitterSnapNs 返回 jitter snap 窗口（纳秒）。
// RECORDING_JITTER_SNAP_MS 环境变量可覆盖，但被 clamp 到 [10, 500] ms 区间
// 防止误配置导致的极端行为：
//
//   - <10ms：连一帧调度抖动都吃不下，相当于关闭 snap，必产生周期性 click；
//   - >500ms：会把 0.5s 内的真实停顿当作连续音频拼接，吞掉合理的回合间静音。
//
// 默认 80ms：参考 pkg/voice/recorder/recorder.go::placePCMTrackBytes 的论证 —
// 大于任何单帧调度卡顿（典型 20ms TTS 帧 + 60ms 瞬时 stall），小于真实回合间
// 停顿（≥100ms），不会误吞真静音。
func RecordingJitterSnapNs() int64 {
	const (
		defaultMs = 80
		minMs     = 10
		maxMs     = 500
	)
	raw := common.GetEnv("RECORDING_JITTER_SNAP_MS")
	if raw == "" {
		return int64(defaultMs) * 1_000_000
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < minMs {
		v = minMs
	}
	if v > maxMs {
		v = maxMs
	}
	return int64(v) * 1_000_000
}
