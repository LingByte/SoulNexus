// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type fwdLeg struct {
	sub   *Peer
	snd   *webrtc.RTPSender
	local *webrtc.TrackLocalStaticRTP
}

// upstreamFanout reads one remote track and clones RTP to many local forward tracks.
type upstreamFanout struct {
	track *webrtc.TrackRemote
	mu    sync.Mutex
	legs  []fwdLeg
	once  sync.Once
}

func (u *upstreamFanout) addLeg(sub *Peer, snd *webrtc.RTPSender, local *webrtc.TrackLocalStaticRTP) {
	u.mu.Lock()
	u.legs = append(u.legs, fwdLeg{sub: sub, snd: snd, local: local})
	u.mu.Unlock()
	u.once.Do(func() { go u.run() })
}

func (u *upstreamFanout) run() {
	for {
		pkt, _, err := u.track.ReadRTP()
		if err != nil {
			return
		}
		u.mu.Lock()
		legs := append([]fwdLeg(nil), u.legs...)
		u.mu.Unlock()
		for _, lg := range legs {
			cp := rtpPacketClone(pkt)
			if cp == nil {
				continue
			}
			_ = lg.local.WriteRTP(cp)
		}
	}
}

// Close removes forwarded legs from subscribers (production cleanup).
func (u *upstreamFanout) Close() {
	u.mu.Lock()
	legs := u.legs
	u.legs = nil
	u.mu.Unlock()
	for _, lg := range legs {
		_ = lg.sub.pc.RemoveTrack(lg.snd)
		lg.sub.scheduleRenegotiate()
	}
	metricUpstreamClosed.Inc()
}

func rtpPacketClone(p *rtp.Packet) *rtp.Packet {
	if p == nil {
		return nil
	}
	raw, err := p.Marshal()
	if err != nil {
		return nil
	}
	q := &rtp.Packet{}
	if err := q.Unmarshal(raw); err != nil {
		return nil
	}
	return q
}
