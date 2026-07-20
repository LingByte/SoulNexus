package knowledge

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
)

// RetrievalRecord is one knowledge search for turn persistence.
type RetrievalRecord struct {
	Query       string
	SearchQuery string
	Strategy    string
	RecallMs    int64
	EmbedMs     int64
	QdrantMs    int64
	HitCount    int
	Hits        []turn.KnowledgeHitRecord
}

var (
	onRecordRetrieval func(callID string, rec RetrievalRecord)
	onClearCall       func(callID string)
)

// SetRetrievalRecorder installs turn persistence for knowledge searches.
func SetRetrievalRecorder(fn func(callID string, rec RetrievalRecord)) {
	onRecordRetrieval = fn
}

// SetClearCallHook runs when ClearCallDID is invoked (pending retrieval cleanup).
func SetClearCallHook(fn func(callID string)) {
	onClearCall = fn
}

func recordRetrieval(callID, rawQuery, searchQuery string, recallMs, embedMs, qdrantMs int64, hits []Hit) {
	if onRecordRetrieval == nil {
		return
	}
	hitRecords := make([]turn.KnowledgeHitRecord, 0, len(hits))
	for _, h := range hits {
		hitRecords = append(hitRecords, turn.KnowledgeHitRecord{
			Title:    h.Title,
			Content:  h.Content,
			Source:   h.Source,
			Score:    h.Score,
			RecordID: h.RecordID,
			ChunkID:  h.ChunkID,
			Quoted:   h.Quoted,
		})
	}
	onRecordRetrieval(callID, RetrievalRecord{
		Query:       rawQuery,
		SearchQuery: searchQuery,
		Strategy:    retrieveStrategyLabel(),
		RecallMs:    recallMs,
		EmbedMs:     embedMs,
		QdrantMs:    qdrantMs,
		HitCount:    len(hits),
		Hits:        hitRecords,
	})
}
