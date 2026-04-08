package parser

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"
)

type TXTParser struct{}

func (p *TXTParser) Provider() string {
	return FileTypeTXT
}

func (p *TXTParser) SupportedTypes() []string {
	return []string{FileTypeTXT, FileTypeMD}
}

func (p *TXTParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	data, fileName, err := readRequestBytes(req)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	text := string(data)
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	ft := req.FileType
	if ft == "" {
		ft = FileTypeTXT
	}

	return &ParseResult{
		FileType: ft,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

func readRequestBytes(req *ParseRequest) ([]byte, string, error) {
	fileName := req.FileName
	if fileName == "" {
		fileName = req.Path
	}

	if len(req.Content) > 0 {
		return req.Content, fileName, nil
	}
	if req.Reader != nil {
		b, err := io.ReadAll(req.Reader)
		return b, fileName, err
	}
	if strings.TrimSpace(req.Path) != "" {
		b, err := os.ReadFile(req.Path)
		return b, fileName, err
	}
	return nil, fileName, ErrEmptyInput
}

func normalizeText(s string, opts *ParseOptions) string {
	if opts == nil {
		return s
	}
	if opts.PreserveLineBreaks {
		return s
	}
	// Collapse all whitespace (including newlines) into single spaces.
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func truncateText(s string, opts *ParseOptions) string {
	if opts == nil {
		return s
	}
	if opts.MaxTextLength <= 0 {
		return s
	}
	if len(s) <= opts.MaxTextLength {
		return s
	}
	// Keep it simple (byte-based). If you need rune-safe truncation later we can adjust.
	return string(bytes.TrimSpace([]byte(s[:opts.MaxTextLength])))
}
