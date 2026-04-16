package scriptlisten

import (
	"strings"
	"sync"
	"time"
)

var dtmfMu sync.Mutex
var dtmfPending = make(map[string]struct {
	D  string
	At time.Time
})

// NormalizeScriptDTMF returns a single DTMF symbol: 0-9, *, #.
func NormalizeScriptDTMF(d string) string {
	d = strings.TrimSpace(d)
	if d == "" {
		return ""
	}
	r := d[0]
	switch r {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '*', '#':
		return string(r)
	default:
		return ""
	}
}

// PublishDTMF records the latest keypad digit for callID (script listen) and wakes script waiters.
func PublishDTMF(callID, digit string) {
	callID = strings.TrimSpace(callID)
	d := NormalizeScriptDTMF(digit)
	if callID == "" || d == "" {
		return
	}
	dtmfMu.Lock()
	dtmfPending[callID] = struct {
		D  string
		At time.Time
	}{D: d, At: time.Now()}
	dtmfMu.Unlock()
	Notify(callID)
}

// TryConsumeDTMF returns (next_id, digit, true) when a pending digit maps digitToNext and is not before notBefore.
// Unknown digits clear the buffer so the caller can keep waiting.
func TryConsumeDTMF(callID string, notBefore time.Time, digitToNext map[string]string) (nextID string, digit string, ok bool) {
	if callID == "" || len(digitToNext) == 0 {
		return "", "", false
	}
	dtmfMu.Lock()
	defer dtmfMu.Unlock()
	p, exists := dtmfPending[callID]
	if !exists || p.D == "" {
		return "", "", false
	}
	if !notBefore.IsZero() && p.At.Before(notBefore) {
		return "", "", false
	}
	nid, mapped := digitToNext[p.D]
	if !mapped || strings.TrimSpace(nid) == "" {
		delete(dtmfPending, callID)
		return "", p.D, false
	}
	delete(dtmfPending, callID)
	return strings.TrimSpace(nid), p.D, true
}

// ClearDTMF removes any buffered digit for callID (e.g. when ASR turn wins).
func ClearDTMF(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	dtmfMu.Lock()
	delete(dtmfPending, callID)
	dtmfMu.Unlock()
}
