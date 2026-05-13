// Package webseat coordinates handoff of an inbound SIP call to a
// browser-based human agent, replacing the AI dialog plane with a
// live operator. It is the VoiceServer counterpart of LingEchoX's
// pkg/sip/webseat — a thinner, more focused design that follows
// VoiceServer's "small interface boundaries" convention.
//
// # Workflow
//
//  1. The SIP route decides "this call goes to a browser agent" (e.g.
//     a REFER target whose URI matches a webseat:// scheme, or an ACD
//     pool row with route_type=web).
//  2. The dialog plane (or the SIP REFER handler) calls
//     Hub.RegisterAwaiting(callID, leg). The AI pipeline detaches
//     and the SIP MediaLeg is held in "ringing for a human" state.
//  3. Connected browsers see the new call in their websocket feed
//     and may POST /webseat/join to accept it. The hub then asks
//     the configured Bridge to splice the SIP audio with the
//     browser's WebRTC PeerConnection (caller-PCM → encoded Opus to
//     browser, browser-Opus → decoded PCM back to SIP).
//  4. Either side hanging up fires Hub.Hangup, which tears the
//     bridge down and lets the SIP server send a normal BYE.
//
// # Why a Bridge interface?
//
// The actual audio splice between SIP MediaLeg and a pion
// PeerConnection is non-trivial: it needs codec transcoding (Opus ↔
// G.711/G.722), sample-rate conversion, and a careful teardown
// ordering so neither side keeps half-encoded frames in flight.
// We isolate that complexity behind the Bridge interface so:
//
//   - The hub's lifecycle and HTTP / WebSocket handlers can be unit
//     tested without spinning up real WebRTC peers.
//   - Different backends (pion, webrtc-rs via FFI, hardware MCU)
//     can be plugged in by satisfying the same two-method contract.
//   - The default no-op implementation lets cmd/voiceserver mount
//     the hub HTTP routes immediately; a real Bridge can be wired
//     in a follow-up commit without API churn here.
//
// # Out of scope (for now)
//
//   - ACD work-state tracking (LingEchoX has it baked in; we keep
//     that policy concern in the calling app).
//   - DTLS-SRTP keying for the SIP side. Browsers always use SRTP
//     to/from VoiceServer; the SIP leg stays plain RTP. The Bridge
//     handles the boundary.
//   - Multiple agents on one call (whisper / coach). One browser
//     per inbound call for now.
package webseat
