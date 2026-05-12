package knowledge

import (
	"context"
	"regexp"
	"strings"
	"unicode"
)

// RuleBasedDocumentTypeDetector classifies documents by simple heuristics.
//
// Targets:
// - Structured (90%): manuals, papers, contracts, reports, markdown
// - Table/KV (5%): resumes, forms, questionnaires, excel-to-text, financial docs
// - Unstructured noisy (5%): OCR text, novels, garbled webpages, chat logs
type RuleBasedDocumentTypeDetector struct{}

func (d *RuleBasedDocumentTypeDetector) DetectDocumentType(ctx context.Context, text string) (DocumentType, error) {
	_ = ctx
	s := strings.TrimSpace(text)
	if s == "" {
		return DocumentTypeUnknown, ErrEmptyText
	}

	// Quick wins for table / kv.
	if isTableKVLike(s) {
		return DocumentTypeTableKV, nil
	}

	// Structured if any common heading patterns exist.
	if hasStructuredHeadings(s) {
		return DocumentTypeStructured, nil
	}

	// Otherwise, determine if it's noisy/unstructured.
	if looksUnstructuredNoisy(s) {
		return DocumentTypeUnstructured, nil
	}

	// Default: treat as structured text (most docs).
	return DocumentTypeStructured, nil
}

var (
	reMarkdownHeading = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+\S+`)
	reChapterCN       = regexp.MustCompile(`(?m)^\s*第\s*[0-9一二三四五六七八九十百千]+\s*章\b`)
	reHeadingNumeric  = regexp.MustCompile(`(?m)^\s*\d+(\.\d+){0,3}\s+`)
	reHeadingNumeric2 = regexp.MustCompile(`(?m)^\s*\d+(\.\d+){0,3}\s*[\.\-、]\s*\S+`) // "1." / "1.1" / "1.1.1"
	reHeadingCN1      = regexp.MustCompile(`(?m)^\s*[一二三四五六七八九十百千]+\s*、\s*\S+`)
	reHeadingCN2      = regexp.MustCompile(`(?m)^\s*（[一二三四五六七八九十百千]+）\s*\S+`)

	reKVLine = regexp.MustCompile(`(?m)^\s*[\p{Han}A-Za-z][\p{Han}A-Za-z0-9_\- /]{0,30}\s*[:：]\s*\S+`)
	// A lightweight table signal: many '|' lines, or a markdown table separator.
	reTableSep = regexp.MustCompile(`(?m)^\s*\|?[-: ]+\|[-:| ]+\|?\s*$`)
)

func hasStructuredHeadings(s string) bool {
	return reMarkdownHeading.FindStringIndex(s) != nil ||
		reChapterCN.FindStringIndex(s) != nil ||
		reHeadingNumeric2.FindStringIndex(s) != nil ||
		reHeadingNumeric.FindStringIndex(s) != nil ||
		reHeadingCN1.FindStringIndex(s) != nil ||
		reHeadingCN2.FindStringIndex(s) != nil
}

func isTableKVLike(s string) bool {
	lines := splitLinesLimit(s, 400)
	if len(lines) == 0 {
		return false
	}

	pipeLines := 0
	kvLines := 0
	longKVRun := 0
	maxKVRun := 0
	tableSep := 0
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			longKVRun = 0
			continue
		}
		if strings.Count(t, "|") >= 2 {
			pipeLines++
		}
		if reTableSep.MatchString(t) {
			tableSep++
		}
		if reKVLine.MatchString(t) {
			kvLines++
			longKVRun++
			if longKVRun > maxKVRun {
				maxKVRun = longKVRun
			}
		} else {
			longKVRun = 0
		}
	}

	// Heuristics: if we see a markdown-like table separator, it's likely table.
	if tableSep >= 1 && pipeLines >= 3 {
		return true
	}
	// Many pipe lines -> likely table.
	if pipeLines >= 8 {
		return true
	}
	// Many key-value lines (or a long contiguous run) -> likely KV/form/resume.
	if kvLines >= 10 || maxKVRun >= 6 {
		return true
	}
	return false
}

func looksUnstructuredNoisy(s string) bool {
	// Signals: very long average line length (few newlines), low punctuation density,
	// few heading markers, and few paragraph breaks.
	if hasStructuredHeadings(s) || isTableKVLike(s) {
		return false
	}

	trim := strings.TrimSpace(s)
	if trim == "" {
		return false
	}

	// Compute punctuation ratio on a sample.
	sample := trim
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	total := 0
	punct := 0
	newlines := 0
	doubleNewline := 0
	for i := 0; i < len(sample); i++ {
		r := rune(sample[i])
		if r == '\n' {
			newlines++
			if i+1 < len(sample) && sample[i+1] == '\n' {
				doubleNewline++
			}
		}
		if r == '。' || r == '！' || r == '？' || r == '；' || r == '，' || r == '.' || r == '!' || r == '?' || r == ';' || r == ',' {
			punct++
		}
		if !unicode.IsSpace(r) {
			total++
		}
	}
	if total == 0 {
		return false
	}
	punctRatio := float64(punct) / float64(total)

	// Very few punctuation and very few paragraph breaks -> likely OCR/noisy/novel dump.
	if punctRatio < 0.006 && doubleNewline == 0 && newlines < 5 {
		return true
	}
	// Very long lines (few newlines) and very low punctuation.
	if punctRatio < 0.01 && newlines < 3 && len(sample) > 2000 {
		return true
	}
	return false
}

func splitLinesLimit(s string, max int) []string {
	if max <= 0 {
		max = 200
	}
	lines := strings.Split(s, "\n")
	if len(lines) > max {
		return lines[:max]
	}
	return lines
}

var _ DocumentTypeDetector = (*RuleBasedDocumentTypeDetector)(nil)

