package sessions

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestFilterWordComponent_ContainsEmoji(t *testing.T) {
	logger := zap.NewNop()
	filter := &FilterWordComponent{
		logger:      logger,
		filterWords: make(map[string]bool),
	}

	tests := []struct {
		name     string
		text     string
		hasEmoji bool
	}{
		{
			name:     "纯文本",
			text:     "你好世界",
			hasEmoji: false,
		},
		{
			name:     "包含笑脸emoji",
			text:     "你好😊",
			hasEmoji: true,
		},
		{
			name:     "包含哭脸emoji",
			text:     "难过😢",
			hasEmoji: true,
		},
		{
			name:     "包含爱心emoji",
			text:     "我爱你❤️",
			hasEmoji: true,
		},
		{
			name:     "包含手势emoji",
			text:     "点赞👍",
			hasEmoji: true,
		},
		{
			name:     "包含天气emoji",
			text:     "今天天气☀️很好",
			hasEmoji: true,
		},
		{
			name:     "包含动物emoji",
			text:     "小狗🐕很可爱",
			hasEmoji: true,
		},
		{
			name:     "包含食物emoji",
			text:     "吃披萨🍕",
			hasEmoji: true,
		},
		{
			name:     "英文文本",
			text:     "Hello World",
			hasEmoji: false,
		},
		{
			name:     "数字和标点",
			text:     "123!@#$%",
			hasEmoji: false,
		},
		{
			name:     "混合中英文",
			text:     "Hello 你好 World 世界",
			hasEmoji: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.containsEmoji(tt.text)
			if result != tt.hasEmoji {
				t.Errorf("containsEmoji(%q) = %v, want %v", tt.text, result, tt.hasEmoji)
			}
		})
	}
}

func TestFilterWordComponent_Process(t *testing.T) {
	logger := zap.NewNop()
	filter := &FilterWordComponent{
		logger:      logger,
		filterWords: make(map[string]bool),
	}
	ctx := context.Background()

	tests := []struct {
		name           string
		input          string
		shouldContinue bool
		expectError    bool
	}{
		{
			name:           "纯文本应该通过",
			input:          "你好世界",
			shouldContinue: true,
			expectError:    false,
		},
		{
			name:           "包含emoji应该被过滤",
			input:          "你好😊",
			shouldContinue: false,
			expectError:    false,
		},
		{
			name:           "英文文本应该通过",
			input:          "Hello World",
			shouldContinue: true,
			expectError:    false,
		},
		{
			name:           "包含多个emoji应该被过滤",
			input:          "今天天气☀️很好😊",
			shouldContinue: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, shouldContinue, err := filter.Process(ctx, tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("期望错误但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误但返回了错误: %v", err)
				return
			}

			if shouldContinue != tt.shouldContinue {
				t.Errorf("shouldContinue = %v, want %v", shouldContinue, tt.shouldContinue)
			}

			if tt.shouldContinue {
				if output == nil {
					t.Error("应该返回输出但返回了nil")
				} else if output.(string) != tt.input {
					t.Errorf("output = %v, want %v", output, tt.input)
				}
			}
		})
	}
}

func TestFilterWordComponent_Process_InvalidType(t *testing.T) {
	logger := zap.NewNop()
	filter := &FilterWordComponent{
		logger:      logger,
		filterWords: make(map[string]bool),
	}
	ctx := context.Background()

	// 测试无效的数据类型
	_, _, err := filter.Process(ctx, 123)
	if err == nil {
		t.Error("期望错误但没有返回错误")
	}
}
