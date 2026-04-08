package conversation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

// SIP 语音对话知识库 RAG：与 Web「知识库管理」中 default_qdrant_kb 等配置对齐。
// 可通过环境变量覆盖；未设置时使用下列默认值（与当前管理端填法一致）。

func sipKnowledgeEnabled() bool {
	s := strings.ToLower(strings.TrimSpace(utils.GetEnv("SIP_KB_ENABLED")))
	if s == "" {
		return true
	}
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

func sipKBTopK() int {
	s := strings.TrimSpace(utils.GetEnv("SIP_KB_TOP_K"))
	if s == "" {
		return 5
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 5
	}
	if n > 20 {
		return 20
	}
	return n
}

func sipKBNamespace() string {
	s := strings.TrimSpace(utils.GetEnv("SIP_KB_NAMESPACE"))
	if s == "" {
		return "ling_kb"
	}
	return s
}

func sipKBMaxSnippetRunes() int {
	s := strings.TrimSpace(utils.GetEnv("SIP_KB_MAX_SNIPPET_RUNES"))
	if s == "" {
		return 400
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 80 {
		return 400
	}
	if n > 2000 {
		return 2000
	}
	return n
}

func buildSIPKnowledgeHandler() (knowledge.KnowledgeHandler, error) {
	baseURL := strings.TrimSpace(utils.GetEnv("SIP_KB_QDRANT_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}
	collection := strings.TrimSpace(utils.GetEnv("SIP_KB_COLLECTION"))
	if collection == "" {
		collection = "ling_collection"
	}
	apiKey := strings.TrimSpace(utils.GetEnv("SIP_KB_API_KEY"))

	embedURL := strings.TrimSpace(utils.GetEnv("SIP_KB_EMBEDDING_URL"))
	if embedURL == "" {
		embedURL = "https://integrate.api.nvidia.com/v1/embeddings"
	}
	embedKey := strings.TrimSpace(utils.GetEnv("SIP_KB_EMBEDDING_KEY"))
	if embedKey == "" {
		// 与知识库管理页默认一致；生产请改用 SIP_KB_EMBEDDING_KEY 环境变量。
		embedKey = "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"
	}
	embedModel := strings.TrimSpace(utils.GetEnv("SIP_KB_EMBEDDING_MODEL"))
	if embedModel == "" {
		embedModel = "nvidia/nv-embed-v1"
	}

	embedder := &knowledge.NvidiaEmbedClient{
		BaseURL: embedURL,
		APIKey:  embedKey,
		Model:   embedModel,
	}
	return knowledge.New(knowledge.KnowledgeQdrant, &knowledge.FactoryOptions{
		Qdrant: &knowledge.QdrantOptions{
			BaseURL:    baseURL,
			APIKey:     apiKey,
			Collection: collection,
			Embedder:   embedder,
		},
	})
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// augmentUserTextWithKnowledge 将向量召回片段拼在用户话术前，供 LLM 单轮理解（不改变持久化的 userText）。
func augmentUserTextWithKnowledge(ctx context.Context, userText string, lg *zap.Logger) string {
	userText = strings.TrimSpace(userText)
	if userText == "" || !sipKnowledgeEnabled() {
		return userText
	}

	h, err := buildSIPKnowledgeHandler()
	if err != nil {
		if lg != nil {
			lg.Warn("sip kb: handler init failed, continue without RAG", zap.Error(err))
		}
		return userText
	}

	topK := sipKBTopK()
	results, err := h.Query(ctx, userText, &knowledge.QueryOptions{
		Namespace: sipKBNamespace(),
		TopK:      topK,
	})
	if err != nil {
		if lg != nil {
			lg.Warn("sip kb: recall failed, continue without RAG", zap.Error(err))
		}
		return userText
	}
	if len(results) == 0 {
		if lg != nil {
			lg.Debug("sip kb: no hits", zap.String("query_preview", truncateRunes(userText, 48)))
		}
		// 无命中时也给一层“话术约束”，避免模型回复“知识库未命中/未查询到”等对用户无价值的措辞。
		return "请直接回答用户问题，语气自然简洁（1-2句）；不要提及“知识库”“检索”“未查询到”“暂无相关信息”等内部过程措辞。\n用户原话：" + userText
	}

	maxR := sipKBMaxSnippetRunes()
	var b strings.Builder
	b.WriteString("以下是与用户问题相关的参考片段。回答时优先依据这些片段，并补充必要的通用建议；保持语音回复简短（1-2句）。不要提及“知识库/检索/未查询到”等内部过程。\n---\n")
	added := 0
	for i, r := range results {
		title := strings.TrimSpace(r.Record.Title)
		content := strings.TrimSpace(r.Record.Content)
		content = truncateRunes(content, maxR)
		if title == "" && content == "" {
			continue
		}
		added++
		fmt.Fprintf(&b, "[%d] ", i+1)
		if title != "" {
			fmt.Fprintf(&b, "标题：%s\n", title)
		}
		if content != "" {
			fmt.Fprintf(&b, "内容：%s\n", content)
		}
		b.WriteByte('\n')
	}
	if added == 0 {
		return userText
	}
	b.WriteString("---\n用户原话：")
	b.WriteString(userText)

	if lg != nil {
		lg.Info("sip kb: injected RAG context",
			zap.Int("hits", added),
			zap.Int("top_k", topK),
		)
	}
	return b.String()
}
