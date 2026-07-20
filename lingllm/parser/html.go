package parser

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"
)

type HTMLParser struct{}

func (p *HTMLParser) Provider() string {
	return FileTypeHTML
}

func (p *HTMLParser) SupportedTypes() []string {
	return []string{FileTypeHTML}
}

func (p *HTMLParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	text, err := extractHTMLText(r)
	if err != nil {
		return nil, err
	}
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeHTML,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
