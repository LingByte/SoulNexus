package knowledge

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	llmknowledge "github.com/LingByte/lingllm/knowledge"
	llmrerank "github.com/LingByte/lingllm/rerank"
	llmretrieve "github.com/LingByte/lingllm/retrieve"
	llmsearch "github.com/LingByte/lingllm/search"
)

// RecallPipelineInfo describes active retrieval pipeline settings for recall debugging UI.
func (s *Service) RecallPipelineInfo() map[string]any {
	if s == nil || s.cfg == nil || s.deps == nil {
		return nil
	}
	p := s.retrievalParams()
	info := map[string]any{
		"strategy":          s.RetrieveStrategy(),
		"rerankEnabled":     s.deps.reranker != nil,
		"compositeRerank":   p.EnableCompositeRerank,
		"enableMMR":         p.EnableMMR,
		"enableDedup":       p.EnableDedup,
		"rrfK":              p.RRFK,
		"rrfVectorWeight":   p.RRFVectorWeight,
		"rrfKeywordWeight":  p.RRFKeywordWeight,
		"rerankModelWeight": p.RerankModelWeight,
		"rerankBaseWeight":  p.RerankBaseWeight,
	}
	if s.deps.reranker != nil && strings.TrimSpace(s.cfg.RerankModel) != "" {
		info["rerankModel"] = s.cfg.RerankModel
	}
	return info
}

// RecallHitPayload builds the recall API item for one search hit.
func RecallHitPayload(hit llmknowledge.QueryResult) map[string]any {
	item := map[string]any{
		"id":      hit.Record.ID,
		"source":  hit.Record.Source,
		"title":   hit.Record.Title,
		"content": hit.Record.Content,
		"score":   hit.Score,
	}
	if scores := recallScoresFromMetadata(hit.Record.Metadata, hit.Score); len(scores) > 0 {
		item["scores"] = scores
	}
	return item
}

func recallScoresFromMetadata(meta map[string]any, finalScore float64) map[string]any {
	if len(meta) == 0 {
		return nil
	}
	scores := map[string]any{"final": finalScore}
	for _, pair := range []struct {
		outKey  string
		metaKey string
	}{
		{"base", "base_score"},
		{"model", "model_score"},
		{"composite", "composite_score"},
		{"rrf", "rrf_score"},
		{"vectorRrf", "vector_rrf"},
		{"keywordRrf", "keyword_rrf"},
	} {
		if v, ok := meta[pair.metaKey]; ok {
			if f, ok := utils.Float64FromAny(v); ok {
				scores[pair.outKey] = f
			}
		}
	}
	for _, pair := range []struct {
		outKey  string
		metaKey string
	}{
		{"vectorRank", "vector_rank"},
		{"keywordRank", "keyword_rank"},
	} {
		if v, ok := meta[pair.metaKey]; ok {
			if n, ok := utils.IntFromAny(v); ok {
				scores[pair.outKey] = n
			}
		}
	}
	if len(scores) <= 1 {
		return nil
	}
	return scores
}

type bleveSearchAdapter struct {
	engine llmsearch.Engine
}

func (a *bleveSearchAdapter) Search(ctx context.Context, query string, fields []string, size int) ([]llmretrieve.SearchHit, error) {
	if a == nil || a.engine == nil {
		return nil, fmt.Errorf("search engine unavailable")
	}
	if size <= 0 {
		size = 10
	}
	result, err := a.engine.Search(ctx, llmsearch.SearchRequest{
		Keyword:      query,
		SearchFields: fields,
		Size:         size,
	})
	if err != nil {
		return nil, err
	}
	out := make([]llmretrieve.SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		out = append(out, llmretrieve.SearchHit{
			ID:     hit.ID,
			Score:  hit.Score,
			Fields: hit.Fields,
		})
	}
	return out, nil
}

type vectorRetriever struct {
	handler   llmknowledge.KnowledgeHandler
	namespace string
	minScore  float64
	filters   []llmknowledge.Filter
}

