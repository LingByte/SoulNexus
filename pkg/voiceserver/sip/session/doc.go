// Package session binds an RTP session and a pkg/media.MediaSession into a
// single "call leg" with codec negotiation, jitter buffering, and optional
// RTP-level taps for recording or metrics.
//
// # Layering
//
// This package sits at Layer 3 of the VoiceServer SIP stack:
//
//	Layer 4: pkg/sip/server       — signalling (registrar/INVITE/BYE)
//	Layer 3: pkg/sip/session      ← (this package) + pkg/sip/bridge
//	Layer 2: pkg/sip/rtp, dtmf, sdp
//	Layer 1: pkg/sip/stack, transaction, dialog, uas
//	Layer 0: pkg/media, pkg/media/encoder, pkg/media/vad, pkg/media/dsp
//
// Nothing in this package reads process environment or talks to databases;
// everything is plumbed through [MediaLegConfig].
//
// # Extension points
//
// Higher layers plug into a MediaLeg via:
//
//   - [MediaLegConfig.RTPInputTap] / [MediaLegConfig.RTPOutputTap] — observe
//     RTP at the wire (build a recorder, SIPREC sink, quality metrics).
//   - [MediaLegConfig.InputFilters] / [MediaLegConfig.OutputFilters] — gate
//     or transform MediaPackets (VAD pre-filter, PII mask, DTMF suppress).
//   - [CodecNegotiator] — register new codecs (G.729, AMR, future video
//     payloads) or override preference without touching library code.
//
// See also pkg/sip/bridge.TwoLegPCMBridge for two-leg call transfer and
// recording taps.
package session
