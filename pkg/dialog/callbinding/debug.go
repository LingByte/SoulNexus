package callbinding

import (
	"strings"
	"sync"
)

var debugCallIDs sync.Map // callID -> struct{}

// RegisterDebugCall marks a call/session leg as assistant debug (no call-row persist).
func RegisterDebugCall(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	debugCallIDs.Store(callID, struct{}{})
}

// IsDebugCall reports whether callID is an assistant/voice debug session.
func IsDebugCall(callID string) bool {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return false
	}
	_, ok := debugCallIDs.Load(callID)
	return ok
}

// UnregisterDebugCall clears the debug marker when a session ends.
func UnregisterDebugCall(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	debugCallIDs.Delete(callID)
}
