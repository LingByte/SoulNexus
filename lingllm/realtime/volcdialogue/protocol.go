package volcdialogue

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Binary framing for Volcengine Realtime Dialogue API (openspeech v3).
// Spec: https://www.volcengine.com/docs — 豆包端到端实时语音大模型

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Client / server event IDs (Realtime Dialogue).
const (
	eventStartConnection  = 1
	eventFinishConnection = 2

	eventStartSession  = 100
	eventFinishSession = 102

	eventTaskRequest = 200

	eventConnectionStarted = 50
	eventSessionStarted    = 150
	eventSessionFailed     = 153

	eventASRStarted  = 450
	eventASRResponse = 451
	eventASREnded    = 459

	eventTTSEnded = 359

	eventChatResponse = 550
	eventChatEnded    = 559

	eventDialogCommonError = 599
)

const (
	msgTypeFullClient      = 0x1
	msgTypeAudioOnlyClient = 0x2
	msgTypeFullServer      = 0x9
	msgTypeAudioOnlyServer = 0xb
	msgTypeError           = 0xf

	flagWithEvent     = 0x4
	flagPositiveSeq   = 0x1
	flagNegativeSeq   = 0x2
	serializationJSON = 0x1
	serializationRaw  = 0x0
	compressionNone   = 0x0
	compressionGzip   = 0x1
)

// frame is a parsed server or client binary message.
type frame struct {
	msgType       byte
	flags         byte
	serialization byte
	compression   byte
	event         int32
	sessionID     string
	payload       []byte
	errorCode     uint32
}

func (f *frame) isAudioServer() bool {
	return f.msgType == msgTypeAudioOnlyServer && len(f.payload) > 0
}

func gzipCompress(in []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(in)
	_ = w.Close()
	return b.Bytes()
}

func gzipDecompress(in []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func buildHeader(msgType, flags, serialization, compression byte) []byte {
	return []byte{
		0x11, // version 1, header size 4 bytes
		(msgType << 4) | (flags & 0x0f),
		(serialization << 4) | (compression & 0x0f),
		0x00,
	}
}

func marshalJSONEvent(event int32, sessionID string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return marshalFrame(msgTypeFullClient, flagWithEvent, serializationJSON, compressionNone, event, sessionID, body)
}

func marshalFrame(msgType, flags, serialization, compression byte, event int32, sessionID string, payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(buildHeader(msgType, flags, serialization, compression))
	if flags&flagWithEvent != 0 {
		if err := binary.Write(&buf, binary.BigEndian, event); err != nil {
			return nil, err
		}
		if shouldWriteSessionID(event) {
			sid := []byte(sessionID)
			if err := binary.Write(&buf, binary.BigEndian, uint32(len(sid))); err != nil {
				return nil, err
			}
			buf.Write(sid)
		}
	}
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(payload))); err != nil {
		return nil, err
	}
	buf.Write(payload)
	return buf.Bytes(), nil
}

func shouldWriteSessionID(event int32) bool {
	switch event {
	case eventStartConnection, eventFinishConnection,
		eventConnectionStarted, 51, 52:
		return false
	default:
		return true
	}
}

// marshalAudioTask sends EVENT_TASK_REQUEST with gzip-compressed PCM.
func marshalAudioTask(sessionID string, pcm []byte) ([]byte, error) {
	compressed := gzipCompress(pcm)
	var buf bytes.Buffer
	buf.Write(buildHeader(msgTypeAudioOnlyClient, flagWithEvent, serializationRaw, compressionGzip))
	if err := binary.Write(&buf, binary.BigEndian, int32(eventTaskRequest)); err != nil {
		return nil, err
	}
	sid := []byte(sessionID)
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(sid))); err != nil {
		return nil, err
	}
	buf.Write(sid)
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(compressed))); err != nil {
		return nil, err
	}
	buf.Write(compressed)
	return buf.Bytes(), nil
}

