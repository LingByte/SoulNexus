package providers

import (
	"strings"
	"testing"
)

func TestCatalogToolsUsageHint_skipsBuiltins(t *testing.T) {
	hint := CatalogToolsUsageHint([]string{"search_knowledge_base", "current_time", "ip_location"})
	if hint == "" {
		t.Fatal("expected hint")
	}
	if strings.Contains(hint, "search_knowledge_base") {
		t.Fatal("built-in knowledge tool should not appear in hint")
	}
	if !strings.Contains(hint, "current_time") || !strings.Contains(hint, "ip_location") {
		t.Fatalf("catalog tools missing from hint: %q", hint)
	}
}

func TestCatalogToolsUsageHint_empty(t *testing.T) {
	if CatalogToolsUsageHint(nil) != "" {
		t.Fatal("nil tools should yield empty hint")
	}
	if CatalogToolsUsageHint([]string{"search_knowledge_base"}) != "" {
		t.Fatal("only built-ins should yield empty hint")
	}
}
