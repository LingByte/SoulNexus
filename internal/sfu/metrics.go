// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricRoomsCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "rtcsfu",
		Name:      "rooms_current",
		Help:      "Number of SFU rooms currently allocated in this process.",
	})
	metricPeersCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "rtcsfu",
		Name:      "peers_current",
		Help:      "Number of SFU signaling peers currently connected.",
	})
	metricPeerJoins = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "rtcsfu",
		Name:      "peer_joins_total",
		Help:      "SFU WebSocket sessions accepted.",
	})
	metricPeerRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "rtcsfu",
		Name:      "peer_rejected_total",
		Help:      "SFU sessions rejected before upgrade.",
	}, []string{"reason"})
	metricUpstreamClosed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "rtcsfu",
		Name:      "upstream_tracks_closed_total",
		Help:      "Publisher upstream fan-outs closed (disconnect or cleanup).",
	})
)

// RecordPeerRejected increments the rejected counter (capacity / auth / upgrade).
func RecordPeerRejected(reason string) {
	metricPeerRejected.WithLabelValues(reason).Inc()
}
