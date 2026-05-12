package knowledge

import (
	"context"
	"regexp"
	"strings"
)

const (
	DefaultRuleChunkMaxChars = 1200
	DefaultRuleChunkMinChars = 80
)

// StructuredRuleChunker chunks structured docs deterministically:
// headings -> paragraphs -> sentences -> fallback hard split, with overlap.
type StructuredRuleChunker struct{}

func (c *StructuredRuleChunker) Provider() string { return "rules_structured" }

func (c *StructuredRuleChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	maxChars := DefaultRuleChunkMaxChars
	minChars := DefaultRuleChunkMinChars
	overlap := int(float64(maxChars) * 0.12) // 10%~15%
	title := ""
	if opts != nil {
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
		if opts.MinChars > 0 {
			minChars = opts.MinChars
		}
		if opts.OverlapChars >= 0 {
			overlap = opts.OverlapChars
		}
		title = strings.TrimSpace(opts.DocumentTitle)
	}
	if maxChars <= 0 {
		return nil, ErrInvalidChunkOpt
	}
	if overlap < 0 {
		overlap = 0
	}

	sections := splitByHeadings(text)
	if len(sections) == 0 {
		sections = []headingSection{{Heading: "", Body: text}}
	}

	var out []Chunk
	for _, sec := range sections {
		secTitle := strings.TrimSpace(sec.Heading)
		if secTitle == "" {
			secTitle = title
		} else if title != "" {
			secTitle = title + " / " + secTitle
		}
		chunks := chunkSectionBody(sec.Body, secTitle, maxChars, minChars)
		out = append(out, chunks...)
	}
	out = applySentenceOverlap(out, overlap)
	out = dropTinyTrailing(out, minChars)
	if len(out) == 0 {
		return nil, ErrNoChunks
	}
	for i := range out {
		out[i].Index = i
	}
	return out, nil
}

// TableKVChunker chunks table/kv docs without breaking tables.
// Strategy: keep table blocks intact, otherwise split by "record" (blank-line blocks, then lines).
type TableKVChunker struct{}

func (c *TableKVChunker) Provider() string { return "table_kv" }

func (c *TableKVChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}
	title := ""
	minChars := DefaultRuleChunkMinChars
	if opts != nil {
		title = strings.TrimSpace(opts.DocumentTitle)
		if opts.MinChars > 0 {
			minChars = opts.MinChars
		}
	}

	blocks := splitByBlankLines(text)
	var out []Chunk
	for _, b := range blocks {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		// If this block looks like a table, keep as-is.
		if looksLikeTableBlock(b) {
			out = append(out, Chunk{Title: title, Text: b})
			continue
		}
		// For kv-like blocks, one line == one record.
		lines := strings.Split(b, "\n")
		kvCount := 0
		for _, ln := range lines {
			if reKVLine.MatchString(strings.TrimSpace(ln)) {
				kvCount++
			}
		}
		if kvCount >= 3 {
			for _, ln := range lines {
				ln = strings.TrimSpace(ln)
				if ln == "" {
					continue
				}
				out = append(out, Chunk{Title: title, Text: ln})
			}
			continue
		}
		// Otherwise treat the whole block as a "record" (e.g. one resume section / Q&A).
		out = append(out, Chunk{Title: title, Text: b})
	}
	out = dropTinyTrailing(out, minChars)
	if len(out) == 0 {
		return nil, ErrNoChunks
	}
	for i := range out {
		out[i].Index = i
	}
	return out, nil
}

var _ Chunker = (*StructuredRuleChunker)(nil)
var _ Chunker = (*TableKVChunker)(nil)

// ---------------- internal helpers ----------------

type headingSection struct {
	Heading string
	Body    string
}