func (v *vectorRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*llmretrieve.Document, error) {
	if v == nil || v.handler == nil {
		return nil, fmt.Errorf("vector handler unavailable")
	}
	hits, err := v.handler.Query(ctx, query, &llmknowledge.QueryOptions{
		Namespace: v.namespace,
		TopK:      topK,
		MinScore:  v.minScore,
		Filters:   v.filters,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*llmretrieve.Document, 0, len(hits))
	for _, hit := range hits {
		meta := map[string]string{}
		if hit.Record.Source != "" {
			meta["source"] = hit.Record.Source
		}
		if hit.Record.Title != "" {
			meta["title"] = hit.Record.Title
		}
		out = append(out, &llmretrieve.Document{
			ID:       hit.Record.ID,
			Content:  hit.Record.Content,
			Score:    hit.Score,
			Metadata: meta,
		})
	}
	return out, nil
}

func recordToSearchDoc(record llmknowledge.Record) llmsearch.Doc {
	fields := map[string]any{
		"title":   record.Title,
		"content": record.Content,
		"body":    record.Content,
		"source":  record.Source,
	}
	if len(record.Metadata) > 0 {
		fields["metadata"] = record.Metadata
		if v, ok := record.Metadata["chunk_index"]; ok {
			fields["chunk_index"] = v
		}
		if v, ok := record.Metadata["doc_id"]; ok {
			fields["doc_id"] = v
		}
		if v, ok := record.Metadata["chunk_strategy"]; ok {
			fields["chunk_strategy"] = v
		}
	}
	if len(record.Tags) > 0 {
		fields["tags"] = record.Tags
	}
	return llmsearch.Doc{
		ID:     record.ID,
		Type:   "chunk",
		Fields: fields,
	}
}

// indexSearchRecords mirrors vector upsert: same chunk IDs and text in Bleve.
func (s *Service) indexSearchRecords(ctx context.Context, collection string, records []llmknowledge.Record) error {
	if len(records) == 0 {
		return nil
	}
	if s == nil || s.deps == nil {
		return fmt.Errorf("knowledge base service is not initialized")
	}
	if s.deps.searchIndexes == nil {
		return fmt.Errorf("search index manager is not initialized")
	}
	engine, err := s.deps.searchIndexes.ForNamespace(collection)
	if err != nil {
		return err
	}
	docs := make([]llmsearch.Doc, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Content) == "" {
			continue
		}
		docs = append(docs, recordToSearchDoc(record))
	}
	if len(docs) == 0 {
		return nil
	}
	return engine.IndexBatch(ctx, docs)
}

func (s *Service) deleteSearchChunks(ctx context.Context, collection string, recordIDs []string) error {
	if s == nil || s.deps == nil || s.deps.searchIndexes == nil || len(recordIDs) == 0 {
		return nil
	}
	engine, err := s.deps.searchIndexes.ForNamespace(collection)
	if err != nil {
		return err
	}
	for _, id := range recordIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := engine.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) buildRetriever(collection string, strategy llmretrieve.Strategy, topK int, minScore float64, filters []llmknowledge.Filter, skipRerank, lean bool) (*llmretrieve.StrategyRetriever, error) {
	vec := &vectorRetriever{handler: s.deps.handler, namespace: collection, minScore: minScore, filters: filters}
	params := s.retrievalParams()
	if lean {
		params.OverRetrieveMult = 1
		params.OverRetrieveMin = topK
		if topK > 0 {
			params.OverRetrieveMax = topK
		}
	}
	if topK <= 0 {
		topK = params.RerankTopK
	}
	if topK <= 0 {
		topK = 10
	}
	cfg := llmretrieve.Config{
		Strategy:         strategy,
		Vector:           vec,
		TopK:             topK,
		MinScore:         minScore,
		VectorWeight:     params.RRFVectorWeight,
		Retrieval:        params,
		RerankCandidates: llmretrieve.OverRetrieveK(topK, params),
		KeywordFields:    []string{"title", "content", "body"},
		Observer:         s.deps.observer,
	}
	if !skipRerank {
		cfg.Reranker = newRerankAdapter(s.deps.reranker)
	}
	if s.deps.searchIndexes != nil {
		engine, err := s.deps.searchIndexes.ForNamespace(collection)
		if err == nil && engine != nil {
			cfg.Search = &bleveSearchAdapter{engine: engine}
		}
	}
	if strategy == llmretrieve.StrategyHybrid && cfg.Search == nil {
		return nil, fmt.Errorf("hybrid retrieval requires keyword index for collection %q", collection)
	}
	if strategy == llmretrieve.StrategyKeyword && cfg.Search == nil {
		return nil, fmt.Errorf("keyword retrieval requires search index for collection %q", collection)
	}
	return llmretrieve.New(cfg)
}

