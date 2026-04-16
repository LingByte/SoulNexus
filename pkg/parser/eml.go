package parser

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"
)

type EMLParser struct{}

func (p *EMLParser) Provider() string { return FileTypeEML }

func (p *EMLParser) SupportedTypes() []string { return []string{FileTypeEML} }

func (p *EMLParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	headersText := buildEMLHeadersText(msg.Header)
	bodyText, _ := extractEMLBodyText(msg)

	text := strings.TrimSpace(strings.Join([]string{headersText, bodyText}, "\n\n"))
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeEML,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

func buildEMLHeadersText(h mail.Header) string {
	get := func(k string) string { return strings.TrimSpace(h.Get(k)) }
	lines := make([]string, 0, 6)
	if v := get("Subject"); v != "" {
		lines = append(lines, "Subject: "+v)
	}
	if v := get("From"); v != "" {
		lines = append(lines, "From: "+v)
	}
	if v := get("To"); v != "" {
		lines = append(lines, "To: "+v)
	}
	if v := get("Cc"); v != "" {
		lines = append(lines, "Cc: "+v)
	}
	if v := get("Date"); v != "" {
		lines = append(lines, "Date: "+v)
	}
	return strings.Join(lines, "\n")
}

func extractEMLBodyText(msg *mail.Message) (string, error) {
	ct := msg.Header.Get("Content-Type")
	mediatype, params, _ := mime.ParseMediaType(ct)
	cte := strings.ToLower(strings.TrimSpace(msg.Header.Get("Content-Transfer-Encoding")))

	readAllDecoded := func(r io.Reader) ([]byte, error) {
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		switch cte {
		case "base64":
			dec, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(b)))
			if err != nil {
				// fallback
				return b, nil
			}
			return dec, nil
		default:
			return b, nil
		}
	}

	if strings.HasPrefix(strings.ToLower(mediatype), "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return "", errors.New("multipart boundary missing")
		}
		mr := multipart.NewReader(msg.Body, boundary)
		var bestPlain string
		var fallback string
		for {
			p, err := mr.NextPart()
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", err
			}
			pCT := p.Header.Get("Content-Type")
			pType, _, _ := mime.ParseMediaType(pCT)
			pb, _ := io.ReadAll(p)
			text := strings.TrimSpace(string(pb))
			if text == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(pType), "text/plain") {
				bestPlain = text
			} else if fallback == "" && strings.HasPrefix(strings.ToLower(pType), "text/") {
				fallback = text
			}
		}
		if bestPlain != "" {
			return bestPlain, nil
		}
		return fallback, nil
	}

	b, err := readAllDecoded(msg.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
