package cascaded

import (
	"context"
	"testing"
)

func TestMaxToolRounds(t *testing.T) {
	ctx := WithMaxToolRounds(context.Background(), 12)
	if got := MaxToolRounds(ctx); got != 12 {
		t.Fatalf("MaxToolRounds=%d want 12", got)
	}
	ctx2 := WithMaxToolRounds(context.Background(), 0)
	if got := MaxToolRounds(ctx2); got != DefaultTextToolRounds {
		t.Fatalf("MaxToolRounds default=%d want %d", got, DefaultTextToolRounds)
	}
}

func TestForcedKnowledgeBlock(t *testing.T) {
	ctx := WithForcedKnowledgeBlock(context.Background(), "kb-block")
	if got := ForcedKnowledgeBlock(ctx); got != "kb-block" {
		t.Fatalf("ForcedKnowledgeBlock=%q", got)
	}
}
