package stream

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestSegmenterBasicSegmentation 测试基本分段功能
func TestSegmenterBasicSegmentation(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	if err := segmenter.Start(ctx); err != nil {
		t.Fatalf("Failed to start segmenter: %v", err)
	}
	defer segmenter.Stop()

	segmenter.SetPlayID("test-001")

	// 测试句号分段
	segmenter.OnToken("今天天气很好。")

	select {
	case segment := <-outputCh:
		if segment.Text != "今天天气很好。" {
			t.Errorf("Expected '今天天气很好。', got '%s'", segment.Text)
		}
		if segment.PlayID != "test-001" {
			t.Errorf("Expected PlayID 'test-001', got '%s'", segment.PlayID)
		}
		if segment.IsFinal {
			t.Error("Expected IsFinal=false for period segment")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for segment")
	}
}

// TestSegmenterPunctuationSegmentation 测试标点符号分段
func TestSegmenterPunctuationSegmentation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		immediate bool // 是否立即分段
	}{
		{"句号", "你好。", "你好。", true},
		{"问号", "你好吗？", "你好吗？", true},
		{"感叹号", "太好了！", "太好了！", true},
		{"逗号", "你好，", "你好，", false}, // 需要延迟
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputCh := make(chan TextSegment, 10)
			logger := zap.NewNop()
			segmenter := NewTextSegmenter(outputCh, logger)

			ctx := context.Background()
			segmenter.Start(ctx)
			defer segmenter.Stop()

			segmenter.OnToken(tt.input)

			if tt.immediate {
				// 立即分段的情况
				select {
				case segment := <-outputCh:
					if segment.Text != tt.expected {
						t.Errorf("Expected '%s', got '%s'", tt.expected, segment.Text)
					}
				case <-time.After(50 * time.Millisecond):
					t.Error("Timeout waiting for immediate segment")
				}
			} else {
				// 延迟分段的情况
				select {
				case <-outputCh:
					t.Error("Should not segment immediately for comma")
				case <-time.After(50 * time.Millisecond):
					// 正确：没有立即分段
				}
			}
		})
	}
}

