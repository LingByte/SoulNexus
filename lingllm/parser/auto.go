package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LingByte/lingllm/utils"
)

// ParsePreviewMeta holds parse-stage metadata for the ingest preview UI.
type ParsePreviewMeta struct {
	Format    string         `json:"format,omitempty"`
	CharCount int            `json:"charCount,omitempty"`
	Preview   string         `json:"preview,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

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

	p, ok := r.parsersByType[ft]
	if !ok || p == nil {
		if isOCRFileType(ft) {
			return nil, fmt.Errorf("%s requires OCR support (build tag 'ocr') and system tesseract: %w", ft, ErrUnsupportedFileType)
		}
		if isASRFileType(ft) {
			return nil, fmt.Errorf("%s requires ASR support (build tag 'asr'), libvosk, and VOSK_MODEL: %w", ft, ErrUnsupportedFileType)
		}
		return nil, ErrUnsupportedFileType
	}
	return p.Parse(ctx, req, opts)
}

func DefaultRouter() *Router {
	return NewRouter(
		&TXTParser{},
		NewMarkdownParser(),
		&MDXParser{},
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
		&EPUBParser{},
		&MHTMLParser{},
		&DOCParser{},
		&OCRParser{Language: "eng"},
		&ASRParser{},
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

func ParseBytesWithMeta(ctx context.Context, fileName string, content []byte, opts *ParseOptions) (string, *ParsePreviewMeta, error) {
	bytes, err := ParseBytes(ctx, fileName, content, opts)
	if err != nil {
		return "", nil, err
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))), ".")
	if ext == "" {
		ext = "text"
	}
	meta := &ParsePreviewMeta{
		Format:    ext,
		CharCount: len([]rune(bytes.Text)),
		Preview:   utils.PreviewText(bytes.Text, 2000),
	}
	return bytes.Text, meta, nil
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
	case "mdx":
		return FileTypeMDX
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
	case "webp":
		return FileTypeWEBP
	case "gif":
		return FileTypeGIF
	case "bmp":
		return FileTypeBMP
	case "tif":
		return FileTypeTIF
	case "tiff":
		return FileTypeTIFF
	case "doc":
		return FileTypeDOC
	case "docx":
		return FileTypeDOCX
	case "pptx":
		return FileTypePPTX
	case "xlsx":
		return FileTypeXLSX
	case "epub":
		return FileTypeEPUB
	case "mhtml", "mht":
		if ext == "mht" {
			return FileTypeMHT
		}
		return FileTypeMHTML
	case "wav":
		return FileTypeWAV
	case "mp3":
		return FileTypeMP3
	case "ogg":
		return FileTypeOGG
	case "flac":
		return FileTypeFLAC
	case "m4a":
		return FileTypeM4A
	case "aac":
		return FileTypeAAC
	default:
		return FileTypeUnknown
	}
}

func isOCRFileType(ft string) bool {
	switch ft {
	case FileTypePNG, FileTypeJPG, FileTypeJPEG, FileTypeWEBP, FileTypeGIF, FileTypeBMP, FileTypeTIFF, FileTypeTIF:
		return true
	default:
		return false
	}
}

func isASRFileType(ft string) bool {
	switch ft {
	case FileTypeWAV, FileTypeMP3, FileTypeOGG, FileTypeFLAC, FileTypeM4A, FileTypeAAC:
		return true
	default:
		return false
	}
}
