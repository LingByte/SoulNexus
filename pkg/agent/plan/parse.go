package plan

import "strings"

func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "```json"); i >= 0 {
		s2 := s[i+len("```json"):]
		if j := strings.Index(s2, "```"); j >= 0 {
			return strings.TrimSpace(s2[:j])
		}
	}
	if i := strings.Index(s, "```"); i >= 0 {
		s2 := s[i+len("```"):]
		if j := strings.LastIndex(s2, "```"); j >= 0 {
			inner := strings.TrimSpace(s2[:j])
			if strings.HasPrefix(inner, "{") {
				return inner
			}
		}
	}
	l := strings.Index(s, "{")
	r := strings.LastIndex(s, "}")
	if l >= 0 && r > l {
		return strings.TrimSpace(s[l : r+1])
	}
	return ""
}
