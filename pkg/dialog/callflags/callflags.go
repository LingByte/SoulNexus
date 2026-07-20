package callflags

import (
	"strings"
	"sync"
)

var scriptModeCalls sync.Map

// MarkScriptMode marks a session as script-mode (suppresses built-in welcome/auto TTS).
func MarkScriptMode(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	scriptModeCalls.Store(callID, true)
}

// ClearScriptMode removes script-mode mark.
func ClearScriptMode(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	scriptModeCalls.Delete(callID)
}

// IsScriptMode reports whether the session is in script mode.
func IsScriptMode(callID string) bool {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return false
	}
	_, ok := scriptModeCalls.Load(callID)
	return ok
}