func parseFrame(data []byte) (*frame, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("volcdialogue: frame too short")
	}
	headerSize := int(data[0]&0x0f) * 4
	if len(data) < headerSize {
		return nil, fmt.Errorf("volcdialogue: incomplete header")
	}
	f := &frame{
		msgType:       data[1] >> 4,
		flags:         data[1] & 0x0f,
		serialization: data[2] >> 4,
		compression:   data[2] & 0x0f,
	}
	off := headerSize

	if f.flags&flagPositiveSeq != 0 || f.flags&flagNegativeSeq != 0 {
		if len(data) < off+4 {
			return nil, fmt.Errorf("volcdialogue: missing sequence")
		}
		off += 4
	}

	if f.msgType == msgTypeError {
		if len(data) < off+8 {
			return nil, fmt.Errorf("volcdialogue: incomplete error frame")
		}
		f.errorCode = binary.BigEndian.Uint32(data[off : off+4])
		off += 4
		size := int(binary.BigEndian.Uint32(data[off : off+4]))
		off += 4
		if len(data) < off+size {
			return nil, fmt.Errorf("volcdialogue: incomplete error payload")
		}
		f.payload = append([]byte(nil), data[off:off+size]...)
		return f, nil
	}

	if f.flags&flagWithEvent != 0 {
		if len(data) < off+4 {
			return nil, fmt.Errorf("volcdialogue: missing event id")
		}
		f.event = int32(binary.BigEndian.Uint32(data[off : off+4]))
		off += 4
		if shouldWriteSessionID(f.event) || f.msgType == msgTypeAudioOnlyServer {
			if len(data) < off+4 {
				return nil, fmt.Errorf("volcdialogue: missing session id size")
			}
			sz := int(binary.BigEndian.Uint32(data[off : off+4]))
			off += 4
			if len(data) < off+sz {
				return nil, fmt.Errorf("volcdialogue: incomplete session id")
			}
			f.sessionID = string(data[off : off+sz])
			off += sz
		}
	}

	if len(data) < off+4 {
		return nil, fmt.Errorf("volcdialogue: missing payload size")
	}
	size := int(binary.BigEndian.Uint32(data[off : off+4]))
	off += 4
	if len(data) < off+size {
		return nil, fmt.Errorf("volcdialogue: incomplete payload")
	}
	raw := data[off : off+size]
	if f.compression == compressionGzip && len(raw) > 0 {
		dec, err := gzipDecompress(raw)
		if err != nil {
			return nil, fmt.Errorf("volcdialogue: gzip decompress: %w", err)
		}
		raw = dec
	}
	f.payload = raw
	return f, nil
}

type startSessionPayload struct {
	ASR    asrPayload    `json:"asr"`
	TTS    ttsPayload    `json:"tts"`
	Dialog dialogPayload `json:"dialog"`
}

type asrPayload struct {
	Format  string         `json:"format"`
	Rate    int            `json:"rate"`
	Bits    int            `json:"bits"`
	Channel int            `json:"channel"`
	Extra   map[string]any `json:"extra,omitempty"`
}

type ttsPayload struct {
	Speaker     string         `json:"speaker"`
	AudioConfig audioConfig    `json:"audio_config"`
	Extra       map[string]any `json:"extra,omitempty"`
}

type audioConfig struct {
	Channel    int    `json:"channel"`
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

type dialogPayload struct {
	BotName           string         `json:"bot_name,omitempty"`
	SystemRole        string         `json:"system_role,omitempty"`
	SpeakingStyle     string         `json:"speaking_style,omitempty"`
	CharacterManifest string         `json:"character_manifest,omitempty"`
	Extra             map[string]any `json:"extra"`
}

type asrResponsePayload struct {
	Results []struct {
		Text      string `json:"text"`
		IsInterim bool   `json:"is_interim"`
	} `json:"results"`
}

type chatResponsePayload struct {
	Content string `json:"content"`
}

type dialogErrorPayload struct {
	StatusCode string `json:"status_code"`
	Message    string `json:"message"`
}
