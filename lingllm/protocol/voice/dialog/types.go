package dialog

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Package dialog defines the control-plane contract between the voice plane
// (ASR/TTS processing) and an external Dialog application that owns the
// conversation flow (LLM, tools, business logic).
//
// This package is transport-agnostic: events and commands are plain Go structs.
// A WebSocket gateway, in-process handler, or message bus adapter can serialize
// them to JSON using the json tags below.
//
// Direction:
//   - Events flow Voice → Dialog (media telemetry).
//   - Commands flow Dialog → Voice (playback / hangup control).

// EventType enumerates messages the voice plane sends to the dialog app.
type EventType string

const (
	EvCallStarted     EventType = "call.started"
	EvCallEnded       EventType = "call.ended"
	EvASRPartial      EventType = "asr.partial"
	EvASRFinal        EventType = "asr.final"
	EvASRError        EventType = "asr.error"
	EvDTMF            EventType = "dtmf"
	EvTTSStarted      EventType = "tts.started"
	EvTTSEnded        EventType = "tts.ended"
	EvTTSInterrupt    EventType = "tts.interrupt"
	EvTransferRequest EventType = "transfer.request"
)

// CommandType enumerates messages the dialog app sends to the voice plane.
type CommandType string

const (
	CmdTTSSpeak     CommandType = "tts.speak"
	CmdTTSStream    CommandType = "tts.stream"     // LLM streaming token/chunk
	CmdTTSStreamEnd CommandType = "tts.stream.end" // flush segmenter tail
	CmdTTSInterrupt CommandType = "tts.interrupt"
	CmdHangup       CommandType = "hangup"
)

// Event is the envelope the voice plane sends to the dialog app.
type Event struct {
	Type   EventType `json:"type"`
	CallID string    `json:"call_id"`

	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	Codec string `json:"codec,omitempty"`
	PCMHz int    `json:"pcm_hz,omitempty"`

	Reason string `json:"reason,omitempty"`

	Text string `json:"text,omitempty"`

	Message string `json:"message,omitempty"`
	Fatal   bool   `json:"fatal,omitempty"`

	Digit string `json:"digit,omitempty"`
	End   bool   `json:"end,omitempty"`

	UtteranceID string `json:"utterance_id,omitempty"`
	OK          bool   `json:"ok,omitempty"`

	Target string `json:"target,omitempty"`
}

// Command is the envelope the dialog app sends to the voice plane.
type Command struct {
	Type   CommandType `json:"type"`
	CallID string      `json:"call_id"`

	Text        string `json:"text,omitempty"`
	UtteranceID string `json:"utterance_id,omitempty"`
	// StreamEnd on tts.stream marks the final LLM chunk (same as tts.stream.end).
	StreamEnd bool `json:"stream_end,omitempty"`

	Reason string `json:"reason,omitempty"`

	Meta *CommandMeta `json:"meta,omitempty"`
}

// CommandMeta carries optional turn-level metadata from the dialog app.
type CommandMeta struct {
	LLMModel   string `json:"llmModel,omitempty"`
	LLMFirstMs int    `json:"llmFirstMs,omitempty"`
	LLMWallMs  int    `json:"llmWallMs,omitempty"`
	UserText   string `json:"userText,omitempty"`
}

// StartMeta describes the call at session start (emitted in call.started).
type StartMeta struct {
	From  string
	To    string
	Codec string
	PCMHz int
}

// FirstAudioEvent fires when the first downlink audio frame of an utterance is sent.
type FirstAudioEvent struct {
	UtteranceID    string
	TTSFirstByteMs int
	E2EFirstByteMs int
}

// TurnEvent is delivered to OnTurn after each tts.speak completes.
type TurnEvent struct {
	UtteranceID      string
	LLMText          string
	Meta             *CommandMeta
	DurationMs       int
	TTSFirstByteMs   int
	E2EFirstByteMs   int
	MoreSpeaksQueued bool
	OK               bool
}

// EventHandler receives voice-plane events. Implementations should return quickly;
// heavy work (LLM calls) belongs in a separate goroutine.
type EventHandler func(Event)
