// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webrtc

// stats.go owns the per-call media-quality telemetry path:
// pion GetStats() → gateway.MediaStatsSample → persister.OnMediaStats →
// `call_media_stats` rows. Split out from session.go so the SDP /
// pipeline lifecycle and the telemetry concerns can be read and
// modified independently.
//
// Behaviour:
//
//   - On ICE Connected/Completed we kick off a single periodic poller
//     (statsPollInterval ticker) — guarded by session.statsOnce so an
//     ICE flap can't spawn duplicates.
//   - The poller exits when session.done is closed (teardown) or when
//     session.closed is observed true at tick time.
//   - On teardown we capture one last sample with Final=true, BEFORE
//     closing the PeerConnection — pion's GetStats returns an empty
//     report after Close, which would corrupt the per-call summary.

import (
	"context"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"

	pionwebrtc "github.com/pion/webrtc/v4"
)

// statsPollInterval is how often the periodic poller pushes a sample.
// 5 s mirrors what most WebRTC dashboards expect and stays under the
// pion stats cache TTL.
const statsPollInterval = 5 * time.Second

// startStatsPoller spawns a background loop that polls pion's
// GetStats() every statsPollInterval and pushes a media-quality sample
// into the persister. The loop exits when the session closes
// (s.done is closed). Idempotent — guarded by s.statsOnce so an ICE
// flap (Connected → Disconnected → Connected) doesn't spawn duplicates.
func (s *session) startStatsPoller() {
	if s.persister == nil {
		return
	}
	s.statsOnce.Do(func() {
		go func() {
			t := time.NewTicker(statsPollInterval)
			defer t.Stop()
			for {
				select {
				case <-s.done:
					return
				case <-t.C:
					if s.closed.Load() {
						return
					}
					s.captureStats(false)
				}
			}
		}()
	})
}

// captureStats reads the current pion stats report, aggregates the
// inbound (browser → us) and outbound (us → browser) RTP streams into a
// single MediaStatsSample, and pushes it into the persister.
//
//   - final=true marks the end-of-call summary; otherwise it's a
//     periodic sample.
//   - Loss rate is computed from PacketsReceived / (PacketsReceived +
//     PacketsLost). We don't trust pion's own loss field because it
//     can spike during the first few packets when the receiver hasn't
//     seen enough sequence numbers to anchor.
//   - Round-trip time comes from RemoteInboundRTPStreamStats — that's
//     the peer's view of OUR outbound stream, which is where the
//     RTCP RR with delay-since-last-SR lives.
func (s *session) captureStats(final bool) {
	if s.pc == nil || s.persister == nil {
		return
	}
	report := s.pc.GetStats()
	sample := gateway.MediaStatsSample{
		At:         time.Now().UTC(),
		Final:      final,
		Codec:      "opus",
		ClockRate:  48000,
		Channels:   2,
		RemoteAddr: s.clientMeta,
	}
	for _, raw := range report {
		switch v := raw.(type) {
		case pionwebrtc.InboundRTPStreamStats:
			if v.Kind != "audio" {
				continue
			}
			sample.PacketsReceived += uint64(v.PacketsReceived)
			sample.BytesReceived += v.BytesReceived
			if v.PacketsLost > 0 {
				sample.PacketsLost += uint64(v.PacketsLost)
			}
			sample.NACKsSent += uint64(v.NACKCount) // we asked the peer to retransmit
			if v.Jitter > 0 {
				// pion reports jitter in seconds.
				sample.JitterMs = int(v.Jitter * 1000)
			}
		case pionwebrtc.OutboundRTPStreamStats:
			if v.Kind != "audio" {
				continue
			}
			sample.PacketsSent += uint64(v.PacketsSent)
			sample.BytesSent += v.BytesSent
			sample.NACKsReceived += uint64(v.NACKCount) // peer asked us to retransmit
		case pionwebrtc.RemoteInboundRTPStreamStats:
			// RTT is reported on the remote-inbound view of our outbound
			// stream. RoundTripTime is in seconds.
			if v.RoundTripTime > 0 {
				sample.RTTMs = int(v.RoundTripTime * 1000)
			}
		}
	}
	if sample.PacketsReceived > 0 && sample.PacketsLost > 0 {
		sample.LossRate = float64(sample.PacketsLost) /
			float64(sample.PacketsReceived+sample.PacketsLost)
	}
	s.persister.OnMediaStats(context.Background(), sample)
}