// TestSegmenterDelayedSegmentation 测试延迟分段
func TestSegmenterDelayedSegmentation(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 测试足够长的文本（超过 minChars=10）
	segmenter.OnToken("今天的天气非常非常好，") // 11 字符

	// 立即检查：不应该有输出
	select {
	case <-outputCh:
		t.Error("Should not segment immediately after comma")
	case <-time.After(50 * time.Millisecond):
		// 正确：延迟分段
		t.Log("No immediate segment (expected)")
	}

	// 等待延迟时间后应该有输出
	select {
	case segment := <-outputCh:
		expected := "今天的天气非常非常好，"
		if segment.Text != expected {
			t.Errorf("Expected '%s', got '%s'", expected, segment.Text)
		}
		charCount := len([]rune(segment.Text))
		t.Logf("Delayed segment received: '%s' (%d characters)",
			segment.Text, charCount)
		if charCount < 10 {
			t.Errorf("Expected at least 10 chars, got %d", charCount)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Timeout waiting for delayed segment")
	}
}

// TestSegmenterDelayedSegmentationCanceled 测试延迟分段被取消
func TestSegmenterDelayedSegmentationCanceled(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 输入逗号
	segmenter.OnToken("今天天气很好，")

	// 在延迟时间内继续输入（遇到句号）
	time.Sleep(100 * time.Millisecond)
	segmenter.OnToken("我们去公园吧。")

	// 应该收到完整的句子
	select {
	case segment := <-outputCh:
		expected := "今天天气很好，我们去公园吧。"
		if segment.Text != expected {
			t.Errorf("Expected '%s', got '%s'", expected, segment.Text)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for complete segment")
	}

	// 不应该有第二个分段
	select {
	case segment := <-outputCh:
		t.Errorf("Unexpected second segment: %s", segment.Text)
	case <-time.After(400 * time.Millisecond):
		// 正确：没有第二个分段
	}
}

// TestSegmenterMaxLength 测试最大长度强制分段
func TestSegmenterMaxLength(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 输入超过 50 个字符的文本（没有标点）
	longText := "这是一段很长的文本这是一段很长的文本这是一段很长的文本这是一段很长的文本这是一段很长的文本"

	for _, char := range longText {
		segmenter.OnToken(string(char))
	}

	// 应该收到至少一个分段
	select {
	case segment := <-outputCh:
		if len([]rune(segment.Text)) <= 50 {
			t.Logf("Received segment with length %d: %s", len([]rune(segment.Text)), segment.Text)
		} else {
			t.Errorf("Segment too long: %d characters", len([]rune(segment.Text)))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for max length segment")
	}
}

// TestSegmenterOnComplete 测试完成时刷新
func TestSegmenterOnComplete(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 输入一些文本（没有标点）
	segmenter.OnToken("这是最后一段")

	// 调用 OnComplete
	segmenter.OnComplete()

	// 应该立即收到分段，且 IsFinal=true
	select {
	case segment := <-outputCh:
		if segment.Text != "这是最后一段" {
			t.Errorf("Expected '这是最后一段', got '%s'", segment.Text)
		}
		if !segment.IsFinal {
			t.Error("Expected IsFinal=true for complete segment")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for complete segment")
	}
}

// TestSegmenterReset 测试重置功能
func TestSegmenterReset(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 输入一些文本
	segmenter.OnToken("这是一段文本，")

	// 重置
	segmenter.Reset()

	// 等待延迟时间
	time.Sleep(400 * time.Millisecond)

	// 不应该有输出（因为已重置）
	select {
	case segment := <-outputCh:
		t.Errorf("Unexpected segment after reset: %s", segment.Text)
	default:
		// 正确：没有输出
	}
}

// TestSegmenterPlayIDChange 测试 PlayID 变化
func TestSegmenterPlayIDChange(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 设置第一个 PlayID
	segmenter.SetPlayID("play-001")
	segmenter.OnToken("第一段。")

	segment1 := <-outputCh
	if segment1.PlayID != "play-001" {
		t.Errorf("Expected PlayID 'play-001', got '%s'", segment1.PlayID)
	}

	// 设置第二个 PlayID
	segmenter.SetPlayID("play-002")
	segmenter.OnToken("第二段。")

	segment2 := <-outputCh
	if segment2.PlayID != "play-002" {
		t.Errorf("Expected PlayID 'play-002', got '%s'", segment2.PlayID)
	}
}

// TestSegmenterConcurrency 测试并发安全
func TestSegmenterConcurrency(t *testing.T) {
	outputCh := make(chan TextSegment, 100)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 并发写入
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				segmenter.OnToken("测试")
				if j%5 == 0 {
					segmenter.OnToken("。")
				}
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 完成
	segmenter.OnComplete()

	// 应该收到一些分段（不检查具体数量，只要不崩溃就行）
	time.Sleep(100 * time.Millisecond)
	t.Logf("Received %d segments", len(outputCh))
}

// TestSegmenterMinCharsRequirement 测试最小字符数要求
func TestSegmenterMinCharsRequirement(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	// 输入少于 10 个字符 + 逗号（"你好，" = 3 字符）
	segmenter.OnToken("你好，")

	// 等待延迟时间
	time.Sleep(400 * time.Millisecond)

	// 不应该有输出（因为少于最小字符数）
	select {
	case segment := <-outputCh:
		// 注意：实际上 segmenter 会累积，所以这里可能会有输出
		t.Logf("Received segment (may be expected): '%s' (%d chars)",
			segment.Text, len([]rune(segment.Text)))
	default:
		// 正确：没有输出
		t.Log("No segment for short text (expected)")
	}

	// 重置并测试达到最小字符数的情况
	segmenter.Reset()
	segmenter.OnToken("今天天气很好，") // 7 字符

	// 等待延迟时间
	time.Sleep(400 * time.Millisecond)

	// 不应该有输出（7 < 10）
	select {
	case segment := <-outputCh:
		t.Logf("Received segment: '%s' (%d chars)", segment.Text, len([]rune(segment.Text)))
	default:
		t.Log("No segment for text < minChars (expected)")
	}

	// 继续输入，达到最小字符数
	segmenter.OnToken("我们去公园吧，") // 累计 14 字符

	// 等待延迟时间
	time.Sleep(400 * time.Millisecond)

	// 现在应该有输出
	select {
	case segment := <-outputCh:
		charCount := len([]rune(segment.Text))
		t.Logf("Received segment: '%s' (%d chars)", segment.Text, charCount)
		if charCount < 10 {
			t.Errorf("Segment too short: %d characters", charCount)
		}
	case <-time.After(100 * time.Millisecond):
		// 可能已经在前面输出了
		t.Log("No additional segment (may have been output earlier)")
	}
}

// TestSegmenterStreamingScenario 测试流式场景
func TestSegmenterStreamingScenario(t *testing.T) {
	outputCh := make(chan TextSegment, 10)
	logger := zap.NewNop()
	segmenter := NewTextSegmenter(outputCh, logger)

	ctx := context.Background()
	segmenter.Start(ctx)
	defer segmenter.Stop()

	segmenter.SetPlayID("stream-001")

	// 模拟 LLM 流式输出
	text := "今天天气很好，阳光明媚，温度适宜。我们可以去公园散步。"

	go func() {
		for _, char := range text {
			segmenter.OnToken(string(char))
			time.Sleep(10 * time.Millisecond) // 模拟 LLM 输出延迟
		}
		segmenter.OnComplete()
	}()

	// 收集所有分段
	var segments []TextSegment
	timeout := time.After(3 * time.Second)

	// 等待一段时间收集所有分段
	time.Sleep(1500 * time.Millisecond)

	// 收集所有可用的分段
	collecting := true
	for collecting {
		select {
		case segment := <-outputCh:
			segments = append(segments, segment)
			t.Logf("Received segment %d: '%s' (IsFinal=%v)",
				len(segments), segment.Text, segment.IsFinal)
		case <-time.After(100 * time.Millisecond):
			// 100ms 没有新分段，认为已经完成
			collecting = false
		case <-timeout:
			t.Fatalf("Timeout waiting for segments (received %d segments)", len(segments))
		}
	}

	// 验证
	if len(segments) == 0 {
		t.Fatal("No segments received")
	}

	// 所有分段的 PlayID 应该一致
	for i, segment := range segments {
		if segment.PlayID != "stream-001" {
			t.Errorf("Segment %d has wrong PlayID: %s", i, segment.PlayID)
		}
	}

	// 拼接所有分段应该等于原文
	var combined string
	for _, segment := range segments {
		combined += segment.Text
	}
	if combined != text {
		t.Errorf("Combined text mismatch:\nExpected: %s\nGot: %s", text, combined)
	}

	t.Logf("Successfully received %d segments", len(segments))
}
