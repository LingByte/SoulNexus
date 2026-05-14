package bridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
)

// CanRawDatagramRelay is true when both legs share the same codec clock/channels so RTP payloads
// can be forwarded bit-transparently (only payload-type remapping in TwoLegPayloadRelay).
// SRTP must match on both legs if used (this helper does not inspect SRTP).
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
	case "g722":
		// Matches pkg/sip/session CallSession wiring (16 kHz PCM path; SDP clock remains 8000).
		return a.Channels == 1 && a.SampleRate == 16000
	case "opus":
		if a.Channels != 1 && a.Channels != 2 {
			return false
		}
		switch a.SampleRate {
		case 8000, 12000, 16000, 24000, 48000:
			return true
		default:
			return false
		}
	case "l16":
		if a.Channels < 1 || a.Channels > 2 {
			return false
		}
		switch a.SampleRate {
		case 8000, 16000, 32000, 48000:
			return true
		default:
			return false
		}
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

// TwoLegPayloadRelay forwards **raw RTP UDP datagrams** between two legs: transparent RTP (preserve
// peer SSRC / sequence number / timestamp); only the 7-bit payload type is remapped when SDP PT
// differs between legs.
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

	recMu                sync.Mutex
	onUserAudio          func(seq uint16, ts uint32, payload []byte)
	onAgentToCallerAudio func(seq uint16, ts uint32, payload []byte)

	startOnce sync.Once
	stopOnce  sync.Once
}

// SetInboundRecording captures RTP audio forwarded during raw relay: user = caller→agent packets;
// agentToCaller = agent→caller packets (written toward the original caller). Optional; nil skips.
func (r *TwoLegPayloadRelay) SetInboundRecording(onUser, onAgentToCaller func(seq uint16, ts uint32, payload []byte)) {
	if r == nil {
		return
	}
	r.recMu.Lock()
	r.onUserAudio = onUser
	r.onAgentToCallerAudio = onAgentToCaller
	r.recMu.Unlock()
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
		return nil, fmt.Errorf("bridge relay: codecs not eligible for raw RTP relay (same pcmu/pcma/g722/opus/l16 required)")
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

// runForward: if fromCaller, read inbound leg → write outbound leg toward last known agent RTP; else agent → caller.
func (r *TwoLegPayloadRelay) runForward(fromCaller bool) {
	src := r.callerSess
	dst := r.agentSess
	if !fromCaller {
		src = r.agentSess
		dst = r.callerSess
	}
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

		curAudioPT := buf[1] & 0x7F
		recordAudio := false
		if fromCaller {
			recordAudio = curAudioPT == (srcAudioPT & 0x7F)
		} else {
			recordAudio = curAudioPT == (srcAudioPT & 0x7F)
		}
		if recordAudio {
			pkt := &rtp.RTPPacket{}
			if err := pkt.Unmarshal(buf[:n]); err == nil && len(pkt.Payload) > 0 {
				p := append([]byte(nil), pkt.Payload...)
				seq := pkt.Header.SequenceNumber
				ts := pkt.Header.Timestamp
				r.recMu.Lock()
				fnU := r.onUserAudio
				fnA := r.onAgentToCallerAudio
				r.recMu.Unlock()
				if fromCaller && fnU != nil {
					fnU(seq, ts, p)
				}
				if !fromCaller && fnA != nil {
					fnA(seq, ts, p)
				}
			}
		}

		if (buf[1] & 0x7F) != (newPT & 0x7F) {
			buf[1] = (buf[1] & 0x80) | (newPT & 0x7F)
		}
		if _, err := dst.Conn.WriteToUDP(buf[:n], dest); err != nil {
			continue
		}
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

