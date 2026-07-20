package retrieve

import "testing"

func TestDeduplicateDocuments_ByIDAndSignature(t *testing.T) {
	docs := []*Document{
		{ID: "a", Content: "Hello World", Score: 0.9},
		{ID: "a", Content: "Hello World", Score: 0.8},
		{ID: "b", Content: "hello   world", Score: 0.7},
		{ID: "c", Content: "Different", Score: 0.6},
	}
	out := DeduplicateDocuments(docs)
	if len(out) != 2 {
		t.Fatalf("expected 2 docs after dedup, got %d", len(out))
	}
	if out[0].ID != "a" || out[1].ID != "c" {
		t.Fatalf("unexpected ids: %v, %v", out[0].ID, out[1].ID)
	}
}

func TestRemovePartialOverlaps_DropsContained(t *testing.T) {
	docs := []*Document{
		{ID: "long", Content: "Go programming language is great for backend services", Score: 0.9},
		{ID: "short", Content: "Go programming language", Score: 0.95},
	}
	out := RemovePartialOverlaps(docs, 0.85)
	if len(out) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(out))
	}
	if out[0].ID != "long" {
		t.Fatalf("expected higher-scored long doc, got %s", out[0].ID)
	}
}

func TestBuildContentSignature_NormalizesWhitespace(t *testing.T) {
	a := BuildContentSignature("Hello   World")
	b := BuildContentSignature("hello world")
	if a == "" || a != b {
		t.Fatalf("signatures should match: %q vs %q", a, b)
	}
}

func TestContentOverlapRatio(t *testing.T) {
	ratio := ContentOverlapRatio("foo bar baz", "foo bar baz qux")
	if ratio < 0.99 {
		t.Fatalf("expected near 1 overlap, got %v", ratio)
	}
}

func TestApplyMMR_ReducesRedundancy(t *testing.T) {
	docs := []*Document{
		{ID: "1", Content: "machine learning neural networks", Score: 0.9},
		{ID: "2", Content: "machine learning deep learning", Score: 0.85},
		{ID: "3", Content: "database sql indexing", Score: 0.8},
	}
	out := ApplyMMR(docs, 2, 0.7)
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	ids := map[string]bool{out[0].ID: true, out[1].ID: true}
	if !ids["3"] {
		t.Fatalf("expected diverse doc 3 in result, got %v %v", out[0].ID, out[1].ID)
	}
}