func (s *Service) hybridRecallFiltered(ctx context.Context, collection, query string, topK int, minScore float64, filters []llmknowledge.Filter, skipRerank, lean bool) ([]llmknowledge.QueryResult, error) {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return nil, fmt.Errorf("knowledge base service is not initialized")
	}
	retriever, err := s.buildRetriever(collection, llmretrieve.StrategyHybrid, topK, minScore, filters, skipRerank, lean)
	if err != nil {
		return s.vectorRecallFiltered(ctx, collection, query, topK, minScore, filters, skipRerank, lean)
	}
	return s.runRetriever(ctx, retriever, query, topK)
}

func (s *Service) keywordRecallFiltered(ctx context.Context, collection, query string, topK int, minScore float64, filters []llmknowledge.Filter, skipRerank, lean bool) ([]llmknowledge.QueryResult, error) {
	if s == nil || s.deps == nil || s.deps.searchIndexes == nil {
		return nil, fmt.Errorf("keyword search is not enabled")
	}
	retriever, err := s.buildRetriever(collection, llmretrieve.StrategyKeyword, topK, minScore, filters, skipRerank, lean)
	if err != nil {
		return nil, err
	}
	return s.runRetriever(ctx, retriever, query, topK)
}

func (s *Service) vectorRecallFiltered(ctx context.Context, collection, query string, topK int, minScore float64, filters []llmknowledge.Filter, skipRerank, lean bool) ([]llmknowledge.QueryResult, error) {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return nil, fmt.Errorf("knowledge base service is not initialized")
	}
	retriever, err := s.buildRetriever(collection, llmretrieve.StrategyVector, topK, minScore, filters, skipRerank, lean)
	if err != nil {
		return nil, err
	}
	return s.runRetriever(ctx, retriever, query, topK)
}

