package knowledge

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"go.uber.org/zap"
)

const defaultRecallTopK = 3

var (
	bindingResolverMu sync.RWMutex
	bindingResolver   func(callID string) Binding
	recallerMu        sync.RWMutex
	recaller          func(ctx context.Context, collection, query string, topK int) ([]Hit, error)
	strategyMu        sync.RWMutex
	retrieveStrategy  func() string

	callDIDMu sync.RWMutex
	callDID   = map[string]string{}

	bindingCache sync.Map

	searchCacheTTL = 5 * time.Minute
	searchCache    sync.Map
)

type searchCacheEntry struct {
	searchQuery string
	resultJSON  string
	at          time.Time
}

// SetBindingResolver installs per-call knowledge binding lookup.
func SetBindingResolver(fn func(callID string) Binding) {
	bindingResolverMu.Lock()
	bindingResolver = fn
	bindingResolverMu.Unlock()
}

// SetRecaller installs recall via vector/keyword/hybrid backend.
func SetRecaller(fn func(ctx context.Context, collection, query string, topK int) ([]Hit, error)) {
	recallerMu.Lock()
	recaller = fn
	recallerMu.Unlock()
}

// SetRetrieveStrategy installs the active retrieve strategy label for logging.
func SetRetrieveStrategy(fn func() string) {
	strategyMu.Lock()
	retrieveStrategy = fn
	strategyMu.Unlock()
}

func retrieveStrategyLabel() string {
	strategyMu.RLock()
	fn := retrieveStrategy
	strategyMu.RUnlock()
	if fn == nil {
		return "vector"
	}
	s := strings.ToLower(strings.TrimSpace(fn()))
	if s == "" {
		return "vector"
	}
	return s
}

// SetCallDID records the inbound called party for per-call KB binding fallback.
func SetCallDID(callID, calledDID string) {
	callID = strings.TrimSpace(callID)
	calledDID = strings.TrimSpace(calledDID)
	if callID == "" || calledDID == "" {
		return
	}
	callDIDMu.Lock()
	callDID[callID] = calledDID
	callDIDMu.Unlock()
}

// ClearCallDID removes the per-call DID fallback and related caches.
func ClearCallDID(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callDIDMu.Lock()
	delete(callDID, callID)
	callDIDMu.Unlock()
	ClearBindingCache(callID)
	ClearSearchCache(callID)
	ClearSearchConfig(callID)
	ClearQuoteState(callID)
	onClearCall(callID)
}

// CallDIDLookup returns the stored called party for a call.
func CallDIDLookup(callID string) string {
	callDIDMu.RLock()
	defer callDIDMu.RUnlock()
	return strings.TrimSpace(callDID[callID])
}

// CacheBinding pins the resolved binding for a call leg.
func CacheBinding(callID string, b Binding) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	bindingCache.Store(callID, b)
}

// BindAssistantOverlay caches assistant-scoped knowledge when configured.
func BindAssistantOverlay(callID string, env tenantcfg.VoiceEnv, searchCfg SearchConfig) {
	ns := strings.TrimSpace(env.KnowledgeCollection)
	if ns == "" {
		ns = strings.TrimSpace(env.KnowledgeNamespace)
	}
	if ns == "" {
		return
	}
	if searchCfg.TopK <= 0 {
		searchCfg.TopK = defaultRecallTopK
	}
	CacheBinding(callID, Binding{
		NamespaceID:  env.KnowledgeNamespaceID,
		Collection:   ns,
		Enabled:      true,
		AssistantID:  env.AssistantID,
		SearchConfig: searchCfg,
	})
	CacheSearchConfig(callID, searchCfg)
}

// PrepareCallKnowledgeBinding pins assistant KB from VoiceEnv, syncs call-scoped
// assistant/tenant ids for resolver fallback, and logs how binding was resolved.
func PrepareCallKnowledgeBinding(callID string, env tenantcfg.VoiceEnv, tenantID uint, searchCfg SearchConfig, lg *zap.Logger) Binding {
	callID = strings.TrimSpace(callID)
	if env.AssistantID > 0 {
		callbinding.SetAssistantID(callID, env.AssistantID)
	}
	if tenantID > 0 {
		callbinding.SetTenantID(callID, tenantID)
	}
	BindAssistantOverlay(callID, env, searchCfg)
	b := ResolveBinding(callID)
	LogBindingResolution(lg, callID, bindingSource(env, b), env, b)
	return b
}

func bindingSource(env tenantcfg.VoiceEnv, b Binding) string {
	if !b.Enabled {
		if env.AssistantID == 0 {
			return "no_assistant"
		}
		if assistantKnowledgeSlug(env) == "" {
			return "assistant_no_kb"
		}
		return "assistant_kb_lookup_failed"
	}
	if assistantKnowledgeSlug(env) != "" {
		return "assistant"
	}
	return "assistant_resolver"
}

