package bootstrap

import (
	"context"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
)

// SetTurnPersist registers turn persistence for voice dialog paths.
func SetTurnPersist(fn func(ctx context.Context, callID string, t turn.Turn)) {
	providers.SetTurnPersist(func(ctx context.Context, callID string, t turn.Turn) {
		prepareTurnForPersist(callID, &t)
		fn(ctx, callID, t)
	})
}


// RecordDialogTurn appends one ASR→assistant turn when persistence is wired.
func RecordDialogTurn(ctx context.Context, callID string, t turn.Turn) {
	providers.RecordTurn(ctx, callID, t)
}

func prepareTurnForPersist(callID string, t *turn.Turn) {
	if t == nil {
		return
	}
	if recs := TakePendingKnowledgeRetrievals(callID); len(recs) > 0 {
		t.KnowledgeRetrievals = recs
	}
	if strings.TrimSpace(t.LLMText) != "" {
		if len(t.KnowledgeRetrievals) > 0 {
			t.KnowledgeRetrievals = ApplyQuoteValidation(t.LLMText, t.KnowledgeRetrievals)
		}
		t.LLMText = stageknow.StripQuoteTags(t.LLMText)
	}
}

// ApplyQuoteValidation marks hits quoted when assistant output contains <quote> tags.
func ApplyQuoteValidation(assistantText string, recs []turn.KnowledgeRetrievalRecord) []turn.KnowledgeRetrievalRecord {
	quotes := stageknow.ExtractQuotes(assistantText)
	if len(quotes) == 0 {
		return recs
	}
	out := make([]turn.KnowledgeRetrievalRecord, len(recs))
	for i, rec := range recs {
		out[i] = rec
		out[i].Hits = make([]turn.KnowledgeHitRecord, len(rec.Hits))
		copy(out[i].Hits, rec.Hits)
		for j := range out[i].Hits {
			for _, q := range quotes {
				if hitMatchesQuote(out[i].Hits[j], q) {
					out[i].Hits[j].Quoted = true
					break
				}
			}
		}
	}
	return out
}

func hitMatchesQuote(h turn.KnowledgeHitRecord, quote string) bool {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return false
	}
	content := strings.TrimSpace(h.Content)
	title := strings.TrimSpace(h.Title)
	if content != "" && (strings.Contains(content, quote) || strings.Contains(quote, content)) {
		return true
	}
	if title != "" && strings.Contains(quote, title) {
		return true
	}
	return false
}
