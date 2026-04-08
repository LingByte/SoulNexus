package parser

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type XLSXParser struct{}

func (p *XLSXParser) Provider() string {
	return FileTypeXLSX
}

func (p *XLSXParser) SupportedTypes() []string {
	return []string{FileTypeXLSX}
}

func (p *XLSXParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	fileName := req.FileName
	if fileName == "" {
		fileName = req.Path
	}

	var f *excelize.File
	var err error

	switch {
	case len(req.Content) > 0:
		f, err = excelize.OpenReader(bytes.NewReader(req.Content))
	case req.Reader != nil:
		b, rerr := io.ReadAll(req.Reader)
		if rerr != nil {
			return nil, rerr
		}
		f, err = excelize.OpenReader(bytes.NewReader(b))
	case strings.TrimSpace(req.Path) != "":
		// excelize needs a reader if we want uniform behavior; keep file reading in stdlib.
		b, rerr := os.ReadFile(req.Path)
		if rerr != nil {
			return nil, rerr
		}
		f, err = excelize.OpenReader(bytes.NewReader(b))
	default:
		return nil, ErrEmptyInput
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	sheets := f.GetSheetList()
	sections := make([]Section, 0, len(sheets))
	allTexts := make([]string, 0, len(sheets))

	for i, sheet := range sheets {
		rows, rerr := f.GetRows(sheet)
		if rerr != nil {
			return nil, rerr
		}
		lines := make([]string, 0, len(rows))
		for _, row := range rows {
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

		sections = append(sections, Section{
			Type:  SectionTypeSheet,
			Index: i,
			Title: sheet,
			Text:  text,
		})
		if text != "" {
			allTexts = append(allTexts, "[Sheet] "+sheet+"\n"+text)
		}
	}

	fullText := strings.Join(allTexts, "\n\n")
	fullText = truncateText(fullText, opts)

	return &ParseResult{
		FileType: FileTypeXLSX,
		FileName: fileName,
		Text:     fullText,
		Sections: sections,
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
