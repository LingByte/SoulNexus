// Package dtmf decodes SIP RTP out-of-band DTMF (RFC 2833 / RFC 4733 telephone-event).
//
// Requirements:
//   - The remote SDP must offer a=rtpmap:PT telephone-event/8000 (or /48000); pkg/sip/session
//     passes that payload type to the RTP input transport.
//   - Key-up events are detected via the E (end) bit; in-band DTMF (audio tones) is not implemented.
//
// SIP INFO: many clients (e.g. Linphone) send DTMF via INFO + application/dtmf-relay; use
// dtmf.DigitFromSIPINFO — handled in pkg/sip/server handleInfo → conversation.HandleSIPINFODTMF.
//
// Env: SIP_TRANSFER_* for agent URI, SetTransferDialer in cmd/sip.
package dtmf
