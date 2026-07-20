package callbinding

import (
	"strings"
	"sync"
)

var (
	callUserIDs       sync.Map // callID -> uint
	callCredentialIDs sync.Map // callID -> uint
)

// SetUserID stores the human tenant user that opened the session (0 for API key).
func SetUserID(callID string, userID uint) {
	callID = strings.TrimSpace(callID)
	if callID == "" || userID == 0 {
		return
	}
	callUserIDs.Store(callID, userID)
}

// GetUserID returns user id bound to callID (0 if unset).
func GetUserID(callID string) uint {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return 0
	}
	if v, ok := callUserIDs.Load(callID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// ClearUserID removes call-scoped user binding.
func ClearUserID(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callUserIDs.Delete(callID)
}

// SetCredentialID stores the API key credential that opened the session.
func SetCredentialID(callID string, credentialID uint) {
	callID = strings.TrimSpace(callID)
	if callID == "" || credentialID == 0 {
		return
	}
	callCredentialIDs.Store(callID, credentialID)
}

// GetCredentialID returns credential id bound to callID (0 if unset).
func GetCredentialID(callID string) uint {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return 0
	}
	if v, ok := callCredentialIDs.Load(callID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// ClearCredentialID removes call-scoped credential binding.
func ClearCredentialID(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callCredentialIDs.Delete(callID)
}
