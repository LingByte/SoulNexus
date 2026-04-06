package conversation

import "sync"

type inboundTransferFlags struct {
	SIPAgent bool
	WebSeat  bool
}

var inboundTransferFlagsMap sync.Map // Call-ID -> *inboundTransferFlags

func markInboundTransfer(callID string, fn func(*inboundTransferFlags)) {
	if callID == "" {
		return
	}
	v, _ := inboundTransferFlagsMap.LoadOrStore(callID, &inboundTransferFlags{})
	fn(v.(*inboundTransferFlags))
}

// MarkInboundHadSIPAgentTransfer records that this inbound Call-ID reached a live SIP transfer bridge.
func MarkInboundHadSIPAgentTransfer(callID string) {
	markInboundTransfer(callID, func(f *inboundTransferFlags) { f.SIPAgent = true })
}

// MarkInboundHadWebSeatHandoff records that this inbound Call-ID entered Web 坐席 handoff (awaiting or bridged).
func MarkInboundHadWebSeatHandoff(callID string) {
	markInboundTransfer(callID, func(f *inboundTransferFlags) { f.WebSeat = true })
}

// TakeInboundTransferFlags returns transfer flags for this inbound Call-ID and clears them (once per dialog).
func TakeInboundTransferFlags(callID string) (sipAgent, webSeat bool) {
	if callID == "" {
		return false, false
	}
	v, ok := inboundTransferFlagsMap.LoadAndDelete(callID)
	if !ok {
		return false, false
	}
	f := v.(*inboundTransferFlags)
	return f.SIPAgent, f.WebSeat
}
