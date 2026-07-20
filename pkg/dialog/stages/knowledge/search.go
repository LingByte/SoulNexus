package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils/common"
	llmknowledge "github.com/LingByte/lingllm/knowledge"
	"go.uber.org/zap"
)

const (
	SearchToolName       = "search_knowledge_base"
	SearchToolDescription = "检索绑定的企业知识库。用户询问产品、功能、价格、优惠、政策、开源扶持、隐私合规、服务器规格等业务事实时必须调用（禁止凭记忆编造）；query 填用户问题关键词或原句。寒暄、告别、确认听清、问时间/算术、联系客服时不要调用。"
	defaultSearchTimeout = 3 * time.Second
)

// SearchToolParams is shared by pipeline ChatLLM tools and realtime FC.
var SearchToolParams = json.RawMessage(`{
	"type":"object",
	"properties":{
		"query":{"type":"string","description":"检索关键词或用户问题的完整表述"}
	},
	"required":["query"],
	"additionalProperties":false
}`)

var (
	searchTimeoutMu sync.RWMutex
	searchTimeout   = defaultSearchTimeout
)

// SetSearchTimeout caps KB recall for voice/tool paths (from KNOWLEDGE_SEARCH_QUERY_TIMEOUT_SECONDS).
// On deadline the dialog continues without waiting for KB hits.
func SetSearchTimeout(d time.Duration) {
	if d <= 0 {
		d = defaultSearchTimeout
	}
	searchTimeoutMu.Lock()
	searchTimeout = d
	searchTimeoutMu.Unlock()
}

func searchTimeoutDuration() time.Duration {
	searchTimeoutMu.RLock()
	defer searchTimeoutMu.RUnlock()
	if searchTimeout <= 0 {
		return defaultSearchTimeout
	}
	return searchTimeout
}

// SearchTimeout is the hard cap for voice/tool KB recall (used by cascaded turn wait).
func SearchTimeout() time.Duration {
	return searchTimeoutDuration()
}

// SearchPromptHint is appended when a session binds a KB.
func SearchPromptHint() string {
	return "【知识库·必用】" +
		"本会话已绑定企业知识库。用户问产品、功能、价格、优惠、政策、合规、技术规格等业务事实时，必须先调用 search_knowledge_base（query=用户问题关键词或原句），再依据检索结果口语作答（30-50字）；禁止仅凭系统提示词猜测或编造。" +
		"禁止在未检索前以「不在服务范围」「无法查询外部知识库」拒绝。" +
		"检索非空时直接作答，勿追问「您想了解哪方面」。" +
		"仅检索为空或与问题无关时，可说暂未查到并建议提交工单。" +
		"单纯问候、寒暄、告别、确认听清、问时间/算术、联系客服不要调用检索。"
}

// EnrichUserText runs server-side recall and appends a compact context block.
// Recall uses the raw user utterance only — NLU/system prompt blocks stay for the LLM, not for search.
func EnrichUserText(ctx context.Context, callID, userText string, lg *zap.Logger) string {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return userText
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	// Already injected this turn (e.g. StreamReply then QueryWithOptions).
	if strings.Contains(userText, "[系统知识库检索") {
		return userText
	}
	searchQuery := UserUtteranceForSearch(userText)
	binding := ResolveBinding(callID)
	if !binding.Enabled {
		lg.Info("knowledge: enrich skipped (kb not bound)",
			zap.String("call_id", callID),
			zap.String("query_preview", previewRunes(searchQuery, 80)),
		)
		return userText
	}
	if !ShouldRunSearch(searchQuery) {
		lg.Info("knowledge: enrich skipped (not a search intent)",
			zap.String("call_id", callID),
			zap.String("query_preview", previewRunes(searchQuery, 80)),
		)
		return userText
	}
	cfg := ResolveSearchConfig(callID)
	if !cfg.AutoEnrich {
		q := normalizeQuery(searchQuery)
		if !strings.Contains(q, "知识库") && !strings.Contains(q, "检索") && !strings.Contains(q, "查询") &&
			!strings.Contains(q, "查一下") && !strings.Contains(q, "搜一下") {
			lg.Info("knowledge: enrich skipped (autoEnrich off)",
				zap.String("call_id", callID),
				zap.String("query_preview", previewRunes(searchQuery, 80)),
			)
			return userText
		}
	}
	recordUserUtterance(callID, searchQuery)
	lg.Info("knowledge: enrich start",
		zap.String("call_id", callID),
		zap.String("collection", binding.Collection),
		zap.String("query_preview", previewRunes(searchQuery, 80)),
	)
	block := SearchBlockForQuery(ctx, callID, searchQuery, lg)
	if block == "" {
		lg.Info("knowledge: enrich empty (timeout or no block)",
			zap.String("call_id", callID),
			zap.String("query_preview", previewRunes(searchQuery, 80)),
		)
		return userText
	}
	return userText + "\n\n" + block + "\n\n" + QuotePromptAddon()
}

