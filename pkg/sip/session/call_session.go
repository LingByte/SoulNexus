package session

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	sipprotocol "github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
)

// CallSession binds an RTP session to a MediaSession for SIP calls.
//
// Uplink: RTP -> decode -> PCM for ASR processors.
// Downlink: only synthesized (TTS) PCM is encoded and sent as RTP; uplink is not echoed
// (see media.KeySIPSuppressUplinkEcho).
type CallSession struct {
	CallID string

	rtpSess *rtp.Session
	media   *media.MediaSession
	neg     sipprotocol.SDPCodec

	// RTP transports and codec (same as used for MediaSession) for handoff to in-process PCM bridge.
	rxTransport *rtp.SIPRTPTransport
	txTransport *rtp.SIPRTPTransport
	srcCodec    media.CodecConfig
	dtmfPT      uint8

	ctx    context.Context
	cancel context.CancelFunc

	startOnce sync.Once
	// For SIP: media starts on ACK, not on INVITE.
	ackOnce sync.Once

	voiceMu       sync.Mutex
	voiceAttached bool
}

// NewCallSession creates a call session with codec negotiation from SDP.
func NewCallSession(callID string, rtpSess *rtp.Session, sdpCodecs []sipprotocol.SDPCodec) (*CallSession, error) {
	if callID == "" {
		return nil, fmt.Errorf("sip: empty callID")
	}
	if rtpSess == nil {
		return nil, fmt.Errorf("sip: nil rtp session")
	}
	if len(sdpCodecs) == 0 {
		return nil, fmt.Errorf("sip: empty sdp codecs")
	}

	// Choose the first supported codec.
	var src media.CodecConfig
	negotiatedPayloadType := uint8(0)
	var negotiatedSDP sipprotocol.SDPCodec
	found := false
	for _, c := range sdpCodecs {
		switch c.Name {
		case "pcmu", "pcma":
			found = true
			negotiatedPayloadType = c.PayloadType
			negotiatedSDP = c
			negotiatedSDP.Channels = 1
			src = media.CodecConfig{
				Codec:         c.Name, // "pcmu" or "pcma"
				SampleRate:    c.ClockRate,
				Channels:      1,
				BitDepth:      8, // PCMU/PCMA payload is 8-bit
				PayloadType:   negotiatedPayloadType,
				// Use 20ms frames for RTP audio so encoder/decoder match
				// typical SIP/RTP expectations and keep payload sizes bounded.
				FrameDuration: "20ms",
			}
			break
		case "g722":
			found = true
			negotiatedPayloadType = c.PayloadType
			negotiatedSDP = c
			negotiatedSDP.Channels = 1
			src = media.CodecConfig{
				Codec:         "g722",
				SampleRate:    c.ClockRate,
				Channels:      1,
				BitDepth:      16,
				PayloadType:   negotiatedPayloadType,
				FrameDuration: "20ms",
			}
			break
		case "opus":
			found = true
			negotiatedPayloadType = c.PayloadType
			// Force mono in the answer: stereo OPUS from softphones often degrades ASR quality
			// when combined with 8k/16k resampling; answering OPUS/48000/1 improves consistency.
			ch := 1
			negotiatedSDP = c
			negotiatedSDP.Channels = ch
			src = media.CodecConfig{
				Codec:         "opus",
				SampleRate:    c.ClockRate, // typically 48000
				Channels:      ch,
				BitDepth:      16,
				PayloadType:   negotiatedPayloadType,
				FrameDuration: "20ms",
			}
			break
		}
		if found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("sip: unsupported codec (need one of: opus/g722/pcmu/pcma)")
	}

	// Target PCM format for ASR/TTS pipelines.
	pcm := media.CodecConfig{
		Codec:         "pcm",
		SampleRate:    16000,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "",
	}

	dec, err := encoder.CreateDecode(src, pcm)
	if err != nil {
		return nil, fmt.Errorf("sip: CreateDecode failed: %w", err)
	}
	dec = passthroughDTMFDecode(dec)
	enc, err := encoder.CreateEncode(src, pcm)
	if err != nil {
		return nil, fmt.Errorf("sip: CreateEncode failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	dtmfPT := telephoneEventPayloadType(sdpCodecs)
	rxTransport := rtp.NewSIPRTPTransport(rtpSess, src, media.DirectionInput, dtmfPT)
	txTransport := rtp.NewSIPRTPTransport(rtpSess, src, media.DirectionOutput, 0)

	ms := media.NewDefaultSession().
		Context(ctx).
		SetSessionID("sip-call-" + callID).
		Decode(dec).
		Encode(enc).
		Input(rxTransport).
		Output(txTransport)
	ms.Set(media.KeySIPSuppressUplinkEcho, true)

	return &CallSession{
		CallID:      callID,
		rtpSess:     rtpSess,
		media:       ms,
		neg:         negotiatedSDP,
		rxTransport: rxTransport,
		txTransport: txTransport,
		srcCodec:    src,
		dtmfPT:      dtmfPT,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// MediaSession exposes the underlying media pipeline for voice processors (ASR/TTS hooks).
func (cs *CallSession) MediaSession() *media.MediaSession {
	if cs == nil {
		return nil
	}
	return cs.media
}

// AttachVoiceConversation runs fn once before media Serve() (typically from ACK) to register
// processors or other hooks. If fn fails, a later call may retry.
func (cs *CallSession) AttachVoiceConversation(fn func() error) error {
	if cs == nil || fn == nil {
		return nil
	}
	cs.voiceMu.Lock()
	defer cs.voiceMu.Unlock()
	if cs.voiceAttached {
		return nil
	}
	if err := fn(); err != nil {
		return err
	}
	cs.voiceAttached = true
	return nil
}

func passthroughDTMFDecode(dec media.EncoderFunc) media.EncoderFunc {
	return func(p media.MediaPacket) ([]media.MediaPacket, error) {
		if _, ok := p.(*media.DTMFPacket); ok {
			return []media.MediaPacket{p}, nil
		}
		return dec(p)
	}
}

func telephoneEventPayloadType(codecs []sipprotocol.SDPCodec) uint8 {
	for _, c := range codecs {
		if strings.EqualFold(strings.TrimSpace(c.Name), "telephone-event") {
			return c.PayloadType
		}
	}
	return 0
}

func (cs *CallSession) NegotiatedCodec() sipprotocol.SDPCodec {
	if cs == nil {
		return sipprotocol.SDPCodec{}
	}
	return cs.neg
}

// RTPSession returns the underlying RTP/UDP session (for building a transfer bridge).
func (cs *CallSession) RTPSession() *rtp.Session {
	if cs == nil {
		return nil
	}
	return cs.rtpSess
}

// SourceCodec is the negotiated RTP codec (PCMU/PCMA/G722/OPUS) for this leg.
func (cs *CallSession) SourceCodec() media.CodecConfig {
	if cs == nil {
		return media.CodecConfig{}
	}
	return cs.srcCodec
}

// DTMFPayloadType is the negotiated telephone-event PT, or 0 if none.
func (cs *CallSession) DTMFPayloadType() uint8 {
	if cs == nil {
		return 0
	}
	return cs.dtmfPT
}

// StopMediaPreserveRTP stops the MediaSession (AI pipeline, RTP read/write loops) but keeps the UDP
// socket open so new SIPRTPTransport instances can attach for bridging.
func (cs *CallSession) StopMediaPreserveRTP() {
	if cs == nil {
		return
	}
	if cs.rxTransport != nil {
		cs.rxTransport.PreserveSessionOnClose = true
	}
	if cs.txTransport != nil {
		cs.txTransport.PreserveSessionOnClose = true
	}
	if cs.cancel != nil {
		cs.cancel()
	}
	if cs.media != nil {
		_ = cs.media.Close()
	}
}

// CloseRTPOnly closes the RTP UDP socket after a bridge or full teardown path.
func (cs *CallSession) CloseRTPOnly() {
	if cs == nil || cs.rtpSess == nil {
		return
	}
	_ = cs.rtpSess.Close()
	cs.rtpSess = nil
}

// Start starts MediaSession serving in background.
func (cs *CallSession) Start() {
	if cs == nil || cs.media == nil {
		return
	}
	cs.startOnce.Do(func() {
		go func() {
			_ = cs.media.Serve()
		}()
	})
}

// StartOnACK starts media pipeline once (idempotent) when ACK is received.
func (cs *CallSession) StartOnACK() {
	if cs == nil {
		return
	}
	cs.ackOnce.Do(func() {
		cs.Start()
	})
}

// Stop stops the session and closes underlying RTP resources.
func (cs *CallSession) Stop() {
	if cs == nil {
		return
	}
	if cs.cancel != nil {
		cs.cancel()
	}
	if cs.media != nil {
		_ = cs.media.Close()
	}
	if cs.rtpSess != nil {
		_ = cs.rtpSess.Close()
		cs.rtpSess = nil
	}
}

