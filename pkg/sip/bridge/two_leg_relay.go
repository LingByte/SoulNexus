package bridge

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
)

// relayRewriteRTPHeader enables the legacy SoulNexus behavior: rewrite SSRC, sequence number,
// and timestamp onto each leg's rtp.Session state. The outer SIPServe project (pkg/sip/session
// Manager.startRTPBridge) forwards RTP **bit-transparent** on a single socket — no header rewrite.
// With two UDP sockets, transparent forward + optional PT remap matches that semantics and avoids
// timestamp/seq discontinuities (especially after TTS) that sound like garbled / "underwater" audio.
// Set SIP_TRANSFER_RELAY_REWRITE_RTP=1 only if a peer strictly requires a single SSRC from our side.
func relayRewriteRTPHeader() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("SIP_TRANSFER_RELAY_REWRITE_RTP")))
	return v == "1" || v == "true" || v == "yes"
}

// CanRawDatagramRelay is true when both legs are the same narrowband G.711 (bit-transparent RTP forward).
func CanRawDatagramRelay(a, b media.CodecConfig) bool {
	na := strings.ToLower(strings.TrimSpace(a.Codec))
	nb := strings.ToLower(strings.TrimSpace(b.Codec))
	if na != nb {
		return false
	}
	if a.SampleRate != b.SampleRate || a.Channels != b.Channels {
		return false
	}
	switch na {
	case "pcmu", "pcma":
		return a.SampleRate == 8000 && a.Channels == 1
	default:
		return false
	}
}

func cloneUDPAddr(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	b := *a
	if len(a.IP) > 0 {
		b.IP = append(net.IP(nil), a.IP...)
	}
	return &b
}

// TwoLegPayloadRelay forwards **raw RTP UDP datagrams** between two legs. Default behavior matches
// the outer repo’s pkg/sip/session.Manager.startRTPBridge: **transparent** RTP (preserve SSRC, seq,
// timestamp from the sending peer); only the 7-bit payload type is remapped when the two SDP
// legs use different PT numbers. Optional SIP_TRANSFER_RELAY_REWRITE_RTP=1 restores SSRC/seq/ts
// rewriting onto each leg's rtp.Session (legacy).
type TwoLegPayloadRelay struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	callerSess *rtp.Session
	agentSess  *rtp.Session

	callerPT   uint8
	agentPT    uint8
	callerDTMF uint8
	agentDTMF  uint8

	// Baseline from SDP at Start(); refreshed from each ReadFromUDP (symmetric RTP), like legacy startRTPBridge addr learning.
	mu sync.Mutex
	lastCallerRTP *net.UDPAddr
	lastAgentRTP  *net.UDPAddr

	// Per-direction timestamp mapping: preserve source RTP clock deltas on our outbound Session clock.
	clockCToA rtpRelayClock
	clockAToC rtpRelayClock

	startOnce sync.Once
	stopOnce  sync.Once
}

type rtpRelayClock struct {
	mu          sync.Mutex
	initialized bool
	lastSrcTS   uint32
	lastOutTS   uint32
}

// nextTimestamp maps the next source RTP timestamp to a value continuous with dst.Timestamp.
func (c *rtpRelayClock) nextTimestamp(dst *rtp.Session, srcTS uint32) uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.initialized {
		c.lastSrcTS = srcTS
		c.lastOutTS = dst.Timestamp
		c.initialized = true
		return c.lastOutTS
	}
	delta := srcTS - c.lastSrcTS
	c.lastOutTS += delta
	c.lastSrcTS = srcTS
	return c.lastOutTS
}

