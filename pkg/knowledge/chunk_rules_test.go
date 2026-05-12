package knowledge

import (
	"context"
	"strings"
	"testing"
)

func TestStructuredRuleChunker_ByHeadingsAndMaxChars(t *testing.T) {
	c := &StructuredRuleChunker{}
	text := `
# A
第一段。第二句！

第二段。第三句？第四句。

## B
这里是第二章的内容。继续。`
	chunks, err := c.Chunk(context.Background(), text, &ChunkOptions{MaxChars: 40, OverlapChars: 0, DocumentTitle: "Doc"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("want >=2 chunks, got %d", len(chunks))
	}
	for _, ch := range chunks {
		if len([]rune(ch.Text)) > 40 {
			t.Fatalf("chunk too large: %d, text=%q", len([]rune(ch.Text)), ch.Text)
		}
		if ch.Title == "" {
			t.Fatalf("expected title")
		}
	}
}

func TestStructuredRuleChunker_OverlapAddsPrefix(t *testing.T) {
	c := &StructuredRuleChunker{}
	text := "第一句。第二句。第三句。第四句。第五句。"
	chunks, err := c.Chunk(context.Background(), text, &ChunkOptions{MaxChars: 12, OverlapChars: 4})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("want >=2 chunks, got %d", len(chunks))
	}
	// Second chunk should start with some overlap content from first.
	if strings.TrimSpace(chunks[1].Text) == strings.TrimSpace(chunks[1].Text) && !strings.Contains(chunks[1].Text, "。") {
		t.Fatalf("expected punctuation overlap, got: %q", chunks[1].Text)
	}
}

func TestTableKVChunker_KeepTableBlock(t *testing.T) {
	c := &TableKVChunker{}
	text := `
| Name | Age |
| ---- | --- |
| A    |  10 |
| B    |  20 |

姓名：张三
年龄：18
项目：LingVoice
`
	chunks, err := c.Chunk(context.Background(), text, &ChunkOptions{DocumentTitle: "T"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("want >=2 chunks, got %d", len(chunks))
	}
	// First block should remain a multi-line table.
	foundTable := false
	for _, ch := range chunks {
		if strings.Count(ch.Text, "\n") >= 3 && strings.Contains(ch.Text, "|") && strings.Contains(ch.Text, "----") {
			foundTable = true
			break
		}
	}
	if !foundTable {
		t.Fatalf("expected a table block chunk")
	}
}

