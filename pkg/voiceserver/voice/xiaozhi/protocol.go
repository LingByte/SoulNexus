// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package xiaozhi implements the xiaozhi-esp32 voice protocol on the
// VoiceServer side. The protocol is a WebSocket dialect — a small JSON
// control surface plus opus / pcm binary audio frames — originally designed
// for the xiaozhi-esp32 firmware but used unchanged by browser web clients
// (the SoulNexus reference server reuses one endpoint for both).
//
// Two client classes share this package because they speak the same wire
// format; only the negotiated audio profile differs:
//
//   - **ESP32 / hardware**  — `audio_params.format = "opus"`, 16 kHz mono,
//     60 ms frames. The device does on-board opus encode + decode.
//   - **Web browser**       — `audio_params.format = "pcm"`, 16 kHz PCM16
//     mono, 60 ms frames. The page captures via WebAudio and sends raw
//     PCM, decoding TTS PCM via an `AudioWorklet`. Browsers that ship
//     opus (most do) may also negotiate `format = "opus"`.
//
// The hello-time format negotiation is the only switch: opus encoders /
// decoders are constructed lazily and skipped entirely in PCM mode, so a
// browser session has zero opus / cgo overhead.
//
// VoiceServer's role here is media-plane only: it terminates the WebSocket,
// decodes inbound audio into PCM, runs ASR, then dials out to a dialog-
// plane WebSocket (the same `pkg/voice/gateway` protocol the SIP path uses)
// where business logic / LLM lives. Commands flowing back from dialog
// (`tts.speak` / `tts.interrupt` / `hangup`) are wrapped in the xiaozhi
// `tts:start ... <binary> ... tts:stop` envelope and written to the client.
//
// Architecture mirrors the SIP path; one dialog plane serves all transports:
//
//	Device / Browser ── xiaozhi WS ──►  VoiceServer  ── gateway WS ──►  Dialog
//	                                                                     (LLM)
//	Device / Browser ◄── audio bin ──  VoiceServer  ◄── tts.speak ──   Dialog
//
// Why a fresh package instead of reusing pkg/voice/gateway.Client?
//
// The gateway.Client is the dialog-plane outbound bridge — it streams ASR
// events and consumes tts.speak commands. We reuse it here too: the
// xiaozhi session builds a `voice.Attached{ASR, TTS}` (with the TTS Sink
// pointing at this WebSocket instead of an RTP socket) and hands it to
// gateway.NewClient. So the dialog application protocol is identical for
// SIP, ESP32, and web — `dialog-example` works for all three with zero
// changes.
package xiaozhi

import (
	"encoding/json"
	"strings"
)

// MessageType enumerates the JSON `type` field on hardware-side text frames.
// Lowercase strings exactly as the firmware sends.
const (
	MsgHello  = "hello"
	MsgListen = "listen"
	MsgAbort  = "abort"
	MsgPing   = "ping"

	// Server-emitted reply types (sent back to the device).
	RespHello        = "hello"
	RespPong         = "pong"
	RespSTT          = "stt"
	RespTTS          = "tts"
	RespError        = "error"
	RespAbortConfirm = "abort"
	RespConnected    = "connected"
)

// ListenState values inside `listen` messages.
const (
	ListenStart  = "start"
	ListenStop   = "stop"
	ListenDetect = "detect"
)

// AudioFormat values seen on hardware hello messages and TTS replies.
const (
	AudioFormatOpus = "opus"
	AudioFormatPCM  = "pcm"
)

// AudioParams describes the audio profile negotiated at hello time. The
// firmware advertises its capabilities; the server may echo back its own
// preferred params in the welcome reply.
type AudioParams struct {
	Format        string `json:"format,omitempty"`         // "opus" | "pcm"
	Codec         string `json:"codec,omitempty"`          // mirrors Format on TTS replies
	SampleRate    int    `json:"sample_rate,omitempty"`    // typically 16000
	Channels      int    `json:"channels,omitempty"`       // 1
	FrameDuration int    `json:"frame_duration,omitempty"` // ms, typically 60
	BitDepth      int    `json:"bit_depth,omitempty"`      // 16
}

// HelloMessage is the firmware's opening JSON. Fields are best-effort;
// missing values get sensible defaults from DefaultHelloAudio().
type HelloMessage struct {
	Type        string                 `json:"type"`
	Version     int                    `json:"version,omitempty"`
	Transport   string                 `json:"transport,omitempty"`
	Features    map[string]interface{} `json:"features,omitempty"`
	AudioParams *AudioParams           `json:"audio_params,omitempty"`
}

