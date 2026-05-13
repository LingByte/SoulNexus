package tts

import (
	"sync"
	"testing"
	"time"
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
	s := NewSegmenter(SegmenterConfig{}, c.append)
	s.Push("你好。")
	s.Push("今天天气不错！")
	segs, fin := c.snap()
	if len(segs) != 2 || segs[0] != "你好。" || segs[1] != "今天天气不错！" {
		t.Fatalf("segments: %#v", segs)
	}
	for _, f := range fin {
		if f {
			t.Fatalf("partial should not be final")
		}
	}
}

func TestSegmenterClauseNeedsMinLen(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(SegmenterConfig{MinRunes: 5, IdleFlush: time.Hour}, c.append)
	s.Push("嗨，") // len < 5 → no flush
	if got, _ := c.snap(); len(got) != 0 {
		t.Fatalf("premature flush: %#v", got)
	}
	s.Push("今天你好吗，") // > min → flush on comma
	if got, _ := c.snap(); len(got) != 1 {
		t.Fatalf("want 1 seg: %#v", got)
	}
}

func TestSegmenterMaxRunesForced(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(SegmenterConfig{MinRunes: 1000, MaxRunes: 5, IdleFlush: time.Hour}, c.append)
	s.Push("abcdefg")
	if got, _ := c.snap(); len(got) != 1 {
		t.Fatalf("hard max must flush: %#v", got)
	}
}

func TestSegmenterComplete(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(SegmenterConfig{IdleFlush: time.Hour}, c.append)
	s.Push("tail piece")
	s.Complete()
	segs, fin := c.snap()
	if len(segs) != 1 || segs[0] != "tail piece" || !fin[0] {
		t.Fatalf("Complete: %#v fin=%v", segs, fin)
	}
}

func TestSegmenterReset(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(SegmenterConfig{IdleFlush: time.Hour}, c.append)
	s.Push("dropme")
	s.Reset()
	s.Complete()
	if got, _ := c.snap(); len(got) != 0 {
		t.Fatalf("Reset leaked: %#v", got)
	}
}

func TestSegmenterIdleFlush(t *testing.T) {
	c := &segCapture{}
	s := NewSegmenter(SegmenterConfig{MinRunes: 2, IdleFlush: 20 * time.Millisecond}, c.append)
	s.Push("ok buffered text")
	time.Sleep(80 * time.Millisecond)
	if got, _ := c.snap(); len(got) != 1 {
		t.Fatalf("idle flush: %#v", got)
	}
}
