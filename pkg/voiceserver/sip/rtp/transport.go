package rtp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/dtmf"
)

// DefaultJitterPlaybackDelay is the default inbound playout delay (reorder + smoothing).
const DefaultJitterPlaybackDelay = 80 * time.Millisecond

const maxTelephoneEventPayload = 24

// jbSlotCount is the reorder window (slots); two sequences that differ by a multiple of alias to the same slot — keep this comfortably above max reorder depth.
const jbSlotCount = 2048

const (
	jbMaxPacketAge = 600 * time.Millisecond
	jbDelayFloor   = 40 * time.Millisecond
	jbDelayRaise   = 15 * time.Millisecond
	jbDelayLower   = 4 * time.Millisecond
	jbPLCMaxBytes  = 1200
)

type jbHeld struct {
	pkt *RTPPacket
	at  time.Time
}

type jbSlot struct {
	valid bool
	seq   uint16
	held  jbHeld
}

// SIPRTPTransport adapts an RTP Session to the media.MediaTransport interface.
//
// It is direction-aware:
//   - direction == media.DirectionInput : Next() reads from UDP & returns AudioPacket, Send() is a no-op
//   - direction == media.DirectionOutput: Send() writes AudioPacket as RTP, Next() is a no-op
type SIPRTPTransport struct {
	sess      *Session
	codec     media.CodecConfig
	direction string

	// telephoneEventPT is the negotiated RTP PT for RFC 2833 (often 101); 0 disables DTMF handling.
	telephoneEventPT uint8

	// PreserveSessionOnClose, if true, Close() does not close the underlying RTP UDP socket.
	// Used when stopping the default MediaSession and handing media to an in-process bridge.
	PreserveSessionOnClose bool

	// OnInputPayload, if set, receives a copy of each incoming RTP payload before DTMF heuristics.
	OnInputPayload func([]byte)
	// OnInputRTP, if set, receives RTP timing metadata + payload copy for recording/reconstruction.
	OnInputRTP func(seq uint16, ts uint32, payload []byte)
	// OnOutputPayload, if set, receives a copy of each outgoing encoded audio RTP payload (output transport only).
	OnOutputPayload func([]byte)
	// OnOutputRTP, if set, receives RTP timing metadata + payload copy for recording/reconstruction.
	// seq/ts correspond to the packet values before Session.SendRTP increments counters.
	OnOutputRTP func(seq uint16, ts uint32, payload []byte)

	// JitterPlaybackDelay, if > 0 on DirectionInput, delays playout and absorbs reorder/jitter (see DefaultJitterPlaybackDelay).
	JitterPlaybackDelay time.Duration

	jbSlots         [jbSlotCount]jbSlot
	jbNext          uint16
	jbStarted       bool
	jbLossWait      time.Time
	jbHoleSkips     int
	jbAdaptiveDelay time.Duration
	jbLastPLC       []byte
	readBuf         []byte

	attached *media.MediaSession
}

// NewSIPRTPTransport creates a new SIPRTPTransport.
//
// codec describes the negotiated RTP codec (from SDP), including sample rate,
// channels, bit depth and payload type.
// telephoneEventPT is the RTP payload type for telephone-event (RFC 2833); use 0 if not negotiated.
func NewSIPRTPTransport(sess *Session, codec media.CodecConfig, direction string, telephoneEventPT uint8) *SIPRTPTransport {
	return &SIPRTPTransport{
		sess:             sess,
		codec:            codec,
		direction:        direction,
		telephoneEventPT: telephoneEventPT,
	}
}

func cloneRTPPacketForJitter(p *RTPPacket) *RTPPacket {
	if p == nil {
		return nil
	}
	out := &RTPPacket{
		Header:           p.Header,
		ExtensionProfile: p.ExtensionProfile,
	}
	if len(p.CSRC) > 0 {
		out.CSRC = append([]uint32(nil), p.CSRC...)
	}
	if len(p.ExtensionPayload) > 0 {
		out.ExtensionPayload = append([]byte(nil), p.ExtensionPayload...)
	}
	if len(p.Payload) > 0 {
		out.Payload = append([]byte(nil), p.Payload...)
	}
	return out
}

func (t *SIPRTPTransport) jbEnsureAdaptiveBaseline() {
	if t == nil {
		return
	}
	if t.jbAdaptiveDelay <= 0 {
		if t.JitterPlaybackDelay > 0 {
			t.jbAdaptiveDelay = t.JitterPlaybackDelay
		} else {
			t.jbAdaptiveDelay = DefaultJitterPlaybackDelay
		}
	}
}

