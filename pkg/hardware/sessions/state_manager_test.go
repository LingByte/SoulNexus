package sessions

import (
	"testing"
)

func TestASRStateManager_UpdateASRText_Final(t *testing.T) {
	manager := NewASRStateManager()

	// 测试最终结果
	result := manager.UpdateASRText("你好，世界", true)
	if result != "你好，世界" {
		t.Errorf("Expected '你好，世界', got '%s'", result)
	}

	// 相同文本应该被过滤
	result = manager.UpdateASRText("你好，世界", true)
	if result != "" {
		t.Errorf("Expected empty string for duplicate, got '%s'", result)
	}

	// 新文本应该返回增量
	result = manager.UpdateASRText("你好，世界！很高兴见到你", true)
	if result == "" {
		t.Error("Expected incremental text, got empty string")
	}
}

func TestASRStateManager_UpdateASRText_Intermediate(t *testing.T) {
	manager := NewASRStateManager()

	// 中间结果：不完整的句子应该被过滤
	result := manager.UpdateASRText("你好", false)
	if result != "" {
		t.Errorf("Expected empty for incomplete sentence, got '%s'", result)
	}

	// 中间结果：完整的句子应该返回
	result = manager.UpdateASRText("你好，世界。", false)
	if result == "" {
		t.Error("Expected complete sentence, got empty string")
	}

	// 继续累积
	result = manager.UpdateASRText("你好，世界。今天天气不错", false)
	if result != "" {
		t.Errorf("Expected empty for incomplete continuation, got '%s'", result)
	}

	// 新的完整句子
	result = manager.UpdateASRText("你好，世界。今天天气不错。", false)
	if result == "" {
		t.Error("Expected new complete sentence, got empty string")
	}
}

func TestASRStateManager_IncompleteThenComplete(t *testing.T) {
	manager := NewASRStateManager()

	// 第一次：不完整的句子（没有结束符）
	result := manager.UpdateASRText("喂喂喂可以听到我说话吗", false)
	if result != "" {
		t.Errorf("Expected empty for incomplete sentence, got '%s'", result)
	}

	// 第二次：加了标点符号，形成完整句子，应该被提取
	result = manager.UpdateASRText("喂喂喂，可以听到我说话吗？", false)
	if result != "喂喂喂，可以听到我说话吗？" {
		t.Errorf("Expected complete sentence to be extracted, got '%s'", result)
	}

	// 验证状态已更新（保留了最新版本）
	if manager.lastProcessedCumulativeText != "喂喂喂，可以听到我说话吗？" {
		t.Errorf("Expected lastProcessedCumulativeText to be updated to latest version, got '%s'",
			manager.lastProcessedCumulativeText)
	}

	// 第三次：ASR 继续返回相同内容，应该被过滤
	result = manager.UpdateASRText("喂喂喂，可以听到我说话吗？", false)
	if result != "" {
		t.Errorf("Expected empty for duplicate content, got '%s'", result)
	}
}

func TestASRStateManager_Clear(t *testing.T) {
	manager := NewASRStateManager()

	// 设置一些状态
	manager.UpdateASRText("你好，世界", true)

	// 清空
	manager.Clear()

	// 应该能再次处理相同的文本
	result := manager.UpdateASRText("你好，世界", true)
	if result != "你好，世界" {
		t.Errorf("Expected '你好，世界' after clear, got '%s'", result)
	}
}

func TestNormalizeTextFast(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"你好，世界！", "你好世界"},
		{"Hello World", "HeloWorld"}, // 'l' 重复会被去重
		{"  空格  测试  ", "空格测试"},
		{"123abc中文", "123abc中文"},
		{"", ""},
		{"aabbcc", "abc"}, // 连续重复字符会被去重
	}

	for _, tt := range tests {
		result := normalizeTextFast(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeTextFast(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCalculateSimilarityFast(t *testing.T) {
	tests := []struct {
		text1  string
		text2  string
		minSim float64
		maxSim float64
	}{
		{"你好", "你好", 1.0, 1.0},
		{"", "", 1.0, 1.0},
		{"你好", "", 0.0, 0.0},
		{"你好世界", "你好", 0.4, 0.6},
		{"完全不同", "totally different", 0.0, 0.1},
	}

	for _, tt := range tests {
		result := calculateSimilarityFast(tt.text1, tt.text2)
		if result < tt.minSim || result > tt.maxSim {
			t.Errorf("calculateSimilarityFast(%q, %q) = %f, want between %f and %f",
				tt.text1, tt.text2, result, tt.minSim, tt.maxSim)
		}
	}
}

func TestStateManager_NewSentenceAfterPreviousConversation(t *testing.T) {
	sm := NewASRStateManager()

	// 模拟之前的对话
	result1 := sm.UpdateASRText("喂，你好啊。", false)
	if result1 != "喂，你好啊。" {
		t.Errorf("第一句应该被提取，got: %s", result1)
	}

	result2 := sm.UpdateASRText("喂，你好啊。喂，你好。", false)
	if result2 != "喂，你好。" {
		t.Errorf("第二句应该被提取，got: %s", result2)
	}

	// 模拟用户说新的内容 "你好，再见。"
	// ASR 会累积所有文本
	result3 := sm.UpdateASRText("喂，你好啊。喂，你好。你好，再见。", false)
	if result3 != "你好，再见。" {
		t.Errorf("新句子应该被提取，got: %s", result3)
	}

	// ASR 继续返回相同的累积文本（没有新内容）
	result4 := sm.UpdateASRText("喂，你好啊。喂，你好。你好，再见。", false)
	if result4 != "" {
		t.Errorf("重复内容应该被过滤，got: %s", result4)
	}
}

func TestStateManager_SimilarityCheckAfterNewSentence(t *testing.T) {
	sm := NewASRStateManager()

	// 第一次识别
	result1 := sm.UpdateASRText("你好。", false)
	if result1 != "你好。" {
		t.Errorf("第一句应该被提取，got: %s", result1)
	}

	// ASR 继续返回相同内容（没有新句子）
	result2 := sm.UpdateASRText("你好。", false)
	if result2 != "" {
		t.Errorf("相同内容应该被过滤，got: %s", result2)
	}

	// 用户说新的内容
	result3 := sm.UpdateASRText("你好。再见。", false)
	if result3 != "再见。" {
		t.Errorf("新句子应该被提取，got: %s", result3)
	}
}
