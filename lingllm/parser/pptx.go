package parser

import (
	"context"
	"encoding/xml"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PPTXParser struct{}

func (p *PPTXParser) Provider() string {
	return FileTypePPTX
}

func (p *PPTXParser) SupportedTypes() []string {
	return []string{FileTypePPTX}
}

func (p *PPTXParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}
	z, fileName, err := openZipFromRequest(req)
	if err != nil {
		return nil, err
	}

	// Slides are under ppt/slides/slideX.xml
	files := listZipFilesWithPrefix(z, "ppt/slides/")
	slides := make([]string, 0)
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasPrefix(base, "slide") && strings.HasSuffix(base, ".xml") {
			slides = append(slides, f)
		}
	}
	if len(slides) == 0 {
		return nil, ErrUnsupportedFileType
	}
	sort.Strings(slides)

	sections := make([]Section, 0, len(slides))
	allTexts := make([]string, 0, len(slides))

	for i, slidePath := range slides {
		b, ok, rerr := readZipFile(z, slidePath)
		if rerr != nil {
			return nil, rerr
		}
		if !ok {
			continue
		}

		text, xerr := extractPPTXText(b)
		if xerr != nil {
			return nil, xerr
		}
		text = normalizeText(text, opts)
		text = truncateText(text, opts)

		title := strings.TrimSuffix(filepath.Base(slidePath), ".xml")
		sections = append(sections, Section{Type: SectionTypeSlide, Index: i, Title: title, Text: text})
		if text != "" {
			allTexts = append(allTexts, "[Slide] "+title+"\n"+text)
		}
	}

	fullText := strings.Join(allTexts, "\n\n")
	fullText = truncateText(fullText, opts)

	return &ParseResult{
		FileType: FileTypePPTX,
		FileName: fileName,
		Text:     fullText,
		Sections: sections,
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

func extractPPTXText(xmlBytes []byte) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))
	var b strings.Builder

	inT := false
	needNewline := false

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inT = true
			case "p":
				needNewline = true
			case "br":
				needNewline = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			}
		case xml.CharData:
			if !inT {
				continue
			}
			s := strings.TrimSpace(string(t))
			if s == "" {
				continue
			}
			if needNewline {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				needNewline = false
			}
			if b.Len() > 0 {
				prev := b.String()
				last := prev[len(prev)-1]
				if last != '\n' && last != ' ' {
					b.WriteByte(' ')
				}
			}
			b.WriteString(s)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
