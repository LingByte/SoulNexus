package parser

import (
	"context"
	"errors"
	"io"
	"time"
)

const (
	FileTypeUnknown = "unknown"
	FileTypeTXT     = "txt"
	FileTypeMD      = "md"
	FileTypeCSV     = "csv"
	FileTypeHTML    = "html"
	FileTypeJSON    = "json"
	FileTypeYAML    = "yaml"
	FileTypeYML     = "yml"
	FileTypeEML     = "eml"
	FileTypeRTF     = "rtf"
	FileTypePDF     = "pdf"
	FileTypePNG     = "png"
	FileTypeJPG     = "jpg"
	FileTypeJPEG    = "jpeg"
	FileTypeDOC     = "doc"
	FileTypeDOCX    = "docx"
	FileTypePPTX    = "pptx"
	FileTypeXLSX    = "xlsx"
)

var (
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrEmptyInput          = errors.New("empty input")
)

type ParseRequest struct {
	FileType    string
	FileName    string
	Path        string
	ContentType string
	Content     []byte
	Reader      io.Reader
	Metadata    map[string]any
}

type ParseOptions struct {
	MaxTextLength      int
	IncludeTables      bool
	IncludeHidden      bool
	PreserveLineBreaks bool
}

type SectionType string

const (
	SectionTypeUnknown  SectionType = "unknown"
	SectionTypeDocument SectionType = "document"
	SectionTypePage     SectionType = "page"
	SectionTypeSheet    SectionType = "sheet"
	SectionTypeSlide    SectionType = "slide"
)

type Section struct {
	Type     SectionType
	Index    int
	Title    string
	Text     string
	Metadata map[string]any
}

type ParseResult struct {
	FileType string
	FileName string
	Text     string
	Sections []Section
	Metadata map[string]any
	ParsedAt time.Time
}

type Parser interface {
	Provider() string

	SupportedTypes() []string

	Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error)
}
