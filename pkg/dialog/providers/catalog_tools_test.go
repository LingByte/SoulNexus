package providers

import (
	"encoding/json"
	"testing"
)

func TestFilterCatalogToolsByAgentConfig_MissingKeyKeepsAll(t *testing.T) {
	rows := []CatalogToolRow{{ID: 1, Name: "a", Enabled: true}, {ID: 2, Name: "b", Enabled: true}}
	got := filterCatalogToolsByAgentConfig(rows, map[string]any{"threshold": 0.4})
	if len(got) != 2 {
		t.Fatalf("want all rows, got %d", len(got))
	}
}

func TestFilterCatalogToolsByAgentConfig_EmptyArrayMeansNone(t *testing.T) {
	rows := []CatalogToolRow{{ID: 1, Name: "a", Enabled: true}}
	got := filterCatalogToolsByAgentConfig(rows, map[string]any{"customToolIds": []any{}})
	if got != nil && len(got) != 0 {
		t.Fatalf("want none, got %+v", got)
	}
}

func TestFilterCatalogToolsByAgentConfig_StringIDs(t *testing.T) {
	rows := []CatalogToolRow{{ID: 10, Name: "a", Enabled: true}, {ID: 20, Name: "b", Enabled: true}}
	got := filterCatalogToolsByAgentConfig(rows, map[string]any{"customToolIds": []any{"20", json.Number("10")}})
	if len(got) != 2 {
		t.Fatalf("got=%+v", got)
	}
}

func TestFilterCatalogToolsByAgentConfig_MCPBindKey(t *testing.T) {
	rows := []CatalogToolRow{
		{ID: 10, Name: "srv", Kind: CatalogToolKindMCPSSE, Enabled: true, McpSSEURL: "http://127.0.0.1:3920/sse"},
		{ID: 20, Name: "http", Kind: CatalogToolKindHTTP, Enabled: true},
	}
	got := filterCatalogToolsByAgentConfig(rows, map[string]any{"customToolIds": []any{"10:order_lookup"}})
	if len(got) != 1 || got[0].ID != 10 {
		t.Fatalf("got=%+v", got)
	}
	sel := parseCatalogToolSelection(map[string]any{"customToolIds": []any{"10:order_lookup"}})
	eps := catalogMCPSSEEndpoints(got, sel)
	if len(eps) != 1 || eps[0].WholeServer || len(eps[0].AllowTools) != 1 {
		t.Fatalf("eps=%+v", eps)
	}
	if _, ok := eps[0].AllowTools["order_lookup"]; !ok {
		t.Fatalf("missing tool allow: %+v", eps[0].AllowTools)
	}
}

func TestSplitCatalogMCPBindKey(t *testing.T) {
	id, name, ok := splitCatalogMCPBindKey("42:order_lookup")
	if !ok || id != 42 || name != "order_lookup" {
		t.Fatalf("got %v %q %v", id, name, ok)
	}
	if _, _, ok := splitCatalogMCPBindKey("42"); ok {
		t.Fatal("expected false")
	}
}
