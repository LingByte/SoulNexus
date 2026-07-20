package parser

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

type PDFParser struct{}

func (p *PDFParser) Provider() string { return FileTypePDF }

func (p *PDFParser) SupportedTypes() []string { return []string{FileTypePDF} }

func (p *PDFParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var texts []string
	var sections []Section
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		content, err := p.GetPlainText(nil)
		if err != nil {
			return nil, err
		}
		t := strings.TrimSpace(content)
		if t != "" {
			texts = append(texts, t)
			sections = append(sections, Section{
				Type:  SectionTypePage,
				Index: i - 1,
				Title: fmt.Sprintf("page %d", i),
				Text:  t,
			})
		}
	}

	text := strings.Join(texts, "\n\n")
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	if len(sections) == 0 {
		sections = []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}}
	}

	return &ParseResult{
		FileType: FileTypePDF,
		FileName: fileName,
		Text:     text,
		Sections: sections,
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
