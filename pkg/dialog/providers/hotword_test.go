package providers

import (
	"strings"
	"testing"
)

func TestHotwordCorrectorExactAndFuzzy(t *testing.T) {
	raw := []byte(`[
		{"word":"龙鳞宫","weight":10,"replacedWords":["农民工","龙陵","龙陵宫"],"enableFuzzyMatch":true},
		{"word":"森林","replacedWords":["生林"]}
	]`)
	c := ParseHotwordBundle(raw, nil, nil).Corrector
	if c == nil {
		t.Fatal("expected corrector")
	}

	cases := []struct {
		in, want string
	}{
		{"我想去农民工", "我想去龙鳞宫"},
		{"你说的是龙林公吗", "你说的是龙鳞宫吗"},
		{"小孩说想去龙陵宫玩", "小孩说想去龙鳞宫玩"},
		{"去龙陵宫怎么走", "去龙鳞宫怎么走"},
		{"保护生林环境", "保护森林环境"},
		{"保护深林环境", "保护深林环境"},
	}
	for _, tc := range cases {
		got := c.Correct(tc.in)
		if got != tc.want {
			t.Errorf("Correct(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTTSTextOptimizerPhonemeAndReplace(t *testing.T) {
	raw := []byte(`[
		{"word":"{零一|ling2yi1}","replacedWords":["01","灵艺"]}
	]`)
	bundle := ParseHotwordBundle(raw, nil, map[string]any{"ssml": true})
	if bundle.TTS == nil {
		t.Fatal("expected TTS optimizer")
	}
	got := bundle.TTS.Prepare("欢迎了解01科技")
	if got == "" {
		t.Fatal("empty TTS text")
	}
	if got == "欢迎了解01科技" {
		t.Fatalf("expected replacement, got %q", got)
	}
	if !strings.Contains(got, "零一") || !strings.Contains(got, "<phoneme") || !strings.Contains(got, "<speak>") {
		t.Fatalf("unexpected TTS output: %q", got)
	}
}
