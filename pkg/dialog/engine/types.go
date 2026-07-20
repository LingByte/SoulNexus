package engine

import (
	"context"
	"time"
)

// Mode is the dialog engine selection key. Implementations register
// themselves under one of these values via Register().
type Mode string

const (
	// ModeCascaded is the classic ASR → LLM → TTS pipeline. Best
	// quality in latency-tolerant settings (1-3s end-of-turn) and
	// when you need explicit control over each stage (e.g. content
	// moderation between LLM and TTS).
	//
	// Production voice attach resolves tenant pipeline mode to
	// ModeCascadedNative (StreamingPort + pkg/dialog/cascaded.Engine).
	// ModeCascaded remains a valid registry key for non-production / tests.
	ModeCascaded Mode = "cascaded"

	// ModeCascadedNative selects the native cascaded engine
	// (pkg/dialog/cascaded.Engine) — ASR → LLM → TTS via pipeline.Stage
	// and StreamingPort. This is the production cascaded path.
	ModeCascadedNative Mode = "cascaded-native"

	// ModeRealtime hands the entire turn to a single multimodal
	// provider (Qwen-Omni / GPT-4o realtime / Doubao realtime). Best
	// latency (<400ms first audio) but you give up per-stage hooks.
	ModeRealtime Mode = "realtime"

	// ModeHybrid runs realtime for duplex audio while a parallel ASR
	// sidecar persists caller transcripts (trigger=hybrid_asr).
	ModeHybrid Mode = "hybrid"
)

// IsValid reports whether m is one of the supported engine modes.
func (m Mode) IsValid() bool {
	switch m {
	case ModeCascaded, ModeCascadedNative, ModeRealtime, ModeHybrid:
		return true
	default:
		return false
	}
}

// PCMFrame carries one frame of linear PCM 16-bit signed audio.
//
// All engine boundaries speak PCM s16le. Transcoding to/from G.711 /
// G.722 / Opus is the media layer's job — engines never see codec
// concerns. Sample rate is tagged on the frame so engines can
// resample when their provider requires a fixed rate (e.g. Qwen-Omni
// only accepts 16kHz mono input).
type PCMFrame struct {
	// Data is mono signed 16-bit little-endian samples. Length MUST
	// be a multiple of 2; odd byte counts indicate a producer bug
	// and SHOULD be rejected by the consumer.
	Data []byte

	// SampleRate of the samples in Data, in Hz. Common values: 8000,
	// 16000, 24000, 44100, 48000.
	SampleRate int

	// Timestamp is the wall-clock time at which the first sample of
	// this frame was captured (input) or should be played (output).
	// Monotonic semantics inside one call.
	Timestamp time.Time

	// Synthesized marks AI/TTS output. Uplink passthrough frames in
	// the cascaded pipeline leave this false and must not be sent to
	// SendOutputPCM (would echo the caller's voice to the browser).
	Synthesized bool
}

// Detach is returned by Engine.Attach. Calling it cleanly tears down
// the engine for this call: stops Provider streams, drains any final
// PCM, persists the last turn, releases resources.
//
// Detach is idempotent — subsequent calls are no-ops.
type Detach func(context.Context) error

// Initiator records who initiated a control event (hangup / transfer
// / barge-in). Persisted in CDR + analytics.
type Initiator string

const (
	InitiatorUser     Initiator = "user"     // PSTN caller
	InitiatorAI       Initiator = "ai"       // dialog engine decision
	InitiatorAgent    Initiator = "agent"    // human seat
	InitiatorSystem   Initiator = "system"   // platform-side (timeout / error)
	InitiatorOperator Initiator = "operator" // ops / admin force
)
