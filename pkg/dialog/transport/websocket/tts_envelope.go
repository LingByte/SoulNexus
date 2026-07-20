package websocket

import (
	"sync"
	"time"
)

// ttsEnvelope wraps binary PCM downlink with xiaozhi tts start/stop frames.
type ttsEnvelope struct {
	sessionID  string
	sampleRate int
	writeJSON  func([]byte) error
	writeBin   func([]byte) error

	mu     sync.Mutex
	active bool
	idle   *time.Timer
}

func newTTSEnvelope(sessionID string, sampleRate int, w WireWriter) *ttsEnvelope {
	return &ttsEnvelope{
		sessionID:  sessionID,
		sampleRate: sampleRate,
		writeJSON:  w.WriteJSON,
		writeBin:   w.WriteBinary,
	}
}

func (t *ttsEnvelope) WritePCM(data []byte) error {
	if t == nil || t.writeBin == nil || len(data) == 0 {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		t.active = true
		if t.writeJSON != nil {
			if b, err := EncodeTTSState(t.sessionID, TTSStart, t.sampleRate); err == nil {
				_ = t.writeJSON(b)
			}
		}
	}
	if t.idle != nil {
		t.idle.Stop()
	}
	t.idle = time.AfterFunc(180*time.Millisecond, t.stopIdle)
	return t.writeBin(data)
}

func (t *ttsEnvelope) stopIdle() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return
	}
	t.active = false
	if t.writeJSON != nil {
		if b, err := EncodeTTSState(t.sessionID, TTSStop, t.sampleRate); err == nil {
			_ = t.writeJSON(b)
		}
	}
}

// ForceStop emits tts:stop if a playback span is open (e.g. on abort).
func (t *ttsEnvelope) ForceStop() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.idle != nil {
		t.idle.Stop()
		t.idle = nil
	}
	if !t.active {
		return
	}
	t.active = false
	if t.writeJSON != nil {
		if b, err := EncodeTTSState(t.sessionID, TTSStop, t.sampleRate); err == nil {
			_ = t.writeJSON(b)
		}
	}
}
