package session

import "testing"

func TestManagerCreateAndEnd(t *testing.T) {
	m := &Manager{
		cfg: Config{WebSocketPath: "ws", WebRTCOfferPath: "offer"},
		m:   make(map[string]*Session),
	}
	info, err := m.Create(CreateParams{TenantID: 1, Transport: TransportWebSocket})
	if err != nil {
		t.Fatal(err)
	}
	if info.SessionID == "" {
		t.Fatal("expected session id")
	}
	if _, ok := m.Get(info.SessionID); !ok {
		t.Fatal("session not found")
	}
	m.End(info.SessionID)
	if _, ok := m.Get(info.SessionID); ok {
		t.Fatal("session should be removed")
	}
}
