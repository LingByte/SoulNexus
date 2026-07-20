package bootstrap

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
)

func init() {
	stageknow.SetRetrievalRecorder(func(callID string, rec stageknow.RetrievalRecord) {
		hitRecords := rec.Hits
		RecordPendingKnowledgeRetrieval(callID, turn.KnowledgeRetrievalRecord{
			Query:       rec.Query,
			SearchQuery: rec.SearchQuery,
			Strategy:    rec.Strategy,
			RecallMs:    rec.RecallMs,
			EmbedMs:     rec.EmbedMs,
			QdrantMs:    rec.QdrantMs,
			HitCount:    rec.HitCount,
			Hits:        hitRecords,
		})
	})
	stageknow.SetClearCallHook(func(callID string) {
		ClearPendingKnowledgeRetrievals(callID)
	})
	session.SetKnowledgeRetrievalHooks(PeekPendingKnowledgeRetrievals, TakePendingKnowledgeRetrievals)
}
