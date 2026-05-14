package voicedialog

import "testing"

func TestIsNoiseOnlyASRFinal(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"嗯嗯", true},
		{"嗯嗯。", true},
		{"嗯，", true},
		{"呃呃哦", true},
		{"  唉。  ", true},
		{"喂，可以听到我说话吗？", false},
		{"嗯，你好", false},
		{"好呀", false},
		{"你。", false},
		{"", true},
		{"。。", true},
	}
	for _, tc := range tests {
		if got := isNoiseOnlyASRFinal(tc.in); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}
