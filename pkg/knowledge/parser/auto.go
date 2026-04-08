package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type Router struct {
	parsersByType map[string]Parser
}

func NewRouter(parsers ...Parser) *Router {
	r := &Router{parsersByType: make(map[string]Parser)}
	for _, p := range parsers {
		_ = r.Register(p)
	}
	return r
}

func (r *Router) Register(p Parser) error {
	if r == nil {
		return fmt.Errorf("nil router")
	}
	if p == nil {
		return fmt.Errorf("nil parser")
	}
	for _, t := range p.SupportedTypes() {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		r.parsersByType[t] = p
	}
	return nil
}

func (r *Router) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	if req == nil {
		return nil, ErrEmptyInput
	}

	ft := strings.ToLower(strings.TrimSpace(req.FileType))
	if ft == "" {
		ft = DetectFileType(req)
		req.FileType = ft
	}

	// Explicitly reject legacy .doc to keep dependencies clean.
	if ft == FileTypeDOC {
		return nil, fmt.Errorf("legacy .doc is not supported; please convert to .docx or .pdf: %w", ErrUnsupportedFileType)
	}

	p, ok := r.parsersByType[ft]
	if !ok || p == nil {
		// Help users understand OCR build tag behavior.
		if ft == FileTypePNG || ft == FileTypeJPG || ft == FileTypeJPEG {
			return nil, fmt.Errorf("%s requires OCR support (build tag 'ocr') and system tesseract: %w", ft, ErrUnsupportedFileType)
		}
		return nil, ErrUnsupportedFileType
	}
	return p.Parse(ctx, req, opts)
}

func DefaultRouter() *Router {
	return NewRouter(
		&TXTParser{},
		&CSVParser{},
		&HTMLParser{},
		&JSONParser{},
		&YAMLParser{},
		&EMLParser{},
		&RTFParser{},
		&XLSXParser{},
		&DOCXParser{},
		&PPTXParser{},
		&PDFParser{},
		&OCRParser{Language: "eng"},
	)
}

func ParseAuto(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	return DefaultRouter().Parse(ctx, req, opts)
}

func ParsePath(ctx context.Context, path string, opts *ParseOptions) (*ParseResult, error) {
	req := &ParseRequest{Path: path, FileName: filepath.Base(path)}
	return ParseAuto(ctx, req, opts)
}

func ParseBytes(ctx context.Context, fileName string, content []byte, opts *ParseOptions) (*ParseResult, error) {
	req := &ParseRequest{FileName: fileName, Content: content}
	return ParseAuto(ctx, req, opts)
}

func DetectFileType(req *ParseRequest) string {
	if req == nil {
		return FileTypeUnknown
	}

	name := strings.TrimSpace(req.FileName)
	if name == "" {
		name = strings.TrimSpace(req.Path)
	}
	name = strings.ToLower(name)

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	switch ext {
	case "txt":
		return FileTypeTXT
	case "md", "markdown":
		return FileTypeMD
	case "csv":
		return FileTypeCSV
	case "html", "htm":
		return FileTypeHTML
	case "json":
		return FileTypeJSON
	case "yaml":
		return FileTypeYAML
	case "yml":
		return FileTypeYML
	case "eml":
		return FileTypeEML
	case "rtf":
		return FileTypeRTF
	case "pdf":
		return FileTypePDF
	case "png":
		return FileTypePNG
	case "jpg":
		return FileTypeJPG
	case "jpeg":
		return FileTypeJPEG
	case "doc":
		return FileTypeDOC
	case "docx":
		return FileTypeDOCX
	case "pptx":
		return FileTypePPTX
	case "xlsx":
		return FileTypeXLSX
	default:
		return FileTypeUnknown
	}
}
