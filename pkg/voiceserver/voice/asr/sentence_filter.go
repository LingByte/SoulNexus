// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import (
	"strings"
	"sync"
)

// SentenceFilter is a higher-quality replacement for IncrementalState
// when the downstream consumer (LLM, dialog plane) prefers complete
// sentences over byte-by-byte deltas. It layers two behaviours on
// top of cumulative-text dedup:
//
//  1. Sentence-boundary partial emission. During non-final updates,
//     the filter only releases text up to the LAST sentence
//     terminator (。！？.!?\n). Half-sentences stay buffered, so the
//     LLM doesn't see "今天的天" followed by "今天的天气". This
//     dramatically reduces LLM thrash without delaying finals.
//
//  2. Levenshtein-similarity dedup. When the recogniser sends two
//     hypotheses that normalise to near-identical text (≥
//     SimilarityThreshold), the second one is silently dropped. This
//     handles QCloud / Volcengine occasionally re-emitting the same
//     hypothesis with cosmetic differences (extra spaces, swapped
//     punctuation) which would otherwise trigger redundant LLM calls.
//
// Both behaviours are bypassed on isFinal=true: the tail of a final
// utterance always emits, even if it ends mid-sentence (the
// recogniser asserted "I'm done", we trust it).
//
// Use SentenceFilter when:
//   - Your dialog plane wants "complete sentence" granularity
//   - You'd rather pay 200-400 ms more end-to-end for cleaner LLM
//     prompts than chase the absolute-lowest-latency tradeoff
//
// Use IncrementalState when:
//   - Your dialog plane streams every delta to the LLM regardless
//   - You're optimising for first-token latency over text quality
type SentenceFilter struct {
	mu                  sync.Mutex
	lastEmittedFull     string  // cumulative text whose tail-up-to-last-period we already pushed
	lastEmittedNorm     string  // normalised form of last emit, cached for similarity checks
	pendingTail         string  // bytes after the last terminator we've seen but haven't emitted
	similarityThreshold float64 // 0..1; ≥this means "same hypothesis"
}

// NewSentenceFilter returns a filter using the provided similarity
// threshold (0.85 is a good default — strict enough to keep distinct
// utterances separate, loose enough to absorb cosmetic ASR jitter).
// A zero or negative threshold disables the similarity check.
func NewSentenceFilter(similarityThreshold float64) *SentenceFilter {
	if similarityThreshold <= 0 {
		similarityThreshold = 0
	}
	if similarityThreshold > 1 {
		similarityThreshold = 1
	}
	return &SentenceFilter{similarityThreshold: similarityThreshold}
}

// Update consumes the latest cumulative ASR text and returns the
// portion the caller should forward downstream. Empty return = nothing
// new to emit. The caller may pass an empty `text` (treated as no-op).
//
// Concurrency: safe to call from multiple goroutines but emissions are
// serialised — a slow downstream consumer will back-pressure further
// Update calls because the filter holds its mutex during the (string)
// computation, not during caller-side I/O.
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

	// Similarity gate: even on partials, drop hypotheses that look
	// like a redo of the last emit. We compare on the normalised
	// form so whitespace / punctuation jitter doesn't fool us.
	if f.similarityThreshold > 0 && f.lastEmittedNorm != "" {
		if Similarity(NormalizeForCompare(text), f.lastEmittedNorm) >= f.similarityThreshold {
			// Treat as no-op without losing track of the latest text;
			// next "real" change still produces a clean delta.
			return ""
		}
	}

	if isFinal {
		// Whatever bit of the final text we haven't shipped yet is
		// the delta. We cover three cases:
		//   a) text strictly extends lastEmittedFull → append the tail
		//   b) text fully equals lastEmittedFull     → empty delta
		//   c) text disagrees with lastEmittedFull   → emit whole text
		//      (recogniser corrected an earlier hypothesis)
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

	// Partial path: only release text up to the LAST sentence
	// terminator. Anything after stays in pendingTail (informational
	// only — useful for callers that want a peek-but-don't-commit
	// behaviour, exposed via Pending()).
	endIdx := FindLastSentenceEnding(text)
	if endIdx < 0 {
		// No sentence boundary yet. Buffer the tail and emit nothing.
		f.pendingTail = text[len(f.lastEmittedFull):]
		if !strings.HasPrefix(text, f.lastEmittedFull) {
			// recogniser revised history; reset pendingTail to whole text
			f.pendingTail = text
		}
		return ""
	}

	// upToSentence is the cumulative text up to and including the
	// last terminator byte (FindLastSentenceEnding returns the LAST
	// byte offset of the multi-byte terminator).
	upToSentence := text[:endIdx+1]
	if upToSentence == f.lastEmittedFull {
		// Same sentence boundary as last time — nothing new to emit.
		f.pendingTail = text[len(f.lastEmittedFull):]
		return ""
	}

	var delta string
	if strings.HasPrefix(upToSentence, f.lastEmittedFull) {
		delta = strings.TrimSpace(upToSentence[len(f.lastEmittedFull):])
	} else {
		// Recogniser revised the head of the utterance; conservative
		// fallback is to re-emit the whole sentence-bounded prefix.
		delta = upToSentence
	}
	f.lastEmittedFull = upToSentence
	f.lastEmittedNorm = NormalizeForCompare(upToSentence)
	f.pendingTail = strings.TrimSpace(text[endIdx+1:])
	return delta
}

// Pending returns the buffered tail (text after the last sentence
// terminator) without committing it. Useful for "what is the user
// likely about to finish saying" UI hints.
func (f *SentenceFilter) Pending() string {
	if f == nil {
		return ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pendingTail
}

// Reset clears state; call between turns or on cancel.
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

// ----- similarity helpers -----

// Similarity returns 1 - Levenshtein(a, b) / max(len(a), len(b)),
// clamped to [0, 1]. 1.0 = identical, 0.0 = completely different.
// Both inputs should already be normalised (see NormalizeForCompare)
// for stable scores; raw text may give surprising results due to
// punctuation/whitespace weight.
//
// Optimised for the short ASR-hypothesis sizes we see in practice
// (<200 runes); above that the O(n*m) cost becomes noticeable but
// is still bounded by a single allocation.
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
	if score > 1 {
		return 1
	}
	return score
}

// runeLen counts runes in s. Local copy because pkg/voice/tts also
// has one; we avoid the dependency to keep this package leaf-level.
func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// levenshteinRunes computes edit distance over runes (so CJK chars
// count as one unit, not three bytes). Two-row dynamic programming
// keeps memory at O(min(|a|, |b|)).
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
			cur[j] = min3(
				cur[j-1]+1,      // insert
				prev[j]+1,       // delete
				prev[j-1]+cost,  // substitute
			)
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
