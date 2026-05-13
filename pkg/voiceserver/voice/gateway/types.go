package gateway

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Package gateway defines the WebSocket control-plane wire format between the
// VoiceServer media plane and an external dialog / business-logic server.
//
// Transport:
//   - One WebSocket per call. VoiceServer is the client; the dialog app is the
//     WebSocket server.
//   - Messages are JSON objects, one per WebSocket text frame.
//   - Every message carries a `type` string and a `call_id`; other fields
//     depend on the type (see EventType / CommandType below).
//
// Direction:
//   - Events flow VoiceServer → Dialog (media-plane telemetry).
//   - Commands flow Dialog → VoiceServer (playback / hangup control).
//
// The two channels are independent and unordered relative to each other; the
// dialog app can send commands at any point after the `call.started` event.

// EventType enumerates messages sent by VoiceServer to the dialog app.
type EventType string

const (
	EvCallStarted EventType = "call.started" // inbound call accepted, MediaLeg up
	EvCallEnded   EventType = "call.ended"   // call torn down (any reason)
	EvASRPartial  EventType = "asr.partial"  // intermediate transcript
	EvASRFinal    EventType = "asr.final"    // end-of-sentence transcript
	EvASRError    EventType = "asr.error"    // recognizer error
	EvDTMF        EventType = "dtmf"         // RFC 2833 DTMF digit
	EvTTSStarted  EventType = "tts.started"  // TTS playback began for an utterance
	EvTTSEnded    EventType = "tts.ended"    // TTS playback finished (ok or error)
	// EvTTSInterrupt is emitted when VoiceServer cut a Speak short
	// because the user started talking during playback (barge-in). The
	// dialog app should treat any remaining un-spoken LLM output as
	// "user never heard this" — common handling is to abandon the
	// tail, process the upcoming asr.final as a fresh turn, and
	// optionally tell the LLM the previous reply was interrupted so
	// it can reference it in context.
	EvTTSInterrupt EventType = "tts.interrupt"
	// EvTransferRequest is emitted when the SIP peer issued a REFER
	// (call transfer) — the customer or operator is asking VoiceServer
	// to redirect the call to a third party. The Target field carries
	// the SIP URI from Refer-To (e.g. sip:agent42@pbx.example.com).
	// Dialog apps decide policy: ignore, allow, prompt the LLM to
	// confirm before transferring, dial a Web seat, ... — VoiceServer
	// auto-emits a NOTIFY 200 OK after this event is fired so the SIP
	// peer's subscription is satisfied; the actual second-leg dial is
	// the dialog plane's responsibility (typically by sending a
	// hangup command and letting the carrier do its own redirect).
	EvTransferRequest EventType = "transfer.request"
)

// CommandType enumerates messages sent by the dialog app to VoiceServer.
type CommandType string

const (
	CmdTTSSpeak     CommandType = "tts.speak"     // synthesize + play text
	CmdTTSInterrupt CommandType = "tts.interrupt" // stop current TTS (barge-in)
	CmdHangup       CommandType = "hangup"        // terminate the call
)

// Event is the envelope VoiceServer sends to the dialog app.
//
// Fields are populated per Type; unused fields are omitted from the JSON.
type Event struct {
	Type   EventType `json:"type"`
	CallID string    `json:"call_id"`

	// call.started
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	Codec string `json:"codec,omitempty"`
	PCMHz int    `json:"pcm_hz,omitempty"`

	// call.ended
	Reason string `json:"reason,omitempty"`

	// asr.partial / asr.final
	Text string `json:"text,omitempty"`

	// asr.error
	Message string `json:"message,omitempty"`
	Fatal   bool   `json:"fatal,omitempty"`

	// dtmf
	Digit string `json:"digit,omitempty"`
	End   bool   `json:"end,omitempty"`

	// tts.*
	UtteranceID string `json:"utterance_id,omitempty"`
	OK          bool   `json:"ok,omitempty"`

	// transfer.request
	// Target carries the SIP URI from the inbound REFER's Refer-To
	// header (e.g. "sip:agent42@pbx.example.com"). Empty for non-
	// transfer events.
	Target string `json:"target,omitempty"`
}

// Command is the envelope the dialog app sends to VoiceServer.
type Command struct {
	Type   CommandType `json:"type"`
	CallID string      `json:"call_id"`

	// tts.speak
	Text        string `json:"text,omitempty"`
	UtteranceID string `json:"utterance_id,omitempty"`

	// hangup
	Reason string `json:"reason,omitempty"`

	// Meta is optional dialog-side metadata about the upstream LLM that
	// produced Text. VoiceServer logs it and persists it on the dialog
	// turn row alongside the spoken text. Set by dialog apps that want
	// rich call records; safe to leave nil.
	Meta *CommandMeta `json:"meta,omitempty"`
}

// CommandMeta carries optional turn-level metadata flowing from the dialog
// app down to VoiceServer. All fields are best-effort and may be empty.
type CommandMeta struct {
	// LLMModel identifies the model that produced the reply (e.g. "qwen-max",
	// "gpt-4o", "ollama/qwen3:4b"). Persisted on the turn row.
	LLMModel string `json:"llmModel,omitempty"`
	// LLMFirstMs is time from the dialog app's user-input timestamp to the
	// first LLM token / chunk (ms). Used for latency dashboards.
	LLMFirstMs int `json:"llmFirstMs,omitempty"`
	// LLMWallMs is total LLM wall-clock time (ms) on the dialog side.
	LLMWallMs int `json:"llmWallMs,omitempty"`
	// UserText overrides the ASR final that VoiceServer would naturally
	// pair with this turn. Useful when the dialog app rewrites the user
	// input (rephrase, intent extraction) before LLM input.
	UserText string `json:"userText,omitempty"`
}