// NewTwoLegPayloadRelay builds a raw-datagram relay; both sessions must already have RemoteAddr (SDP or learned RTP).
func NewTwoLegPayloadRelay(
	callerSess, agentSess *rtp.Session,
	callerCodec, agentCodec media.CodecConfig,
	callerDTMF, agentDTMF uint8,
) (*TwoLegPayloadRelay, error) {
	if callerSess == nil || agentSess == nil {
		return nil, fmt.Errorf("bridge relay: nil session")
	}
	if !CanRawDatagramRelay(callerCodec, agentCodec) {
		return nil, fmt.Errorf("bridge relay: codecs not eligible for raw RTP relay (need same narrowband PCMU/PCMA)")
	}
	if callerSess.RemoteAddr == nil || agentSess.RemoteAddr == nil {
		return nil, fmt.Errorf("bridge relay: remote RTP address not set on both legs")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &TwoLegPayloadRelay{
		ctx:        ctx,
		cancel:     cancel,
		callerSess: callerSess,
		agentSess:  agentSess,
		callerPT:   callerCodec.PayloadType,
		agentPT:    agentCodec.PayloadType,
		callerDTMF: callerDTMF,
		agentDTMF:  agentDTMF,
	}, nil
}

func (r *TwoLegPayloadRelay) Start() {
	if r == nil {
		return
	}
	r.startOnce.Do(func() {
		r.lastCallerRTP = cloneUDPAddr(r.callerSess.RemoteAddr)
		r.lastAgentRTP = cloneUDPAddr(r.agentSess.RemoteAddr)
		if r.lastCallerRTP == nil || r.lastAgentRTP == nil {
			return
		}
		r.wg.Add(2)
		go func() { defer r.wg.Done(); r.runForward(true) }()
		go func() { defer r.wg.Done(); r.runForward(false) }()
	})
}

func (r *TwoLegPayloadRelay) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		r.cancel()
		unblockUDPRead(r.callerSess)
		unblockUDPRead(r.agentSess)
		r.wg.Wait()
	})
}

func unblockUDPRead(s *rtp.Session) {
	if s == nil || s.Conn == nil {
		return
	}
	_ = s.Conn.SetReadDeadline(time.Now())
}

// rtpInPlaceHeaderOK is true for the common SIP case: RTP v2, no CSRC, no header extension, no padding.
// Rewriting only the first 12 bytes preserves the UDP payload bit-for-bit after the header, which
// avoids Marshal edge cases that can shift Opus data and sound like static/hiss.
func rtpInPlaceHeaderOK(buf []byte, n int) bool {
	if n < 12 {
		return false
	}
	if buf[0]>>6 != 2 {
		return false
	}
	if buf[0]&0x0F != 0 { // CSRC count
		return false
	}
	if buf[0]&0x10 != 0 { // extension
		return false
	}
	if buf[0]&0x20 != 0 { // padding — payload length needs trim; use parse path
		return false
	}
	return true
}