func (t *SIPRTPTransport) jbEffectiveDelay() time.Duration {
	if t == nil {
		return DefaultJitterPlaybackDelay
	}
	t.jbEnsureAdaptiveBaseline()
	base := t.JitterPlaybackDelay
	if base <= 0 {
		base = DefaultJitterPlaybackDelay
	}
	d := t.jbAdaptiveDelay
	if d < jbDelayFloor {
		d = jbDelayFloor
	}
	maxD := base + 200*time.Millisecond
	if d > maxD {
		d = maxD
	}
	return d
}

func (t *SIPRTPTransport) jbNoteHole() {
	if t == nil {
		return
	}
	t.jbEnsureAdaptiveBaseline()
	t.jbAdaptiveDelay += jbDelayRaise
}

func (t *SIPRTPTransport) jbNoteGoodPop() {
	if t == nil {
		return
	}
	t.jbEnsureAdaptiveBaseline()
	if t.jbAdaptiveDelay > jbDelayFloor+jbDelayLower {
		t.jbAdaptiveDelay -= jbDelayLower
	}
}

func (t *SIPRTPTransport) jbExpireStale(now time.Time) {
	if t == nil {
		return
	}
	for i := range t.jbSlots {
		sl := &t.jbSlots[i]
		if !sl.valid {
			continue
		}
		if now.Sub(sl.held.at) > jbMaxPacketAge {
			sl.valid = false
		}
	}
}

func (t *SIPRTPTransport) jbRememberPLC(payload []byte) {
	if t == nil || len(payload) == 0 {
		return
	}
	n := len(payload)
	if n > jbPLCMaxBytes {
		n = jbPLCMaxBytes
	}
	if cap(t.jbLastPLC) < n {
		t.jbLastPLC = make([]byte, n)
	} else {
		t.jbLastPLC = t.jbLastPLC[:n]
	}
	copy(t.jbLastPLC, payload[:n])
}

func (t *SIPRTPTransport) jbPLCReplacement() []byte {
	if t == nil || len(t.jbLastPLC) == 0 {
		return nil
	}
	c := strings.ToLower(strings.TrimSpace(t.codec.Codec))
	switch c {
	case "pcmu", "g711u", "ulaw", "pcma", "g711a", "alaw":
		out := make([]byte, len(t.jbLastPLC))
		copy(out, t.jbLastPLC)
		return out
	default:
		return nil
	}
}

func (t *SIPRTPTransport) jbPush(pkt *RTPPacket, now time.Time) {
	if t == nil || pkt == nil || t.JitterPlaybackDelay <= 0 {
		return
	}
	t.jbEnsureAdaptiveBaseline()
	seq := pkt.Header.SequenceNumber
	t.jbExpireStale(now)

	idx := int(seq) % jbSlotCount
	slot := &t.jbSlots[idx]
	if slot.valid && slot.seq != seq {
		// Index collision: cannot disambiguate without a map; drop new packet (rare if jbSlotCount is large).
		return
	}
	if slot.valid && slot.seq == seq {
		return
	}
	slot.valid = true
	slot.seq = seq
	slot.held = jbHeld{pkt: cloneRTPPacketForJitter(pkt), at: now}
	if !t.jbStarted {
		t.jbNext = seq
		t.jbStarted = true
	}
}

func (t *SIPRTPTransport) jbTryPop(now time.Time) *RTPPacket {
	if t == nil || t.JitterPlaybackDelay <= 0 || !t.jbStarted {
		return nil
	}
	const lossSkipAfter = 120 * time.Millisecond
	const maxHoleSkips = 64

	delay := t.jbEffectiveDelay()
	for {
		idx := int(t.jbNext) % jbSlotCount
		slot := &t.jbSlots[idx]
		if slot.valid && slot.seq == t.jbNext {
			if now.Sub(slot.held.at) >= delay {
				slot.valid = false
				pkt := slot.held.pkt
				t.jbNext++
				t.jbLossWait = time.Time{}
				t.jbHoleSkips = 0
				t.jbNoteGoodPop()
				if pkt != nil && len(pkt.Payload) > 0 {
					t.jbRememberPLC(pkt.Payload)
				}
				return pkt
			}
			return nil
		}
		if t.jbLossWait.IsZero() {
			t.jbLossWait = now
		} else if now.Sub(t.jbLossWait) >= lossSkipAfter && t.jbHoleSkips < maxHoleSkips {
			t.jbNext++
			t.jbHoleSkips++
			t.jbLossWait = now
			t.jbNoteHole()
			if plc := t.jbPLCReplacement(); len(plc) > 0 {
				return &RTPPacket{Payload: plc}
			}
			continue
		}
		return nil
	}
}

