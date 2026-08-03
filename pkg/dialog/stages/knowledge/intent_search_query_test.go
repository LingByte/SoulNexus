package knowledge

import (
	"strings"
	"testing"
)

func TestUserUtteranceForSearch_StripsLegacySystemBlock(t *testing.T) {
	raw := "开源扶持政策是什么\n\n【系统·提示】请在业务意图与知识库范围内作答；不要编造未配置的意图。"
	got := UserUtteranceForSearch(raw)
	want := "开源扶持政策是什么"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCompactSearchQuery_IgnoresLegacySystemPrompt(t *testing.T) {
	raw := "查一下价格\n\n【系统·提示】请在业务意图与知识库范围内作答。"
	got := CompactSearchQuery(raw)
	if strings.Contains(got, "系统·提示") || strings.Contains(got, "业务意图") {
		t.Fatalf("compact query still contains legacy system prompt: %q", got)
	}
	if got == "" {
		t.Fatal("expected non-empty compact query")
	}
}
