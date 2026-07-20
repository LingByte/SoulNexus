package websocket

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/transport/pcm"
)

const ProtocolVersion = 1

// Wire control types. Includes legacy debug UI types and xiaozhi-esp32 dialect.
const (
	TypeHello               = "hello"
	TypePCM                 = "pcm"
	TypeHangup              = "hangup"
	TypeError               = "error"
	TypeTranscriptUser      = "transcript.user"
	TypeTranscriptAssistant = "transcript.assistant"
	TypeStatus              = "status"

	// Xiaozhi client → server
	TypeListen = "listen"
	TypeAbort  = "abort"
	TypePing   = "ping"

	// Xiaozhi server → client
	TypeSTT         = "stt"
	TypeTTS         = "tts"
	TypeLLMResponse = "llm_response"
	TypePong        = "pong"
)

const (
	ListenStart = "start"
	ListenStop  = "stop"

	TTSStart = "start"
	TTSStop  = "stop"

	AudioFormatPCM = "pcm"
)

// AudioParams is the xiaozhi hello / tts audio profile.
type AudioParams struct {
	Format        string `json:"format,omitempty"`
	Codec         string `json:"codec,omitempty"`
	SampleRate    int    `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	FrameDuration int    `json:"frame_duration,omitempty"`
	BitDepth      int    `json:"bit_depth,omitempty"`
}

// Frame is the JSON wire format for control and transcript messages.
type Frame struct {
	Type        string       `json:"type"`
	V           int          `json:"v,omitempty"`
	Version     int          `json:"version,omitempty"`
	SessionID   string       `json:"sessionId,omitempty"`
	SessionID2  string       `json:"session_id,omitempty"` // xiaozhi snake_case
	SampleRate  int          `json:"sampleRateHz,omitempty"`
	Data        string       `json:"data,omitempty"`
	Message     string       `json:"message,omitempty"`
	Text        string       `json:"text,omitempty"`
	Final       bool         `json:"final,omitempty"`
	State       string       `json:"state,omitempty"`
	Mode        string       `json:"mode,omitempty"`
	Transport   string       `json:"transport,omitempty"`
	Fatal       bool         `json:"fatal,omitempty"`
	AudioParams *AudioParams `json:"audio_params,omitempty"`
}

// WireWriter sends JSON control frames and binary PCM downlink.
type WireWriter struct {
	WriteJSON   func([]byte) error
	WriteBinary func([]byte) error
}

// NewPort creates a WebSocket-backed MediaPort.
// AI PCM is sent as binary frames wrapped with xiaozhi tts start/stop envelopes.
func NewPort(cfg pcm.Config, w WireWriter) *pcm.Port {
	env := newTTSEnvelope(cfg.SessionID, cfg.SampleRate, w)
	p := pcm.NewPort(cfg)
	p.OutputFn = func(f engine.PCMFrame) error {
		if len(f.Data) == 0 {
			return nil
		}
		return env.WritePCM(f.Data)
	}
	return p
}

// DecodeBase64PCM decodes a base64 PCM payload from a legacy wire frame.
func DecodeBase64PCM(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// PushInputFromWire decodes a client JSON PCM frame into the port.
func PushInputFromWire(p *pcm.Port, fr Frame) error {
	if p == nil {
		return errors.New("websocket: nil port")
	}
	if fr.Type != TypePCM {
		return nil
	}
	raw, err := DecodeBase64PCM(fr.Data)
	if err != nil {
		return err
	}
	return p.PushInput(raw)
}

// PushInputFromBinary enqueues raw PCM16LE from a binary WebSocket frame.
func PushInputFromBinary(p *pcm.Port, raw []byte) error {
	if p == nil {
		return errors.New("websocket: nil port")
	}
	return p.PushInput(raw)
}

func defaultAudioParams(sampleRate int) AudioParams {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	return AudioParams{
		Format:        AudioFormatPCM,
		Codec:         AudioFormatPCM,
		SampleRate:    sampleRate,
		Channels:      1,
		FrameDuration: 60,
		BitDepth:      16,
	}
}

// EncodeHello builds a dual-dialect welcome (debug camelCase + xiaozhi audio_params).
func EncodeHello(sessionID string, sampleRate int) ([]byte, error) {
	ap := defaultAudioParams(sampleRate)
	return json.Marshal(Frame{
		Type:        TypeHello,
		V:           ProtocolVersion,
		Version:     ProtocolVersion,
		SessionID:   sessionID,
		SessionID2:  sessionID,
		SampleRate:  sampleRate,
		Transport:   "websocket",
		AudioParams: &ap,
	})
}

// EncodeTTSState builds a xiaozhi tts start/stop envelope.
func EncodeTTSState(sessionID, state string, sampleRate int) ([]byte, error) {
	body := Frame{
		Type:       TypeTTS,
		State:      state,
		SessionID:  sessionID,
		SessionID2: sessionID,
	}
	if state == TTSStart {
		ap := defaultAudioParams(sampleRate)
		body.AudioParams = &ap
	}
	return json.Marshal(body)
}

// EncodeSTT builds a xiaozhi stt caption frame.
func EncodeSTT(sessionID, text string) ([]byte, error) {
	return json.Marshal(Frame{
		Type:       TypeSTT,
		Text:       text,
		SessionID:  sessionID,
		SessionID2: sessionID,
	})
}

// EncodeLLMResponse builds a xiaozhi llm_response frame.
func EncodeLLMResponse(text string) ([]byte, error) {
	return json.Marshal(Frame{Type: TypeLLMResponse, Text: text})
}

// EncodeAbortConfirm acknowledges a client abort.
func EncodeAbortConfirm(sessionID string) ([]byte, error) {
	return json.Marshal(Frame{
		Type:       TypeAbort,
		State:      "confirmed",
		SessionID:  sessionID,
		SessionID2: sessionID,
	})
}

// EncodePong replies to ping.
func EncodePong(sessionID string) ([]byte, error) {
	return json.Marshal(Frame{
		Type:       TypePong,
		SessionID:  sessionID,
		SessionID2: sessionID,
	})
}

// DecodeFrame parses a JSON wire frame.
func DecodeFrame(data []byte) (Frame, error) {
	var fr Frame
	err := json.Unmarshal(data, &fr)
	if err != nil {
		return fr, err
	}
	if fr.SessionID == "" && fr.SessionID2 != "" {
		fr.SessionID = fr.SessionID2
	}
	fr.Type = strings.TrimSpace(fr.Type)
	fr.State = strings.TrimSpace(fr.State)
	return fr, nil
}

// EncodeStatus builds a status frame for the debug UI.
func EncodeStatus(sessionID, state, message string) ([]byte, error) {
	return json.Marshal(Frame{
		Type:       TypeStatus,
		V:          ProtocolVersion,
		SessionID:  sessionID,
		SessionID2: sessionID,
		State:      state,
		Message:    message,
	})
}

// EncodeTranscript builds a user/assistant transcript frame.
func EncodeTranscript(sessionID, typ, text string, final bool) ([]byte, error) {
	return json.Marshal(Frame{
		Type:       typ,
		V:          ProtocolVersion,
		SessionID:  sessionID,
		SessionID2: sessionID,
		Text:       text,
		Final:      final,
	})
}

func errorFrame(msg string) []byte {
	b, _ := json.Marshal(Frame{Type: TypeError, Message: msg, Fatal: false})
	return b
}
