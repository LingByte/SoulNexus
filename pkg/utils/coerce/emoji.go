// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package coerce

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

func ContainsFourByteRune(s string) bool {
	for _, r := range s {
		if utf8.RuneLen(r) >= 4 {
			return true
		}
	}
	return false
}

func StripFourByteRunes(s string) string {
	if !ContainsFourByteRune(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if utf8.RuneLen(r) >= 4 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func ReplaceFourByteRunes(s, placeholder string) string {
	if !ContainsFourByteRune(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if utf8.RuneLen(r) >= 4 {
			b.WriteString(placeholder)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// RemoveEmoji strips emoji runes from text to avoid DB charset issues.
func RemoveEmoji(text string) string {
	var result []rune
	for _, r := range text {
		if (r >= 0x1F300 && r <= 0x1F9FF) ||
			(r >= 0x1F600 && r <= 0x1F64F) ||
			(r >= 0x1F680 && r <= 0x1F6FF) ||
			(r >= 0x2600 && r <= 0x26FF) ||
			(r >= 0x2700 && r <= 0x27BF) ||
			(r >= 0xFE00 && r <= 0xFE0F) ||
			(r >= 0x1F900 && r <= 0x1F9FF) ||
			(r >= 0x1F1E0 && r <= 0x1F1FF) {
			continue
		}
		result = append(result, r)
	}
	return string(result)
}

// RemoveEmojiFromJSON removes emoji from JSON string values while preserving structure.
func RemoveEmojiFromJSON(jsonStr string) string {
	re := regexp.MustCompile(`("(?:[^"\\]|\\.)*")`)
	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		if len(match) > 2 {
			content := match[1 : len(match)-1]
			cleaned := RemoveEmoji(content)
			return `"` + cleaned + `"`
		}
		return match
	})
}
