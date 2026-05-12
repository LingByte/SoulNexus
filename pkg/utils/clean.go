package utils

import (
	"regexp"
	"strings"
	"unicode"
)

type Options struct {
	Lowercase            bool
	StripMarkdown        bool
	StripSymbols         bool
	DedupLines           bool
	DropBoilerplateLines bool
}

func CleanText(s string, opts *Options) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.ReplaceAll(s, "\uFEFF", "")
	s = strings.Map(func(r rune) rune {
		if r == unicode.ReplacementChar {
			return ' '
		}
		if r == 0 {
			return ' '
		}
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)

	if opts != nil && opts.Lowercase {
		s = strings.ToLower(s)
	}

	if opts != nil && opts.StripMarkdown {
		s = stripMarkdown(s)
	}

	if opts != nil && opts.StripSymbols {
		s = stripSymbols(s)
	}

	s = normalizeWhitespace(s)

	if opts != nil && (opts.DedupLines || opts.DropBoilerplateLines) {
		lines := strings.Split(s, "\n")
		if opts.DropBoilerplateLines {
			lines = dropBoilerplate(lines)
		}
		if opts.DedupLines {
			lines = dedupConsecutive(lines)
		}
		s = strings.TrimSpace(strings.Join(lines, "\n"))
	}

	return strings.TrimSpace(s)
}

var (
	reCodeFence  = regexp.MustCompile("(?s)```.*?```")
	reInlineCode = regexp.MustCompile("`[^`]+`")
	reImageLink  = regexp.MustCompile("!\\[[^\\]]*\\]\\([^\\)]*\\)")
	reLink       = regexp.MustCompile("\\[([^\\]]+)\\]\\([^\\)]*\\)")
	reHeading    = regexp.MustCompile("(?m)^#{1,6}\\s+")
	reBlockQuote = regexp.MustCompile("(?m)^>\\s+")
	reListMarker = regexp.MustCompile("(?m)^\\s*([-*+]\\s+|\\d+\\.\\s+)")
	reEmphasis   = regexp.MustCompile("(\\*\\*|__|\\*|_)")
)

func stripMarkdown(s string) string {
	s = reCodeFence.ReplaceAllString(s, " ")
	s = reInlineCode.ReplaceAllString(s, " ")
	s = reImageLink.ReplaceAllString(s, " ")
	s = reLink.ReplaceAllString(s, "$1")
	s = reHeading.ReplaceAllString(s, "")
	s = reBlockQuote.ReplaceAllString(s, "")
	s = reListMarker.ReplaceAllString(s, "")
	s = reEmphasis.ReplaceAllString(s, "")
	return s
}

func stripSymbols(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return ' '
	}, s)
	return s
}

func normalizeWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = regexp.MustCompile(`[ \t\f\v]+`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func dedupConsecutive(lines []string) []string {
	out := make([]string, 0, len(lines))
	prev := ""
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			if prev == "" {
				continue
			}
			out = append(out, "")
			prev = ""
			continue
		}
		if l == prev {
			continue
		}
		out = append(out, l)
		prev = l
	}
	return out
}

func dropBoilerplate(lines []string) []string {
	freq := map[string]int{}
	trimmed := make([]string, 0, len(lines))
	for _, l := range lines {
		t := strings.TrimSpace(l)
		trimmed = append(trimmed, t)
		if t != "" {
			freq[t]++
		}
	}
	threshold := 3
	if len(trimmed) < 6 {
		threshold = 2
	}
	out := make([]string, 0, len(trimmed))
	for _, t := range trimmed {
		if t != "" && freq[t] >= threshold {
			continue
		}
		out = append(out, t)
	}
	return out
}
