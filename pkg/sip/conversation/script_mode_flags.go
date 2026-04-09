package conversation

import (
	"strings"
	"sync"
)

var sipScriptModeCalls sync.Map // call-id -> true

// MarkSIPScriptMode marks a call-id as script-mode.
// Script mode suppresses built-in welcome and auto TTS replies; script runtime controls prompts.
func MarkSIPScriptMode(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	sipScriptModeCalls.Store(callID, true)
}

// ClearSIPScriptMode removes script-mode mark for a call-id.
func ClearSIPScriptMode(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	sipScriptModeCalls.Delete(callID)
}

func isSIPScriptMode(callID string) bool {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return false
	}
	_, ok := sipScriptModeCalls.Load(callID)
	return ok
}
