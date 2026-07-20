package tts

import (
	"context"
	"strings"
	"testing"
)

func TestTextSegmenterComponentCreation(t *testing.T) {
	config := DefaultTextSegmenterConfig()
	segmenter := NewTextSegmenterComponent(config, func(segment TextSegment) {})

	if segmenter == nil {
		t.Error("TextSegmenterComponent should not be nil")
	}

	if segmenter.Name() != "text_segmenter" {
		t.Errorf("Name = %s, want 'text_segmenter'", segmenter.Name())
	}
}

func TestTextSegmenterComponentDefaultConfig(t *testing.T) {
	config := DefaultTextSegmenterConfig()

	if config.FirstMinChars != 2 {
		t.Errorf("FirstMinChars = %d, want 2", config.FirstMinChars)
	}
	if config.FirstMaxChars != 18 {
		t.Errorf("FirstMaxChars = %d, want 18", config.FirstMaxChars)
	}
	if config.RestForceMaxChars != 120 {
		t.Errorf("RestForceMaxChars = %d, want 120", config.RestForceMaxChars)
	}
}

func TestTextSegmenterComponentEmptyText(t *testing.T) {
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(TextSegment) {})

	result, shouldContinue, err := segmenter.Process(context.Background(), "")
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}
	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}
	if result != "" {
		t.Errorf("Result = %q, want empty string", result)
	}
}

func TestTextSegmenterComponentSentenceEnding(t *testing.T) {
	segmentCount := 0
	var lastSegment string

	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		segmentCount++
		lastSegment = segment.Text
	})

	segmenter.Process(context.Background(), "这是一个句子。")

	if segmentCount != 1 {
		t.Errorf("Segment count = %d, want 1", segmentCount)
	}
	if lastSegment != "这是一个句子。" {
		t.Errorf("Last segment = %q, want '这是一个句子。'", lastSegment)
	}
}

func TestTextSegmenterFirstSegmentCommaEarly(t *testing.T) {
	var segments []string
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		segments = append(segments, segment.Text)
	})

	segmenter.Process(context.Background(), "你好，")
	segmenter.Process(context.Background(), "我是小智。")

	if len(segments) < 2 {
		t.Fatalf("expected 2 segments, got %d: %v", len(segments), segments)
	}
	if segments[0] != "你好，" {
		t.Errorf("first segment = %q, want '你好，'", segments[0])
	}
	if segments[1] != "我是小智。" {
		t.Errorf("second segment = %q, want '我是小智。'", segments[1])
	}
}

func TestTextSegmenterRestSegmentCommaHold(t *testing.T) {
	var segments []string
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		segments = append(segments, segment.Text)
	})

	segmenter.Process(context.Background(), "你好，")
	if len(segments) != 1 {
		t.Fatalf("first segment count = %d, want 1", len(segments))
	}

	segmenter.Process(context.Background(), "今天天气不错，明天也好。")
	segmenter.OnComplete()

	full := strings.Join(segments, "")
	if !strings.Contains(full, "今天天气不错，明天也好。") {
		t.Errorf("unexpected segments: %v", segments)
	}
	// Comma in the middle of rest buffer must not create an extra early segment.
	for i, seg := range segments {
		if i == 0 {
			continue
		}
		if strings.HasSuffix(seg, "，") && !strings.ContainsAny(seg, "。！？") {
			t.Errorf("rest segment should not end at comma only: %q", seg)
		}
	}
}

func TestTextSegmenterRestOnlySentenceSplit(t *testing.T) {
	var segments []string
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		segments = append(segments, segment.Text)
	})

	segmenter.Process(context.Background(), "第一句。")
	segmenter.Process(context.Background(), "第二句。")
	segmenter.OnComplete()

	if len(segments) != 2 {
		t.Errorf("segment count = %d, want 2 (%v)", len(segments), segments)
	}
}

func TestTextSegmenterComponentOnComplete(t *testing.T) {
	segmentCount := 0
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		segmentCount++
	})

	segmenter.Process(context.Background(), "这是一个句子")
	initialCount := segmentCount
	segmenter.OnComplete()

	if segmentCount <= initialCount {
		t.Error("OnComplete should flush remaining text")
	}
}

func TestTextSegmenterComponentSetPlayID(t *testing.T) {
	var capturedPlayID string
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		capturedPlayID = segment.PlayID
	})

	segmenter.SetPlayID("test-play-id")
	segmenter.Process(context.Background(), "测试。")

	if capturedPlayID != "test-play-id" {
		t.Errorf("PlayID = %s, want 'test-play-id'", capturedPlayID)
	}
}

func TestTextSegmenterComponentSetLogger(t *testing.T) {
	logCalled := false
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(TextSegment) {})
	segmenter.SetLogger(func(string) { logCalled = true })
	segmenter.Process(context.Background(), "测试。")

	if !logCalled {
		t.Error("Logger should have been called")
	}
}

func TestTextSegmenterComponentReset(t *testing.T) {
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(TextSegment) {})
	segmenter.Process(context.Background(), "测试文本")
	segmenter.Reset()

	if segmenter.buffer != "" {
		t.Errorf("Buffer should be empty after reset, got %q", segmenter.buffer)
	}
	if segmenter.firstFlushed {
		t.Error("firstFlushed should reset")
	}
}

func TestTextSegmenterComponentEnglishPunctuation(t *testing.T) {
	segmentCount := 0
	var lastSegment string
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		segmentCount++
		lastSegment = segment.Text
	})

	segmenter.Process(context.Background(), "This is a sentence.")
	if segmentCount != 1 {
		t.Errorf("Segment count = %d, want 1", segmentCount)
	}
	if lastSegment != "This is a sentence." {
		t.Errorf("Last segment = %q, want 'This is a sentence.'", lastSegment)
	}
}

func TestTextSegmenterComponentFinalSegment(t *testing.T) {
	var finalSegment TextSegment
	segmenter := NewTextSegmenterComponent(DefaultTextSegmenterConfig(), func(segment TextSegment) {
		finalSegment = segment
	})

	segmenter.Process(context.Background(), "这是最后一句")
	segmenter.OnComplete()

	if !finalSegment.IsFinal {
		t.Error("Final segment should have IsFinal=true")
	}
	if finalSegment.Text != "这是最后一句" {
		t.Errorf("Final segment text = %q, want '这是最后一句'", finalSegment.Text)
	}
}

func TestSplitAtPunctuationBoundary(t *testing.T) {
	head, tail, ok := splitAtPunctuationBoundary("你好我是AI助手今天天气", 6, true)
	if !ok || head == "" || tail == "" {
		t.Fatalf("expected split, got head=%q tail=%q ok=%v", head, tail, ok)
	}
}
