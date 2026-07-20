package websocket

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestHubBroadcastToSubscriber(t *testing.T) {
	h := NewHub()
	var got atomic.Int32
	h.Subscribe(func(msg *Message) {
		if msg != nil && msg.Type == "workflow_log" {
			got.Add(1)
		}
	})
	h.Start()
	h.Broadcast("workflow_log", map[string]string{"ok": "1"})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got.Load() == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("handler not invoked, got=%d", got.Load())
}
