package chunk

import (
	"context"
	"strings"
)

const (
	DefaultRuleChunkMaxChars = 1200
	DefaultRuleChunkMinChars = 80
)

// StructuredRuleChunker 结构化文档规则分块器（标题 / 启发式 / 递归策略自动选择）
type StructuredRuleChunker struct {
	config *Config
}

// NewStructuredRuleChunker 创建结构化规则分块器
func NewStructuredRuleChunker(cfg *Config) *StructuredRuleChunker {
	return &StructuredRuleChunker{config: cfg}
}

func (c *StructuredRuleChunker) Provider() string {
	return "rules_structured"
}

func (c *StructuredRuleChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	minChars := DefaultRuleChunkMinChars
	if opts != nil && opts.MinChars > 0 {
		minChars = opts.MinChars
	}

	strategy := ChunkStrategyAuto
	if opts != nil && strings.TrimSpace(opts.Strategy) != "" {
		strategy = strings.TrimSpace(opts.Strategy)
	} else if c.config != nil && strings.TrimSpace(c.config.Strategy) != "" {
		strategy = strings.TrimSpace(c.config.Strategy)
	} else if c.config != nil && c.config.CustomConfig != nil {
		if s, ok := c.config.CustomConfig["strategy"].(string); ok && strings.TrimSpace(s) != "" {
			strategy = strings.TrimSpace(s)
		}
	}

	out := splitToPublicChunks(Split(text, splitterConfigFromOptions(opts, strategy)), opts)
	out = dropTinyTrailing(out, minChars)

	if len(out) == 1 && len(strings.TrimSpace(out[0].Text)) < minChars {
		return nil, ErrNoChunks
	}

	if len(out) == 0 {
		return nil, ErrNoChunks
	}

	for i := range out {
		out[i].Index = i
	}

	return out, nil
}

// TableKVChunker 表格/键值对分块器
type TableKVChunker struct {
	config *Config
}

// NewTableKVChunker 创建表格/键值对分块器
func NewTableKVChunker(cfg *Config) *TableKVChunker {
	return &TableKVChunker{config: cfg}
}

func (c *TableKVChunker) Provider() string {
	return "rules_table_kv"
}

func (c *TableKVChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	maxChars := DefaultRuleChunkMaxChars
	if opts != nil && opts.MaxChars > 0 {
		maxChars = opts.MaxChars
	}

	records := strings.Split(text, "\n\n")
	var chunks []Chunk

	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		if len(record) <= maxChars {
			chunks = append(chunks, Chunk{
				Text:     record,
				Metadata: make(map[string]interface{}),
			})
			continue
		}

		lines := strings.Split(record, "\n")
		var currentChunk strings.Builder
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if currentChunk.Len()+len(line)+1 > maxChars {
				if currentChunk.Len() > 0 {
					chunks = append(chunks, Chunk{
						Text:     currentChunk.String(),
						Metadata: make(map[string]interface{}),
					})
					currentChunk.Reset()
				}
			}
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n")
			}
			currentChunk.WriteString(line)
		}
		if currentChunk.Len() > 0 {
			chunks = append(chunks, Chunk{
				Text:     currentChunk.String(),
				Metadata: make(map[string]interface{}),
			})
		}
	}

	if len(chunks) == 0 {
		return nil, ErrNoChunks
	}

	for i := range chunks {
		chunks[i].Index = i
	}
	return chunks, nil
}

func dropTinyTrailing(chunks []Chunk, minChars int) []Chunk {
	if len(chunks) == 0 {
		return chunks
	}
	for len(chunks) > 1 && len(chunks[len(chunks)-1].Text) < minChars {
		chunks[len(chunks)-2].Text += "\n" + chunks[len(chunks)-1].Text
		chunks = chunks[:len(chunks)-1]
	}
	return chunks
}
