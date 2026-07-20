package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"net/http"

	"github.com/google/uuid"
)

// ComputeSampleByteCount computes the number of bytes for audio samples
// based on sample rate, bit depth, and number of channels.
// Formula: (sampleRate * bitDepth * channels) / 8
func ComputeSampleByteCount(sampleRate, bitDepth, channels int) int {
	return (sampleRate * bitDepth * channels) / 8
}

// ProtocolHeader represents the binary protocol header for ASR requests
type ProtocolHeader struct {
	messageType              MessageType
	messageTypeSpecificFlags MessageTypeSpecificFlags
	serializationType        SerializationType
	compressionType          CompressionType
	reservedData             []byte
}

// Serialize converts the header to binary format
func (h *ProtocolHeader) Serialize() []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(byte(ProtocolVersionV1<<4 | 1))
	buf.WriteByte(byte(h.messageType<<4) | byte(h.messageTypeSpecificFlags))
	buf.WriteByte(byte(h.serializationType<<4) | byte(h.compressionType))
	buf.Write(h.reservedData)
	return buf.Bytes()
}

// SetMessageType sets the message type
func (h *ProtocolHeader) SetMessageType(msgType MessageType) *ProtocolHeader {
	h.messageType = msgType
	return h
}

// SetMessageTypeFlags sets the message type specific flags
func (h *ProtocolHeader) SetMessageTypeFlags(flags MessageTypeSpecificFlags) *ProtocolHeader {
	h.messageTypeSpecificFlags = flags
	return h
}

// SetSerializationType sets the serialization type
func (h *ProtocolHeader) SetSerializationType(serType SerializationType) *ProtocolHeader {
	h.serializationType = serType
	return h
}

// SetCompressionType sets the compression type
func (h *ProtocolHeader) SetCompressionType(compType CompressionType) *ProtocolHeader {
	h.compressionType = compType
	return h
}

// SetReservedData sets the reserved data bytes
func (h *ProtocolHeader) SetReservedData(data []byte) *ProtocolHeader {
	h.reservedData = data
	return h
}

// NewDefaultHeader creates a default protocol header
func NewDefaultHeader() *ProtocolHeader {
	return &ProtocolHeader{
		messageType:              MessageTypeClientFullRequest,
		messageTypeSpecificFlags: FlagPosSequence,
		serializationType:        SerializationJSON,
		compressionType:          CompressionGZIP,
		reservedData:             []byte{0x00},
	}
}

// BuildAuthHeader creates HTTP headers for authentication
func BuildAuthHeader(auth AuthConfig) http.Header {
	reqID := uuid.New().String()
	header := http.Header{}

	header.Add("X-Api-Resource-Id", auth.ResourceId)
	header.Add("X-Api-Request-Id", reqID)
	header.Add("X-Api-Access-Key", auth.AccessKey)
	header.Add("X-Api-App-Key", auth.AppKey)
	return header
}
