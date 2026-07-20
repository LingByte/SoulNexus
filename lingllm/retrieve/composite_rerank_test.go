package retrieve

import "testing"

func TestCompositeRerankScore(t *testing.T) {
	// model=0.8, base normalized to 1.0 -> 0.6*0.8 + 0.3*1.0 = 0.78
	got := CompositeRerankScore(0.8, 0.9, 0.1, 0.9, 0.6, 0.3)
	want := 0.6*0.8 + 0.3*1.0
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("composite=%v want=%v", got, want)
	}
}

func TestCompositeRerankScore_DisabledWeightsFallback(t *testing.T) {
	got := CompositeRerankScore(1.0, 0.5, 0, 1, 0, 0)
	if got <= 0 {
		t.Fatalf("expected positive composite, got %v", got)
	}
}

func TestApplyCompositeClone_Metadata(t *testing.T) {
	doc := &Document{ID: "x", Score: 0.4, Content: "hello"}
	params := DefaultRetrievalParams()
	out := applyCompositeClone(doc, 0.9, 0.4, 0.2, 0.8, params)
	if out.Metadata["model_score"] == "" || out.Metadata["base_score"] == "" {
		t.Fatalf("metadata=%v", out.Metadata)
	}
	if out.Score <= 0 {
		t.Fatalf("score=%v", out.Score)
	}
}

func TestApplyCompositeClone_PlainModelWhenDisabled(t *testing.T) {
	doc := &Document{ID: "x", Score: 0.4}
	params := DefaultRetrievalParams()
	params.EnableCompositeRerank = false
	out := applyCompositeClone(doc, 0.75, 0.4, 0, 1, params)
	if out.Score != 0.75 {
		t.Fatalf("score=%v", out.Score)
	}
}
