package retrieve

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Retrieve runs the configured strategy with optional over-retrieval, RRF fusion, rerank, and MMR.
func (r *StrategyRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	if r == nil {
		return nil, fmt.Errorf("retrieve: nil retriever")
	}
	if topK <= 0 {
		topK = r.cfg.TopK
	}

	obs := r.activeObserver(ctx)
	meta := map[string]string{
		"strategy": string(r.cfg.Strategy),
		"top_k":    fmt.Sprintf("%d", topK),
	}
	ctx = obs.BeginRetrieval(ctx, query, meta)
	start := time.Now()

	params := r.cfg.Retrieval
	fetchK := OverRetrieveK(topK, params)
	if r.cfg.Reranker != nil {
		candidates := r.cfg.RerankCandidates
		if candidates <= 0 {
			candidates = fetchK
		}
		if candidates < fetchK {
			candidates = fetchK
		}
		fetchK = candidates
	}

	var docs []*Document
	var err error

	recallStart := time.Now()
	switch r.cfg.Strategy {
	case StrategyVector:
		docs, err = r.cfg.Vector.Retrieve(ctx, query, fetchK)
	case StrategyKeyword:
		docs, err = r.keywordSearch(ctx, query, fetchK)
	case StrategyHybrid:
		docs, err = r.hybridSearch(ctx, query, fetchK, params)
	default:
		err = fmt.Errorf("retrieve: unsupported strategy")
	}
	obs.RecordStage(ctx, StageRetrieve, StageEvent{
		DurationMs:  time.Since(recallStart).Milliseconds(),
		InputCount:  fetchK,
		OutputCount: len(docs),
		Metadata:    map[string]string{"strategy": string(r.cfg.Strategy)},
		Err:         err,
	})
	if err != nil {
		obs.EndRetrieval(ctx, nil, err)
		return nil, err
	}

	rerankStart := time.Now()
	beforeRerank := len(docs)
	docs = applyRerankPipeline(ctx, r.cfg.Reranker, query, docs, topK, params)
	if r.cfg.Reranker != nil {
		obs.RecordStage(ctx, StageRerank, StageEvent{
			DurationMs:  time.Since(rerankStart).Milliseconds(),
			InputCount:  beforeRerank,
			OutputCount: len(docs),
			Metadata: map[string]string{
				"composite": fmt.Sprintf("%v", params.EnableCompositeRerank),
			},
		})
	}

	if params.EnableDedup {
		dedupStart := time.Now()
		before := len(docs)
		docs = DeduplicateDocuments(docs)
		docs = RemovePartialOverlaps(docs, params.OverlapThreshold)
		obs.RecordStage(ctx, StageDedup, StageEvent{
			DurationMs:  time.Since(dedupStart).Milliseconds(),
			InputCount:  before,
			OutputCount: len(docs),
		})
	}
	if len(docs) > topK {
		docs = docs[:topK]
	}

	_ = start // total duration captured by EndRetrieval via observer
	obs.EndRetrieval(ctx, docs, nil)
	return docs, nil
}

// keywordSearch performs full-text search.
func (r *StrategyRetriever) keywordSearch(ctx context.Context, query string, topK int) ([]*Document, error) {
	hits, err := r.cfg.Search.Search(ctx, query, r.cfg.KeywordFields, topK)
	if err != nil {
		return nil, err
	}
	params := r.cfg.Retrieval
	minScore := r.cfg.MinScore
	if params.KeywordThreshold > minScore {
		minScore = params.KeywordThreshold
	}
	return hitsToDocuments(hits, minScore), nil
}

