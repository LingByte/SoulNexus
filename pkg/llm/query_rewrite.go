package llm

import "strings"

// BuildQueryRewriteUserPrompt builds a one-shot user message for LLM-based query rewrite.
func BuildQueryRewriteUserPrompt(userQuery string, extraInstruction string) string {
	userQuery = strings.TrimSpace(userQuery)
	extra := strings.TrimSpace(extraInstruction)
	base := "你是查询改写助手。将用户的表述改写为更适合检索与问答的一条中文短句：保留原意，去掉口语废话，不要回答问题，不要分点，不要解释。\n原始输入：" +
		userQuery +
		"\n只输出改写后的一行句子。\n"
	if extra != "" {
		return base + "额外要求：" + extra + "\n"
	}
	return base
}

// NormalizeRewrittenQuery trims fences/quotes and keeps the first line of model output.
func NormalizeRewrittenQuery(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimPrefix(s, "plaintext")
		s = strings.TrimPrefix(s, "text")
		s = strings.TrimSpace(s)
		if i := strings.Index(s, "```"); i >= 0 {
			s = strings.TrimSpace(s[:i])
		}
	}
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		switch {
		case first == '"' && last == '"':
			s = strings.TrimSpace(s[1 : len(s)-1])
		case first == '\'' && last == '\'':
			s = strings.TrimSpace(s[1 : len(s)-1])
		case strings.HasPrefix(s, "「") && strings.HasSuffix(s, "」") && len([]rune(s)) >= 2:
			rs := []rune(s)
			s = strings.TrimSpace(string(rs[1 : len(rs)-1]))
		}
	}
	return strings.TrimSpace(s)
}

// queryRewriteModel picks the model name for the rewrite one-shot call.
func queryRewriteModel(opts *QueryOptions, fallback string) string {
	if opts == nil {
		return strings.TrimSpace(fallback)
	}
	if m := strings.TrimSpace(opts.QueryRewriteModel); m != "" {
		return m
	}
	if m := strings.TrimSpace(opts.Model); m != "" {
		return m
	}
	return strings.TrimSpace(fallback)
}
