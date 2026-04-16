package conversation

import (
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/sip/bridge"
	siprtp "github.com/LingByte/SoulNexus/pkg/sip/rtp"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"go.uber.org/zap"
)

// CallStore removes sessions from the SIP server map without invoking Stop (RTP already handled).
type CallStore interface {
	RemoveCallSession(callID string)
}

var (
	lookupInbound func(callID string) *sipSession.CallSession
	callStore     CallStore

	bridgeSendOutboundBYE func(callID string) error
	bridgeHangupInbound   func(callID string) error

	bridgeMu sync.Mutex
	bridges  map[string]*transferBridgeState // keyed by inbound or outbound Call-ID
)

// legBridge is PCM transcode or raw G.711 RTP relay.
type legBridge interface {
	Start()
	Stop()
}

type transferBridgeState struct {
	br         legBridge
	inboundID  string
	outboundID string
	inboundCS  *sipSession.CallSession
	outboundCS *sipSession.CallSession
}

// TransferBridgeByePersist carries inbound-leg recording + media metadata for sippersist.OnBye after a transfer bridge ends.
type TransferBridgeByePersist struct {
	InboundCallID      string
	RawPayload         []byte
	CodecName          string
	Initiator          string
	RecordSampleRate   int
	RecordOpusChannels int
}

func transferBridgePersistSnapshot(bs *transferBridgeState, initiator string) *TransferBridgeByePersist {
	if bs == nil || bs.inboundID == "" {
		return nil
	}
	p := &TransferBridgeByePersist{
		InboundCallID: bs.inboundID,
		Initiator:     initiator,
	}
	if initiator == "" {
		p.Initiator = "remote"
	}
	if bs.inboundCS != nil {
		p.RawPayload = bs.inboundCS.TakeRecording()
		p.CodecName = bs.inboundCS.NegotiatedCodec().Name
		src := bs.inboundCS.SourceCodec()
		p.RecordSampleRate = src.SampleRate
		p.RecordOpusChannels = src.OpusDecodeChannels
		if p.RecordOpusChannels < 1 {
			p.RecordOpusChannels = src.Channels
		}
	}
	return p
}

// SetInboundSessionLookup resolves the inbound UAS CallSession by Call-ID (set from cmd/sip).
func SetInboundSessionLookup(fn func(string) *sipSession.CallSession) {
	lookupInbound = fn
}

// SetCallStore removes call entries when a transfer bridge ends (set from cmd/sip).
func SetCallStore(cs CallStore) {
	callStore = cs
}

// SetTransferPeerCallbacks wires BYE to the peer leg when one side hangs up (outbound Manager.SendBYE, server SendUASBye).
func SetTransferPeerCallbacks(sendOutboundBYE func(callID string) error, hangupInboundRemote func(callID string) error) {
	bridgeSendOutboundBYE = sendOutboundBYE
	bridgeHangupInbound = hangupInboundRemote
}

// ActiveTransferBridgeForCallID is true when this Call-ID (inbound or outbound) is in an active media bridge.
func ActiveTransferBridgeForCallID(callID string) bool {
	if callID == "" {
		return false
	}
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if bridges == nil {
		return false
	}
	_, ok := bridges[callID]
	return ok
}

