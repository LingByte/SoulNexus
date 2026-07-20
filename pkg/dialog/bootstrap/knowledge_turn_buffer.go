package bootstrap

import (
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
)

var pendingKnowledgeRetrievals sync.Map // callID -> *[]turn.KnowledgeRetrievalRecord

// RecordPendingKnowledgeRetrieval buffers KB hits for the next dialog turn persist (non-blocking).
func RecordPendingKnowledgeRetrieval(callID string, rec turn.KnowledgeRetrievalRecord) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	v, _ := pendingKnowledgeRetrievals.LoadOrStore(callID, &[]turn.KnowledgeRetrievalRecord{})
	slicePtr := v.(*[]turn.KnowledgeRetrievalRecord)
	*slicePtr = append(*slicePtr, rec)
}

// PeekPendingKnowledgeRetrievals returns a copy without clearing (debug metrics).
func PeekPendingKnowledgeRetrievals(callID string) []turn.KnowledgeRetrievalRecord {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	v, ok := pendingKnowledgeRetrievals.Load(callID)
	if !ok {
		return nil
	}
	slicePtr := v.(*[]turn.KnowledgeRetrievalRecord)
	out := make([]turn.KnowledgeRetrievalRecord, len(*slicePtr))
	copy(out, *slicePtr)
	return out
}

// TakePendingKnowledgeRetrievals returns and clears buffered KB hits for a call leg.
func TakePendingKnowledgeRetrievals(callID string) []turn.KnowledgeRetrievalRecord {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	v, ok := pendingKnowledgeRetrievals.LoadAndDelete(callID)
	if !ok {
		return nil
	}
	slicePtr := v.(*[]turn.KnowledgeRetrievalRecord)
	out := make([]turn.KnowledgeRetrievalRecord, len(*slicePtr))
	copy(out, *slicePtr)
	return out
}

// ClearPendingKnowledgeRetrievals drops buffered KB hits when a call leg ends.
func ClearPendingKnowledgeRetrievals(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	pendingKnowledgeRetrievals.Delete(callID)
}
