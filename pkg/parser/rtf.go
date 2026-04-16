package parser

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"time"
)

type RTFParser struct{}

func (p *RTFParser) Provider() string { return FileTypeRTF }

func (p *RTFParser) SupportedTypes() []string { return []string{FileTypeRTF} }

func (p *RTFParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
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

	text := stripRTF(string(data))
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeRTF,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

var (
	rtfControlWord = regexp.MustCompile(`\\[a-zA-Z]+-?\d*\s?`)
	rtfHex         = regexp.MustCompile(`\\'([0-9a-fA-F]{2})`)
	rtfGroup       = regexp.MustCompile(`[{}]`)
)

func stripRTF(s string) string {
	// Basic decoding of hex escapes: \'hh
	s = rtfHex.ReplaceAllStringFunc(s, func(m string) string {
		sub := rtfHex.FindStringSubmatch(m)
		if len(sub) != 2 {
			return ""
		}
		b, err := hexToByte(sub[1])
		if err != nil {
			return ""
		}
		return string([]byte{b})
	})

	// Replace paragraph markers
	s = strings.ReplaceAll(s, "\\par", "\n")
	s = strings.ReplaceAll(s, "\\line", "\n")
	s = strings.ReplaceAll(s, "\\tab", "\t")

	// Remove remaining control words
	s = rtfControlWord.ReplaceAllString(s, "")
	// Drop group braces
	s = rtfGroup.ReplaceAllString(s, "")
	// Unescape escaped braces/backslash
	s = strings.ReplaceAll(s, "\\{", "{")
	s = strings.ReplaceAll(s, "\\}", "}")
	s = strings.ReplaceAll(s, "\\\\", "\\")

	return strings.TrimSpace(s)
}

func hexToByte(h string) (byte, error) {
	h = strings.TrimSpace(h)
	if len(h) != 2 {
		return 0, ErrUnsupportedFileType
	}
	var b byte
	for i := 0; i < 2; i++ {
		c := h[i]
		var v byte
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v = c - 'A' + 10
		default:
			return 0, ErrUnsupportedFileType
		}
		b = (b << 4) | v
	}
	return b, nil
}
