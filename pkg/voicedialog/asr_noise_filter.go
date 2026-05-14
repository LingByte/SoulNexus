package voicedialog

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import "strings"

// Separators / punctuation stripped before deciding if a final transcript is
// "filler only" (common false-positive ASR when the user did not really speak).
var asrNoiseSeparators = " \t\n\r.。!?,!?？，、；：…—~～:（）【】《》「」『』"

// isNoiseOnlyASRFinal reports whether the ASR final should be ignored for
// dialog/LLM: the whole content (after removing punctuation/spaces) consists
// only of short interjection syllables, e.g. "嗯嗯", "呃呃。", "哦哦".
func isNoiseOnlyASRFinal(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(asrNoiseSeparators, r) {
			continue
		}
		b.WriteRune(r)
	}
	core := b.String()
	if core == "" {
		return true
	}
	runes := []rune(core)
	// Longer strings are unlikely to be pure hallucinated filler.
	if len(runes) > 12 {
		return false
	}
	for _, r := range runes {
		if !asrFillerRune(r) {
			return false
		}
	}
	return true
}

func asrFillerRune(r rune) bool {
	switch r {
	case '嗯', '唔', '呣',
		'呃', '额',
		'啊', '哦', '喔', '噢',
		'唉', '哎', '诶', '欸',
		'哼', '呵', '哈', '嘻',
		'呀', '哟', '呦', '咯', '呐', '哩':
		return true
	default:
		return false
	}
}
