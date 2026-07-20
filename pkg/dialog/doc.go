// Package dialog is the voice conversation engine for SoulNexus.
//
// It owns how the product talks with users over websocket / WebRTC media,
// without depending on telephony signaling.
//
// Engines interact with media through MediaPort / StreamingPort adapters in
// pkg/dialog/transport (pcm, websocket, webrtc).
package dialog
