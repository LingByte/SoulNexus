package knowledge

import (
	"strings"
	"testing"
)

func TestParseLLMChunks_WrapperObject(t *testing.T) {
	raw := `{"chunks":[{"title":"A","text":"hello","metadata":{"k":1}},{"title":"B","text":"world"}]}`
	chunks, err := parseLLMChunks(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("want 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Title != "A" || chunks[0].Text != "hello" {
		t.Fatalf("unexpected chunk0: %+v", chunks[0])
	}
	if chunks[0].Metadata == nil || chunks[0].Metadata["k"] == nil {
		t.Fatalf("expected metadata")
	}
}

func TestParseLLMChunks_ArrayLegacy_CodeFence(t *testing.T) {
	raw := "```json\n" + `[{"title":"A","text":"t1"},{"title":"B","text":"t2"}]` + "\n```"
	chunks, err := parseLLMChunks(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("want 2, got %d", len(chunks))
	}
	if chunks[1].Text != "t2" {
		t.Fatalf("unexpected: %+v", chunks[1])
	}
}

func TestExtractBalancedDelimiters_IgnoreBracketsInStrings(t *testing.T) {
	s := `prefix {"chunks":[{"title":"A","text":"has [brackets] and {braces} inside"}]} suffix`
	i := strings.IndexByte(s, '{')
	frag, ok := extractBalancedDelimiters(s, i, '{', '}')
	if !ok {
		t.Fatalf("expected ok")
	}
	if !strings.Contains(frag, `"chunks"`) || !strings.Contains(frag, `has [brackets]`) {
		t.Fatalf("unexpected frag: %s", frag)
	}
}