// SearchBlockForQuery runs server-side recall and returns a compact block for prompts.
// Used only when assistant autoEnrich is enabled (pipeline); realtime omni uses the tool.
func SearchBlockForQuery(ctx context.Context, callID, query string, lg *zap.Logger) string {
	query = UserUtteranceForSearch(query)
	if !ShouldServerEnrich(callID, query) {
		return ""
	}
	resultJSON := ExecuteSearchTool(ctx, callID, map[string]any{"query": query}, lg)
	return FormatSearchBlock(resultJSON)
}

// ForceSearchBlockForQuery runs KB recall for text-dialog remediation,
// skipping AutoEnrich/ShouldServerEnrich gates but still requiring binding + non-chitchat.
func ForceSearchBlockForQuery(ctx context.Context, callID, query string, lg *zap.Logger) string {
	query = UserUtteranceForSearch(query)
	if !ResolveBinding(callID).Enabled {
		return ""
	}
	if !ShouldRunSearch(query) {
		// Plan fallback: non-empty and length > 2
		runes := []rune(strings.TrimSpace(query))
		if len(runes) <= 2 {
			return ""
		}
		if queryIsChitchat(normalizeQuery(query)) && !queryHasIntent(normalizeQuery(query)) {
			return ""
		}
	}
	resultJSON := ExecuteSearchTool(ctx, callID, map[string]any{"query": query}, lg)
	return FormatSearchBlock(resultJSON)
}

// FormatSearchBlock formats tool JSON into a prompt block.
func FormatSearchBlock(resultJSON string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(resultJSON)), &payload); err != nil {
		return "[系统知识库检索：结果解析失败]"
	}
	if timedOut, _ := payload["timedOut"].(bool); timedOut {
		// Voice path: do not inject a failure block; continue without KB context.
		return ""
	}
	ok, _ := payload["ok"].(bool)
	if !ok {
		if msg, _ := payload["error"].(string); strings.TrimSpace(msg) != "" {
			return "[系统知识库检索：失败，" + strings.TrimSpace(msg) + "]"
		}
		return "[系统知识库检索：失败]"
	}
	results, _ := payload["results"].([]any)
	if len(results) == 0 {
		return "[系统知识库检索：未命中相关内容]"
	}
	var b strings.Builder
	b.WriteString("[系统知识库检索结果·供作答参考，勿向用户宣读本标记]")
	for i, item := range results {
		m, _ := item.(map[string]any)
		title := strings.TrimSpace(fmt.Sprint(m["title"]))
		content := strings.TrimSpace(fmt.Sprint(m["content"]))
		if title == "" && content == "" {
			continue
		}
		if title == "" {
			b.WriteString(fmt.Sprintf("\n%d. %s", i+1, content))
			continue
		}
		b.WriteString(fmt.Sprintf("\n%d. %s：%s", i+1, title, content))
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "[系统知识库检索：未命中相关内容]"
	}
	return out
}