var (
	reHeadingLine = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+\S+.*$|^\s*第\s*[0-9一二三四五六七八九十百千]+\s*章\b.*$|^\s*\d+(\.\d+){0,3}\s*[\.\-、]?\s*\S+.*$|^\s*[一二三四五六七八九十百千]+\s*、\s*\S+.*$|^\s*（[一二三四五六七八九十百千]+）\s*\S+.*$`)
)

func splitByHeadings(s string) []headingSection {
	lines := strings.Split(s, "\n")
	var secs []headingSection
	cur := headingSection{}
	var buf []string

	flush := func() {
		body := strings.TrimSpace(strings.Join(buf, "\n"))
		if strings.TrimSpace(cur.Heading) == "" && body == "" {
			buf = nil
			return
		}
		cur.Body = body
		secs = append(secs, cur)
		cur = headingSection{}
		buf = nil
	}

	for _, ln := range lines {
		if reHeadingLine.MatchString(ln) {
			// start new section
			if cur.Heading != "" || len(buf) > 0 {
				flush()
			}
			cur.Heading = strings.TrimSpace(ln)
			continue
		}
		buf = append(buf, ln)
	}
	if cur.Heading != "" || len(buf) > 0 {
		flush()
	}

	// If first section has empty heading and empty body, drop.
	var out []headingSection
	for _, x := range secs {
		if strings.TrimSpace(x.Heading) == "" && strings.TrimSpace(x.Body) == "" {
			continue
		}
		out = append(out, x)
	}
	return out
}

func splitByBlankLines(s string) []string {
	// Normalize Windows newlines.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	parts := strings.Split(s, "\n\n")
	if len(parts) <= 1 {
		return []string{s}
	}
	return parts
}

func splitIntoSentences(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// A small heuristic: split by common punctuation; keep delimiters.
	var out []string
	var buf strings.Builder
	for _, r := range s {
		buf.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '；' || r == '.' || r == '!' || r == '?' || r == ';' {
			t := strings.TrimSpace(buf.String())
			if t != "" {
				out = append(out, t)
			}
			buf.Reset()
		}
	}
	if t := strings.TrimSpace(buf.String()); t != "" {
		out = append(out, t)
	}
	return out
}

func chunkSectionBody(body, title string, maxChars, minChars int) []Chunk {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}

	paras := splitByBlankLines(body)
	var chunks []Chunk
	var cur []string
	curLen := 0

	flush := func() {
		txt := strings.TrimSpace(strings.Join(cur, "\n\n"))
		cur = nil
		curLen = 0
		if txt == "" {
			return
		}
		chunks = append(chunks, Chunk{Title: title, Text: txt})
	}

	appendUnit := func(unit string) {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			return
		}
		ulen := len([]rune(unit))
		sep := 0
		if len(cur) > 0 {
			sep = 2 // "\n\n"
		}
		if curLen+sep+ulen <= maxChars || len(cur) == 0 {
			if sep > 0 {
				curLen += sep
			}
			cur = append(cur, unit)
			curLen += ulen
			return
		}
		flush()
		cur = append(cur, unit)
		curLen = ulen
	}

	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len([]rune(p)) <= maxChars {
			appendUnit(p)
			continue
		}
		// Too long paragraph: split by sentences.
		sents := splitIntoSentences(p)
		if len(sents) == 0 {
			// Fallback hard split.
			for _, part := range hardSplitRunes(p, maxChars) {
				appendUnit(part)
			}
			continue
		}
		var sb strings.Builder
		for _, sen := range sents {
			sen = strings.TrimSpace(sen)
			if sen == "" {
				continue
			}
			if len([]rune(sen)) > maxChars {
				// Very long sentence: hard split.
				if sb.Len() > 0 {
					appendUnit(sb.String())
					sb.Reset()
				}
				for _, part := range hardSplitRunes(sen, maxChars) {
					appendUnit(part)
				}
				continue
			}
			if sb.Len() == 0 {
				sb.WriteString(sen)
				continue
			}
			// Add to current sentence-group if fits, else flush as one unit.
			cand := sb.String() + sen
			if len([]rune(cand)) <= maxChars {
				sb.WriteString(sen)
				continue
			}
			appendUnit(sb.String())
			sb.Reset()
			sb.WriteString(sen)
		}
		if sb.Len() > 0 {
			appendUnit(sb.String())
		}
	}
	if len(cur) > 0 {
		flush()
	}

	// Merge tiny chunks into previous when possible.
	if minChars > 0 && len(chunks) > 1 {
		var merged []Chunk
		for _, ch := range chunks {
			if len(merged) == 0 {
				merged = append(merged, ch)
				continue
			}
			if runeLen(ch.Text) < minChars && runeLen(merged[len(merged)-1].Text)+2+runeLen(ch.Text) <= maxChars {
				merged[len(merged)-1].Text = strings.TrimSpace(merged[len(merged)-1].Text + "\n\n" + ch.Text)
				continue
			}
			merged = append(merged, ch)
		}
		chunks = merged
	}
	return chunks
}

func hardSplitRunes(s string, maxChars int) []string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) == 0 {
		return nil
	}
	if maxChars <= 0 {
		return []string{string(rs)}
	}
	var out []string
	for start := 0; start < len(rs); start += maxChars {
		end := start + maxChars
		if end > len(rs) {
			end = len(rs)
		}
		part := strings.TrimSpace(string(rs[start:end]))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func applySentenceOverlap(chunks []Chunk, overlapChars int) []Chunk {
	if overlapChars <= 0 || len(chunks) < 2 {
		return chunks
	}
	out := make([]Chunk, len(chunks))
	copy(out, chunks)

	for i := 1; i < len(out); i++ {
		prev := strings.TrimSpace(out[i-1].Text)
		if prev == "" {
			continue
		}
		// Take overlap from end of prev by sentence units if possible.
		sents := splitIntoSentences(prev)
		if len(sents) == 0 {
			tail := tailRunes(prev, overlapChars)
			if tail != "" {
				out[i].Text = strings.TrimSpace(tail + "\n" + out[i].Text)
			}
			continue
		}
		var picked []string
		total := 0
		for j := len(sents) - 1; j >= 0; j-- {
			rn := runeLen(sents[j])
			if total+rn > overlapChars && len(picked) > 0 {
				break
			}
			picked = append([]string{sents[j]}, picked...)
			total += rn
			if total >= overlapChars {
				break
			}
		}
		if len(picked) == 0 {
			continue
		}
		ov := strings.TrimSpace(strings.Join(picked, ""))
		if ov != "" {
			out[i].Text = strings.TrimSpace(ov + "\n" + out[i].Text)
		}
	}
	return out
}

func tailRunes(s string, n int) string {
	rs := []rune(s)
	if n <= 0 || len(rs) == 0 {
		return ""
	}
	if len(rs) <= n {
		return strings.TrimSpace(string(rs))
	}
	return strings.TrimSpace(string(rs[len(rs)-n:]))
}

func runeLen(s string) int { return len([]rune(s)) }

func looksLikeTableBlock(b string) bool {
	lines := strings.Split(b, "\n")
	pipeLines := 0
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		if strings.Count(t, "|") >= 2 || reTableSep.MatchString(t) {
			pipeLines++
		}
	}
	return pipeLines >= 2
}

func dropTinyTrailing(chunks []Chunk, minChars int) []Chunk {
	if minChars <= 0 || len(chunks) == 0 {
		return chunks
	}
	var out []Chunk
	for _, ch := range chunks {
		ch.Text = strings.TrimSpace(ch.Text)
		if ch.Text == "" {
			continue
		}
		out = append(out, ch)
	}
	return out
}

