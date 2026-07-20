package rewrite

import (
	"context"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
)

// Config is assistant queryRewriter JSON.
type Config struct {
	UseRewriter   bool   `json:"useRewriter"`
	RewritePrompt string `json:"rewritePrompt"`
}

// ParseConfig decodes queryRewriter from agentConfig or dedicated JSON.
func ParseConfig(raw map[string]any) Config {
	if len(raw) == 0 {
		return Config{}
	}
	if nested, ok := raw["queryRewriter"].(map[string]any); ok {
		raw = nested
	}
	cfg := Config{}
	if v, ok := raw["useRewriter"].(bool); ok {
		cfg.UseRewriter = v
	}
	cfg.RewritePrompt = strings.TrimSpace(str(raw, "rewritePrompt"))
	return cfg
}

func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// Rewrite optionally rewrites user ASR text before LLM/knowledge search.
func Rewrite(ctx context.Context, llm providers.ChatLLM, model, callID string, cfg Config, userText string) string {
	userText = strings.TrimSpace(userText)
	if userText == "" || !cfg.UseRewriter || llm == nil {
		return userText
	}
	prompt := cfg.RewritePrompt
	if prompt == "" {
		prompt = "你是语音对话查询改写助手。将用户的口语化 ASR 文本改写为简洁、完整的检索/对话 query，保留原意，不要回答用户。只输出改写后的文本。"
	}
	combined := prompt + "\n\n请改写以下用户语句，只输出改写结果：\n" + userText
	reply, err := llm.QueryWithOptions(combined, providers.LLMQueryOptions{
		Model: model,
		// Do not pass KnowledgeCallID: rewrite prompt must not trigger KB recall.
	})
	_ = ctx
	if err != nil {
		return userText
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return userText
	}
	return reply
}
