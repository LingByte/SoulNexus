package sip

import (
	"sync"
	"time"
)

// QoSStats 通话质量统计
type QoSStats struct {
	CallID    string    `json:"callId"`
	Timestamp time.Time `json:"timestamp"`

	// 延迟统计
	RTT         float64 `json:"rtt"`         // 往返时延（ms）
	OneWayDelay float64 `json:"oneWayDelay"` // 单向延迟（ms）

	// 丢包统计
	PacketLoss      float64 `json:"packetLoss"`      // 丢包率（%）
	PacketsSent     uint64  `json:"packetsSent"`     // 发送包数
	PacketsReceived uint64  `json:"packetsReceived"` // 接收包数
	PacketsLost     uint64  `json:"packetsLost"`     // 丢失包数

	// 抖动统计
	Jitter       float64 `json:"jitter"`       // 抖动（ms）
	JitterBuffer int     `json:"jitterBuffer"` // 抖动缓冲区大小

	// 带宽统计
	BandwidthUsed uint64 `json:"bandwidthUsed"` // 已使用带宽（bps）
	BandwidthMax  uint64 `json:"bandwidthMax"`  // 最大带宽（bps）

	// 音频质量
	MOS     float64 `json:"mos"`     // 平均意见分（1-5）
	Codec   string  `json:"codec"`   // 编解码器
	Bitrate uint32  `json:"bitrate"` // 码率（bps）

	// RTCP统计
	RTCPPackets  uint64    `json:"rtcpPackets"`  // RTCP包数
	LastRTCPTime time.Time `json:"lastRtcpTime"` // 最后RTCP时间
}

// QoSMonitor QoS监控器
type QoSMonitor struct {
	stats      map[string]*QoSStats // CallID -> Stats
	statsMutex sync.RWMutex

	// 配置
	monitoringInterval time.Duration
	rtcpInterval       time.Duration
}

// NewQoSMonitor 创建QoS监控器
func NewQoSMonitor() *QoSMonitor {
	return &QoSMonitor{
		stats:              make(map[string]*QoSStats),
		monitoringInterval: 1 * time.Second,
		rtcpInterval:       5 * time.Second,
	}
}

// StartMonitoring 开始监控指定通话
func (q *QoSMonitor) StartMonitoring(callID string) {
	q.statsMutex.Lock()
	defer q.statsMutex.Unlock()

	if _, exists := q.stats[callID]; !exists {
		q.stats[callID] = &QoSStats{
			CallID:    callID,
			Timestamp: time.Now(),
			Codec:     "PCMU",
			Bitrate:   64000,
		}
	}
}

// StopMonitoring 停止监控指定通话
func (q *QoSMonitor) StopMonitoring(callID string) {
	q.statsMutex.Lock()
	defer q.statsMutex.Unlock()
	delete(q.stats, callID)
}

// UpdateRTPStats 更新RTP统计信息
func (q *QoSMonitor) UpdateRTPStats(callID string, packetsSent, packetsReceived, packetsLost uint64, jitter float64) {
	q.statsMutex.Lock()
	defer q.statsMutex.Unlock()

	stats, exists := q.stats[callID]
	if !exists {
		return
	}

	stats.PacketsSent = packetsSent
	stats.PacketsReceived = packetsReceived
	stats.PacketsLost = packetsLost
	stats.Jitter = jitter
	stats.Timestamp = time.Now()

	// 计算丢包率
	totalPackets := packetsSent + packetsReceived
	if totalPackets > 0 {
		stats.PacketLoss = float64(packetsLost) / float64(totalPackets) * 100.0
	}

	// 计算MOS分数（简化版）
	stats.MOS = q.calculateMOS(stats.PacketLoss, stats.Jitter, stats.RTT)
}

// UpdateDelay 更新延迟统计
func (q *QoSMonitor) UpdateDelay(callID string, rtt, oneWayDelay float64) {
	q.statsMutex.Lock()
	defer q.statsMutex.Unlock()

	stats, exists := q.stats[callID]
	if !exists {
		return
	}

	stats.RTT = rtt
	stats.OneWayDelay = oneWayDelay
	stats.Timestamp = time.Now()
}

// UpdateBandwidth 更新带宽统计
func (q *QoSMonitor) UpdateBandwidth(callID string, used, max uint64) {
	q.statsMutex.Lock()
	defer q.statsMutex.Unlock()

	stats, exists := q.stats[callID]
	if !exists {
		return
	}

	stats.BandwidthUsed = used
	stats.BandwidthMax = max
	stats.Timestamp = time.Now()
}

// GetStats 获取指定通话的统计信息
func (q *QoSMonitor) GetStats(callID string) (*QoSStats, bool) {
	q.statsMutex.RLock()
	defer q.statsMutex.RUnlock()

	stats, exists := q.stats[callID]
	if !exists {
		return nil, false
	}

	// 返回副本
	statsCopy := *stats
	return &statsCopy, true
}

// GetAllStats 获取所有通话的统计信息
func (q *QoSMonitor) GetAllStats() map[string]*QoSStats {
	q.statsMutex.RLock()
	defer q.statsMutex.RUnlock()

	result := make(map[string]*QoSStats)
	for callID, stats := range q.stats {
		statsCopy := *stats
		result[callID] = &statsCopy
	}
	return result
}

// calculateMOS 计算MOS分数（简化版）
// MOS范围：1-5，5为最佳
func (q *QoSMonitor) calculateMOS(packetLoss, jitter, rtt float64) float64 {
	// 基础MOS分数
	mos := 4.5

	// 丢包率影响（每1%丢包降低0.1分）
	if packetLoss > 0 {
		mos -= packetLoss * 0.1
	}

	// 抖动影响（每10ms抖动降低0.05分）
	if jitter > 0 {
		mos -= (jitter / 10.0) * 0.05
	}

	// 延迟影响（每50ms延迟降低0.05分）
	if rtt > 0 {
		mos -= (rtt / 50.0) * 0.05
	}

	// 限制在1-5范围内
	if mos < 1.0 {
		mos = 1.0
	} else if mos > 5.0 {
		mos = 5.0
	}

	return mos
}
