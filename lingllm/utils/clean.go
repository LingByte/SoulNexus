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
	reEmoji      = regexp.MustCompile(`[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F004}\x{1F0CF}\x{1F170}-\x{1F251}\x{1F300}-\x{1F5FF}\x{1F600}-\x{1F64F}\x{1F680}-\x{1F6FF}\x{1F910}-\x{1F93E}\x{1F940}-\x{1F94C}\x{1F950}-\x{1F96B}\x{1F980}-\x{1F997}\x{1F9C0}-\x{1F9E6}\x{1FA70}-\x{1FA74}\x{1FA78}-\x{1FA7A}\x{1FA80}-\x{1FA86}\x{1FA90}-\x{1FAA8}\x{1FAB0}-\x{1FAB6}\x{1FAC0}-\x{1FAC2}\x{1FAD0}-\x{1FAD6}\x{1F1E6}-\x{1F1FF}\x{200D}\x{FE0F}]`)
	reSSMLUnsafe = regexp.MustCompile(`[<>&]`)
	reHTMLTag    = regexp.MustCompile(`</?[^>\s]+[^>]*>`)
)

// SanitizeForSpeech prepares LLM or ASR text for cloud TTS (strip markdown/emoji, etc.).
func SanitizeForSpeech(s string) string {
	s = CleanText(s, &Options{StripMarkdown: true})
	s = reHTMLTag.ReplaceAllString(s, "")
	s = reEmoji.ReplaceAllString(s, "")
	s = replaceSmartQuotes(s)
	s = reSSMLUnsafe.ReplaceAllString(s, " ")
	s = normalizeWhitespace(s)
	s = strings.TrimSpace(s)
	s = trimPunctuationEdges(s)
	if !IsCloudTTSAcceptable(s) {
		return ""
	}
	return s
}

// IsCloudTTSAcceptable reports whether text is safe to send to cloud TTS APIs.
func IsCloudTTSAcceptable(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !HasSpeakableContent(s) {
		return false
	}
	if strings.ContainsAny(s, "<>") {
		return false
	}
	return CountSpeakableRunes(s) >= 1
}

// CountSpeakableRunes counts letter/digit/CJK runes in s.
func CountSpeakableRunes(s string) int {
	n := 0
	for _, r := range s {
		if isSpeakableRune(r) {
			n++
		}
	}
	return n
}

func trimPunctuationEdges(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	start := 0
	for start < len(runes) && isLeadingJunk(runes[start]) {
		start++
	}
	if start >= len(runes) {
		return ""
	}
	trimmed := string(runes[start:])
	if !HasSpeakableContent(trimmed) {
		return ""
	}
	return trimmed
}

func isLeadingJunk(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsNumber(r) || isSpeakableRune(r) {
		return false
	}
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// HasSpeakableContent reports whether text contains letters, digits, or CJK characters.
func HasSpeakableContent(s string) bool {
	for _, r := range s {
		if isSpeakableRune(r) {
			return true
		}
	}
	return false
}

func isSpeakableRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return true
	}
	switch {
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	case r >= 0x3040 && r <= 0x30FF: // Japanese kana
		return true
	case r >= 0xAC00 && r <= 0xD7AF: // Korean hangul syllables
		return true
	}
	return false
}

func replaceSmartQuotes(s string) string {
	return strings.NewReplacer(
		"\u2018", "'",
		"\u2019", "'",
		"\u201C", "\"",
		"\u201D", "\"",
		"\u2026", "...",
		"\u200B", "",
		"\u200C", "",
		"\u200D", "",
	).Replace(s)
}

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