func (s *Service) runRetriever(ctx context.Context, retriever *llmretrieve.StrategyRetriever, query string, topK int) ([]llmknowledge.QueryResult, error) {
	if s != nil && s.deps != nil && s.deps.observer != nil {
		ctx = llmretrieve.WithObserver(ctx, s.deps.observer)
	}
	docs, err := retriever.Retrieve(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	out := make([]llmknowledge.QueryResult, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		meta := map[string]any{}
		for k, v := range doc.Metadata {
			meta[k] = v
		}
		out = append(out, llmknowledge.QueryResult{
			Record: llmknowledge.Record{
				ID:       doc.ID,
				Source:   doc.Metadata["source"],
				Title:    doc.Metadata["title"],
				Content:  doc.Content,
				Metadata: meta,
			},
			Score: doc.Score,
		})
	}
	return out, nil
}

type rerankAdapter struct {
	inner llmrerank.Reranker
}

func (a rerankAdapter) Rerank(ctx context.Context, query string, documents []string, topN int) ([]llmretrieve.RerankResult, error) {
	if a.inner == nil {
		return nil, nil
	}
	results, err := a.inner.Rerank(ctx, query, documents, topN)
	if err != nil {
		return nil, err
	}
	out := make([]llmretrieve.RerankResult, len(results))
	for i, r := range results {
		out[i] = llmretrieve.RerankResult{Index: r.Index, Score: r.Score}
	}
	return out, nil
}

func newRerankAdapter(r llmrerank.Reranker) llmretrieve.Reranker {
	if r == nil {
		return nil
	}
	return rerankAdapter{inner: r}
}

func (s *Service) retrievalParams() llmretrieve.RetrievalParams {
	if s == nil || s.cfg == nil {
		return llmretrieve.DefaultRetrievalParams()
	}
	c := s.cfg
	p := llmretrieve.RetrievalParams{
		EmbeddingTopK:         c.RetrievalEmbeddingTopK,
		VectorThreshold:       c.RetrievalVectorThreshold,
		KeywordThreshold:      c.RetrievalKeywordThreshold,
		RerankTopK:            c.RetrievalRerankTopK,
		RerankThreshold:       c.RetrievalRerankThreshold,
		RRFK:                  c.RetrievalRRFK,
		RRFVectorWeight:       c.RetrievalRRFVectorWeight,
		RRFKeywordWeight:      c.RetrievalRRFKeywordWeight,
		OverRetrieveMult:      c.RetrievalOverRetrieveMult,
		OverRetrieveMin:       c.RetrievalOverRetrieveMin,
		OverRetrieveMax:       c.RetrievalOverRetrieveMax,
		MMRLambda:             c.RetrievalMMRLambda,
		EnableMMR:             c.RetrievalEnableMMR,
		EnableDedup:           c.RetrievalEnableDedup,
		OverlapThreshold:      c.RetrievalOverlapThreshold,
		EnableCompositeRerank: c.RetrievalEnableCompositeRerank,
		RerankModelWeight:     c.RetrievalRerankModelWeight,
		RerankBaseWeight:      c.RetrievalRerankBaseWeight,
	}
	if p.EmbeddingTopK <= 0 {
		return llmretrieve.DefaultRetrievalParams()
	}
	return p
}

// Hit is one retrieval result used by query enhancement and analytics.
type Hit struct {
	Title    string
	Content  string
	Source   string
	Score    float64
	RecordID string
	ChunkID  uint
	Quoted   bool
}

// expandRecallParentContext swaps child hit content with parent return_content when set.
func expandRecallParentContext(hits []llmknowledge.QueryResult) []llmknowledge.QueryResult {
	if len(hits) == 0 {
		return hits
	}
	out := make([]llmknowledge.QueryResult, 0, len(hits))
	for _, hit := range hits {
		if hit.Record.Metadata != nil {
			if rc, ok := hit.Record.Metadata["return_content"]; ok {
				if text := strings.TrimSpace(rc.(string)); text != "" {
					hit.Record.Content = text
				}
			}
		}
		out = append(out, hit)
	}
	return out
}

// CallQueryContext holds per-call dialogue memory for query enhancement.
type CallQueryContext struct {
	RecentUserUtterances []string
	RecentHitRecordIDs   []string
	RecentHitContents    []string
	LastSearchAt         time.Time
}

var (
	callContextMu sync.RWMutex
	callContexts  = map[string]*CallQueryContext{}
)

// ClearCallQueryContext removes per-call memory on hangup.
func ClearCallQueryContext(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callContextMu.Lock()
	delete(callContexts, callID)
	callContextMu.Unlock()
}

func getOrCreateCallContext(callID string) *CallQueryContext {
	callContextMu.Lock()
	defer callContextMu.Unlock()
	if ctx, ok := callContexts[callID]; ok {
		return ctx
	}
	ctx := &CallQueryContext{}
	callContexts[callID] = ctx
	return ctx
}

// RecordUserUtterance stores recent user text for query rewrite.
func RecordUserUtterance(callID, text string) {
	callID = strings.TrimSpace(callID)
	text = strings.TrimSpace(text)
	if callID == "" || text == "" {
		return
	}
	ctx := getOrCreateCallContext(callID)
	callContextMu.Lock()
	defer callContextMu.Unlock()
	ctx.RecentUserUtterances = appendRecent(ctx.RecentUserUtterances, text, 6)
}

// RecordSearchHits stores recent retrieval hits for slice reuse.
func RecordSearchHits(callID string, hits []Hit) {
	callID = strings.TrimSpace(callID)
	if callID == "" || len(hits) == 0 {
		return
	}
	ctx := getOrCreateCallContext(callID)
	callContextMu.Lock()
	defer callContextMu.Unlock()
	for _, h := range hits {
		if id := strings.TrimSpace(h.RecordID); id != "" {
			ctx.RecentHitRecordIDs = appendRecent(ctx.RecentHitRecordIDs, id, 8)
		}
		if c := strings.TrimSpace(h.Content); c != "" {
			ctx.RecentHitContents = appendRecent(ctx.RecentHitContents, c, 4)
		}
	}
	ctx.LastSearchAt = time.Now()
}

func appendRecent(items []string, v string, max int) []string {
	if max <= 0 {
		max = 4
	}
	out := append(items, v)
	if len(out) > max {
		out = out[len(out)-max:]
	}
	return out
}

// EnhanceSearchQuery optionally prefixes recent user utterances for context.
// It never appends prior hit bodies — that bloated embed inputs and inflated latency.
func EnhanceSearchQuery(callID, rawQuery string, cfg SearchConfig) string {
	rawQuery = strings.TrimSpace(rawQuery)
	if rawQuery == "" {
		return rawQuery
	}
	if !cfg.UseMemoEnhanceQuery {
		return rawQuery
	}
	callContextMu.RLock()
	ctx := callContexts[strings.TrimSpace(callID)]
	callContextMu.RUnlock()
	if ctx == nil || len(ctx.RecentUserUtterances) == 0 {
		return rawQuery
	}
	parts := []string{rawQuery}
	budget := 160 // runes; keep embed payload short
	used := len([]rune(rawQuery))
	for i := len(ctx.RecentUserUtterances) - 1; i >= 0; i-- {
		u := strings.TrimSpace(ctx.RecentUserUtterances[i])
		if u == "" || u == rawQuery {
			continue
		}
		n := len([]rune(u))
		if used+n+1 > budget {
			break
		}
		parts = append([]string{u}, parts...)
		used += n + 1
		if len(parts) >= 3 {
			break
		}
	}
	return strings.Join(parts, " ")
}

// SearchConfig is per-assistant knowledge retrieval tuning.
type SearchConfig struct {
	TopK                   int
	MinScore               float64
	UseMemoEnhanceQuery    bool
	UsePreviousRoundsSlice int
	AutoEnrich             bool
	Threshold              float64
}

// DefaultSearchConfig returns platform defaults.
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		TopK:       3,
		MinScore:   0,
		Threshold:  0.4,
		AutoEnrich: false,
	}
}

