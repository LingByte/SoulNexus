package chunk

import (
	"context"
	"errors"
)

var (
	ErrEmptyText = errors.New("empty text")
)

type Chunk struct {
	Index    int
	Title    string
	Text     string
	Metadata map[string]any
}

type ChunkOptions struct {
	// Target size for each chunk.
	MaxChars int
	// Overlap between consecutive chunks.
	OverlapChars int
	// Minimum characters to keep a chunk; smaller ones may be merged/dropped by implementations.
	MinChars int
	// Optional: original document title.
	DocumentTitle string
}

type Chunker interface {
	Provider() string
	Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error)
}
