package dialog

import (
	"fmt"
	"strings"
	"sync"
)

// Registry stores dialogs keyed by Call-ID (single dialog per Call-ID for this minimal registry).
type Registry struct {
	mu   sync.RWMutex
	byID map[string]*Dialog
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		byID: make(map[string]*Dialog),
	}
}

// Put registers d under its Call-ID (must be non-empty).
func (r *Registry) Put(d *Dialog) error {
	if r == nil || d == nil {
		return fmt.Errorf("sip1/dialog: nil registry or dialog")
	}
	id := strings.TrimSpace(d.CallID)
	if id == "" {
		return fmt.Errorf("sip1/dialog: empty Call-ID")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byID == nil {
		r.byID = make(map[string]*Dialog)
	}
	r.byID[id] = d
	return nil
}

// Get returns the dialog for Call-ID or nil.
func (r *Registry) Get(callID string) *Dialog {
	if r == nil {
		return nil
	}
	callID = strings.TrimSpace(callID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[callID]
}

// Delete removes a dialog by Call-ID.
func (r *Registry) Delete(callID string) {
	if r == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, callID)
}
