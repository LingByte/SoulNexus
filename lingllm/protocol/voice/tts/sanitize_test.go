package tts

import "testing"

func TestSanitizeSpeech(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Hello world!", "Hello world!"},
		{"**Sure**, I can hear you.", "Sure, I can hear you."},
		{"😊", ""},
		{"---", ""},
		{"* * *", ""},
		{"What's your name?", "What's your name?"},
		{"<speak>bad</speak>", "bad"},
		{"，", ""},
		{"...", ""},
		{"* * *", ""},
	}
	for _, tc := range tests {
		got := SanitizeSpeech(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeSpeech(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
