package knowledge

import (
	"strings"
	"testing"
)

func TestUserUtteranceForSearch_StripsNLUBlock(t *testing.T) {
	raw := "开源扶持政策是什么\n\n【系统·NLU】初步意图=查询，置信度=0.910。请在业务意图与知识库范围内作答；不要编造未配置的意图。"
	got := UserUtteranceForSearch(raw)
	want := "开源扶持政策是什么"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCompactSearchQuery_IgnoresNLUPrompt(t *testing.T) {
	raw := "查一下价格\n\n【系统·NLU】初步意图=询价，置信度=0.900。请在业务意图与知识库范围内作答。"
	got := CompactSearchQuery(raw)
	if strings.Contains(got, "系统·NLU") || strings.Contains(got, "置信度") {
		t.Fatalf("compact query still contains NLU prompt: %q", got)
	}
	if got == "" {
		t.Fatal("expected non-empty compact query")
	}
}