func (t *SIPRTPTransport) jbReset() {
	if t == nil {
		return
	}
	for i := range t.jbSlots {
		t.jbSlots[i].valid = false
	}
	t.jbStarted = false
	t.jbLossWait = time.Time{}
	t.jbHoleSkips = 0
	t.jbAdaptiveDelay = 0
	t.jbLastPLC = nil
}

func (t *SIPRTPTransport) inputCallbacks(pkt *RTPPacket) {
	if t == nil || pkt == nil || len(pkt.Payload) == 0 {
		return
	}
	if t.OnInputPayload != nil {
		cp := make([]byte, len(pkt.Payload))
		copy(cp, pkt.Payload)
		t.OnInputPayload(cp)
	}
	if t.OnInputRTP != nil {
		cp := make([]byte, len(pkt.Payload))
		copy(cp, pkt.Payload)
		t.OnInputRTP(pkt.Header.SequenceNumber, pkt.Header.Timestamp, cp)
	}
}

func (t *SIPRTPTransport) String() string {
	return fmt.Sprintf("SIPRTPTransport{dir=%s, codec=%s, local=%v, remote=%v}",
		t.direction, t.codec.String(), addrString(t.sessLocalAddr()), addrString(t.sessRemoteAddr()))
}

func (t *SIPRTPTransport) sessLocalAddr() *net.UDPAddr {
	if t == nil || t.sess == nil {
		return nil
	}
	return t.sess.LocalAddr
}

func (t *SIPRTPTransport) sessRemoteAddr() *net.UDPAddr {
	if t == nil || t.sess == nil {
		return nil
	}
	return t.sess.RemoteAddr
}

// Attach is called by MediaSession when the transport is registered.
func (t *SIPRTPTransport) Attach(s *media.MediaSession) {
	t.attached = s
}

