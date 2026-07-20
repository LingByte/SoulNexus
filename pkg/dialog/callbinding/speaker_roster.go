package callbinding

import (
	"strings"
	"sync"
)

var callSpeakerRosters sync.Map // callID -> []string (bound voiceprint display names)

// SetSpeakerRoster stores the assistant-bound voiceprint names for this call (pre-identify).
func SetSpeakerRoster(callID string, names []string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	cleaned := make([]string, 0, len(names))
	for _, n := range names {
		if v := strings.TrimSpace(n); v != "" {
			cleaned = append(cleaned, v)
		}
	}
	if len(cleaned) == 0 {
		callSpeakerRosters.Delete(callID)
		return
	}
	callSpeakerRosters.Store(callID, cleaned)
}

// GetSpeakerRoster returns assistant-bound speaker names for this call.
func GetSpeakerRoster(callID string) []string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	v, ok := callSpeakerRosters.Load(callID)
	if !ok {
		return nil
	}
	names, _ := v.([]string)
	return names
}

// ClearSpeakerRoster removes the roster for a call.
func ClearSpeakerRoster(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callSpeakerRosters.Delete(callID)
}
