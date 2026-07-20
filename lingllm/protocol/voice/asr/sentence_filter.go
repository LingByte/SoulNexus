package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// SentenceFilter buffers ASR partials and emits complete sentences to reduce
// LLM thrash. Finals always pass through.
type SentenceFilter struct {
	mu                  sync.Mutex
	lastEmittedFull     string
	lastEmittedNorm     string
	pendingTail         string
	similarityThreshold float64
}

// NewSentenceFilter returns a filter with the given similarity threshold (0 disables).
func NewSentenceFilter(similarityThreshold float64) *SentenceFilter {
	if similarityThreshold < 0 {
		similarityThreshold = 0
	}
	if similarityThreshold > 1 {
		similarityThreshold = 1
	}
	return &SentenceFilter{similarityThreshold: similarityThreshold}
}

// Update returns the text to forward downstream. Empty means suppress.
func (f *SentenceFilter) Update(text string, isFinal bool) string {
	if f == nil {
		return text
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.similarityThreshold > 0 && f.lastEmittedNorm != "" {
		if Similarity(NormalizeForCompare(text), f.lastEmittedNorm) >= f.similarityThreshold {
			return ""
		}
	}

	if isFinal {
		var delta string
		switch {
		case text == f.lastEmittedFull:
			delta = ""
		case strings.HasPrefix(text, f.lastEmittedFull):
			delta = strings.TrimSpace(text[len(f.lastEmittedFull):])
		default:
			delta = text
		}
		f.lastEmittedFull = text
		f.lastEmittedNorm = NormalizeForCompare(text)
		f.pendingTail = ""
		return delta
	}

	endIdx := FindLastSentenceEnding(text)
	if endIdx < 0 {
		if strings.HasPrefix(text, f.lastEmittedFull) {
			f.pendingTail = text[len(f.lastEmittedFull):]
		} else {
			// ASR revised/shrank the hypothesis — stale prefix must not slice text.
			f.pendingTail = text
		}
		return ""
	}

	upToSentence := text[:endIdx+1]
	if upToSentence == f.lastEmittedFull {
		if strings.HasPrefix(text, f.lastEmittedFull) {
			f.pendingTail = text[len(f.lastEmittedFull):]
		} else {
			f.pendingTail = text
		}
		return ""
	}

	var delta string
	if strings.HasPrefix(upToSentence, f.lastEmittedFull) {
		delta = strings.TrimSpace(upToSentence[len(f.lastEmittedFull):])
	} else {
		delta = upToSentence
	}
	f.lastEmittedFull = upToSentence
	f.lastEmittedNorm = NormalizeForCompare(upToSentence)
	f.pendingTail = strings.TrimSpace(text[endIdx+1:])
	return delta
}

// Reset clears filter state between turns.
func (f *SentenceFilter) Reset() {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.lastEmittedFull = ""
	f.lastEmittedNorm = ""
	f.pendingTail = ""
	f.mu.Unlock()
}

// SentenceEndings are treated as sentence boundaries.
var SentenceEndings = []rune{'。', '！', '？', '.', '!', '?', '\n'}

// FindLastSentenceEnding returns the byte offset of the last sentence terminator, or -1.
func FindLastSentenceEnding(text string) int {
	if text == "" {
		return -1
	}
	last := -1
	for i, r := range text {
		for _, e := range SentenceEndings {
			if r == e {
				last = i + utf8.RuneLen(r)
			}
		}
	}
	if last < 0 {
		return -1
	}
	return last - 1
}

// NormalizeForCompare strips punctuation/whitespace for fuzzy ASR dedup.
func NormalizeForCompare(text string) string {
	if text == "" {
		return ""
	}
	out := make([]rune, 0, len(text))
	var last rune
	hasLast := false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !hasLast || r != last {
				out = append(out, r)
				last = r
				hasLast = true
			}
		}
	}
	return string(out)
}

// Similarity returns normalized Levenshtein similarity in [0, 1].
func Similarity(a, b string) float64 {
	if a == "" && b == "" {
		return 1
	}
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	d := levenshteinRunes(a, b)
	maxLen := runeLen(a)
	if l := runeLen(b); l > maxLen {
		maxLen = l
	}
	if maxLen == 0 {
		return 1
	}
	score := 1 - float64(d)/float64(maxLen)
	if score < 0 {
		return 0
	}
	return score
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func levenshteinRunes(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) < len(rb) {
		ra, rb = rb, ra
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
