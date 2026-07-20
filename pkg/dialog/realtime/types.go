// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

import "context"

// EventKind enumerates the high-level events the agentStage cares
// about. Mirrors the relevant subset of pkg/realtime.EventType but
// kept local so this package does not depend on pkg/realtime
// directly.
type EventKind uint8

const (
	// EventUserTranscript carries the caller's ASR transcript.
	// Final=true means the model finished hearing the utterance and
	// the text is committed.
	EventUserTranscript EventKind = iota + 1

	// EventUserSpeechStarted fires when server-side VAD detects the
	// caller began speaking. The agentStage forwards this as
	// pipeline.KindBargeIn so a downstream player drains in-flight
	// AI audio.
	EventUserSpeechStarted

	// EventUserSpeechEnded fires when server VAD detects silence.
	// Currently informational; the agentStage does not forward it.
	EventUserSpeechEnded

	// EventAssistantText carries one assistant text fragment. When
	// Final=true it marks the end of the current assistant reply.
	EventAssistantText

	// EventAssistantAudio carries one chunk of synthesised AI audio
	// at the rate negotiated in Options.OutputSampleRate.
	EventAssistantAudio

	// EventAssistantTurnEnd fires when the model finishes a reply.
	// agentStage emits a synthetic KindAITextDone on this signal if
	// EventAssistantText{Final:true} did not already deliver one.
	EventAssistantTurnEnd

	// EventSessionClose fires once when the underlying transport
	// disconnects for any reason. Recoverable closes should NOT use
	// this — only surface it on terminal teardown.
	EventSessionClose

	// EventError surfaces a transport / protocol error. Fatal=true
	// means the session is unrecoverable; the agentStage shuts down
	// cleanly when it sees one.
	EventError
)

// Event is the union type Agent.OnEvent delivers. Only the fields
// documented for each EventKind are populated; the rest are zero.
type Event struct {
	Kind  EventKind
	Text  string
	Final bool
	// Audio is the PCM16LE payload for EventAssistantAudio.
	Audio []byte
	// SampleRate is the sample rate (Hz) of Audio. Producers MUST
	// set this on EventAssistantAudio so downstream stages can
	// resample without out-of-band metadata. 0 on non-audio events.
	SampleRate int
	// Err carries the transport / protocol error for EventError.
	Err   error
	Fatal bool
	// Vendor is a free-form provider tag ("aliyun_omni", "openai",
	// …) for diagnostics. Optional.
	Vendor string
}

// Agent is the minimal full-duplex contract the agentStage needs
// from a multimodal provider. It is a strict subset of
// pkg/realtime.Agent — the fields the media adapter does not
// surface (CommitInputAudio, UpdateInstructions) intentionally do
// not appear here. Future stages that need them can extend Agent
// behind an additional optional interface so this core stays small.
//
// Concurrency contract:
//
//   - PushAudio is called from a single goroutine (the agentStage's
//     input fan-out) — implementations MAY be non-thread-safe on
//     this method.
//   - Cancel and Close MUST be safe to call from any goroutine at
//     any time; both MUST be idempotent.
type Agent interface {
	// Start opens the underlying transport and arms event delivery.
	// It MUST return only after the session is ready (or with an
	// error). The agentStage will not push audio before Start
	// returns.
	Start(ctx context.Context) error

	// PushAudio appends one PCM16LE chunk at the rate negotiated at
	// agent construction time. Returning an error halts the audio
	// feed for the rest of the call but does not by itself tear the
	// stage down — the next agent error event (if any) will.
	PushAudio(pcm []byte) error

	// Cancel asks the model to stop the in-flight reply (barge-in).
	// MUST NOT block on the model's acknowledgement; agentStage
	// pairs Cancel with a local audio drain so the user is not left
	// hearing the cancelled tail.
	Cancel() error

	// Close tears the agent down. Idempotent.
	Close() error
}

// EventSink is the channel-shaped consumer the agentStage exposes to
// Agent implementations. Adapters wire their callback to push
// translated Events through this sink.
//
// Producers MUST NOT block on the sink — the agentStage drains it
// from a dedicated goroutine, but if that goroutine is paused (e.g.
// downstream pipeline is draining) producers should drop instead of
// stalling the agent's read loop.
type EventSink interface {
	// Emit delivers one event. Returns false when the sink is
	// closed; producers should treat that as a signal to stop
	// emitting (typically because the call ended).
	Emit(ev Event) bool
}

// AgentBuilder constructs an Agent bound to the supplied EventSink.
// The returned Agent's event callback MUST forward through the
// sink. Production wiring implements AgentBuilder by
// closing over tenant credentials and calling
// pkg/realtime.NewAgentFromCredential with an OnEvent that adapts to
// Emit. Tests inject in-memory builders that drive the sink
// synchronously.
type AgentBuilder interface {
	Build(sink EventSink) (Agent, error)
}

// AgentBuilderFunc adapts a plain function to AgentBuilder. Useful
// for tests and one-shot wiring.
type AgentBuilderFunc func(sink EventSink) (Agent, error)

// Build calls f.
func (f AgentBuilderFunc) Build(sink EventSink) (Agent, error) {
	return f(sink)
}
