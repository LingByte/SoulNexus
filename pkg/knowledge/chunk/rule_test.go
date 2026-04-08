package chunk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleChunker_Chunk_ParagraphPacking(t *testing.T) {
	c := &RuleChunker{}
	text := "para1 line1\npara1 line2\n\npara2\n\npara3"
	chunks, err := c.Chunk(context.Background(), text, &ChunkOptions{MaxChars: 20, OverlapChars: 0, MinChars: 1})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 2)
}

func TestRuleChunker_Chunk_Overlap(t *testing.T) {
	c := &RuleChunker{}
	text := "a1234567890\n\nB1234567890\n\nC1234567890"
	chunks, err := c.Chunk(context.Background(), text, &ChunkOptions{MaxChars: 12, OverlapChars: 3, MinChars: 1})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 2)
	assert.Contains(t, chunks[1].Text, "789")
}
