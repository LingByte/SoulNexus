//go:build ocr

package parser

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	"github.com/otiai10/gosseract/v2"
)

type OCRParser struct {
	Language string
}

func (p *OCRParser) Provider() string { return "ocr" }

func (p *OCRParser) SupportedTypes() []string {
	return []string{FileTypePNG, FileTypeJPG, FileTypeJPEG}
}

func (p *OCRParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	// Decode to ensure it is a supported image and to guard against invalid data.
	_, _, err = image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	cl := gosseract.NewClient()
	defer cl.Close()

	if p != nil && strings.TrimSpace(p.Language) != "" {
		cl.SetLanguage(p.Language)
	}
	// gosseract doesn't accept context directly; best-effort cancellation.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := cl.SetImageFromBytes(data); err != nil {
		return nil, err
	}
	text, err := cl.Text()
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: req.FileType,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