// StartTransferBridge stops AI media on both legs and bridges audio.
// Raw UDP RTP relay only when both legs are the same narrowband G.711 (e.g. PCMU↔PCMU); transfer agent
// legs are offered PCMU, so the usual path is PCM transcode (e.g. inbound Opus ↔ agent PCMU).
// Raw relay keeps peer SSRC/seq/timestamp; only PT is remapped when needed. SIP_TRANSFER_RELAY_REWRITE_RTP=1
// restores legacy SSRC/seq/ts rewrite.
// inboundCallID is the original caller's Call-ID (CorrelationID on the outbound DialRequest).
func StartTransferBridge(inboundCallID string, outboundCS *sipSession.CallSession, outboundCallID string, lg *zap.Logger) {
	stopTransferRinging(inboundCallID)
	if lg == nil && logger.Lg != nil {
		lg = logger.Lg
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	if lookupInbound == nil {
		lg.Warn("sip transfer bridge: SetInboundSessionLookup not configured")
		if outboundCS != nil {
			outboundCS.Stop()
		}
		if callStore != nil && outboundCallID != "" {
			callStore.RemoveCallSession(outboundCallID)
		}
		return
	}
	inbound := lookupInbound(inboundCallID)
	if inbound == nil {
		lg.Warn("sip transfer bridge: inbound session not found",
			zap.String("inbound_call_id", inboundCallID),
			zap.String("outbound_call_id", outboundCallID))
		if outboundCS != nil {
			outboundCS.Stop()
		}
		if callStore != nil && outboundCallID != "" {
			callStore.RemoveCallSession(outboundCallID)
		}
		return
	}
	if outboundCS == nil {
		lg.Warn("sip transfer bridge: nil outbound session")
		return
	}

	inbound.StopMediaPreserveRTP()
	outboundCS.StopMediaPreserveRTP()

	ccIn := inbound.SourceCodec()
	ccOut := outboundCS.SourceCodec()
	var br legBridge
	var err error
	var mode string
	var pcmReason string
	rawOK := bridge.CanRawDatagramRelay(ccIn, ccOut)
	if rawOK {
		br, err = bridge.NewTwoLegPayloadRelay(
			inbound.RTPSession(), outboundCS.RTPSession(),
			ccIn, ccOut,
			inbound.DTMFPayloadType(), outboundCS.DTMFPayloadType(),
		)
		if err != nil {
			pcmReason = "raw_relay_error: " + err.Error()
			lg.Warn("sip transfer bridge: raw relay failed, falling back to pcm", zap.Error(err))
			br = nil
		} else {
			mode = "raw_rtp_forward"
		}
	} else {
		pcmReason = "codecs_not_eligible_for_raw_relay"
	}
	if br == nil {
		callerRx := siprtp.NewSIPRTPTransport(inbound.RTPSession(), ccIn, media.DirectionInput, inbound.DTMFPayloadType())
		callerTx := siprtp.NewSIPRTPTransport(inbound.RTPSession(), ccIn, media.DirectionOutput, 0)
		agentRx := siprtp.NewSIPRTPTransport(outboundCS.RTPSession(), ccOut, media.DirectionInput, outboundCS.DTMFPayloadType())
		agentTx := siprtp.NewSIPRTPTransport(outboundCS.RTPSession(), ccOut, media.DirectionOutput, 0)
		br, err = bridge.NewTwoLegPCMBridge(callerRx, callerTx, agentRx, agentTx)
		if err != nil {
			lg.Warn("sip transfer bridge: build failed", zap.Error(err))
			inbound.CloseRTPOnly()
			outboundCS.CloseRTPOnly()
			if callStore != nil {
				callStore.RemoveCallSession(inboundCallID)
				callStore.RemoveCallSession(outboundCallID)
			}
			return
		}
		mode = "pcm_transcode"
	}

	bs := &transferBridgeState{
		br:         br,
		inboundID:  inboundCallID,
		outboundID: outboundCallID,
		inboundCS:  inbound,
		outboundCS: outboundCS,
	}
	bridgeMu.Lock()
	if bridges == nil {
		bridges = make(map[string]*transferBridgeState)
	}
	bridges[inboundCallID] = bs
	bridges[outboundCallID] = bs
	bridgeMu.Unlock()

	br.Start()

	logFields := []zap.Field{
		zap.String("inbound_call_id", inboundCallID),
		zap.String("outbound_call_id", outboundCallID),
		zap.String("mode", mode),
		zap.String("in_codec", ccIn.Codec),
		zap.String("out_codec", ccOut.Codec),
	}
	if strings.EqualFold(strings.TrimSpace(ccIn.Codec), "opus") {
		logFields = append(logFields, zap.Int("in_opus_decode_ch", ccIn.OpusDecodeChannels))
	}
	if mode == "pcm_transcode" && pcmReason != "" {
		logFields = append(logFields, zap.String("pcm_reason", pcmReason))
	}
	lg.Info("sip transfer bridge started", logFields...)
	MarkInboundHadSIPAgentTransfer(inboundCallID)
}

func hangPeerIfNeeded(bs *transferBridgeState, hungCallID string) {
	if bs == nil {
		return
	}
	if hungCallID == bs.inboundID {
		if bridgeSendOutboundBYE != nil {
			_ = bridgeSendOutboundBYE(bs.outboundID)
		}
		return
	}
	if hungCallID == bs.outboundID {
		if bridgeHangupInbound != nil {
			_ = bridgeHangupInbound(bs.inboundID)
		}
	}
}

func teardownBridge(bs *transferBridgeState) {
	if bs == nil {
		return
	}
	if bs.br != nil {
		bs.br.Stop()
	}
	if bs.inboundCS != nil {
		bs.inboundCS.CloseRTPOnly()
	}
	if bs.outboundCS != nil {
		bs.outboundCS.CloseRTPOnly()
	}
	if callStore != nil {
		callStore.RemoveCallSession(bs.inboundID)
		callStore.RemoveCallSession(bs.outboundID)
	}
}

// HangupTransferBridgeIfAny stops bridging and RTP for both legs when either Call-ID hangs up.
// Notifies the peer leg with BYE before local teardown when callbacks are wired.
// When non-nil, the caller must run sippersist.OnBye once for TransferBridgeByePersist.InboundCallID (BYE arrived on this stack).
func HangupTransferBridgeIfAny(callID string) *TransferBridgeByePersist {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if bridges == nil {
		return nil
	}
	bs, ok := bridges[callID]
	if !ok {
		return nil
	}
	persist := transferBridgePersistSnapshot(bs, "remote")
	hangPeerIfNeeded(bs, callID)
	delete(bridges, bs.inboundID)
	delete(bridges, bs.outboundID)
	teardownBridge(bs)
	if logger.Lg != nil {
		logger.Lg.Info("sip transfer bridge ended", zap.String("hangup_call_id", callID),
			zap.String("inbound_call_id", bs.inboundID), zap.String("outbound_call_id", bs.outboundID))
	}
	return persist
}

// HangupTransferBridgeFull tears down an active transfer bridge and BYE both SIP legs (e.g. keyword hangup).
// When non-nil, the caller must run sippersist.OnBye for TransferBridgeByePersist.InboundCallID with initiator "local".
func HangupTransferBridgeFull(callID string) *TransferBridgeByePersist {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if bridges == nil {
		return nil
	}
	bs, ok := bridges[callID]
	if !ok {
		return nil
	}
	persist := transferBridgePersistSnapshot(bs, "local")
	if bridgeSendOutboundBYE != nil {
		_ = bridgeSendOutboundBYE(bs.outboundID)
	}
	if bridgeHangupInbound != nil {
		_ = bridgeHangupInbound(bs.inboundID)
	}
	delete(bridges, bs.inboundID)
	delete(bridges, bs.outboundID)
	teardownBridge(bs)
	if logger.Lg != nil {
		logger.Lg.Info("sip transfer bridge full hangup", zap.String("trigger_call_id", callID),
			zap.String("inbound_call_id", bs.inboundID), zap.String("outbound_call_id", bs.outboundID))
	}
	return persist
}
