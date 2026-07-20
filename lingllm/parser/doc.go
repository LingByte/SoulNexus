package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
)

type DOCParser struct{}

func (p *DOCParser) Provider() string { return FileTypeDOC }

func (p *DOCParser) SupportedTypes() []string { return []string{FileTypeDOC} }

func (p *DOCParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	data, fileName, err := readRequestBytes(req)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, ErrEmptyInput
	}

	text, err := extractLegacyDOCText(data)
	if err != nil {
		return nil, err
	}
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeDOC,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

func extractLegacyDOCText(data []byte) (string, error) {
	doc, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("legacy .doc: not a valid OLE document: %w", err)
	}

	var wordDoc []byte
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		if strings.EqualFold(strings.TrimSpace(entry.Name), "WordDocument") {
			wordDoc, err = io.ReadAll(entry)
			if err != nil {
				return "", fmt.Errorf("legacy .doc: read WordDocument stream: %w", err)
			}
			break
		}
	}
	if len(wordDoc) == 0 {
		return "", fmt.Errorf("legacy .doc: WordDocument stream not found: %w", ErrUnsupportedFileType)
	}

	text := extractTextFromWordDocument(wordDoc)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("legacy .doc: no extractable text found")
	}
	return text, nil
}

// extractTextFromWordDocument heuristically extracts readable text from a Word 97-2003
// WordDocument binary stream (UTF-16LE runs and printable ASCII sequences).
func extractTextFromWordDocument(data []byte) string {
	candidates := []string{
		extractUTF16LERuns(data),
		extractASCIIRuns(data),
	}

	best := ""
	bestScore := -1
	for _, c := range candidates {
		score := textQualityScore(c)
		if score > bestScore {
			bestScore = score
			best = c
		}
	}
	return strings.TrimSpace(best)
}

func textQualityScore(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var letters int
	runes := []rune(s)
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters++
		}
	}
	if len(runes) == 0 {
		return 0
	}
	return letters * 100 / len(runes)
}

func extractUTF16LERuns(data []byte) string {
	var runs [][]uint16
	var current []uint16

	flush := func() {
		if len(current) >= 2 {
			runs = append(runs, append([]uint16(nil), current...))
		}
		current = current[:0]
	}

	for i := 0; i+1 < len(data); i += 2 {
		r := binaryUint16(data[i], data[i+1])
		if r == 0 || r == 0x000D || r == 0x000A || r == 0x0009 {
			if r == 0x000D || r == 0x000A {
				current = append(current, '\n')
			} else if r == 0x0009 {
				current = append(current, '\t')
			} else {
				flush()
			}
			continue
		}
		if r < 32 || r == 0xFFFD {
			flush()
			continue
		}
		if !unicode.IsPrint(rune(r)) && r < 127 {
			flush()
			continue
		}
		current = append(current, r)
	}
	flush()

	var b strings.Builder
	for _, run := range runs {
		s := strings.TrimSpace(string(utf16.Decode(run)))
		if len(s) < 2 {
			continue
		}
		if mostlyBinaryNoise(s) {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s)
	}
	return b.String()
}

func extractASCIIRuns(data []byte) string {
	var b strings.Builder
	var current strings.Builder

	flush := func() {
		s := strings.TrimSpace(current.String())
		if len(s) >= 4 && !mostlyBinaryNoise(s) {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(s)
		}
		current.Reset()
	}

	for _, ch := range data {
		if ch >= 32 && ch < 127 {
			current.WriteByte(ch)
			continue
		}
		if ch == '\n' || ch == '\r' || ch == '\t' {
			current.WriteByte(' ')
			continue
		}
		flush()
	}
	flush()
	return b.String()
}

func binaryUint16(lo, hi byte) uint16 {
	return uint16(lo) | uint16(hi)<<8
}

func mostlyBinaryNoise(s string) bool {
	if s == "" {
		return true
	}
	var letters int
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters++
		}
	}
	return letters*3 < len([]rune(s))
}
