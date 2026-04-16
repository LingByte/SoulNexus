package conversation

import (
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/webseat"
	"go.uber.org/zap"
)

// StartWebSeatHandoff stops AI media and waits for the browser to POST {APIPrefix}/lingecho/webseat/v1/join with this Call-ID.
func StartWebSeatHandoff(inboundCallID string, lg *zap.Logger) {
	if lg == nil && logger.Lg != nil {
		lg = logger.Lg
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	if lookupInbound == nil {
		lg.Warn("web seat: SetInboundSessionLookup not configured")
		ReleaseTransferStartDedupe(inboundCallID)
		return
	}
	inbound := lookupInbound(inboundCallID)
	if inbound == nil {
		lg.Warn("web seat: inbound session not found", zap.String("call_id", inboundCallID))
		ReleaseTransferStartDedupe(inboundCallID)
		return
	}
	if err := webseat.RegisterAwaiting(inboundCallID, inbound, lg); err != nil {
		lg.Warn("web seat: register failed", zap.String("call_id", inboundCallID), zap.Error(err))
		ReleaseTransferStartDedupe(inboundCallID)
		return
	}
	MarkInboundHadWebSeatHandoff(inboundCallID)
}

// HangupWebSeatBridgeIfAny handles customer BYE while on web seat bridge or while awaiting join.
func HangupWebSeatBridgeIfAny(callID string) bool {
	return webseat.HangupIfCustomerBye(callID)
}

// HangupWebSeatBridgeFull ends web seat and BYEs the customer (browser disconnect / operator hangup).
func HangupWebSeatBridgeFull(callID string) bool {
	return webseat.HangupFull(callID)
}

// ActiveWebSeatSession is true while awaiting browser join or while the WebRTC bridge is up.
func ActiveWebSeatSession(callID string) bool {
	return webseat.IsPendingOrActive(callID)
}

// ActiveWebSeatBridge is true only after browser join completed and bridge is running.
func ActiveWebSeatBridge(callID string) bool {
	return webseat.IsActive(callID)
}

// ReleaseTransferStartDedupe clears the per-call transfer lock (join timeout, failed register).
func ReleaseTransferStartDedupe(callID string) {
	stopTransferRinging(callID)
	transferStarted.Delete(callID)
}
