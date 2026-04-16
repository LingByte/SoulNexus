package conversation

var sipHangupFn func(callID string)

// SetSIPHangup wires server-side hangup (e.g. *server.SIPServer.HangupInboundCall) from cmd/sip.
func SetSIPHangup(fn func(callID string)) {
	sipHangupFn = fn
}

// RequestSIPHangup ends the inbound call immediately (transfer bridge or AI leg).
func RequestSIPHangup(callID string) {
	if callID == "" || sipHangupFn == nil {
		return
	}
	sipHangupFn(callID)
}
