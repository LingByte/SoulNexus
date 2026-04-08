package parser

import (
	"context"
	"fmt"
	"time"
)

type DOCParser struct{}

func (p *DOCParser) Provider() string { return FileTypeDOC }

func (p *DOCParser) SupportedTypes() []string { return []string{FileTypeDOC} }

func (p *DOCParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	_ = req
	_ = opts
	return nil, fmt.Errorf("legacy .doc is not supported; please convert to .docx or .pdf: %w", ErrUnsupportedFileType)
}

func init() { _ = time.Now() }