func assistantKnowledgeSlug(env tenantcfg.VoiceEnv) string {
	if ns := strings.TrimSpace(env.KnowledgeCollection); ns != "" {
		return ns
	}
	return strings.TrimSpace(env.KnowledgeNamespace)
}

// LogBindingResolution emits structured diagnostics for KB binding checks.
func LogBindingResolution(lg *zap.Logger, callID, source string, env tenantcfg.VoiceEnv, b Binding) {
	if lg == nil {
		return
	}
	lg.Info("knowledge: binding resolved",
		zap.String("call_id", callID),
		zap.String("source", source),
		zap.Uint("assistant_id", env.AssistantID),
		zap.Uint("assistant_version_id", env.AssistantVersionID),
		zap.String("knowledge_namespace_slug", strings.TrimSpace(env.KnowledgeNamespace)),
		zap.String("knowledge_collection", strings.TrimSpace(env.KnowledgeCollection)),
		zap.Bool("enabled", b.Enabled),
		zap.Uint("namespace_id", b.NamespaceID),
		zap.String("collection", b.Collection),
		zap.Uint("bound_assistant_id", b.AssistantID),
	)
}

// ClearBindingCache removes the per-call binding cache.
func ClearBindingCache(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	bindingCache.Delete(callID)
}

// ResolveBinding returns the knowledge base bound to this call leg.
func ResolveBinding(callID string) Binding {
	callID = strings.TrimSpace(callID)
	if callID != "" {
		if v, ok := bindingCache.Load(callID); ok {
			if b, ok := v.(Binding); ok {
				return b
			}
		}
	}
	bindingResolverMu.RLock()
	fn := bindingResolver
	bindingResolverMu.RUnlock()
	if fn == nil {
		return Binding{}
	}
	b := fn(callID)
	if callID != "" && b.Enabled {
		bindingCache.Store(callID, b)
	}
	return b
}

func recall(ctx context.Context, collection, query string, topK int) ([]Hit, error) {
	recallerMu.RLock()
	fn := recaller
	recallerMu.RUnlock()
	if fn == nil {
		return nil, fmt.Errorf("knowledge: recaller not wired")
	}
	if topK <= 0 {
		topK = defaultRecallTopK
	}
	hits, err := fn(ctx, collection, query, topK)
	if err != nil {
		return nil, err
	}
	return truncateHits(hits, topK), nil
}

func truncateHits(hits []Hit, topK int) []Hit {
	if topK <= 0 {
		topK = defaultRecallTopK
	}
	if len(hits) <= topK {
		return hits
	}
	out := make([]Hit, topK)
	copy(out, hits[:topK])
	return out
}

// RecallForCall runs retrieval when the call has a bound knowledge base.
func RecallForCall(ctx context.Context, callID, query string) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	binding := ResolveBinding(callID)
	if !binding.Enabled || strings.TrimSpace(binding.Collection) == "" {
		return nil, nil
	}
	cfg := ResolveSearchConfig(callID)
	return recall(ctx, binding.Collection, query, cfg.TopK)
}

// FormatHitsJSON builds a tool-friendly payload for LLM/realtime function calling.
func FormatHitsJSON(hits []Hit) map[string]any {
	strategy := retrieveStrategyLabel()
	topK := defaultRecallTopK
	items := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		items = append(items, map[string]any{
			"title":    h.Title,
			"content":  h.Content,
			"source":   h.Source,
			"score":    h.Score,
			"recordId": h.RecordID,
			"chunkId":  h.ChunkID,
			"quoted":   h.Quoted,
		})
	}
	return map[string]any{
		"ok":       true,
		"strategy": strategy,
		"topK":     topK,
		"hitCount": len(items),
		"results":  items,
	}
}

func lookupSearchCache(callID, searchQuery string) (string, bool) {
	callID = strings.TrimSpace(callID)
	searchQuery = strings.TrimSpace(searchQuery)
	if callID == "" || searchQuery == "" {
		return "", false
	}
	v, ok := searchCache.Load(callID)
	if !ok {
		return "", false
	}
	entry, ok := v.(searchCacheEntry)
	if !ok || entry.searchQuery != searchQuery || time.Since(entry.at) > searchCacheTTL {
		return "", false
	}
	return entry.resultJSON, true
}

func storeSearchCache(callID, searchQuery, resultJSON string) {
	callID = strings.TrimSpace(callID)
	searchQuery = strings.TrimSpace(searchQuery)
	resultJSON = strings.TrimSpace(resultJSON)
	if callID == "" || searchQuery == "" || resultJSON == "" {
		return
	}
	searchCache.Store(callID, searchCacheEntry{
		searchQuery: searchQuery,
		resultJSON:  resultJSON,
		at:          time.Now(),
	})
}

// ClearSearchCache removes the per-call recall dedupe cache.
func ClearSearchCache(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	searchCache.Delete(callID)
}