// ParseSearchConfigFromAgentJSON reads knowledgeConfig from assistant agent_config JSON.
func ParseSearchConfigFromAgentJSON(raw map[string]any) SearchConfig {
	cfg := DefaultSearchConfig()
	if raw == nil {
		return cfg
	}
	kc, ok := raw["knowledgeConfig"].(map[string]any)
	if !ok {
		// flat legacy keys
		if v, ok := raw["topK"]; ok {
			cfg.TopK = utils.IntDefault(v, cfg.TopK)
		}
		if v, ok := raw["threshold"]; ok {
			cfg.Threshold = utils.Float64Default(v, cfg.Threshold)
			cfg.MinScore = cfg.Threshold
		}
		return cfg
	}
	if v, ok := kc["topK"]; ok {
		cfg.TopK = utils.IntDefault(v, cfg.TopK)
	}
	if v, ok := kc["sliceMinimumScore"]; ok {
		cfg.MinScore = utils.Float64Default(v, cfg.MinScore)
	}
	if v, ok := kc["threshold"]; ok {
		cfg.Threshold = utils.Float64Default(v, cfg.Threshold)
		if cfg.MinScore == 0 {
			cfg.MinScore = cfg.Threshold
		}
	}
	if v, ok := kc["useMemoEnhanceQuery"]; ok {
		cfg.UseMemoEnhanceQuery = utils.BoolFromAny(v)
	}
	if v, ok := kc["usePreviousRoundsSlice"]; ok {
		cfg.UsePreviousRoundsSlice = utils.IntDefault(v, 0)
	}
	if v, ok := kc["autoEnrich"]; ok {
		cfg.AutoEnrich = utils.BoolFromAny(v)
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 3
	}
	return cfg
}

// FilterHitsByMinScore drops hits below configured threshold.
func FilterHitsByMinScore(hits []Hit, minScore float64) []Hit {
	if minScore <= 0 || len(hits) == 0 {
		return hits
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		if h.Score >= minScore {
			out = append(out, h)
		}
	}
	return out
}

// ReusePreviousRoundHits returns cached hits when slice reuse is enabled and recent hits exist.
func ReusePreviousRoundHits(callID string, cfg SearchConfig) ([]Hit, bool) {
	if cfg.UsePreviousRoundsSlice <= 0 {
		return nil, false
	}
	callContextMu.RLock()
	ctx := callContexts[strings.TrimSpace(callID)]
	callContextMu.RUnlock()
	if ctx == nil || len(ctx.RecentHitContents) == 0 {
		return nil, false
	}
	n := cfg.UsePreviousRoundsSlice
	if n > len(ctx.RecentHitContents) {
		n = len(ctx.RecentHitContents)
	}
	hits := make([]Hit, 0, n)
	start := len(ctx.RecentHitContents) - n
	for i := start; i < len(ctx.RecentHitContents); i++ {
		h := Hit{Content: ctx.RecentHitContents[i], Score: 1}
		if i < len(ctx.RecentHitRecordIDs) {
			h.RecordID = ctx.RecentHitRecordIDs[i]
		}
		hits = append(hits, h)
	}
	return hits, len(hits) > 0
}

// ValidateSearchConfig sanity-checks assistant KB config.
func ValidateSearchConfig(cfg SearchConfig) error {
	if cfg.TopK < 1 || cfg.TopK > 20 {
		return fmt.Errorf("topK must be between 1 and 20")
	}
	if cfg.MinScore < 0 || cfg.MinScore > 1 {
		return fmt.Errorf("minScore must be between 0 and 1")
	}
	return nil
}
