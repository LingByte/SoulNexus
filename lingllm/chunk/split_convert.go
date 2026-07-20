package chunk

import "strings"

// ChunkStrategy* are aliases for SplitterConfig.Strategy values.
const (
	ChunkStrategyAuto      = StrategyAuto
	ChunkStrategyHeading   = StrategyHeading
	ChunkStrategyHeuristic = StrategyHeuristic
	ChunkStrategyRecursive = StrategyRecursive
	ChunkStrategyLegacy    = StrategyLegacy
)

func splitterConfigFromOptions(opts *ChunkOptions, strategy string) SplitterConfig {
	cfg := DefaultConfig()
	if opts != nil {
		if opts.MaxChars > 0 {
			cfg.ChunkSize = opts.MaxChars
		}
		if opts.OverlapChars >= 0 {
			cfg.ChunkOverlap = opts.OverlapChars
		}
		if opts.PreChunkClean != nil {
			if tokenLimit, ok := opts.PreChunkClean["token_limit"].(int); ok && tokenLimit > 0 {
				cfg.TokenLimit = tokenLimit
			}
		}
	}
	if strategy != "" {
		cfg.Strategy = strategy
	}
	return cfg
}

func splitToPublicChunks(raw []splitChunk, opts *ChunkOptions) []Chunk {
	docTitle := ""
	if opts != nil {
		docTitle = strings.TrimSpace(opts.DocumentTitle)
	}

	out := make([]Chunk, 0, len(raw))
	for i, c := range raw {
		title := docTitle
		if c.ContextHeader != "" {
			if title != "" {
				title = title + "\n" + c.ContextHeader
			} else {
				title = c.ContextHeader
			}
		}

		out = append(out, Chunk{
			Index: i,
			Title: title,
			Text:  strings.TrimSpace(c.EmbeddingContent()),
			Metadata: map[string]interface{}{
				"start":          c.Start,
				"end":            c.End,
				"context_header": c.ContextHeader,
			},
		})
	}
	return out
}
