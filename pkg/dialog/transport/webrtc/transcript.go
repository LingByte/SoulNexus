package webrtc

import (
	"strings"
	"sync"

	dialogws "github.com/LingByte/SoulNexus/pkg/dialog/transport/websocket"
	"github.com/pion/webrtc/v3"
)

const maxPendingTranscriptFrames = 64

type transcriptSink struct {
	mu      sync.Mutex
	send    func([]byte) error
	sessID  string
	pending [][]byte
}

func newTranscriptSink(sessionID string) *transcriptSink {
	return &transcriptSink{sessID: sessionID}
}

func (t *transcriptSink) bindDC(dc *webrtc.DataChannel) {
	if t == nil || dc == nil {
		return
	}
	dc.OnOpen(func() {
		t.mu.Lock()
		t.send = func(b []byte) error { return dc.SendText(string(b)) }
		pending := t.pending
		t.pending = nil
		send := t.send
		t.mu.Unlock()
		if b, err := dialogws.EncodeStatus(t.sessID, "ready", "transcript channel open"); err == nil {
			_ = t.write(b)
		}
		for _, payload := range pending {
			if send != nil {
				_ = send(payload)
			}
		}
	})
	dc.OnClose(func() {
		t.mu.Lock()
		t.send = nil
		t.mu.Unlock()
	})
}

func (t *transcriptSink) write(payload []byte) error {
	if t == nil || len(payload) == 0 {
		return nil
	}
	t.mu.Lock()
	send := t.send
	if send == nil {
		if len(t.pending) < maxPendingTranscriptFrames {
			cp := make([]byte, len(payload))
			copy(cp, payload)
			t.pending = append(t.pending, cp)
		}
		t.mu.Unlock()
		return nil
	}
	t.mu.Unlock()
	return send(payload)
}

func (t *transcriptSink) sendAssistantText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if b, err := dialogws.EncodeTranscript(t.sessID, dialogws.TypeTranscriptAssistant, text, true); err == nil {
		_ = t.write(b)
	}
}
