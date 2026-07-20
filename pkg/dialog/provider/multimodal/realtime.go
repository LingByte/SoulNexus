// Package multimodal defines the realtime (full-duplex) multimodal
// provider interface used by the realtime dialog engine. Examples:
// Qwen-Omni, GPT-4o realtime, Doubao realtime.
//
// Unlike the cascaded ASR/LLM/TTS triad, realtime providers consume
// PCM directly and produce both AI audio (PCM) and tool calls. The
// engine becomes a thin pass-through.
package multimodal

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/provider/llm"
)

// Provider is the factory abstraction for one realtime vendor.
type Provider interface {
	Name() string

	// Open starts a full-duplex realtime session.
	Open(ctx context.Context, cfg SessionConfig) (Session, error)
}

// SessionConfig is the per-call realtime configuration.
type SessionConfig struct {
	Model        string
	Voice        string
	Language     string
	SampleRate   int
	SystemPrompt string

	// Tools is the function-call tool schema (shared shape with
	// pkg/dialog/provider/llm) — realtime models that don't support
	// function calling ignore this field.
	Tools []llm.ToolSchema
}

// Session is an open full-duplex realtime session.
type Session interface {
	// PushPCM feeds user-side audio into the model. The model
	// continuously listens; there's no explicit "end of utterance".
	PushPCM(pcm []byte) error

	// Events returns a receive-only channel of model events
	// (AI audio, tool calls, transcripts, errors). Closed when the
	// session ends.
	Events() <-chan Event

	// PushToolResult delivers the result of a tool call back to the
	// model so it can continue the conversation. id must match the
	// ToolCall.ID from a prior Event.
	PushToolResult(id string, result []byte) error

	// Interrupt asks the model to stop speaking immediately (used
	// for barge-in). Vendors that don't support interruption ignore
	// the call; the engine MUST also gate output frame emission.
	Interrupt() error

	// Close terminates the session. Idempotent.
	Close() error
}

// Event is one model-side update during a realtime session.
type Event struct {
	// Type categorises this event. Use type-switch on the payload
	// fields rather than relying on Type alone.
	Type EventType

	// AudioPCM is set for AudioOutput events (PCM at
	// SessionConfig.SampleRate).
	AudioPCM []byte

	// Transcript is set for UserTranscript / AITranscript events.
	Transcript string

	// ToolCall is set for ToolCallRequested events.
	ToolCall *llm.ToolCall

	// Err is set for Error events.
	Err error
}

// EventType enumerates event kinds.
type EventType uint8

const (
	EventUnknown EventType = iota
	EventAudioOutput
	EventUserTranscript
	EventAITranscript
	EventToolCallRequested
	EventTurnCompleted
	EventError
)
