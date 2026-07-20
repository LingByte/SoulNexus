// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package gateway

// SessionPersister is the optional persistence sink that every transport
// (websocket, xiaozhi, WebRTC, …) can plug into to record a call's lifecycle.
//
// Hook order (per call):
//
//	OnAccept    — exactly once, after media negotiation completes
//	OnASRFinal  — N times, one per final ASR transcript
//	OnTurn      — N times, paired 1:1 with OnASRFinal in the persister
//	OnEvent     — any number, for ad-hoc timeline entries
//	OnMediaStats — periodic + once at teardown
//	OnRecording — once per recording artefact (typically at teardown)
//	OnTerminate — exactly once, on tear-down (any reason)
//
// Implementations must be goroutine-safe.
//
// Note (SoulNexus divergence from VoiceServer):
//   - TurnEvent is declared here rather than in client.go so callers that
//     only need the persister contract (e.g. pkg/voice/recorder) do not
//     transitively import the WebSocket gateway client and its
//     pkg/voice/{vad,metrics} dependencies.

import (
	"context"
	"time"
)

// MediaStatsSample is a transport-neutral snapshot of media-quality
// counters and instantaneous metrics for one call. Zero numeric values
// mean "unknown / not applicable", not "zero observed".
type MediaStatsSample struct {
	At              time.Time
	Final           bool
	Codec           string
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

// TurnEvent is delivered after each TTS Speak completes. It pairs with
// the most recent ASR final to form one dialog turn row.
type TurnEvent struct {
	UtteranceID    string
	LLMText        string
	Meta           *CommandMeta
	DurationMs     int
	TTSFirstByteMs int
	E2EFirstByteMs int
	OK             bool
}

// SessionPersister captures a single call from media-negotiation through
// teardown. Methods are best-effort: implementations should log and
// swallow errors rather than fail a call for a persistence hiccup.
type SessionPersister interface {
	OnAccept(ctx context.Context, codec string, sampleRate int, remoteAddr string)
	OnASRFinal(ctx context.Context, text string)
	OnTurn(ctx context.Context, t TurnEvent)
	OnTerminate(ctx context.Context, reason string)
	OnEvent(ctx context.Context, kind, level string, detail []byte)
	OnMediaStats(ctx context.Context, sample MediaStatsSample)
	OnRecording(ctx context.Context, rec RecordingInfo)
}
