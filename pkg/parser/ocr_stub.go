//go:build !ocr

package parser

import (
	"context"
)

type OCRParser struct {
	Language string
}

func (p *OCRParser) Provider() string { return "ocr" }

func (p *OCRParser) SupportedTypes() []string {
	return []string{FileTypePNG, FileTypeJPG, FileTypeJPEG}
}

func (p *OCRParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	_ = req
	_ = opts
	return nil, ErrUnsupportedFileType
}
