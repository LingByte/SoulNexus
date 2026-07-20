// Package transport is the protocol layer for voice dialog.
//
// Voice layer (pkg/dialog/session, engine, cascaded, realtime) speaks
// only engine.MediaPort — PCM in/out, call metadata. It must not import
// WebSocket or WebRTC packages.
//
// Protocol layer (pkg/dialog/transport/*) adapts each wire format to
// MediaPort and delegates engine wiring to session.AttachEngine.
//
// Business hooks (transfer, welcome WAV, tenant env loading) are injected
// via transport adapters (e.g. ProtocolHooks on the wire adapter).
package transport
