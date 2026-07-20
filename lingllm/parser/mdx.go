package parser

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// MDXParser parses MDX (Markdown with JSX) files.
// MDX is a format that allows JSX components within Markdown.
type MDXParser struct{}

func (p *MDXParser) Provider() string {
	return FileTypeMDX
}

func (p *MDXParser) SupportedTypes() []string {
	return []string{FileTypeMDX}
}

func (p *MDXParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	if req == nil {
		return nil, ErrEmptyInput
	}

	data, fileName, err := readRequestBytes(req)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	text := string(data)

	// Extract text content from MDX
	text = extractMDXContent(text)
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	return &ParseResult{
		FileType: FileTypeMDX,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}

// extractMDXContent extracts text content from MDX, removing JSX components and code blocks.
func extractMDXContent(content string) string {
	// Remove JSX components (e.g., <Component prop="value">content</Component>)
	// This is a simplified approach that handles most common cases
	jsxRegex := regexp.MustCompile(`<[A-Z][^>]*>.*?</[A-Z][^>]*>`)
	content = jsxRegex.ReplaceAllString(content, "")

	// Remove self-closing JSX components (e.g., <Component prop="value" />)
	selfClosingRegex := regexp.MustCompile(`<[A-Z][^>]*/\s*>`)
	content = selfClosingRegex.ReplaceAllString(content, "")

	// Remove code blocks (both ``` and ~~~)
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```|~~~[\\s\\S]*?~~~")
	content = codeBlockRegex.ReplaceAllString(content, "")

	// Remove inline code (backticks)
	inlineCodeRegex := regexp.MustCompile("`[^`]+`")
	content = inlineCodeRegex.ReplaceAllString(content, "")

	// Remove HTML comments
	commentRegex := regexp.MustCompile(`<!--[\s\S]*?-->`)
	content = commentRegex.ReplaceAllString(content, "")

	// Remove frontmatter (YAML or TOML at the beginning)
	frontmatterRegex := regexp.MustCompile(`^---\n[\s\S]*?\n---\n|^\+\+\+\n[\s\S]*?\n\+\+\+\n`)
	content = frontmatterRegex.ReplaceAllString(content, "")

	// Remove markdown links but keep the text: [text](url) -> text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	content = linkRegex.ReplaceAllString(content, "$1")

	// Remove markdown images: ![alt](url) -> alt
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\([^\)]+\)`)
	content = imageRegex.ReplaceAllString(content, "$1")

	// Remove markdown formatting
	// Bold: **text** or __text__ -> text
	boldRegex := regexp.MustCompile(`\*\*([^\*]+)\*\*|__([^_]+)__`)
	content = boldRegex.ReplaceAllString(content, "$1$2")

	// Italic: *text* or _text_ -> text
	italicRegex := regexp.MustCompile(`\*([^\*]+)\*|_([^_]+)_`)
	content = italicRegex.ReplaceAllString(content, "$1$2")

	// Strikethrough: ~~text~~ -> text
	strikeRegex := regexp.MustCompile(`~~([^~]+)~~`)
	content = strikeRegex.ReplaceAllString(content, "$1")

	// Remove heading markers: # text -> text
	headingRegex := regexp.MustCompile(`(?m)^#+\s+`)
	content = headingRegex.ReplaceAllString(content, "")

	// Remove list markers: - text or * text or + text -> text
	listRegex := regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	content = listRegex.ReplaceAllString(content, "")

	// Remove blockquote markers: > text -> text
	blockquoteRegex := regexp.MustCompile(`(?m)^>\s+`)
	content = blockquoteRegex.ReplaceAllString(content, "")

	// Remove horizontal rules
	hrRegex := regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	content = hrRegex.ReplaceAllString(content, "")

	// Clean up extra whitespace
	content = strings.TrimSpace(content)

	return content
}

// MarkdownParser parses standard Markdown files (alias for TXT parser with MD support).
type MarkdownParser struct {
	txtParser *TXTParser
}

func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		txtParser: &TXTParser{},
	}
}

func (p *MarkdownParser) Provider() string {
	return FileTypeMD
}

func (p *MarkdownParser) SupportedTypes() []string {
	return []string{FileTypeMD}
}

func (p *MarkdownParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	if req == nil {
		return nil, ErrEmptyInput
	}

	// Use TXT parser for basic markdown parsing
	result, err := p.txtParser.Parse(ctx, req, opts)
	if err != nil {
		return nil, err
	}

	// Override file type to markdown
	result.FileType = FileTypeMD

	// Extract markdown sections (headings)
	result.Sections = extractMarkdownSections(string(result.Text))

	return result, nil
}

// extractMarkdownSections extracts sections from markdown based on headings.
func extractMarkdownSections(content string) []Section {
	lines := strings.Split(content, "\n")
	sections := []Section{}
	currentSection := Section{
		Type:  SectionTypeDocument,
		Index: 0,
	}
	sectionIndex := 0

	for _, line := range lines {
		// Check for heading
		if strings.HasPrefix(line, "#") {
			// Save previous section if it has content
			if strings.TrimSpace(currentSection.Text) != "" {
				sections = append(sections, currentSection)
				sectionIndex++
			}

			// Create new section
			level := 0
			for i := 0; i < len(line); i++ {
				if line[i] == '#' {
					level++
				} else {
					break
				}
			}

			title := strings.TrimSpace(strings.TrimPrefix(line, strings.Repeat("#", level)))
			currentSection = Section{
				Type:  SectionTypeDocument,
				Index: sectionIndex,
				Title: title,
				Text:  "",
			}
		} else {
			// Add line to current section
			if currentSection.Text != "" {
				currentSection.Text += "\n"
			}
			currentSection.Text += line
		}
	}

	// Add last section
	if strings.TrimSpace(currentSection.Text) != "" {
		sections = append(sections, currentSection)
	}

	// If no sections were found, return a single document section
	if len(sections) == 0 {
		sections = []Section{{
			Type:  SectionTypeDocument,
			Index: 0,
			Title: "Document",
			Text:  content,
		}}
	}

	return sections
}
