package server

import (
	"strings"
	"sync"
)

// TrunkCapacityTracker holds per-trunk-number concurrent counters for this process only.
// It tightens enforcement versus DB row counts alone on a single SIP worker.
// Multiple replicas do not share these counters — coordinate externally if you scale out.
type TrunkCapacityTracker struct {
	mu sync.Mutex

	inboundCount   map[uint]int
	inboundByCall  map[string]uint
	outboundCount  map[uint]int
	outboundByCall map[string]uint
}

// NewTrunkCapacityTracker constructs an empty tracker.
func NewTrunkCapacityTracker() *TrunkCapacityTracker {
	return &TrunkCapacityTracker{
		inboundCount:   make(map[uint]int),
		inboundByCall:  make(map[string]uint),
		outboundCount:  make(map[uint]int),
		outboundByCall: make(map[string]uint),
	}
}

// TryAcquireInbound reserves one inbound slot for trunkNumberID. Idempotent per callID (re-INVITE).
func (t *TrunkCapacityTracker) TryAcquireInbound(callID string, trunkNumberID uint, limit uint) bool {
	if t == nil || trunkNumberID == 0 || limit == 0 {
		return true
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.inboundByCall[callID]; exists {
		return true
	}
	if t.inboundCount[trunkNumberID] >= int(limit) {
		return false
	}
	t.inboundCount[trunkNumberID]++
	t.inboundByCall[callID] = trunkNumberID
	return true
}

// ReleaseInbound frees the slot for callID if held. Safe to call multiple times.
func (t *TrunkCapacityTracker) ReleaseInbound(callID string) {
	if t == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	trunkID, ok := t.inboundByCall[callID]
	if !ok {
		return
	}
	delete(t.inboundByCall, callID)
	if t.inboundCount[trunkID] > 0 {
		t.inboundCount[trunkID]--
	}
}

// TryAcquireOutbound reserves one outbound slot for trunkNumberID. Idempotent per callID.
func (t *TrunkCapacityTracker) TryAcquireOutbound(callID string, trunkNumberID uint, limit uint) bool {
	if t == nil || trunkNumberID == 0 || limit == 0 {
		return true
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.outboundByCall[callID]; exists {
		return true
	}
	if t.outboundCount[trunkNumberID] >= int(limit) {
		return false
	}
	t.outboundCount[trunkNumberID]++
	t.outboundByCall[callID] = trunkNumberID
	return true
}

// ReleaseOutbound frees the outbound slot for callID if held. Safe to call multiple times.
func (t *TrunkCapacityTracker) ReleaseOutbound(callID string) {
	if t == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	trunkID, ok := t.outboundByCall[callID]
	if !ok {
		return
	}
	delete(t.outboundByCall, callID)
	if t.outboundCount[trunkID] > 0 {
		t.outboundCount[trunkID]--
	}
}
