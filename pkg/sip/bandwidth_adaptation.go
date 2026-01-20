package sip

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BandwidthAdaptor 带宽自适应器
type BandwidthAdaptor struct {
	currentBitrate uint32 // 当前码率（bps）
	minBitrate     uint32 // 最小码率
	maxBitrate     uint32 // 最大码率

	// 网络质量指标
	packetLoss float64
	jitter     float64
	rtt        float64

	// 自适应参数
	adaptationInterval time.Duration
	lastAdaptation     time.Time

	mutex sync.RWMutex
}

// NewBandwidthAdaptor 创建带宽自适应器
func NewBandwidthAdaptor() *BandwidthAdaptor {
	return &BandwidthAdaptor{
		currentBitrate:     64000,  // 默认64kbps (G.711)
		minBitrate:         16000,  // 最小16kbps
		maxBitrate:         128000, // 最大128kbps
		adaptationInterval: 5 * time.Second,
		lastAdaptation:     time.Now(),
	}
}

// UpdateNetworkMetrics 更新网络质量指标
func (ba *BandwidthAdaptor) UpdateNetworkMetrics(packetLoss, jitter, rtt float64) {
	ba.mutex.Lock()
	defer ba.mutex.Unlock()

	ba.packetLoss = packetLoss
	ba.jitter = jitter
	ba.rtt = rtt

	// 如果距离上次自适应超过间隔时间，执行自适应
	if time.Since(ba.lastAdaptation) >= ba.adaptationInterval {
		ba.adapt()
		ba.lastAdaptation = time.Now()
	}
}

// adapt 自适应调整码率
func (ba *BandwidthAdaptor) adapt() {
	oldBitrate := ba.currentBitrate

	// 根据网络质量调整码率
	if ba.packetLoss > 5.0 || ba.jitter > 50.0 || ba.rtt > 300.0 {
		// 网络质量差，降低码率
		ba.currentBitrate = ba.currentBitrate * 3 / 4
		if ba.currentBitrate < ba.minBitrate {
			ba.currentBitrate = ba.minBitrate
		}
		logrus.WithFields(logrus.Fields{
			"old_bitrate": oldBitrate,
			"new_bitrate": ba.currentBitrate,
			"packet_loss": ba.packetLoss,
			"jitter":      ba.jitter,
			"rtt":         ba.rtt,
		}).Info("Bitrate decreased due to poor network quality")
	} else if ba.packetLoss < 1.0 && ba.jitter < 20.0 && ba.rtt < 100.0 {
		// 网络质量好，可以尝试提高码率
		ba.currentBitrate = ba.currentBitrate * 5 / 4
		if ba.currentBitrate > ba.maxBitrate {
			ba.currentBitrate = ba.maxBitrate
		}
		logrus.WithFields(logrus.Fields{
			"old_bitrate": oldBitrate,
			"new_bitrate": ba.currentBitrate,
			"packet_loss": ba.packetLoss,
			"jitter":      ba.jitter,
			"rtt":         ba.rtt,
		}).Info("Bitrate increased due to good network quality")
	}
}

// GetCurrentBitrate 获取当前码率
func (ba *BandwidthAdaptor) GetCurrentBitrate() uint32 {
	ba.mutex.RLock()
	defer ba.mutex.RUnlock()
	return ba.currentBitrate
}

// SetBitrateRange 设置码率范围
func (ba *BandwidthAdaptor) SetBitrateRange(min, max uint32) {
	ba.mutex.Lock()
	defer ba.mutex.Unlock()

	ba.minBitrate = min
	ba.maxBitrate = max

	if ba.currentBitrate < min {
		ba.currentBitrate = min
	} else if ba.currentBitrate > max {
		ba.currentBitrate = max
	}
}

// GetCodecForBitrate 根据码率选择合适的编解码器
func (ba *BandwidthAdaptor) GetCodecForBitrate(bitrate uint32) string {
	if bitrate >= 64000 {
		return "PCMU" // G.711 μ-law (64kbps)
	} else if bitrate >= 32000 {
		return "G.726" // G.726 (32kbps)
	} else if bitrate >= 16000 {
		return "G.729" // G.729 (8kbps, 但需要16kbps带宽)
	} else {
		return "G.729" // 最低码率
	}
}
