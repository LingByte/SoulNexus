package callbinding

import (
	"strings"
	"sync"
)

var callTransitPoolIDs sync.Map // callID -> []uint

// SetTransitPoolIDs records which provider pools were selected for this call (platform transit metering).
func SetTransitPoolIDs(callID string, poolIDs []uint) {
	callID = strings.TrimSpace(callID)
	if callID == "" || len(poolIDs) == 0 {
		return
	}
	cp := append([]uint(nil), poolIDs...)
	callTransitPoolIDs.Store(callID, cp)
}

// GetTransitPoolIDs returns pool ids used on this call.
func GetTransitPoolIDs(callID string) []uint {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	if v, ok := callTransitPoolIDs.Load(callID); ok {
		if ids, ok := v.([]uint); ok {
			return append([]uint(nil), ids...)
		}
	}
	return nil
}

// ClearTransitPoolIDs removes transit pool bindings for a call.
func ClearTransitPoolIDs(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callTransitPoolIDs.Delete(callID)
}