// runForward: if fromCaller, read inbound leg → write outbound leg toward last known agent RTP; else agent → caller.
func (r *TwoLegPayloadRelay) runForward(fromCaller bool) {
	src := r.callerSess
	dst := r.agentSess
	clock := &r.clockCToA
	if !fromCaller {
		src = r.agentSess
		dst = r.callerSess
		clock = &r.clockAToC
	}
	rewriteHdr := relayRewriteRTPHeader()
	if src == nil || dst == nil || src.Conn == nil || dst.Conn == nil {
		return
	}
	buf := make([]byte, 4096)
	for {
		if r.ctx.Err() != nil {
			return
		}
		_ = src.Conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		n, from, err := src.Conn.ReadFromUDP(buf)
		if err != nil {
			if r.ctx.Err() != nil {
				return
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			continue
		}
		if n < 12 {
			continue
		}
		if buf[0]>>6 != 2 {
			continue
		}
		if pt := buf[1] & 0x7F; pt >= 192 && pt <= 223 {
			continue
		}

		var srcAudioPT, srcDTMF, dstAudioPT, dstDTMF uint8
		var dest *net.UDPAddr
		if fromCaller {
			srcAudioPT, srcDTMF = r.callerPT, r.callerDTMF
			dstAudioPT, dstDTMF = r.agentPT, r.agentDTMF
			r.mu.Lock()
			if from != nil {
				r.lastCallerRTP = cloneUDPAddr(from)
			}
			dest = r.lastAgentRTP
			r.mu.Unlock()
		} else {
			srcAudioPT, srcDTMF = r.agentPT, r.agentDTMF
			dstAudioPT, dstDTMF = r.callerPT, r.callerDTMF
			r.mu.Lock()
			if from != nil {
				r.lastAgentRTP = cloneUDPAddr(from)
			}
			dest = r.lastCallerRTP
			r.mu.Unlock()
		}
		if dest == nil {
			continue
		}

		newPT, ok := mapRelayPayloadType(buf[1], srcAudioPT, srcDTMF, dstAudioPT, dstDTMF)
		if !ok {
			continue
		}

		if !rewriteHdr {
			// Outer-repo style: same RTP header as received (peer's SSRC / seq / timestamp), optional PT nibble only.
			if (buf[1] & 0x7F) != (newPT & 0x7F) {
				buf[1] = (buf[1] & 0x80) | (newPT & 0x7F)
			}
			if _, err := dst.Conn.WriteToUDP(buf[:n], dest); err != nil {
				continue
			}
			continue
		}

		if rtpInPlaceHeaderOK(buf, n) {
			srcTS := binary.BigEndian.Uint32(buf[4:8])
			outTS := clock.nextTimestamp(dst, srcTS)
			buf[1] = (buf[1] & 0x80) | (newPT & 0x7F)
			binary.BigEndian.PutUint16(buf[2:4], dst.SeqNum)
			binary.BigEndian.PutUint32(buf[4:8], outTS)
			binary.BigEndian.PutUint32(buf[8:12], dst.SSRC)
			if _, err := dst.Conn.WriteToUDP(buf[:n], dest); err != nil {
				continue
			}
			dst.SeqNum++
			dst.Timestamp = outTS
			continue
		}

		pkt := &rtp.RTPPacket{}
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}
		outTS := clock.nextTimestamp(dst, pkt.Header.Timestamp)
		out := rtp.RTPPacket{
			Header: rtp.RTPHeader{
				Version:        pkt.Header.Version,
				Padding:        false,
				Extension:      pkt.Header.Extension,
				CSRCCount:      pkt.Header.CSRCCount,
				Marker:         pkt.Header.Marker,
				PayloadType:    newPT,
				SequenceNumber: dst.SeqNum,
				Timestamp:      outTS,
				SSRC:           dst.SSRC,
			},
			CSRC:             append([]uint32(nil), pkt.CSRC...),
			ExtensionProfile: pkt.ExtensionProfile,
			ExtensionPayload: append([]byte(nil), pkt.ExtensionPayload...),
			Payload:          append([]byte(nil), pkt.Payload...),
		}
		outData, err := out.Marshal()
		if err != nil {
			continue
		}
		if _, err := dst.Conn.WriteToUDP(outData, dest); err != nil {
			continue
		}
		dst.SeqNum++
		dst.Timestamp = outTS
	}
}

// mapRelayPayloadType maps negotiated audio / telephone-event PT across legs.
// Unknown payload types must not be forwarded: the peer leg's PT numbering is unrelated to ours
// (e.g. comfort-noise, RED, or dynamic types); relaying them unchanged decodes as garbage/noise.
func mapRelayPayloadType(cur uint8, srcAudioPT, srcDTMF, dstAudioPT, dstDTMF uint8) (newPT uint8, ok bool) {
	cur &= 0x7F
	if cur == srcAudioPT&0x7F {
		return dstAudioPT & 0x7F, true
	}
	if srcDTMF != 0 && cur == srcDTMF&0x7F {
		if dstDTMF == 0 {
			return 0, false
		}
		return dstDTMF & 0x7F, true
	}
	return 0, false
}