// DefaultHelloAudio returns the assumed profile when the device omits or
// partially fills audio_params. Mirrors xiaozhi-esp32 firmware defaults.
func DefaultHelloAudio() AudioParams {
	return AudioParams{
		Format:        AudioFormatOpus,
		SampleRate:    16000,
		Channels:      1,
		FrameDuration: 60,
		BitDepth:      16,
	}
}

// MergeHelloAudio fills missing fields in h from defaults. h is mutated
// in-place. Format is normalised to lowercase.
func MergeHelloAudio(h *AudioParams) {
	if h == nil {
		return
	}
	d := DefaultHelloAudio()
	h.Format = strings.ToLower(strings.TrimSpace(h.Format))
	if h.Format == "" {
		h.Format = d.Format
	}
	if h.SampleRate <= 0 {
		h.SampleRate = d.SampleRate
	}
	if h.Channels <= 0 {
		h.Channels = d.Channels
	}
	if h.FrameDuration <= 0 {
		h.FrameDuration = d.FrameDuration
	}
	if h.BitDepth <= 0 {
		h.BitDepth = d.BitDepth
	}
}

// ListenMessage signals start / stop of microphone capture.
type ListenMessage struct {
	Type  string `json:"type"`
	State string `json:"state"`          // start | stop | detect
	Mode  string `json:"mode,omitempty"` // auto | manual
}

// ParseTextFrame extracts the `type` field from a JSON text frame. Returns
// ("", err) if the payload is not a JSON object with a string `type` field.
// Used by the read loop to dispatch to typed handlers.
func ParseTextFrame(raw []byte) (string, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return "", err
	}
	return strings.TrimSpace(head.Type), nil
}

// MakeWelcomeReply builds the JSON server reply to a hardware hello. The
// reply mirrors back the agreed audio profile and a fresh session_id so
// the device can stamp it on later frames.
func MakeWelcomeReply(sessionID string, ap AudioParams) []byte {
	msg := map[string]interface{}{
		"type":         RespHello,
		"version":      1,
		"transport":    "websocket",
		"session_id":   sessionID,
		"audio_params": ap,
	}
	b, _ := json.Marshal(msg)
	return b
}

// MakeSTTReply emits an interim or final ASR text result to the device.
// xiaozhi firmware does not distinguish partial vs final on the wire; it
// just renders whatever text the server pushes. Servers typically only
// push final results to avoid flicker.
func MakeSTTReply(sessionID, text string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":       RespSTT,
		"text":       text,
		"session_id": sessionID,
	})
	return b
}

// MakeTTSStateReply signals the start / stop of a TTS playback span.
// codec ("opus" or "pcm") tells the firmware how to decode the binary
// frames that follow until the matching stop. TTS start uses 60 ms frames
// unless you call MakeTTSStateReplyFrames (PCM browser path often uses 20).
func MakeTTSStateReply(sessionID, state, codec string) []byte {
	return MakeTTSStateReplyFrames(sessionID, state, codec, 60)
}

// MakeTTSStateReplyFrames is like MakeTTSStateReply but sets audio_params.
// frame_duration_ms on tts:start so it matches the binary chunk cadence.
func MakeTTSStateReplyFrames(sessionID, state, codec string, frameMs int) []byte {
	if codec == "" {
		codec = AudioFormatOpus
	}
	body := map[string]interface{}{
		"type":       RespTTS,
		"state":      state,
		"session_id": sessionID,
	}
	if state == "start" {
		if frameMs <= 0 {
			frameMs = 60
		}
		body["audio_params"] = AudioParams{
			Codec:         codec,
			SampleRate:    16000,
			Channels:      1,
			FrameDuration: frameMs,
			BitDepth:      16,
		}
	}
	b, _ := json.Marshal(body)
	return b
}

// MakePongReply replies to a device-initiated keepalive ping.
func MakePongReply(sessionID string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":       RespPong,
		"session_id": sessionID,
	})
	return b
}

// MakeAbortConfirm acknowledges a device-initiated abort (interrupt).
func MakeAbortConfirm(sessionID string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":       RespAbortConfirm,
		"state":      "confirmed",
		"session_id": sessionID,
	})
	return b
}

// MakeError surfaces a server-side error to the device. fatal=true tells
// the firmware to drop the connection and re-handshake.
func MakeError(message string, fatal bool) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":    RespError,
		"message": message,
		"fatal":   fatal,
	})
	return b
}
