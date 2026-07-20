package tts

import (
	"strings"
	"sync"
	"testing"
)

type segCapture struct {
	mu   sync.Mutex
	segs []string
	fin  []bool
}

func (c *segCapture) append(s string, final bool) {
	c.mu.Lock()
	c.segs = append(c.segs, s)
	c.fin = append(c.fin, final)
	c.mu.Unlock()
}
func (c *segCapture) snap() ([]string, []bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.segs...), append([]bool(nil), c.fin...)
}

func TestSegmenterSentenceEnder(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	s.Push("你好。")
	s.Push("今天天气不错！")
	segs, _ := c.snap()
	if len(segs) != 2 || segs[0] != "你好。" || segs[1] != "今天天气不错！" {
		t.Fatalf("segments: %#v", segs)
	}
}

func TestSegmenterFirstSegmentCommaEarly(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	// Below FirstMinRunes — hold through the greeting comma.
	s.Push("你好，")
	if segs, _ := c.snap(); len(segs) != 0 {
		t.Fatalf("short greeting must not flush yet, got %v", segs)
	}
	// ≥ FirstMinRunes ending in comma — early flush.
	s.Push("我是智能客服，")
	segs, _ := c.snap()
	if len(segs) != 1 {
		t.Fatalf("expected 1 early segment, got %d: %v", len(segs), segs)
	}
	if !strings.HasSuffix(segs[0], "，") {
		t.Errorf("first segment = %q, want comma ending", segs[0])
	}
	s.Push("今天为您服务。")
	s.Complete()
	segs, _ = c.snap()
	if len(segs) < 2 {
		t.Fatalf("expected rest segment, got %v", segs)
	}
}

func TestSegmenterRestSegmentCommaHold(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	s.Push("您好，我是小智，")
	if segs, _ := c.snap(); len(segs) != 1 {
		t.Fatalf("first segment count = %d, want 1", len(segs))
	}
	s.Push("今天天气不错，明天也好。")
	s.Complete()
	segs, _ := c.snap()
	full := strings.Join(segs, "")
	if !strings.Contains(full, "今天天气不错，明天也好。") {
		t.Errorf("unexpected segments: %v", segs)
	}
	for i, seg := range segs {
		if i == 0 {
			continue
		}
		if strings.HasSuffix(seg, "，") && !strings.ContainsAny(seg, "。！？") {
			t.Errorf("rest segment should not end at comma only: %q", seg)
		}
	}
}

func TestSegmenterRestOnlySentenceSplit(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	s.Push("第一句。")
	s.Push("第二句。")
	s.Complete()
	segs, _ := c.snap()
	if len(segs) != 2 {
		t.Errorf("segment count = %d, want 2 (%v)", len(segs), segs)
	}
}

func TestSegmenterOneShotSentenceKeepsGreeting(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	// One-shot short reply: must NOT peel「您好，」into its own Speak.
	s.Push("您好，有什么可以帮您的？")
	s.Complete()
	segs, _ := c.snap()
	if len(segs) != 1 || segs[0] != "您好，有什么可以帮您的？" {
		t.Fatalf("want single full segment, got %v", segs)
	}
}

func TestSegmenterFirstMaxSkipsTinyPauseCut(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	// Streaming past FirstMax without sentence end — hold through tiny comma.
	s.Push("您好，有什么可以帮")
	if segs, _ := c.snap(); len(segs) != 0 {
		t.Fatalf("must not flush tiny greeting cut, got %v", segs)
	}
	s.Push("您的？")
	segs, _ := c.snap()
	if len(segs) != 1 || segs[0] != "您好，有什么可以帮您的？" {
		t.Fatalf("sentence end should flush full reply, got %v", segs)
	}
}

func TestSegmenterFirstMaxRunesForced(t *testing.T) {
	c := &segCapture{}
	cfg := DefaultSegmenterConfig()
	cfg.FirstMaxRunes = 8
	cfg.FirstMinRunes = 1 // allow forced cut without sentence punct
	s := NewSegmenter(cfg, c.append)
	s.Push("这是一段没有句号但足够长的开头文字继续")
	segs, _ := c.snap()
	if len(segs) != 1 {
		t.Fatalf("first max should force one segment, got %v", segs)
	}
}

func TestSegmenterComplete(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	s.Push("尾巴文字")
	s.Complete()
	segs, fin := c.snap()
	if len(segs) != 1 || segs[0] != "尾巴文字" || !fin[0] {
		t.Fatalf("Complete: %#v fin=%v", segs, fin)
	}
}

func TestSegmenterReset(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(DefaultSegmenterConfig(), c.append)
	s.Push("dropme")
	s.Reset()
	s.Complete()
	if got, _ := c.snap(); len(got) != 0 {
		t.Fatalf("Reset leaked: %#v", got)
	}
}

func TestSplitAtPunctuationBoundary(t *testing.T) {
	head, tail, ok := splitAtPunctuationBoundary("你好我是AI助手今天天气", 6, true)
	if !ok || head == "" || tail == "" {
		t.Fatalf("expected split, got head=%q tail=%q ok=%v", head, tail, ok)
	}
}
