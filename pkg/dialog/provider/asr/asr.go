// Package asr defines the streaming ASR (Automatic Speech Recognition)
// provider interface. Implementations live in sibling sub-packages
// (qcloud, deepgram, baidu, ...) and self-register via
// provider.ASRRegistry.
//
// The interface targets STREAMING ASR — single-utterance batch APIs
// can be wrapped behind the same surface by buffering until silence.
package asr

import (
	"context"
	"time"
)

// Provider is the factory abstraction for one ASR vendor. Each
// vendor's package registers exactly one Provider under its name.
type Provider interface {
	// Name returns the registered identifier (e.g. "qcloud",
	// "deepgram"). Used in metrics labels and tenant config.
	Name() string

	// Open starts a new streaming session. Returns a Stream that
	// accepts PCM and emits Results. Closing the Stream MUST release
	// all vendor resources (WebSocket, gRPC stream, ...).
	Open(ctx context.Context, cfg StreamConfig) (Stream, error)
}

// StreamConfig is the per-session ASR configuration.
type StreamConfig struct {
	// SampleRate of the PCM data that will be pushed (Hz). Vendors
	// that don't support the requested rate MUST resample
	// internally rather than fail; engines never resample.
	SampleRate int

	// Language is a BCP-47 tag (e.g. "zh-CN", "en-US", "yue-CN").
	// Empty = vendor-default (often Chinese).
	Language string

	// EnableInterim asks the vendor to emit interim (partial)
	// results. Disabling reduces network chatter when the engine
	// only consumes final transcripts.
	EnableInterim bool

	// HotWords are vendor-specific bias terms (proper nouns,
	// product names, ...). Empty when unsupported by the vendor.
	HotWords []string

	// VoiceActivityTimeout is the max silence before the vendor
	// flushes the current utterance as final. Zero = vendor default.
	VoiceActivityTimeout time.Duration
}

// Stream is an open ASR streaming session. Goroutine ownership:
//   - Caller goroutine pushes PCM via Push.
//   - Vendor or Stream-internal goroutine produces Results.
//   - Close terminates both.
type Stream interface {
	// Push feeds one frame of linear PCM 16-bit signed (mono,
	// matching StreamConfig.SampleRate) into the recogniser.
	// Returns ErrClosed once Close has been called.
	Push(pcm []byte) error

	// Results returns a receive-only channel of recognition results.
	// Closed when the stream ends (Close or vendor disconnect).
	Results() <-chan Result

	// Close terminates the stream and releases vendor resources.
	// Idempotent. SHOULD return quickly; never blocks on network.
	Close() error
}

// Result is one ASR recognition event.
type Result struct {
	// Text is the transcribed text. For interim results this is
	// the current best guess of the partial utterance.
	Text string

	// IsFinal indicates whether this is the vendor's committed
	// final result for the current utterance. Engines typically
	// trigger LLM/tool dispatch only on final=true.
	IsFinal bool

	// Confidence is the vendor's confidence score in [0,1].
	// Zero when the vendor does not report confidence.
	Confidence float64

	// StartedAt and CompletedAt are the wall-clock times of the
	// utterance start and final result emission, used for analytics.
	// Zero when the vendor does not report timestamps.
	StartedAt   time.Time
	CompletedAt time.Time
}
