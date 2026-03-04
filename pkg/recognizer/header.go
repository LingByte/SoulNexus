package recognizer

import (
	"bytes"
	"net/http"

	"github.com/google/uuid"
)

type RequestHeader struct {
	messageType              MessageType
	messageTypeSpecificFlags MessageTypeSpecificFlags
	serializationType        SerializationType
	compressionType          CompressionType
	reservedData             []byte
}

func (h *RequestHeader) toBytes() []byte {
	header := bytes.NewBuffer([]byte{})
	header.WriteByte(byte(ProtocolVersionV1<<4 | 1))
	header.WriteByte(byte(h.messageType<<4) | byte(h.messageTypeSpecificFlags))
	header.WriteByte(byte(h.serializationType<<4) | byte(h.compressionType))
	header.Write(h.reservedData)
	return header.Bytes()
}

func (h *RequestHeader) WithMessageType(messageType MessageType) *RequestHeader {
	h.messageType = messageType
	return h
}

func (h *RequestHeader) WithMessageTypeSpecificFlags(messageTypeSpecificFlags MessageTypeSpecificFlags) *RequestHeader {
	h.messageTypeSpecificFlags = messageTypeSpecificFlags
	return h
}

func (h *RequestHeader) WithSerializationType(serializationType SerializationType) *RequestHeader {
	h.serializationType = serializationType
	return h
}

func (h *RequestHeader) WithCompressionType(compressionType CompressionType) *RequestHeader {
	h.compressionType = compressionType
	return h
}

func (h *RequestHeader) WithReservedData(reservedData []byte) *RequestHeader {
	h.reservedData = reservedData
	return h
}

func DefaultHeader() *RequestHeader {
	return &RequestHeader{
		messageType:              MessageTypeClientFullRequest,
		messageTypeSpecificFlags: FlagPosSequence,
		serializationType:        SerializationJSON,
		compressionType:          CompressionGZIP,
		reservedData:             []byte{0x00},
	}
}

func NewAuthHeader(auth AuthConfig) http.Header {
	reqid := uuid.New().String()
	header := http.Header{}

	header.Add("X-Api-Resource-Id", auth.ResourceId)
	header.Add("X-Api-Request-Id", reqid)
	header.Add("X-Api-Access-Key", auth.AccessKey)
	header.Add("X-Api-App-Key", auth.AppKey)
	return header
}
