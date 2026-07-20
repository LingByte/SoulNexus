package session

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
)

// RegisterDebugCall marks a call/session leg that must not create call rows or recordings.
func RegisterDebugCall(callID string) {
	callbinding.RegisterDebugCall(callID)
}

// IsDebugCall reports whether callID is an assistant/voice debug session.
func IsDebugCall(callID string) bool {
	return callbinding.IsDebugCall(callID)
}

// UnregisterDebugCall clears the debug marker when a session ends.
func UnregisterDebugCall(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callbinding.UnregisterDebugCall(callID)
}
