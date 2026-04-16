// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

// startSubscriberRTCPRelay forwards video feedback (PLI / FIR) from a subscriber's outbound
// sender to the publisher PeerConnection so the browser can emit a keyframe — required for
// decoded video after SSRC-rewriting SFU paths.
func startSubscriberRTCPRelay(sender *webrtc.RTPSender, publisher *Peer, upstream *webrtc.TrackRemote) {
	if sender == nil || publisher == nil || upstream == nil {
		return
	}
	srcSSRC := uint32(upstream.SSRC())
	go func() {
		for {
			pkts, _, err := sender.ReadRTCP()
			if err != nil {
				return
			}
			var out []rtcp.Packet
			for _, pkt := range pkts {
				switch pkt.(type) {
				case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
					out = append(out, &rtcp.PictureLossIndication{MediaSSRC: srcSSRC})
				}
			}
			if len(out) > 0 {
				_ = publisher.pc.WriteRTCP(out)
			}
		}
	}()
}

func burstKeyframeRequestsToPublisher(publisher *Peer, upstream *webrtc.TrackRemote, rounds int) {
	if publisher == nil || upstream == nil {
		return
	}
	go func() {
		for i := 0; i < rounds; i++ {
			time.Sleep(time.Duration(60+140*i) * time.Millisecond)
			ssrc := uint32(upstream.SSRC())
			if ssrc == 0 {
				continue
			}
			_ = publisher.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}})
		}
	}()
}
