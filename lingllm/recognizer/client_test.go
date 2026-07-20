package recognizer

import (
	"testing"
)

func TestClientCreation(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Auth: AuthConfig{
			ResourceId: "test-resource",
			AccessKey:  "test-key",
			AppKey:     "test-app",
		},
	}

	client := NewClient(config)
	if client == nil {
		t.Error("NewClient should not return nil")
	}

	if client.IsClosed() {
		t.Error("New client should not be closed")
	}
}

func TestClientTimeouts(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("NewClient failed")
	}

	// Verify default timeouts are set
	if client.sendTimeout == 0 {
		t.Error("sendTimeout should be set")
	}

	if client.recvTimeout == 0 {
		t.Error("recvTimeout should be set")
	}
}

func TestClientTraceID(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
	}

	client := NewClient(config)
	traceID := client.GetTraceID()

	// Initially empty
	if traceID != "" {
		t.Errorf("Initial traceID should be empty, got %s", traceID)
	}
}

func TestClientClose(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
	}

	client := NewClient(config)

	// Mark as closed before calling Close to avoid waiting for loops
	client.mu.Lock()
	client.isClosed = true
	client.mu.Unlock()

	client.Close()

	if !client.IsClosed() {
		t.Error("Client should be closed after Close()")
	}

	// Closing again should not panic
	client.Close()
}
