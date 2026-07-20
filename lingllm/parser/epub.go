package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

type EPUBParser struct{}

func (p *EPUBParser) Provider() string { return FileTypeEPUB }

func (p *EPUBParser) SupportedTypes() []string { return []string{FileTypeEPUB} }

func (p *EPUBParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	z, fileName, err := openZipFromRequest(req)
	if err != nil {
		return nil, err
	}

	containerXML, ok, err := readZipFile(z, "META-INF/container.xml")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("epub missing META-INF/container.xml")
	}

	rootPath, err := parseEPUBRootPath(containerXML)
	if err != nil {
		return nil, err
	}
	rootDir := path.Dir(rootPath)

	opfData, ok, err := readZipFile(z, rootPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("epub missing rootfile %q", rootPath)
	}

	chapters, err := parseEPUBSpine(opfData)
	if err != nil {
		return nil, err
	}

	var sections []Section
	var texts []string
	for i, href := range chapters {
		itemPath := path.Join(rootDir, href)
		itemPath = strings.TrimPrefix(path.Clean("/"+itemPath), "/")
		htmlData, ok, err := readZipFile(z, itemPath)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		text, err := extractHTMLTextFromBytes(htmlData)
		if err != nil {
			return nil, fmt.Errorf("epub chapter %q: %w", href, err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		texts = append(texts, text)
		sections = append(sections, Section{
			Type:  SectionTypePage,
			Index: i,
			Title: path.Base(href),
			Text:  text,
		})
	}

	fullText := strings.Join(texts, "\n\n")
	fullText = normalizeText(fullText, opts)
	fullText = truncateText(fullText, opts)

	if len(sections) == 0 {
		sections = []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: fullText}}
	}

	return &ParseResult{
		FileType: FileTypeEPUB,
		FileName: fileName,
		Text:     fullText,
		Sections: sections,
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

type epubContainer struct {
	Rootfiles []struct {
		FullPath  string `xml:"full-path,attr"`
		MediaType string `xml:"media-type,attr"`
	} `xml:"rootfiles>rootfile"`
}

func parseEPUBRootPath(containerXML []byte) (string, error) {
	var c epubContainer
	if err := xml.Unmarshal(containerXML, &c); err != nil {
		return "", fmt.Errorf("epub container.xml: %w", err)
	}
	for _, rf := range c.Rootfiles {
		if strings.TrimSpace(rf.FullPath) != "" {
			return strings.TrimSpace(rf.FullPath), nil
		}
	}
	return "", fmt.Errorf("epub container.xml: no rootfile")
}

type epubPackage struct {
	Manifest []struct {
		ID   string `xml:"id,attr"`
		Href string `xml:"href,attr"`
	} `xml:"manifest>item"`
	Spine []struct {
		IDRef string `xml:"idref,attr"`
	} `xml:"spine>itemref"`
}

func parseEPUBSpine(opfData []byte) ([]string, error) {
	var pkg epubPackage
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return nil, fmt.Errorf("epub opf: %w", err)
	}

	manifest := make(map[string]string, len(pkg.Manifest))
	for _, item := range pkg.Manifest {
		if item.ID != "" && item.Href != "" {
			manifest[item.ID] = item.Href
		}
	}

	out := make([]string, 0, len(pkg.Spine))
	for _, ref := range pkg.Spine {
		if href, ok := manifest[ref.IDRef]; ok {
			out = append(out, href)
		}
	}
	if len(out) == 0 {
		// Fallback: read all HTML/XHTML manifest items in stable order.
		type pair struct{ id, href string }
		items := make([]pair, 0)
		for _, item := range pkg.Manifest {
			lower := strings.ToLower(item.Href)
			if strings.HasSuffix(lower, ".xhtml") || strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
				items = append(items, pair{item.ID, item.Href})
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].id < items[j].id })
		for _, item := range items {
			out = append(out, item.href)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("epub opf: no readable spine items")
	}
	return out, nil
}
