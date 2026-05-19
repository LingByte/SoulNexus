package realtime

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Realtime multimodal Agent — provider-agnostic abstraction.
//
// What lives here:
// ----------------
// A single interface (`Agent`) that wraps any vendor's full-duplex realtime
// voice model. Caller PCM goes in, AI PCM/text/transcription events come out
// over the same WebSocket. This is a peer abstraction to:
//
//   pkg/recognizer    — ASR only
//   pkg/llm           — text-in / text-out chat
//   pkg/synthesizer   — TTS only
//
// The realtime layer collapses ASR+LLM+TTS into a single end-to-end stream
// (Qwen-Omni realtime, GPT-4o realtime, Gemini Live, …). Plugging it into any
// of the existing three layers is a category mistake — the SIP voice
// pipeline picks `pipeline` (3-layer) or `realtime` (this layer) at attach
// time; the two paths coexist and remain independently testable.
//
// Lifecycle:
//
//   agent, _ := realtime.NewAgentFromCredential(cfg, opts)
//   _ = agent.Start(ctx)              // opens WS, sends session.update
//   for { agent.PushAudio(pcm16le) }  // caller PCM 16k mono
//   agent.Cancel()                    // optional barge-in: stop current AI reply
//   agent.Close()                     // teardown
//
// All callbacks fire from the WS read goroutine. Implementations MUST NOT
// block in callbacks; SIP attach sites push events into channels handled by
// dedicated workers.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// EventType enumerates the high-level events surfaced to callers. Provider
// implementations are responsible for translating their wire protocol into
// this enum so SIP integration code is provider-agnostic.
type EventType string

const (
	// EventSessionOpen fires once after a successful WS handshake +
	// session.update ack. Used by SIP to mark "ready to forward audio".
	EventSessionOpen EventType = "session.open"
	// EventSessionClose fires once when the WS closes for any reason.
	EventSessionClose EventType = "session.close"

	// EventUserTranscript carries the caller's ASR transcript. `Final=true`
	// means the model finished hearing this utterance.
	EventUserTranscript EventType = "user.transcript"
	// EventUserSpeechStarted fires when the server-side VAD detects the
	// caller began speaking. SIP must immediately stop forwarding any
	// in-flight AI PCM to the call leg (barge-in).
	EventUserSpeechStarted EventType = "user.speech.started"
	// EventUserSpeechEnded fires when the server-side VAD detects the
	// caller stopped speaking.
	EventUserSpeechEnded EventType = "user.speech.ended"

	// EventAssistantText carries an AI text fragment. `Final=true` ends
	// the current assistant response. Used by SIP for hangup-phrase /
	// transfer keyword detection (Omni realtime has no tool calling).
	EventAssistantText EventType = "assistant.text"
	// EventAssistantAudio carries a chunk of AI audio at OutputSampleRate.
	EventAssistantAudio EventType = "assistant.audio"
	// EventAssistantTurnEnd fires when the model finishes one full reply.
	EventAssistantTurnEnd EventType = "assistant.turn.end"

	// EventError surfaces a server-reported error. `Fatal=true` means the
	// session is unrecoverable and SIP should fall back to playback.
	EventError EventType = "error"
)

// Event is the union type passed to Options.OnEvent. Only the fields
// relevant to the EventType are populated; the rest are zero-value.
type Event struct {
	Type    EventType
	Text    string
	Final   bool
	AudioPC []byte // PCM16LE mono @ Options.OutputSampleRate
	Err     error
	Fatal   bool
	// Vendor is the provider slug ("aliyun_omni", …) for logging.
	Vendor string
	// Raw carries the unparsed event JSON for diagnostic logs (optional;
	// implementations may leave nil to avoid the allocation).
	Raw []byte
}

