// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package gateway

// SessionPersister is the optional persistence sink that every transport
// (SIP, xiaozhi, WebRTC) can plug into to record a call's lifecycle into
// `voiceserver.db`. The interface lives in pkg/voice/gateway because all
// three transports already depend on this package — declaring it here
// avoids a circular import while keeping the cmd-side implementation
// (cmd/voiceserver/persister.go) free of transport-specific types.
//
// Hook order (per call):
//
//	OnAccept   — exactly once, after media negotiation completes
//	OnASRFinal — N times, one per final ASR transcript
//	OnTurn     — N times, paired 1:1 with OnASRFinal in the persister
//	OnTerminate — exactly once, on tear-down (any reason)
//
// Implementations must be goroutine-safe; OnASRFinal and OnTurn fire
// from gateway.Client's read / TTS-worker goroutines.

import (
	"context"
	"time"
)

// MediaStatsSample is a transport-neutral snapshot of media-quality
// counters and instantaneous metrics for one call. Not every transport
// fills every field — zero values mean "unknown / not applicable", not
// "zero observed". Persisters should treat zero-valued numeric fields
// the same as absent.
type MediaStatsSample struct {
	At              time.Time
	Final           bool   // true = end-of-call summary
	Codec           string // wire name, lowercase
	ClockRate       int
	Channels        int
	RemoteAddr      string
	PacketsSent     uint64
	PacketsReceived uint64
	BytesSent       uint64
	BytesReceived   uint64
	PacketsLost     uint64
	NACKsSent       uint64
	NACKsReceived   uint64
	RTTMs           int
	JitterMs        int
	LossRate        float64
	BitrateKbps     int
	Note            string
}

// RecordingInfo captures one finished recording artefact. Transports
// emit this once per recording (typically at teardown) so the persister
// can write a row into call_recording.
type RecordingInfo struct {
	Bucket     string
	Key        string
	URL        string
	Format     string // "wav", "opus", …
	Layout     string // "mono", "stereo-l-r", …
	SampleRate int
	Channels   int
	Bytes      int
	DurationMs int64
	Hash       string
	Note       string
}

// SessionPersister captures a single call from media-negotiation through
// teardown. Methods are best-effort: implementations should log and
// swallow errors rather than fail a call for a persistence hiccup.
type SessionPersister interface {
	// OnAccept stamps the negotiated media profile on the call row.
	// codec is the wire-name (opus / pcm / pcmu / …); sampleRate is
	// the negotiated clock; remoteAddr is the peer's media address
	// (RTP host:port for SIP, WS remote for xiaozhi, ICE pair string
	// for WebRTC). Any field may be empty when the transport doesn't
	// expose it.
	OnAccept(ctx context.Context, codec string, sampleRate int, remoteAddr string)

	// OnASRFinal is invoked for every final transcript, after the
	// dialog plane has already received the asr.final event. The
	// persister typically buffers the text so the next OnTurn can
	// pair it into a complete dialog turn row.
	OnASRFinal(ctx context.Context, text string)

	// OnTurn is invoked after each TTS Speak completes. The TurnEvent
	// carries the text spoken plus optional dialog-side metadata
	// (LLM model, latency).
	OnTurn(ctx context.Context, t TurnEvent)

	// OnTerminate finalises the call row with end timestamps,
	// duration, and a classified end status derived from `reason`.
	OnTerminate(ctx context.Context, reason string)

	// OnEvent records a timestamped, classified entry into the
	// per-call timeline (call_events). Use the persist.EventKind*
	// constants from pkg/persist when possible so dashboards can
	// aggregate consistently across transports. detail is a small
	// JSON blob (passed through unchanged); pass nil when there's no
	// extra context to attach.
	OnEvent(ctx context.Context, kind, level string, detail []byte)

	// OnMediaStats records a media-quality sample. Called periodically
	// from transports that can compute stats (WebRTC) and once at
	// teardown (any transport) with a final summary.
	OnMediaStats(ctx context.Context, sample MediaStatsSample)

	// OnRecording records one recording artefact. Called once per
	// recording (typically at teardown after the WAV / opus file has
	// been flushed to disk).
	OnRecording(ctx context.Context, rec RecordingInfo)
}
