package recognizer

import (
	"encoding/binary"
	"encoding/json"
)

type ResponsePayload struct {
	AudioInfo struct {
		Duration int `json:"duration"`
	} `json:"audio_info"`
	Result struct {
		Text       string `json:"text"`
		Utterances []struct {
			Definite  bool   `json:"definite"`
			EndTime   int    `json:"end_time"`
			StartTime int    `json:"start_time"`
			Text      string `json:"text"`
			Words     []struct {
				EndTime   int    `json:"end_time"`
				StartTime int    `json:"start_time"`
				Text      string `json:"text"`
			} `json:"words"`
		} `json:"utterances,omitempty"`
	} `json:"result"`
	Error string `json:"error,omitempty"`
}

type Response struct {
	Code            int              `json:"code"`
	Event           int              `json:"event"`
	IsLastPackage   bool             `json:"is_last_package"`
	PayloadSequence int32            `json:"payload_sequence"`
	PayloadSize     int              `json:"payload_size"`
	PayloadMsg      *ResponsePayload `json:"payload_msg"`
	Err             error
}

// ParseResponse parses the response message
func ParseResponse(msg []byte) *Response {
	var result Response

	headerSize := msg[0] & 0x0f
	messageType := MessageType(msg[1] >> 4)
	messageTypeSpecificFlags := MessageTypeSpecificFlags(msg[1] & 0x0f)
	serializationMethod := SerializationType(msg[2] >> 4)
	messageCompression := CompressionType(msg[2] & 0x0f)
	payload := msg[headerSize*4:]

	// Parse messageTypeSpecificFlags
	if messageTypeSpecificFlags&0x01 != 0 {
		result.PayloadSequence = int32(binary.BigEndian.Uint32(payload[:4]))
		payload = payload[4:]
	}
	// Check if this is the last audio result (0b0011 = 3)
	if messageTypeSpecificFlags == FlagNegWithSequence {
		result.IsLastPackage = true
	}
	if messageTypeSpecificFlags&0x04 != 0 {
		result.Event = int(binary.BigEndian.Uint32(payload[:4]))
		payload = payload[4:]
	}

	// Parse messageType
	switch messageType {
	case MessageTypeServerFullResponse:
		result.PayloadSize = int(binary.BigEndian.Uint32(payload[:4]))
		payload = payload[4:]
	case MessageTypeServerErrorResponse:
		result.Code = int(binary.BigEndian.Uint32(payload[:4]))
		result.PayloadSize = int(binary.BigEndian.Uint32(payload[4:8]))
		payload = payload[8:]
	}

	if len(payload) == 0 {
		return &result
	}

	// Decompress if needed
	if messageCompression == CompressionGZIP {
		payload = GzipDecompress(payload)
	}

	// Parse payload
	var asrResponse ResponsePayload
	switch serializationMethod {
	case SerializationJSON:
		_ = json.Unmarshal(payload, &asrResponse)
	case SerializationNone:
	}
	result.PayloadMsg = &asrResponse
	return &result
}
