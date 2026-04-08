package chunk

import (
	"context"
	"strings"
)

type RuleChunker struct{}

func (c *RuleChunker) Provider() string { return "rule" }

func (c *RuleChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	maxChars := 1200
	overlap := 120
	minChars := 50
	if opts != nil {
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
		if opts.OverlapChars >= 0 {
			overlap = opts.OverlapChars
		}
		if opts.MinChars > 0 {
			minChars = opts.MinChars
		}
	}
	if overlap >= maxChars {
		overlap = maxChars / 10
	}

	paras := splitParagraphs(text)
	chunksText := packParagraphs(paras, maxChars)
	chunksText = applyOverlap(chunksText, overlap)

	out := make([]Chunk, 0, len(chunksText))
	for i, ct := range chunksText {
		ct = strings.TrimSpace(ct)
		if ct == "" {
			continue
		}
		if len(ct) < minChars && len(out) > 0 {
			// merge into previous
			out[len(out)-1].Text = strings.TrimSpace(out[len(out)-1].Text + "\n" + ct)
			continue
		}
		out = append(out, Chunk{Index: i, Text: ct})
	}
	return out, nil
}

func splitParagraphs(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Split by blank lines first
	blocks := strings.Split(s, "\n\n")
	out := make([]string, 0, len(blocks))
	for _, b := range blocks {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return []string{strings.TrimSpace(s)}
	}
	return out
}

func packParagraphs(paras []string, maxChars int) []string {
	out := make([]string, 0)
	var cur strings.Builder
	flush := func() {
		t := strings.TrimSpace(cur.String())
		if t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}

	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) > maxChars {
			// Oversized paragraph: split before appending into current buffer.
			if cur.Len() > 0 {
				flush()
			}
			for start := 0; start < len(p); start += maxChars {
				end := start + maxChars
				if end > len(p) {
					end = len(p)
				}
				out = append(out, strings.TrimSpace(p[start:end]))
			}
			continue
		}
		if cur.Len() == 0 {
			cur.WriteString(p)
			continue
		}
		if cur.Len()+2+len(p) <= maxChars {
			cur.WriteString("\n\n")
			cur.WriteString(p)
			continue
		}
		flush()
		cur.WriteString(p)
	}
	flush()
	return out
}

func applyOverlap(chunks []string, overlap int) []string {
	if overlap <= 0 || len(chunks) <= 1 {
		return chunks
	}
	out := make([]string, 0, len(chunks))
	out = append(out, chunks[0])
	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1]
		cur := chunks[i]
		prefix := prev
		if len(prefix) > overlap {
			prefix = prefix[len(prefix)-overlap:]
		}
		out = append(out, strings.TrimSpace(prefix+"\n"+cur))
	}
	return out
}
