package utils

import (
	"strings"
	"unicode/utf8"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

func PreviewText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	var b strings.Builder
	count := 0
	for _, r := range text {
		b.WriteRune(r)
		count++
		if count >= maxRunes {
			break
		}
	}
	b.WriteString("…")
	return b.String()
}
