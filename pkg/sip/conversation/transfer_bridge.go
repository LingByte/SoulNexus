package conversation

import (
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

	bridgeMu sync.Mutex
	bridges  map[string]*transferBridgeState // keyed by inbound or outbound Call-ID
)

type transferBridgeState struct {
	br         *bridge.TwoLegPCMBridge
	inboundID  string
	outboundID string
	inboundCS  *sipSession.CallSession
	outboundCS *sipSession.CallSession
}

// SetInboundSessionLookup resolves the inbound UAS CallSession by Call-ID (set from cmd/sip).
func SetInboundSessionLookup(fn func(string) *sipSession.CallSession) {
	lookupInbound = fn
}

// SetCallStore removes call entries when a transfer bridge ends (set from cmd/sip).
func SetCallStore(cs CallStore) {
	callStore = cs
}

// ActiveTransferBridgeForCallID is true when this Call-ID (inbound or outbound) is in an active PCM bridge.
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

// StartTransferBridge stops AI media on both legs and runs PCM bridging between them.
// inboundCallID is the original caller's Call-ID (CorrelationID on the outbound DialRequest).
func StartTransferBridge(inboundCallID string, outboundCS *sipSession.CallSession, outboundCallID string, lg *zap.Logger) {
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

	callerRx := siprtp.NewSIPRTPTransport(inbound.RTPSession(), inbound.SourceCodec(), media.DirectionInput, inbound.DTMFPayloadType())
	callerTx := siprtp.NewSIPRTPTransport(inbound.RTPSession(), inbound.SourceCodec(), media.DirectionOutput, 0)
	agentRx := siprtp.NewSIPRTPTransport(outboundCS.RTPSession(), outboundCS.SourceCodec(), media.DirectionInput, outboundCS.DTMFPayloadType())
	agentTx := siprtp.NewSIPRTPTransport(outboundCS.RTPSession(), outboundCS.SourceCodec(), media.DirectionOutput, 0)

	br, err := bridge.NewTwoLegPCMBridge(callerRx, callerTx, agentRx, agentTx)
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

	lg.Info("sip transfer bridge started",
		zap.String("inbound_call_id", inboundCallID),
		zap.String("outbound_call_id", outboundCallID))
}

// HangupTransferBridgeIfAny stops bridging and RTP for both legs when either Call-ID hangs up.
// Returns true if this call-id was part of an active transfer bridge.
func HangupTransferBridgeIfAny(callID string) bool {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if bridges == nil {
		return false
	}
	bs, ok := bridges[callID]
	if !ok {
		return false
	}
	delete(bridges, bs.inboundID)
	delete(bridges, bs.outboundID)

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
	if logger.Lg != nil {
		logger.Lg.Info("sip transfer bridge ended", zap.String("hangup_call_id", callID),
			zap.String("inbound_call_id", bs.inboundID), zap.String("outbound_call_id", bs.outboundID))
	}
	return true
}
