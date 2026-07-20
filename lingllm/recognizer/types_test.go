package recognizer

import (
	"net/http"
	"testing"
)

func TestProtocolHeaderSerialization(t *testing.T) {
	header := NewDefaultHeader()
	data := header.Serialize()

	if len(data) == 0 {
		t.Error("Serialize should return non-empty bytes")
	}

	// Verify header structure (4 bytes minimum)
	if len(data) < 4 {
		t.Errorf("Serialized header should be at least 4 bytes, got %d", len(data))
	}
}

func TestProtocolHeaderSetters(t *testing.T) {
	header := NewDefaultHeader()

	// Test SetMessageType
	header.SetMessageType(MessageTypeClientAudioOnlyRequest)
	if header.messageType != MessageTypeClientAudioOnlyRequest {
		t.Error("SetMessageType should update messageType")
	}

	// Test SetSerializationType
	header.SetSerializationType(SerializationJSON)
	if header.serializationType != SerializationJSON {
		t.Error("SetSerializationType should update serializationType")
	}

	// Test SetCompressionType
	header.SetCompressionType(CompressionGZIP)
	if header.compressionType != CompressionGZIP {
		t.Error("SetCompressionType should update compressionType")
	}

	// Test SetReservedData
	data := []byte{0x01, 0x02}
	header.SetReservedData(data)
	if len(header.reservedData) != 2 {
		t.Error("SetReservedData should update reservedData")
	}
}

func TestProtocolHeaderChaining(t *testing.T) {
	header := NewDefaultHeader().
		SetMessageType(MessageTypeClientAudioOnlyRequest).
		SetSerializationType(SerializationJSON).
		SetCompressionType(CompressionGZIP)

	if header == nil {
		t.Error("Method chaining should return non-nil header")
	}

	data := header.Serialize()
	if len(data) == 0 {
		t.Error("Chained header should serialize to non-empty bytes")
	}
}

func TestNewDefaultHeader(t *testing.T) {
	header := NewDefaultHeader()

	if header.messageType != MessageTypeClientFullRequest {
		t.Error("Default messageType should be MessageTypeClientFullRequest")
	}

	if header.serializationType != SerializationJSON {
		t.Error("Default serializationType should be SerializationJSON")
	}

	if header.compressionType != CompressionGZIP {
		t.Error("Default compressionType should be CompressionGZIP")
	}

	if len(header.reservedData) == 0 {
		t.Error("Default reservedData should not be empty")
	}
}

func TestBuildAuthHeader(t *testing.T) {
	auth := AuthConfig{
		ResourceId: "resource-123",
		AccessKey:  "access-key",
		AppKey:     "app-key",
	}

	header := BuildAuthHeader(auth)

	if header == nil {
		t.Error("BuildAuthHeader should return non-nil header")
	}

	// Verify header contains expected fields
	resourceID := header.Get("X-Api-Resource-Id")
	if resourceID != "resource-123" {
		t.Errorf("X-Api-Resource-Id = %s, want resource-123", resourceID)
	}

	accessKey := header.Get("X-Api-Access-Key")
	if accessKey != "access-key" {
		t.Errorf("X-Api-Access-Key = %s, want access-key", accessKey)
	}

	appKey := header.Get("X-Api-App-Key")
	if appKey != "app-key" {
		t.Errorf("X-Api-App-Key = %s, want app-key", appKey)
	}

	// Verify request ID is set
	requestID := header.Get("X-Api-Request-Id")
	if requestID == "" {
		t.Error("X-Api-Request-Id should be set")
	}
}

func TestBuildAuthHeaderType(t *testing.T) {
	auth := AuthConfig{
		ResourceId: "test",
		AccessKey:  "test",
		AppKey:     "test",
	}

	header := BuildAuthHeader(auth)

	// Verify it's an http.Header
	var _ http.Header = header
}
