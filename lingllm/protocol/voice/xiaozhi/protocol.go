// Package xiaozhi implements the xiaozhi-esp32 WebSocket voice protocol.
//
// Device / Browser ── xiaozhi WS ──► Server ── dialog WS ──► Dialog (LLM)
package xiaozhi

import (
	"encoding/json"
	"strings"
)

const (
	MsgHello         = "hello"
	MsgListen        = "listen"
	MsgAbort         = "abort"
	MsgPing          = "ping"
	RespHello        = "hello"
	RespPong         = "pong"
	RespSTT          = "stt"
	RespTTS          = "tts"
	RespError        = "error"
	RespAbortConfirm = "abort"
)

const (
	ListenStart  = "start"
	ListenStop   = "stop"
	ListenDetect = "detect"
)

const (
	AudioFormatOpus = "opus"
	AudioFormatPCM  = "pcm"
)

const (
	// ModePipeline: ASR + dialog-plane WebSocket (LLM) + TTS.
	ModePipeline = "pipeline"
	// ModeRealtime: realtime multimodal agent (e.g. Qwen-Omni).
	ModeRealtime = "realtime"
)

type AudioParams struct {
	Format        string `json:"format,omitempty"`
	Codec         string `json:"codec,omitempty"`
	SampleRate    int    `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	FrameDuration int    `json:"frame_duration,omitempty"`
	BitDepth      int    `json:"bit_depth,omitempty"`
}

type HelloMessage struct {
	Type        string                 `json:"type"`
	Version     int                    `json:"version,omitempty"`
	Transport   string                 `json:"transport,omitempty"`
	Features    map[string]interface{} `json:"features,omitempty"`
	AudioParams *AudioParams           `json:"audio_params,omitempty"`
	Mode        string                 `json:"mode,omitempty"`
}

func DefaultHelloAudio() AudioParams {
	return AudioParams{
		Format:        AudioFormatOpus,
		SampleRate:    16000,
		Channels:      1,
		FrameDuration: 60,
		BitDepth:      16,
	}
}

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

type ListenMessage struct {
	Type  string `json:"type"`
	State string `json:"state"`
	Mode  string `json:"mode,omitempty"`
}

func ParseTextFrame(raw []byte) (string, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return "", err
	}
	return strings.TrimSpace(head.Type), nil
}

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

func MakeSTTReply(sessionID, text string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":       RespSTT,
		"text":       text,
		"session_id": sessionID,
	})
	return b
}

func MakeLLMReply(text string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type": "llm_response",
		"text": text,
	})
	return b
}

func MakeTTSStateReply(sessionID, state, codec string) []byte {
	return MakeTTSStateReplyFrames(sessionID, state, codec, 60)
}

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

func MakePongReply(sessionID string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":       RespPong,
		"session_id": sessionID,
	})
	return b
}

func MakeAbortConfirm(sessionID string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":       RespAbortConfirm,
		"state":      "confirmed",
		"session_id": sessionID,
	})
	return b
}

func MakeError(message string, fatal bool) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type":    RespError,
		"message": message,
		"fatal":   fatal,
	})
	return b
}

func normalizeMode(m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	if m == "" {
		return ""
	}
	switch m {
	case ModeRealtime:
		return ModeRealtime
	default:
		return ModePipeline
	}
}
