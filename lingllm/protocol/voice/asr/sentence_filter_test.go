package asr

import "testing"

func TestSentenceFilter_ShorterPartialDoesNotPanic(t *testing.T) {
	f := NewSentenceFilter(0)
	// Simulate a longer partial that established lastEmittedFull.
	_ = f.Update("这是一段比较长的识别结果。", false)
	// Tencent ASR may revise to a shorter partial without sharing prefix.
	if delta := f.Update("喂，你好", false); delta != "" {
		t.Fatalf("unexpected delta %q", delta)
	}
}

func TestSentenceFilter_EmitsOnSentenceBoundary(t *testing.T) {
	f := NewSentenceFilter(0)
	delta := f.Update("今天天气很好。", false)
	if delta == "" {
		t.Fatal("expected sentence-boundary partial emit")
	}
}

func TestFindLastSentenceEnding(t *testing.T) {
	idx := FindLastSentenceEnding("喂，听得到吗？")
	if idx < 0 {
		t.Fatal("expected sentence ending")
	}
	if got := "喂，听得到吗？"[:idx+1]; got != "喂，听得到吗？" {
		t.Fatalf("got %q", got)
	}
}
