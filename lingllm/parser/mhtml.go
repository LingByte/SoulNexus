package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"
)

type MHTMLParser struct{}

func (p *MHTMLParser) Provider() string { return FileTypeMHTML }

func (p *MHTMLParser) SupportedTypes() []string {
	return []string{FileTypeMHTML, FileTypeMHT}
}

func (p *MHTMLParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	htmlBody, err := extractMHTMLHTML(data)
	if err != nil {
		return nil, err
	}

	text, err := extractHTMLTextFromBytes(htmlBody)
	if err != nil {
		return nil, err
	}
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: req.FileType,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

func extractMHTMLHTML(data []byte) ([]byte, error) {
	if html, ok := firstHTMLPartFromMultipart(data); ok {
		return html, nil
	}

	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err == nil {
		if html, ok := readMessageHTMLBody(msg); ok {
			return html, nil
		}
	}

	// Some MHTML files are raw HTML with embedded boundaries; try locating text/html directly.
	lower := strings.ToLower(string(data))
	if idx := strings.Index(lower, "<html"); idx >= 0 {
		return data[idx:], nil
	}

	return nil, fmt.Errorf("mhtml: no text/html part found")
}

func firstHTMLPartFromMultipart(data []byte) ([]byte, bool) {
	// Try common MHTML boundary markers.
	for _, marker := range []string{"boundary=", "Boundary="} {
		if idx := strings.Index(string(data[:min(4096, len(data))]), marker); idx >= 0 {
			line := string(data[idx:])
			end := strings.IndexAny(line, "\r\n")
			if end > 0 {
				line = line[:end]
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				boundary := strings.Trim(parts[1], `" \t`)
				if boundary != "" {
					if html, ok := readMultipartHTML(data, boundary); ok {
						return html, true
					}
				}
			}
		}
	}

	// Fallback: let net/mail try to parse Content-Type from the first lines.
	ct := detectContentTypeHeader(data)
	if ct == "" {
		return nil, false
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, false
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, false
	}
	return readMultipartHTML(data, boundary)
}

func readMultipartHTML(data []byte, boundary string) ([]byte, bool) {
	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, false
		}
		ct := part.Header.Get("Content-Type")
		mediaType, _, _ := mime.ParseMediaType(ct)
		if strings.EqualFold(mediaType, "text/html") || strings.Contains(strings.ToLower(ct), "text/html") {
			body, err := io.ReadAll(part)
			if err != nil {
				return nil, false
			}
			return body, true
		}
	}
	return nil, false
}

func readMessageHTMLBody(msg *mail.Message) ([]byte, bool) {
	ct := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return nil, false
	}
	if strings.EqualFold(mediaType, "text/html") {
		body, err := io.ReadAll(msg.Body)
		return body, err == nil
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return nil, false
		}
		mr := multipart.NewReader(msg.Body, boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, false
			}
			partCT := part.Header.Get("Content-Type")
			pmt, _, _ := mime.ParseMediaType(partCT)
			if strings.EqualFold(pmt, "text/html") || strings.Contains(strings.ToLower(partCT), "text/html") {
				body, err := io.ReadAll(part)
				return body, err == nil
			}
		}
	}
	return nil, false
}

func detectContentTypeHeader(data []byte) string {
	lines := strings.Split(string(data[:min(8192, len(data))]), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "content-type:") {
			return strings.TrimSpace(line[len("content-type:"):])
		}
		if line == "" {
			break
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
