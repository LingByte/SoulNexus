// Package session is the transport-agnostic voice dialog session layer.
//
// It sits above pkg/dialog/engine and below concrete transports (voice,
// WebSocket PCM, WebRTC). One Session corresponds to one voice
// conversation regardless of how audio enters or leaves the system.
//
// Typical flow (web):
//
//	POST /api/lingecho/voice-session/v1/sessions  → session_id + endpoints
//	WS   …/ws?session_id=…                        → PCM duplex + engine attach
//	POST …/webrtc/offer { session_id, sdp }         → SDP answer + engine attach
//
// voice reuses the same attach path via voiceattach → session.AttachEngine.
package session
