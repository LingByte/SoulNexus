package chat

import (
	"strings"
	"testing"
)

func TestFormatToolsMemoryAppendix(t *testing.T) {
	raw := `[{"name":"order_lookup","arguments":"{\"id\":\"1\"}"},{"name":"run_skill_demo","arguments":"{}"}]`
	got := FormatToolsMemoryAppendix(raw)
	if got == "" || !strings.Contains(got, "【工具记忆") || !strings.Contains(got, "order_lookup") || !strings.Contains(got, "run_skill_demo") {
		t.Fatalf("got %q", got)
	}
	if FormatToolsMemoryAppendix("") != "" {
		t.Fatal("empty should yield empty")
	}
	if FormatToolsMemoryAppendix("{") != "" {
		t.Fatal("invalid json should yield empty")
	}
}
