// Package tts defines the streaming TTS (Text-to-Speech) provider
// interface. Implementations live in sibling sub-packages
// (qcloud, openai, azure, xunfei, volcengine, ...) and self-register
// via provider.TTSRegistry.
package tts

import (
	"context"
	"time"
)

// Provider is the factory abstraction for one TTS vendor.
type Provider interface {
	Name() string

	// Open starts a streaming synthesis session. The engine feeds
	// text via Speak (potentially in multiple calls) and reads PCM
	// from PCM() until the channel closes.
	Open(ctx context.Context, cfg StreamConfig) (Stream, error)
}

// StreamConfig is the per-session TTS configuration.
type StreamConfig struct {
	// Voice is a vendor-specific voice id (e.g. "zh-CN-XiaoyiNeural",
	// "alloy", "101003").
	Voice string

	// Language is a BCP-47 tag. May be inferred from Voice for some
	// vendors.
	Language string

	// SampleRate of the PCM frames the engine wants out. Vendors
	// that synthesise at a fixed rate MUST resample to match this.
	SampleRate int

	// Speed is a multiplier; 1.0 = vendor default. Zero falls back
	// to default.
	Speed float64

	// Pitch is a vendor-specific pitch shift; zero = default.
	Pitch float64

	// MaxLatency is the synthesis budget for "first byte" — useful
	// for engines that need to barge-in promptly. Zero = vendor
	// default.
	MaxLatency time.Duration
}

// Stream is an open TTS streaming session.
type Stream interface {
	// Speak appends text to be synthesised. May be called multiple
	// times to stream incremental tokens from an LLM. Final text
	// for a turn ends with Finalize() — the vendor will flush any
	// residual buffer and produce the closing PCM frames.
	Speak(text string) error

	// Finalize signals end-of-text. After Finalize, no more Speak
	// calls; the PCM channel closes once the vendor finishes
	// producing audio.
	Finalize() error

	// PCM returns a receive-only channel of synthesised PCM frames
	// at StreamConfig.SampleRate. Closed when synthesis completes
	// or Close is called.
	PCM() <-chan []byte

	// Close cancels any in-flight synthesis and releases vendor
	// resources. Idempotent.
	Close() error
}
