package providers

import "testing"

func TestNeedsNonStreamToolRound_BuiltinOnly(t *testing.T) {
	if NeedsNonStreamToolRound([]string{"search_knowledge_base"}, "喂，听得到吗？") {
		t.Fatal("builtin tools must not force non-stream")
	}
}

func TestNeedsNonStreamToolRound_CatalogShortChitchat(t *testing.T) {
	tools := []string{"search_knowledge_base", "cloudsteps_book_lesson"}
	if NeedsNonStreamToolRound(tools, "喂，听得到我说话吗？") {
		t.Fatal("short chitchat with catalog tools should still stream")
	}
	if NeedsNonStreamToolRound(tools, "自我介绍一下") {
		t.Fatal("short intro request should stream")
	}
}

func TestNeedsNonStreamToolRound_CatalogAction(t *testing.T) {
	tools := []string{"cloudsteps_book_lesson"}
	if !NeedsNonStreamToolRound(tools, "帮我预约明天的课程") {
		t.Fatal("booking utterance should force tool round")
	}
}
