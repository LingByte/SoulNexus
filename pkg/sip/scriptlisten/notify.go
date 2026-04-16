// Package scriptlisten wakes campaign script waiters when a dialog turn is persisted for a call.
package scriptlisten

import "sync"

var mu sync.RWMutex
var subs = make(map[string]chan struct{})

// Subscribe registers a wake channel for callID. cancel must be called when the waiter stops.
func Subscribe(callID string) (wake <-chan struct{}, cancel func()) {
	ch := make(chan struct{}, 32)
	mu.Lock()
	subs[callID] = ch
	mu.Unlock()
	return ch, func() {
		mu.Lock()
		if cur, ok := subs[callID]; ok && cur == ch {
			delete(subs, callID)
		}
		mu.Unlock()
	}
}

// Notify signals waiters that new turn data may be available for callID (non-blocking).
func Notify(callID string) {
	if callID == "" {
		return
	}
	mu.RLock()
	ch, ok := subs[callID]
	mu.RUnlock()
	if !ok {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}
