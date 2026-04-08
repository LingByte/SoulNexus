package session

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	sipprotocol "github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

const maxInboundRecordingBytes = 50 * 1024 * 1024

// SIP recording blob format v2 (sippersist): magic "SN2" then repeated
// [dir u8][seq u16LE][ts u32LE][len u16LE][payload].
// Includes RTP sequence/timestamp to allow reordering and smoother offline reconstruction.
const recBlobMagic = "SN2"

const (
	recDirUser = 0
	recDirAI   = 1
)

// EnvSIPMediaMaxSeconds caps the SIP AI voice pipeline (MediaSession) for one call, in seconds.
// 0 means no limit. When unset, a 1-hour default is used (legacy NewDefaultSession was 10 minutes).
const EnvSIPMediaMaxSeconds = "SIP_MEDIA_MAX_SECONDS"

const defaultSIPMediaMaxSeconds = 3600

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

	recMu  sync.Mutex
	recBuf []byte
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

	// Prefer higher quality codecs instead of blindly following offer order.
	// Many SIP UAs advertise PCMU/PCMA first for compatibility, which would lock
	// us to 8k narrowband and sound "muffled" even when Opus/G.722 is available.
	preferredCodecs := map[string]int{
		"opus": 0,
		"g722": 1,
		"pcmu": 2,
		"pcma": 3,
	}
	codecs := make([]sipprotocol.SDPCodec, len(sdpCodecs))
	copy(codecs, sdpCodecs)
	sort.SliceStable(codecs, func(i, j int) bool {
		ci := strings.ToLower(strings.TrimSpace(codecs[i].Name))
		cj := strings.ToLower(strings.TrimSpace(codecs[j].Name))
		ri, okI := preferredCodecs[ci]
		rj, okJ := preferredCodecs[cj]
		if !okI {
			ri = 100
		}
		if !okJ {
			rj = 100
		}
		return ri < rj
	})

	// Choose the first supported codec by preference.
	var src media.CodecConfig
	negotiatedPayloadType := uint8(0)
	var negotiatedSDP sipprotocol.SDPCodec
	found := false
	for _, c := range codecs {
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
			// SDP rtpmap uses 8000 Hz clock (RFC 3551) but G.722 decode/encode PCM is 16 kHz.
			// SampleRate here is the PCM rate for encoder resamplers; RTP timestamp ticks use 8 kHz
			// in pkg/sip/rtp SIPRTPTransport.Send.
			src = media.CodecConfig{
				Codec:         "g722",
				SampleRate:    16000,
				Channels:      1,
				BitDepth:      16,
				PayloadType:   negotiatedPayloadType,
				FrameDuration: "20ms",
			}
			break
		case "opus":
			found = true
			negotiatedPayloadType = c.PayloadType
			decodeCh := c.Channels
			if decodeCh < 1 {
				decodeCh = 1
			}
			if decodeCh > 2 {
				decodeCh = 2
			}
			negotiatedSDP = c
			// 200 OK SDP must match offered channel count (e.g. OPUS/48000/2). Answering /1 while
			// the peer sends stereo RTP breaks several stacks; we still encode TTS mono (Channels:1).
			negotiatedSDP.Channels = decodeCh
			src = media.CodecConfig{
				Codec:              "opus",
				SampleRate:         c.ClockRate, // typically 48000
				Channels:           1,
				OpusDecodeChannels: decodeCh,
				BitDepth:           16,
				PayloadType:        negotiatedPayloadType,
				FrameDuration:      "20ms",
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
	cs := &CallSession{
		CallID:   callID,
		rtpSess:  rtpSess,
		neg:      negotiatedSDP,
		srcCodec: src,
		dtmfPT:   dtmfPT,
		ctx:      ctx,
		cancel:   cancel,
	}
	rxTransport := rtp.NewSIPRTPTransport(rtpSess, src, media.DirectionInput, dtmfPT)
	rxTransport.OnInputRTP = func(seq uint16, ts uint32, p []byte) { cs.appendRecordingFrame(recDirUser, seq, ts, p) }
	txTransport := rtp.NewSIPRTPTransport(rtpSess, src, media.DirectionOutput, 0)
	txTransport.OnOutputRTP = func(seq uint16, ts uint32, p []byte) { cs.appendRecordingFrame(recDirAI, seq, ts, p) }
	cs.rxTransport = rxTransport
	cs.txTransport = txTransport

	ms := media.NewDefaultSession().
		Context(ctx).
		SetSessionID("sip-call-" + callID).
		Decode(dec).
		Encode(enc).
		Input(rxTransport).
		Output(txTransport)
	ms.Set(media.KeySIPSuppressUplinkEcho, true)
	ms.MaxSessionDuration = sipMediaMaxSecondsFromEnv()
	cs.media = ms

	return cs, nil
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
	// With PreserveSessionOnClose, Transport.Close() does not close the UDP socket, so a goroutine
	// blocked in ReceiveRTP would otherwise keep running. The transfer bridge then reads the same
	// socket and two readers split packets → noise. Wake the blocked read before tearing down media.
	if cs.rtpSess != nil && cs.rtpSess.Conn != nil {
		_ = cs.rtpSess.Conn.SetReadDeadline(time.Now())
	}
	if cs.cancel != nil {
		cs.cancel()
	}
	if cs.media != nil {
		_ = cs.media.Close()
		// Do not hand the RTP socket to the transfer bridge until MediaSession transport goroutines
		// have stopped calling ReadFromUDP — two readers on one UDP socket steal packets.
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = cs.media.WaitServeShutdown(drainCtx)
		drainCancel()
	}
	// The wakeup above leaves a past deadline on the conn; the next Read (transfer bridge) would
	// otherwise return i/o timeout immediately and silence audio. Clear the deadline for new readers.
	if cs.rtpSess != nil && cs.rtpSess.Conn != nil {
		_ = cs.rtpSess.Conn.SetReadDeadline(time.Time{})
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
		cs.media.NotifyServeStarting()
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

func (cs *CallSession) appendRecordingFrame(dir byte, seq uint16, ts uint32, p []byte) {
	if cs == nil || len(p) == 0 {
		return
	}
	if dir != recDirUser && dir != recDirAI {
		return
	}
	maxB := maxInboundRecordingBytes
	cs.recMu.Lock()
	defer cs.recMu.Unlock()
	if len(cs.recBuf) >= maxB {
		return
	}
	rem := maxB - len(cs.recBuf)
	if rem <= 0 {
		return
	}
	frameOverhead := 1 + 2 + 4 + 2 // dir + seq + ts + uint16 len
	if len(cs.recBuf) == 0 {
		if len(recBlobMagic) > rem {
			return
		}
		cs.recBuf = append(cs.recBuf, recBlobMagic...)
		rem = maxB - len(cs.recBuf)
	}
	if frameOverhead+len(p) > rem {
		return
	}
	cs.recBuf = append(cs.recBuf, dir)
	var hdr [8]byte
	binary.LittleEndian.PutUint16(hdr[0:2], seq)
	binary.LittleEndian.PutUint32(hdr[2:6], ts)
	binary.LittleEndian.PutUint16(hdr[6:8], uint16(len(p)))
	cs.recBuf = append(cs.recBuf, hdr[:]...)
	cs.recBuf = append(cs.recBuf, p...)
}

// TakeRecording returns buffered RTP recording (SN2 + per-frame dir/seq/ts/len/payload) and clears the buffer.
func (cs *CallSession) TakeRecording() []byte {
	if cs == nil {
		return nil
	}
	cs.recMu.Lock()
	defer cs.recMu.Unlock()
	if len(cs.recBuf) == 0 {
		return nil
	}
	out := make([]byte, len(cs.recBuf))
	copy(out, cs.recBuf)
	cs.recBuf = cs.recBuf[:0]
	return out
}

func sipMediaMaxSecondsFromEnv() int {
	v := strings.TrimSpace(utils.GetEnv(EnvSIPMediaMaxSeconds))
	if v == "" {
		return defaultSIPMediaMaxSeconds
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultSIPMediaMaxSeconds
	}
	return n
}