// Options configure a single realtime session. The caller-facing PCM
// contract is fixed at PCM16LE mono; rates may differ per provider but
// 16 kHz in / 24 kHz out is the de-facto standard (Qwen-Omni, GPT-4o).
type Options struct {
	// SystemPrompt is sent as the model's `instructions` (Qwen-Omni) or
	// `instructions` field on session.update (GPT-4o realtime).
	SystemPrompt string
	// Voice selects the AI voice. Provider-specific values (Ethan / Cherry
	// for Qwen, alloy / nova for GPT-4o).
	Voice string
	// InputSampleRate is the sample rate of PCM frames passed to PushAudio.
	// Defaults to 16000 if 0.
	InputSampleRate int
	// OutputSampleRate is the rate the provider will emit audio at.
	// Defaults to 24000 if 0.
	OutputSampleRate int
	// OnEvent receives all session events. Required.
	OnEvent func(Event)
	// EnableServerVAD opts in to server-side voice activity detection.
	// When false, caller is responsible for sending response.create /
	// equivalent boundary signals (advanced; not used by SIP attach
	// today). Defaults to true.
	DisableServerVAD bool
	// Modalities selects which output streams the model emits. Empty =
	// vendor default (audio + text). The realtime SIP path always wants
	// audio, but text-only is useful for dry-run tests.
	Modalities []string
	// Temperature controls sampling randomness on the model side. 0
	// means "use vendor default" (caller didn't set it). Provider
	// implementations clamp to their supported range. Lower values
	// produce more deterministic / consistent replies which is what
	// telephony deployments usually want (script adherence > variety).
	Temperature float64
}

// Agent is the provider-agnostic full-duplex realtime client. Implementations
// are NOT required to be safe for concurrent calls into PushAudio from
// multiple goroutines — SIP attach uses a single audio-feed goroutine.
// Cancel / Close are safe to call from any goroutine at any time.
type Agent interface {
	// Start opens the underlying transport (WebSocket for current vendors)
	// and configures the session. Returns once the session is ready to
	// receive audio (or with an error before EventSessionOpen fires).
	Start(ctx context.Context) error
	// PushAudio appends one PCM16LE chunk at Options.InputSampleRate.
	// Implementations base64-encode internally as required by the wire
	// protocol. Safe to call from a single goroutine.
	PushAudio(pcm []byte) error
	// CommitInputAudio signals end-of-utterance manually. No-op when
	// server VAD is enabled (the default).
	CommitInputAudio() error
	// Cancel asks the model to stop the current reply (barge-in). The
	// EventAssistantTurnEnd will still fire so callers can drain state
	// machines uniformly.
	Cancel() error
	// Close tears the session down. Idempotent.
	Close() error
}

// Provider is a credential-driven factory. cfg is the parsed JSON the
// tenant control plane stored (must include "provider"). Each provider's
// expected fields are documented next to its registration call.
type Provider func(cfg map[string]any, opts Options) (Agent, error)

// --- Registry ---------------------------------------------------------------

var (
	registryMu sync.RWMutex
	registry   = map[string]Provider{}
)

// Register installs a Provider under one or more provider slugs. Slugs are
// normalised to lowercase. Re-registering a slug overwrites — useful for
// tests; production code should call Register exactly once per provider in
// init() of its sub-package.
func Register(provider Provider, slugs ...string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, s := range slugs {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		registry[s] = provider
	}
}

// Lookup returns the Provider registered under slug, or nil.
func Lookup(slug string) Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[strings.ToLower(strings.TrimSpace(slug))]
}

// RegisteredProviders returns the sorted list of registered slugs (handy
// for /api/tenant/voice/providers introspection).
func RegisteredProviders() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}

// --- Errors -----------------------------------------------------------------

// ErrAgentClosed is returned by PushAudio / Cancel after Close has run.
var ErrAgentClosed = errors.New("realtime: agent already closed")

// ErrUnknownProvider is returned by NewAgentFromCredential when the
// "provider" field doesn't match any Register'ed slug.
type ErrUnknownProvider struct {
	Provider string
}

func (e *ErrUnknownProvider) Error() string {
	return fmt.Sprintf("realtime: unknown provider %q (registered: %v)",
		e.Provider, RegisteredProviders())
}
