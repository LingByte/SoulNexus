package rtp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/sip/dtmf"
)

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
		return nil, nil
	}

	buf := make([]byte, 1500) // enough for typical RTP over UDP
	for {
		// If the media session is shutting down, stop waiting.
		if ctx != nil && ctx.Err() != nil {
			return nil, nil
		}

		n, _, pkt, err := t.sess.ReceiveRTP(buf)
		if err != nil {
			// When shutdown triggers UDP close, suppress the error to avoid publishing
			// into a closed EventBus.
			if ctx != nil && ctx.Err() != nil {
				return nil, nil
			}
			return nil, err
		}
		if pkt == nil {
			if n == 0 {
				return nil, nil
			}
			return nil, fmt.Errorf("siprtp: got nil packet from ReceiveRTP")
		}

		// RFC 2833 telephone-event (out-of-band DTMF) — do not feed to audio decoder.
		if t.telephoneEventPT != 0 && pkt.Header.PayloadType == t.telephoneEventPT {
			digit, end, ok := dtmf.EventFromRFC2833(pkt.Payload)
			if ok && end && digit != "" {
				return &media.DTMFPacket{Digit: digit, End: end}, nil
			}
			continue
		}

		// Only accept the negotiated audio RTP payload type.
		if t.codec.PayloadType != 0 && pkt.Header.PayloadType != t.codec.PayloadType {
			continue
		}

		return &media.AudioPacket{Payload: pkt.Payload}, nil
	}
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

	// RTP timestamp increment must be based on codec clock rate, not payload bytes.
	// For codecs like OPUS (variable bitrate), deriving samples from payload length
	// causes timestamp drift and audible artifacts (noise/choppiness).
	samples := uint32(0)
	if t.codec.SampleRate > 0 {
		if t.codec.FrameDuration != "" {
			if d, err := time.ParseDuration(t.codec.FrameDuration); err == nil && d > 0 {
				samples = uint32((int64(t.codec.SampleRate) * d.Milliseconds()) / 1000)
			}
		}
		// Default to 20ms frames if not specified/parsable.
		if samples == 0 {
			samples = uint32((t.codec.SampleRate * 20) / 1000)
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

