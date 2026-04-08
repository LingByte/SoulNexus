package parser

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type HTMLParser struct{}

func (p *HTMLParser) Provider() string {
	return FileTypeHTML
}

func (p *HTMLParser) SupportedTypes() []string {
	return []string{FileTypeHTML}
}

func (p *HTMLParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	fileName := req.FileName
	if fileName == "" {
		fileName = req.Path
	}

	var r io.Reader
	switch {
	case len(req.Content) > 0:
		r = bytes.NewReader(req.Content)
	case req.Reader != nil:
		r = req.Reader
	case strings.TrimSpace(req.Path) != "":
		b, err := os.ReadFile(req.Path)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	default:
		return nil, ErrEmptyInput
	}

	n, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(text)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	text := b.String()
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeHTML,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
