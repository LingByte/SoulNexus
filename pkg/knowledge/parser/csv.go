package parser

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"os"
	"strings"
	"time"
)

type CSVParser struct{}

func (p *CSVParser) Provider() string {
	return FileTypeCSV
}

func (p *CSVParser) SupportedTypes() []string {
	return []string{FileTypeCSV}
}

func (p *CSVParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	fileName := req.FileName
	if fileName == "" {
		fileName = req.Path
	}

	var r io.Reader
	switch {
	case len(req.Content) > 0:
		r = bytes.NewReader(req.Content)
	case req.Reader != nil:
		r = req.Reader
	case strings.TrimSpace(req.Path) != "":
		b, err := os.ReadFile(req.Path)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	default:
		return nil, ErrEmptyInput
	}

	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0, len(records))
	for _, row := range records {
		if len(row) == 0 {
			continue
		}
		line := strings.Join(row, "\t")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}

	text := strings.Join(lines, "\n")
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeCSV,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
