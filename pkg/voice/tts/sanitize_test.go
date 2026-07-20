package tts

import "testing"

func TestSanitizeForSpeech(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"你好！", "你好！"},
		{"### 标题\n正文。", "标题 正文。"},
		{"**粗体**和`code`", "粗体和code"},
		{"[链接](https://example.com)", "链接"},
		{"```go\nfmt.Println()\n```结束", "结束"},
		{"---\n\n- 列表项", "列表项"},
		{"`ERR_CONNECTION_TIMED_OUT`", "ERR_CONNECTION_TIMED_OUT"},
		{"### 🔍 一、确认", "🔍 一、确认"},
		{"   ", ""},
		{"***", ""},
	}
	for _, tc := range tests {
		got := SanitizeForSpeech(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeForSpeech(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
