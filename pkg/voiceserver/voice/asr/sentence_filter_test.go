// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import "testing"

func TestSentenceFilter_PartialBuffersUntilTerminator(t *testing.T) {
	f := NewSentenceFilter(0) // disable similarity for clarity
	if got := f.Update("今天", false); got != "" {
		t.Errorf("partial without terminator should buffer, got %q", got)
	}
	if got := f.Update("今天的天气", false); got != "" {
		t.Errorf("still no terminator, got %q", got)
	}
	got := f.Update("今天的天气不错。", false)
	if got != "今天的天气不错。" {
		t.Errorf("first complete sentence should emit whole text, got %q", got)
	}
}

func TestSentenceFilter_NextSentenceEmitsDeltaOnly(t *testing.T) {
	f := NewSentenceFilter(0)
	f.Update("你好。", false)
	got := f.Update("你好。今天怎么样？", false)
	if got != "今天怎么样？" {
		t.Errorf("delta after sentence boundary, got %q", got)
	}
}

func TestSentenceFilter_FinalEmitsTailEvenWithoutTerminator(t *testing.T) {
	f := NewSentenceFilter(0)
	f.Update("先说一句。", false)
	got := f.Update("先说一句。然后", true)
	if got != "然后" {
		t.Errorf("final should emit untruncated tail, got %q", got)
	}
}

func TestSentenceFilter_RecogniserRevisionFallsBackToFullEmit(t *testing.T) {
	f := NewSentenceFilter(0)
	f.Update("你好世界。", false)
	// Recogniser changes its mind: replaces 世界 with 大家.
	got := f.Update("你好大家。", false)
	if got == "" {
		t.Fatalf("revision should emit something, got empty")
	}
	// We accept either "你好大家。" (whole revised sentence) or any
	// non-empty string, but it must NOT be a stale prefix-delta.
	if got == "大家。" {
		t.Errorf("must not pretend revision is a clean append: %q", got)
	}
}

func TestSentenceFilter_DuplicateHypothesisDropped(t *testing.T) {
	f := NewSentenceFilter(0.85)
	first := f.Update("我想订一张机票。", false)
	if first == "" {
		t.Fatal("first emit should be non-empty")
	}
	// Same-meaning hypothesis with cosmetic jitter — different
	// punctuation, an extra space.
	second := f.Update("我想订一张机票 。", false)
	if second != "" {
		t.Errorf("cosmetically-identical hypothesis should drop, got %q", second)
	}
}

func TestSentenceFilter_ZeroThresholdDisablesSimilarity(t *testing.T) {
	f := NewSentenceFilter(0)
	f.Update("你好。", false)
	// With similarity disabled, near-duplicates still trigger only
	// if they introduce a fresh delta — which the SAME terminated
	// text does NOT, so it still suppresses (via lastEmittedFull
	// equality), confirming the disabled-similarity path doesn't
	// regress the equality path.
	if got := f.Update("你好。", false); got != "" {
		t.Errorf("identical re-emit should still drop, got %q", got)
	}
}

func TestSentenceFilter_EmptyInputIsNoOp(t *testing.T) {
	f := NewSentenceFilter(0.85)
	if got := f.Update("", false); got != "" {
		t.Errorf("empty input must be no-op, got %q", got)
	}
	if got := f.Update("   ", true); got != "" {
		t.Errorf("whitespace-only input must be no-op, got %q", got)
	}
}

func TestSentenceFilter_PendingPeek(t *testing.T) {
	f := NewSentenceFilter(0)
	f.Update("已经说完。还在说", false)
	if got := f.Pending(); got != "还在说" {
		t.Errorf("Pending()=%q, want %q", got, "还在说")
	}
}

func TestSentenceFilter_ResetClearsState(t *testing.T) {
	f := NewSentenceFilter(0.85)
	f.Update("你好。", false)
	f.Reset()
	got := f.Update("你好。", false)
	if got != "你好。" {
		t.Errorf("after Reset, same text should emit again, got %q", got)
	}
}

func TestSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		min  float64
		max  float64
	}{
		{"", "", 1, 1},
		{"abc", "abc", 1, 1},
		{"abc", "", 0, 0},
		{"abc", "abd", 0.6, 0.7},   // 1 sub out of 3
		{"你好世界", "你好世界", 1, 1}, // CJK identity
		{"你好世界", "你好大家", 0.4, 0.6},
	}
	for _, c := range cases {
		got := Similarity(c.a, c.b)
		if got < c.min || got > c.max {
			t.Errorf("Similarity(%q,%q)=%.3f, want in [%.2f, %.2f]", c.a, c.b, got, c.min, c.max)
		}
	}
}

func TestLevenshteinRunes_CJKCountsAsOne(t *testing.T) {
	// "你" is 3 bytes UTF-8; the byte-level Levenshtein would say 3
	// substitutions vs "我". Rune-level must say 1.
	if d := levenshteinRunes("你", "我"); d != 1 {
		t.Errorf("CJK distance=%d, want 1", d)
	}
}
