package audit

import (
	"encoding/json"
	"testing"
)

func TestRedact_masksSensitiveFields(t *testing.T) {
	out := Redact(map[string]any{
		"name":     "demo",
		"password": "secret123",
		"nested": map[string]any{
			"secretKey": "sk_live",
		},
	})
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !json.Valid(b) {
		t.Fatalf("invalid json: %s", s)
	}
	if contains(s, "secret123") || contains(s, "sk_live") {
		t.Fatalf("sensitive value leaked: %s", s)
	}
}

func TestBuildDetailJSON_createOmitsChanges(t *testing.T) {
	d := BuildDetailJSON(nil, map[string]any{"name": "seat-a", "workState": "offline"}, nil, "")
	m, ok := d.(map[string]any)
	if !ok {
		t.Fatalf("got %T", d)
	}
	if _, ok := m["changes"]; ok {
		t.Fatalf("create should not include changes: %+v", m)
	}
}

func TestComputeChanges_detectsFieldDiffs(t *testing.T) {
	before := map[string]any{"workState": "offline", "name": "seat-a"}
	after := map[string]any{"workState": "available", "name": "seat-a"}
	ch := ComputeChanges(before, after)
	if ch == nil {
		t.Fatal("expected changes")
	}
	ws, ok := ch["workState"].(map[string]any)
	if !ok || ws["from"] != "offline" || ws["to"] != "available" {
		t.Fatalf("workState change: %+v", ch["workState"])
	}
	if _, ok := ch["name"]; ok {
		t.Fatalf("unchanged name should not appear: %+v", ch)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
