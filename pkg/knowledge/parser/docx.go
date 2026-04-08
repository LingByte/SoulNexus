package parser

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strings"
	"time"
)

type DOCXParser struct{}

func (p *DOCXParser) Provider() string {
	return FileTypeDOCX
}

func (p *DOCXParser) SupportedTypes() []string {
	return []string{FileTypeDOCX}
}

func (p *DOCXParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}
	z, fileName, err := openZipFromRequest(req)
	if err != nil {
		return nil, err
	}

	b, ok, err := readZipFile(z, "word/document.xml")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrUnsupportedFileType
	}

	text, err := extractDOCXText(b)
	if err != nil {
		return nil, err
	}
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeDOCX,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

func extractDOCXText(xmlBytes []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(xmlBytes))
	var out strings.Builder

	inT := false
	// For nicer spacing
	needNewline := false
	needTab := false

	writeSep := func(s string) {
		if out.Len() == 0 {
			out.WriteString(s)
			return
		}
		// Avoid piling separators
		prev := out.String()
		if strings.HasSuffix(prev, s) {
			return
		}
		out.WriteString(s)
	}

	for {
		tok, err := dec.Token()
		if err != nil {
			if errorsIsEOF(err) {
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
				// Paragraph
				needNewline = true
			case "br":
				needNewline = true
			case "tab":
				needTab = true
			case "tr":
				needNewline = true
			case "tc":
				needTab = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			}
		case xml.CharData:
			if !inT {
				continue
			}
			s := string(t)
			// Keep original spaces inside runs; trim only extremes.
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}

			if needNewline {
				writeSep("\n")
				needNewline = false
			}
			if needTab {
				writeSep("\t")
				needTab = false
			}
			if out.Len() > 0 {
				// If previous char isn't a separator, add a space between tokens.
				prev := out.String()
				last := prev[len(prev)-1]
				if last != '\n' && last != '\t' && last != ' ' {
					out.WriteByte(' ')
				}
			}
			out.WriteString(s)
		}
	}

	return strings.TrimSpace(out.String()), nil
}

func errorsIsEOF(err error) bool {
	return err == io.EOF
}