// ExecuteSearchTool runs recall via KNOWLEDGE_RETRIEVE_STRATEGY (vector|keyword|hybrid).
// Recall is hard-capped by SetSearchTimeout; on timeout returns timedOut with empty hits.
func ExecuteSearchTool(ctx context.Context, callID string, args map[string]any, lg *zap.Logger) string {
	if lg == nil {
		lg = zap.NewNop()
	}
	binding := ResolveBinding(callID)
	if !binding.Enabled || strings.TrimSpace(binding.Collection) == "" {
		lg.Info("knowledge: search skipped (not bound)", zap.String("call_id", callID))
		return toolJSON(map[string]any{"ok": false, "error": "knowledge base not bound for this call"})
	}
	cfg := ResolveSearchConfig(callID)
	rawQuery, _ := args["query"].(string)
	rawQuery = UserUtteranceForSearch(rawQuery)
	if rawQuery == "" {
		return toolJSON(map[string]any{"ok": false, "error": "query is required"})
	}
	recordUserUtterance(callID, rawQuery)
	searchQuery := CompactSearchQuery(rawQuery)
	if searchQuery == "" {
		searchQuery = rawQuery
	}
	searchQuery = enhanceQuery(callID, searchQuery, cfg)
	if cached, ok := lookupSearchCache(callID, searchQuery); ok {
		lg.Info("knowledge: search cache hit",
			zap.String("call_id", callID),
			zap.String("search_query", previewRunes(searchQuery, 80)),
		)
		return cached
	}

	// Never inherit a cancelled attach/turn ctx — voice enrich must use a fresh timeout budget.
	searchCtx, cancel := context.WithTimeout(context.Background(), searchTimeoutDuration())
	defer cancel()
	_ = ctx

	timing := &llmknowledge.QueryTiming{}
	searchCtx = llmknowledge.WithQueryTiming(searchCtx, timing)

	recallT0 := time.Now()
	var hits []Hit
	var err error
	if reused, ok := reuseHits(callID, cfg); ok {
		hits = reused
	} else {
		hits, err = recall(searchCtx, binding.Collection, searchQuery, cfg.TopK)
	}
	recallMs := time.Since(recallT0).Milliseconds()
	if err != nil {
		timedOut := errors.Is(err, context.DeadlineExceeded) || errors.Is(searchCtx.Err(), context.DeadlineExceeded)
		if timedOut {
			lg.Warn("knowledge: search timed out, skipping KB context",
				zap.String("call_id", callID),
				zap.String("strategy", retrieveStrategyLabel()),
				zap.Duration("timeout", searchTimeoutDuration()),
				zap.Int64("recall_ms", recallMs),
				zap.Int64("embed_ms", timing.EmbedMs),
				zap.Int64("qdrant_ms", timing.SearchMs),
				zap.String("query_preview", previewRunes(rawQuery, 80)),
			)
			return toolJSON(map[string]any{
				"ok":       true,
				"timedOut": true,
				"strategy": retrieveStrategyLabel(),
				"hitCount": 0,
				"results":  []any{},
			})
		}
		lg.Warn("knowledge: search failed",
			zap.String("call_id", callID),
			zap.String("strategy", retrieveStrategyLabel()),
			zap.Int64("recall_ms", recallMs),
			zap.Int64("embed_ms", timing.EmbedMs),
			zap.Int64("qdrant_ms", timing.SearchMs),
			zap.Error(err),
		)
		return toolJSON(map[string]any{"ok": false, "error": err.Error()})
	}
	minScore := cfg.MinScore
	if minScore <= 0 {
		minScore = cfg.Threshold
	}
	// Voice default Threshold 0.4 was wiping Bleve/weak vector hits; floor for tool/enrich path.
	if minScore > 0.25 {
		minScore = 0.25
	}
	beforeFilter := len(hits)
	hits = filterHits(hits, minScore)
	recordSearchHits(callID, hits)
	lg.Info("knowledge: search",
		zap.String("call_id", callID),
		zap.String("strategy", retrieveStrategyLabel()),
		zap.String("collection", binding.Collection),
		zap.Int("top_k", cfg.TopK),
		zap.Float64("min_score", minScore),
		zap.Int("hits_before_filter", beforeFilter),
		zap.Int("hit_count", len(hits)),
		zap.Int64("recall_ms", recallMs),
		zap.Int64("embed_ms", timing.EmbedMs),
		zap.Int64("qdrant_ms", timing.SearchMs),
		zap.String("query_preview", previewRunes(rawQuery, 80)),
		zap.String("search_query", previewRunes(searchQuery, 80)),
	)
	resultJSON := toolJSON(FormatHitsJSON(hits))
	storeSearchCache(callID, searchQuery, resultJSON)
	recordRetrieval(callID, rawQuery, searchQuery, recallMs, timing.EmbedMs, timing.SearchMs, hits)
	return resultJSON
}

func toolJSON(v map[string]any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"ok":false,"error":"marshal failed"}`
	}
	return string(b)
}

func previewRunes(s string, max int) string {
	return common.TruncateRunes(s, max)
}
