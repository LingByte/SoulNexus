package media

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventBusPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eb := NewEventBus(ctx, 10, 2)
	defer eb.Close()

	var mu sync.Mutex
	received := false
	eb.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		mu.Lock()
		received = true
		mu.Unlock()
		return nil
	})

	eb.PublishPacket("test-session", &AudioPacket{}, nil)

	// Give handler time to process
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !received {
		t.Error("Event handler was not called")
	}
}

func TestEventBusSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eb := NewEventBus(ctx, 10, 2)
	defer eb.Close()

	var mu sync.Mutex
	count := 0
	eb.Subscribe(EventTypeState, func(ctx context.Context, event *MediaEvent) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})

	eb.PublishState("session-1", StateChange{State: "begin"}, nil)
	eb.PublishState("session-1", StateChange{State: "end"}, nil)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 2 {
		t.Errorf("Expected 2 events, got %d", count)
	}
}

func TestEventBusError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eb := NewEventBus(ctx, 10, 2)
	defer eb.Close()

	var mu sync.Mutex
	errorReceived := false
	eb.Subscribe(EventTypeError, func(ctx context.Context, event *MediaEvent) error {
		mu.Lock()
		errorReceived = true
		mu.Unlock()
		return nil
	})

	eb.PublishError("session-1", nil, nil)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !errorReceived {
		t.Error("Error event handler was not called")
	}
}

func TestEventBusClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eb := NewEventBus(ctx, 10, 2)

	// Close should not panic
	eb.Close()
}

func TestMediaEventTimestamp(t *testing.T) {
	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}

	if event.Timestamp.IsZero() {
		t.Error("MediaEvent.Timestamp should not be zero")
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name string
		et   EventType
		want string
	}{
		{"packet", EventTypePacket, "packet"},
		{"state", EventTypeState, "state"},
		{"error", EventTypeError, "error"},
		{"lifecycle", EventTypeLifecycle, "lifecycle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.et) != tt.want {
				t.Errorf("EventType = %v, want %v", string(tt.et), tt.want)
			}
		})
	}
}
