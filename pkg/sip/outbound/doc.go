// Package outbound implements SIP UAC (outbound) signaling and call legs without
// entangling with the UAS (inbound) code path in pkg/sip/server.
//
// Cross-cutting SIP helpers (CSeq parsing, PCM RMS, etc.) live in pkg/sip/siputil.
//
// Boundaries:
//   - Uses the same UDP socket as SIPServer via SignalingSender (SendSIP).
//   - Handles responses via Manager.HandleSIPResponse wired to protocol.Server.OnSIPResponse.
//   - Media uses the same pkg/sip/session.CallSession + pkg/media as inbound.
//
// Scenarios (see docs/SIP_OUTBOUND_MODULE.md):
//   - Manual / scripted outbound (campaign, callback)
//   - Inbound AI → transfer to human (second leg + optional RTP bridge)
package outbound
