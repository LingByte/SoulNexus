package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type JSONParser struct{}

func (p *JSONParser) Provider() string { return FileTypeJSON }

func (p *JSONParser) SupportedTypes() []string { return []string{FileTypeJSON} }

func (p *JSONParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	text := string(pretty)
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeJSON,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

type YAMLParser struct{}

func (p *YAMLParser) Provider() string { return FileTypeYAML }

func (p *YAMLParser) SupportedTypes() []string { return []string{FileTypeYAML, FileTypeYML} }

func (p *YAMLParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
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

	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(out))
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	ft := req.FileType
	if ft == "" {
		ft = FileTypeYAML
	}

	return &ParseResult{
		FileType: ft,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
