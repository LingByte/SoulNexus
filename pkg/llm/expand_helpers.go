package llm

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

import "strings"

// BuildQueryExpansionUserPrompt builds the user message for one-shot query expansion.
func BuildQueryExpansionUserPrompt(text string, maxTerms int) string {
	if maxTerms <= 0 {
		maxTerms = 8
	}
	text = strings.TrimSpace(text)
	return "对于<原始查询>,补充其上位概念、下位具体场景及相关关联词,用|分隔。\n" +
		"原始查询:" + text + "\n" +
		"要求: 输出不超过" + itoa(maxTerms) + "个词条; 只输出词条,不要解释。\n" +
		"示例: 跑步→运动|慢跑|马拉松|跑鞋|运动手环\n"
}

// ExpandedQueryFromModelAnswer parses expansion model output and joins terms with the original query.
func ExpandedQueryFromModelAnswer(original, modelOut string, maxTerms int, separator string) (expanded string, terms []string) {
	original = strings.TrimSpace(original)
	modelOut = strings.TrimSpace(modelOut)
	if modelOut == "" {
		return original, nil
	}
	terms = parseBarSeparatedTerms(modelOut)
	terms = dedupKeepOrder(terms)
	if maxTerms > 0 && len(terms) > maxTerms {
		terms = terms[:maxTerms]
	}
	if len(terms) == 0 {
		return original, nil
	}
	sep := strings.TrimSpace(separator)
	if sep == "" {
		sep = " "
	}
	return joinExpanded(original, terms, sep), terms
}

// expansionMaxTerms returns a positive max term count from options.
func expansionMaxTerms(opts *QueryOptions) int {
	if opts == nil {
		return 8
	}
	n := opts.ExpansionMaxTerms
	if n <= 0 {
		return 8
	}
	return n
}

// expansionSeparator returns the join separator for expanded terms.
func expansionSeparator(opts *QueryOptions) string {
	if opts == nil || opts.ExpansionSeparator == "" {
		return " "
	}
	return opts.ExpansionSeparator
}

// parseBarSeparatedTerms parses bar-separated terms
func parseBarSeparatedTerms(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// common patterns: "跑步→运动|慢跑..." or just "运动|慢跑..."
	if idx := strings.Index(s, "→"); idx >= 0 {
		s = s[idx+len("→"):]
	}
	// Some models may output newlines; normalize to single line.
	s = strings.ReplaceAll(s, "\n", "|")
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `\"""'`)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// joinExpanded joins original query with expanded terms
func joinExpanded(original string, terms []string, sep string) string {
	sep = strings.TrimSpace(sep)
	if sep == "" {
		sep = " "
	}
	parts := make([]string, 0, 1+len(terms))
	parts = append(parts, strings.TrimSpace(original))
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, sep)
}

// dedupKeepOrder removes duplicates while keeping order
func dedupKeepOrder(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// itoa converts int to string (avoiding strconv)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		d := n % 10
		buf = append(buf, byte('0'+d))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