// hybridSearch merges vector and keyword search via RRF (WeKnora-style).
func (r *StrategyRetriever) hybridSearch(ctx context.Context, query string, topK int, params RetrievalParams) ([]*Document, error) {
	vecMin := r.cfg.MinScore
	if params.VectorThreshold > vecMin {
		vecMin = params.VectorThreshold
	}
	kwMin := r.cfg.MinScore
	if params.KeywordThreshold > kwMin {
		kwMin = params.KeywordThreshold
	}

	vecDocs, err := r.cfg.Vector.Retrieve(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	vecDocs = applyThreshold(vecDocs, vecMin)

	kwDocs, err := r.keywordSearch(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	kwDocs = applyThreshold(kwDocs, kwMin)

	merged := fuseRRF(vecDocs, kwDocs, params)
	if len(merged) > topK {
		merged = merged[:topK]
	}
	return merged, nil
}

func applyRerankPipeline(ctx context.Context, reranker Reranker, query string, docs []*Document, topK int, params RetrievalParams) []*Document {
	if reranker == nil {
		return docs
	}
	if len(docs) == 0 {
		return docs
	}
	rerankTopK := params.RerankTopK
	if rerankTopK <= 0 {
		rerankTopK = topK
	}
	if rerankTopK < topK {
		rerankTopK = topK
	}

	baseMin, baseMax := baseScoreBounds(docs)
	baseScores := make([]float64, len(docs))
	for i, d := range docs {
		if d != nil {
			baseScores[i] = d.Score
		}
	}

	texts := make([]string, len(docs))
	for i, d := range docs {
		if d == nil {
			continue
		}
		texts[i] = CleanPassageForRerank(d.Content)
	}

	results, err := reranker.Rerank(ctx, query, texts, len(docs))
	if err != nil {
		return docs
	}

	threshold := params.RerankThreshold
	filtered := filterRerankResults(results, docs, baseScores, baseMin, baseMax, threshold, params)
	if len(filtered) == 0 && len(results) > 0 && results[0].Score >= 0.15 {
		filtered = []*Document{applyCompositeClone(docs[results[0].Index], results[0].Score, baseScores[results[0].Index], baseMin, baseMax, params)}
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Score > filtered[j].Score })

	if params.EnableMMR && len(filtered) > 1 {
		filtered = ApplyMMR(filtered, rerankTopK, params.MMRLambda)
	} else if len(filtered) > rerankTopK {
		filtered = filtered[:rerankTopK]
	}
	return filtered
}

func filterRerankResults(results []RerankResult, docs []*Document, baseScores []float64, baseMin, baseMax, threshold float64, params RetrievalParams) []*Document {
	out := make([]*Document, 0, len(results))
	for _, rr := range results {
		if rr.Index < 0 || rr.Index >= len(docs) || docs[rr.Index] == nil {
			continue
		}
		if threshold > 0 && rr.Score < threshold {
			continue
		}
		base := 0.0
		if rr.Index < len(baseScores) {
			base = baseScores[rr.Index]
		}
		out = append(out, applyCompositeClone(docs[rr.Index], rr.Score, base, baseMin, baseMax, params))
	}
	return out
}

func applyCompositeClone(d *Document, modelScore, baseScore, baseMin, baseMax float64, params RetrievalParams) *Document {
	if d == nil {
		return nil
	}
	copy := *d
	if copy.Metadata == nil {
		copy.Metadata = map[string]string{}
	}
	copy.Metadata["model_score"] = fmt.Sprintf("%.4f", modelScore)
	copy.Metadata["base_score"] = fmt.Sprintf("%.4f", baseScore)
	final := modelScore
	if params.EnableCompositeRerank {
		final = CompositeRerankScore(modelScore, baseScore, baseMin, baseMax, params.RerankModelWeight, params.RerankBaseWeight)
	}
	copy.Metadata["composite_score"] = fmt.Sprintf("%.4f", final)
	copy.Score = final
	return &copy
}

func cloneDoc(d *Document, score float64) *Document {
	if d == nil {
		return nil
	}
	copy := *d
	copy.Score = score
	return &copy
}

// applyRerank applies optional reranking to documents (legacy helper).
func applyRerank(ctx context.Context, reranker Reranker, query string, docs []*Document, topK, candidates int) ([]*Document, error) {
	params := DefaultRetrievalParams()
	params.RerankTopK = topK
	out := applyRerankPipeline(ctx, reranker, query, docs, topK, params)
	return out, nil
}

// hitsToDocuments converts search hits to documents.
func hitsToDocuments(hits []SearchHit, minScore float64) []*Document {
	out := make([]*Document, 0, len(hits))
	for _, h := range hits {
		if h.Score < minScore {
			continue
		}

		content := fieldString(h.Fields, "content")
		if content == "" {
			content = fieldString(h.Fields, "body")
		}

		meta := map[string]string{}
		for _, k := range []string{"title", "source", "parent_id", "chunk_strategy"} {
			if v := fieldString(h.Fields, k); v != "" {
				meta[k] = v
			}
		}

		out = append(out, &Document{
			ID:       h.ID,
			Content:  content,
			Score:    h.Score,
			Metadata: meta,
		})
	}

	return out
}

// fieldString extracts string value from fields map.
func fieldString(fields map[string]interface{}, key string) string {
	if fields == nil {
		return ""
	}
	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	}
	return fmt.Sprint(v)
}
