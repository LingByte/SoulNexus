package callbinding

import (
	"strings"
	"sync"
)

var callTenantIDs sync.Map // callID -> uint

// SetTenantID stores tenant scope for a call/session leg (LLM cost, etc.).
func SetTenantID(callID string, tenantID uint) {
	callID = strings.TrimSpace(callID)
	if callID == "" || tenantID == 0 {
		return
	}
	callTenantIDs.Store(callID, tenantID)
}

// GetTenantID returns tenant id bound to callID (0 if unset).
func GetTenantID(callID string) uint {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return 0
	}
	if v, ok := callTenantIDs.Load(callID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// ClearTenantID removes call-scoped tenant binding.
func ClearTenantID(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callTenantIDs.Delete(callID)
}
