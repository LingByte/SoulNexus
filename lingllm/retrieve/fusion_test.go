package retrieve

import (
	"testing"
)

func TestFuseRRF_PrefersDualHits(t *testing.T) {
	vec := []*Document{
		{ID: "a", Content: "alpha", Score: 0.9},
		{ID: "b", Content: "beta", Score: 0.5},
	}
	kw := []*Document{
		{ID: "b", Content: "beta", Score: 12.0},
		{ID: "c", Content: "gamma", Score: 8.0},
	}
	out := fuseRRF(vec, kw, DefaultRetrievalParams())
	if len(out) == 0 {
		t.Fatal("expected results")
	}
	if out[0].ID != "b" {
		t.Fatalf("expected b on top, got %s", out[0].ID)
	}
	if out[0].Metadata["rrf_score"] == "" {
		t.Fatalf("expected rrf metadata, got %v", out[0].Metadata)
	}
	if out[0].Metadata["vector_rank"] == "" || out[0].Metadata["keyword_rank"] == "" {
		t.Fatalf("expected rank metadata, got %v", out[0].Metadata)
	}
}

func TestOverRetrieveK(t *testing.T) {
	p := DefaultRetrievalParams()
	if got := OverRetrieveK(5, p); got != 50 {
		t.Fatalf("expected min 50, got %d", got)
	}
	if got := OverRetrieveK(100, p); got != 500 {
		t.Fatalf("expected cap 500, got %d", got)
	}
}

func TestCleanPassageForRerank(t *testing.T) {
	in := "# Title\n\nSee [link](https://example.com) and `code`"
	out := CleanPassageForRerank(in)
	if out == "" {
		t.Fatal("expected non-empty")
	}
	if contains(out, "http") {
		t.Fatalf("url should be stripped: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && (s == sub || len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
