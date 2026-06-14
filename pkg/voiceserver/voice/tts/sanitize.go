package tts

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"regexp"
	"strings"
)

var (
	reMarkdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reHTMLTag      = regexp.MustCompile(`<[^>]+>`)
	reFencedCode   = regexp.MustCompile("(?s)```.*?```")
	reInlineCode   = regexp.MustCompile("`+([^`]*)`+")
	reMDHeader     = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	reHR           = regexp.MustCompile(`(?m)^\s*[-*_]{3,}\s*$`)
	reBullet       = regexp.MustCompile(`(?m)^\s*[-*+•]\s+`)
	reOrderedList  = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
	reBoldStar     = regexp.MustCompile(`\*\*([^*]*)\*\*`)
	reBoldUnder    = regexp.MustCompile(`__([^_]*)__`)
	reLoneStars    = regexp.MustCompile(`\*+`)
)

// SanitizeForSpeech strips markdown / markup noise before TTS synthesis.
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
	return collapseWhitespace(s)
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