// Next reads one RTP packet from the underlying Session and converts it
// to a media.AudioPacket for the input direction. For output transports it
// returns (nil, nil).
func (t *SIPRTPTransport) Next(ctx context.Context) (media.MediaPacket, error) {
	// Output transports don't provide incoming packets.
	if t.direction == media.DirectionOutput {
		return nil, nil
	}

	if t.sess == nil {
		return nil, fmt.Errorf("siprtp: nil session")
	}

	// If the media session is shutting down, avoid returning errors that would
	// be published into EventBus after it is closed.
	if ctx != nil && ctx.Err() != nil {
		t.clearReadDeadline()
		t.jbReset()
		return nil, nil
	}

	if cap(t.readBuf) < 2048 {
		t.readBuf = make([]byte, 2048)
	}
	buf := t.readBuf[:2048]
	for {
		// If the media session is shutting down, stop waiting.
		if ctx != nil && ctx.Err() != nil {
			t.clearReadDeadline()
			t.jbReset()
			return nil, nil
		}

		// Bounded wait so bridge teardown (cancel + WakeupRead) and PCM direct loops can exit;
		// also avoids relying on EventBus queue depth for real-time audio.
		if t.sess.Conn != nil {
			_ = t.sess.Conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		}

		jitter := t.JitterPlaybackDelay > 0 && t.direction == media.DirectionInput
		if jitter {
			if pkt := t.jbTryPop(time.Now()); pkt != nil {
				t.clearReadDeadline()
				return &media.AudioPacket{Payload: pkt.Payload}, nil
			}
		}

		n, _, pkt, err := t.sess.ReceiveRTP(buf)
		if err != nil {
			if errors.Is(err, ErrRTPDiscard) {
				continue
			}
			if ctx != nil && ctx.Err() != nil {
				t.clearReadDeadline()
				t.jbReset()
				return nil, nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			t.clearReadDeadline()
			if jitter {
				t.jbReset()
			}
			return nil, err
		}
		if pkt == nil {
			if n == 0 {
				t.clearReadDeadline()
				return nil, nil
			}
			t.clearReadDeadline()
			if jitter {
				t.jbReset()
			}
			return nil, fmt.Errorf("siprtp: got nil packet from ReceiveRTP")
		}

		// Recording / taps: always run first (large payloads on telephone-event PT must not be skipped).
		t.inputCallbacks(pkt)

		if jitter {
			if pt := pkt.Header.PayloadType & 0x7F; pt >= 192 && pt <= 223 {
				continue
			}
			if t.telephoneEventPT != 0 && pkt.Header.PayloadType == t.telephoneEventPT && len(pkt.Payload) <= maxTelephoneEventPayload {
				digit, end, ok := dtmf.EventFromRFC2833(pkt.Payload)
				if ok && end && digit != "" {
					t.clearReadDeadline()
					return &media.DTMFPacket{Digit: digit, End: end}, nil
				}
				continue
			}
			if t.codec.PayloadType != 0 && pkt.Header.PayloadType != t.codec.PayloadType {
				continue
			}
			t.jbPush(pkt, time.Now())
			continue
		}

		// RFC 2833 telephone-event — only when payload is short (real event frames are a few octets; audio is much larger).
		if t.telephoneEventPT != 0 && pkt.Header.PayloadType == t.telephoneEventPT && len(pkt.Payload) <= maxTelephoneEventPayload {
			digit, end, ok := dtmf.EventFromRFC2833(pkt.Payload)
			if ok && end && digit != "" {
				t.clearReadDeadline()
				return &media.DTMFPacket{Digit: digit, End: end}, nil
			}
			if ok && !end {
				continue
			}
		}

		// Only decode the negotiated audio RTP payload type into PCM for ASR/media.
		if t.codec.PayloadType != 0 && pkt.Header.PayloadType != t.codec.PayloadType {
			continue
		}

		t.clearReadDeadline()
		return &media.AudioPacket{Payload: pkt.Payload}, nil
	}
}

func (t *SIPRTPTransport) clearReadDeadline() {
	if t == nil || t.sess == nil || t.sess.Conn == nil {
		return
	}
	_ = t.sess.Conn.SetReadDeadline(time.Time{})
}

// WakeupRead unblocks a goroutine stuck in Next() (same idea as transfer RTP relay stop).
func (t *SIPRTPTransport) WakeupRead() {
	if t == nil || t.sess == nil || t.sess.Conn == nil {
		return
	}
	_ = t.sess.Conn.SetReadDeadline(time.Now())
}

// Send sends a media.AudioPacket as a single RTP packet for the output direction.
// For input transports it is a no-op.
func (t *SIPRTPTransport) Send(ctx context.Context, packet media.MediaPacket) (int, error) {
	// Input transports don't send outgoing packets.
	if t.direction == media.DirectionInput {
		return 0, nil
	}

	if t.sess == nil {
		return 0, fmt.Errorf("siprtp: nil session")
	}

	audio, ok := packet.(*media.AudioPacket)
	if !ok {
		// Ignore non-audio media packets at this transport level.
		return 0, nil
	}

	payload := audio.Payload
	if len(payload) == 0 {
		return 0, nil
	}

	if t.OnOutputPayload != nil {
		cp := make([]byte, len(payload))
		copy(cp, payload)
		t.OnOutputPayload(cp)
	}
	if t.OnOutputRTP != nil {
		cp := make([]byte, len(payload))
		copy(cp, payload)
		t.OnOutputRTP(t.sess.SeqNum, t.sess.Timestamp, cp)
	}

	// RTP timestamp increment must be based on codec clock rate, not payload bytes.
	// For codecs like OPUS (variable bitrate), deriving samples from payload length
	// causes timestamp drift and audible artifacts (noise/choppiness).
	samples := audio.RTPSamples
	if samples == 0 {
		clockRate := t.codec.SampleRate
		if strings.EqualFold(strings.TrimSpace(t.codec.Codec), "g722") {
			// G.722 SDP clock is 8000 Hz even though PCM is 16 kHz (RFC 3551).
			clockRate = 8000
		}
		if clockRate > 0 {
			if t.codec.FrameDuration != "" {
				if d, err := time.ParseDuration(t.codec.FrameDuration); err == nil && d > 0 {
					samples = uint32((int64(clockRate) * d.Milliseconds()) / 1000)
				}
			}
			// Default to 20ms frames if not specified/parsable.
			if samples == 0 {
				samples = uint32((clockRate * 20) / 1000)
			}
		}
		if samples == 0 {
			// Fallback: approximate from raw PCM payload size (works for 8-bit PCMU/PCMA).
			bytesPerSample := (t.codec.BitDepth / 8) * t.codec.Channels
			if bytesPerSample <= 0 {
				bytesPerSample = 2
			}
			samples = uint32(len(payload) / bytesPerSample)
			if samples == 0 {
				samples = 1
			}
		}
	}

	if err := t.sess.SendRTP(payload, t.codec.PayloadType, samples); err != nil {
		return 0, err
	}

	return len(payload), nil
}

// Codec returns the negotiated codec configuration.
func (t *SIPRTPTransport) Codec() media.CodecConfig {
	return t.codec
}

// Close closes the underlying RTP session.
func (t *SIPRTPTransport) Close() error {
	if t == nil || t.sess == nil {
		return nil
	}
	if t.PreserveSessionOnClose {
		return nil
	}
	return t.sess.Close()
}

func addrString(addr *net.UDPAddr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}
