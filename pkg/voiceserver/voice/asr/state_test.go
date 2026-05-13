package asr

import "testing"

func TestIncrementalStateCumulative(t *testing.T) {
	s := NewIncrementalState()
	if d := s.Update("你好", false); d != "你好" {
		t.Fatalf("1st partial: %q", d)
	}
	if d := s.Update("你好今天", false); d != "今天" {
		t.Fatalf("2nd partial: %q", d)
	}
	if d := s.Update("你好今天天气", false); d != "天气" {
		t.Fatalf("3rd partial: %q", d)
	}
	if d := s.Update("你好今天天气不错", true); d != "不错" {
		t.Fatalf("final: %q", d)
	}
	if s.LastFinal() != "你好今天天气不错" {
		t.Fatalf("LastFinal: %q", s.LastFinal())
	}
	if s.Utterances() != 1 {
		t.Fatalf("Utterances: %d", s.Utterances())
	}
}

func TestIncrementalStateRestartHypothesis(t *testing.T) {
	s := NewIncrementalState()
	_ = s.Update("hello wor", false)
	// Different prefix → treated as new full text.
	if d := s.Update("world", false); d != "world" {
		t.Fatalf("reset hypothesis: %q", d)
	}
}

func TestIncrementalStateReset(t *testing.T) {
	s := NewIncrementalState()
	_ = s.Update("abc", true)
	s.Reset()
	if s.LastFinal() != "" || s.Utterances() != 0 {
		t.Fatalf("Reset did not clear")
	}
	if d := s.Update("xyz", false); d != "xyz" {
		t.Fatalf("after reset: %q", d)
	}
}

func TestIncrementalStateNilSafe(t *testing.T) {
	var s *IncrementalState
	if d := s.Update("abc", true); d != "abc" {
		t.Fatalf("nil-safe Update: %q", d)
	}
	s.Reset()
	_ = s.LastFinal()
	_ = s.Utterances()
}
