package tts

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	reMarkdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reHTMLTag      = regexp.MustCompile(`<[^>]+>`)
	reFencedCode   = regexp.MustCompile("(?s)```.*?```")
	reInlineCode   = regexp.MustCompile("`+([^`]*)`+")
	reMDHeader     = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	reHR           = regexp.MustCompile(`(?m)^\s*[-*_]{3,}\s*$`)
	reBullet      = regexp.MustCompile(`(?m)^\s*[-*+•]\s+`)
	reOrderedList = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
	reBoldStar    = regexp.MustCompile(`\*\*([^*]*)\*\*`)
	reBoldUnder   = regexp.MustCompile(`__([^_]*)__`)
	reLoneStars   = regexp.MustCompile(`\*+`)
)

// SanitizeForSpeech strips markdown / markup noise before TTS synthesis.
// Returns empty string when nothing speakable remains.
func SanitizeForSpeech(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = reFencedCode.ReplaceAllString(s, " ")
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reMarkdownLink.ReplaceAllString(s, "$1")
	s = reHTMLTag.ReplaceAllString(s, " ")
	s = reMDHeader.ReplaceAllString(s, "")
	s = reHR.ReplaceAllString(s, " ")
	s = reBullet.ReplaceAllString(s, "")
	s = reOrderedList.ReplaceAllString(s, "")
	s = reBoldStar.ReplaceAllString(s, "$1")
	s = reBoldUnder.ReplaceAllString(s, "$1")
	s = reLoneStars.ReplaceAllString(s, "")

	var b strings.Builder
	for _, r := range s {
		if r < 0x20 && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	s = strings.TrimSpace(b.String())
	s = collapseWhitespace(s)
	return normalizeSpeakable(s)
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if r == '\n' || r == '\r' || unicode.IsSpace(r) {
			if !space && b.Len() > 0 {
				b.WriteRune(' ')
				space = true
			}
			continue
		}
		space = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func normalizeSpeakable(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	hasSpeech := false
	for _, r := range s {
		if isSpeechRune(r) {
			hasSpeech = true
			break
		}
	}
	if !hasSpeech {
		return ""
	}
	return s
}

func isSpeechRune(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= 0x4e00 && r <= 0x9fff {
		return true
	}
	return false
}
