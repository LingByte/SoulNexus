package callbinding

import (
	"strings"
	"sync"
)

var callAssistantIDs sync.Map // callID -> uint

// SetAssistantID stores the assistant selected for one call/session leg.
func SetAssistantID(callID string, assistantID uint) {
	callID = strings.TrimSpace(callID)
	if callID == "" || assistantID == 0 {
		return
	}
	callAssistantIDs.Store(callID, assistantID)
}

// GetAssistantID returns the call-scoped assistant id (0 if unset).
func GetAssistantID(callID string) uint {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return 0
	}
	if v, ok := callAssistantIDs.Load(callID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// ClearAssistantID removes call-scoped assistant binding.
func ClearAssistantID(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callAssistantIDs.Delete(callID)
}
