package parser

import (
	"context"
	"strings"
	"testing"
)

func TestMDXParserProvider(t *testing.T) {
	parser := &MDXParser{}
	if parser.Provider() != FileTypeMDX {
		t.Errorf("Provider() = %s, want %s", parser.Provider(), FileTypeMDX)
	}
}

func TestMDXParserSupportedTypes(t *testing.T) {
	parser := &MDXParser{}
	types := parser.SupportedTypes()
	if len(types) != 1 || types[0] != FileTypeMDX {
		t.Errorf("SupportedTypes() = %v, want [%s]", types, FileTypeMDX)
	}
}

func TestMDXParserBasic(t *testing.T) {
	parser := &MDXParser{}
	content := `# Hello World
This is a paragraph.

<Component prop="value">
  Some content
</Component>

More text here.`

	req := &ParseRequest{
		FileType: FileTypeMDX,
		FileName: "test.mdx",
		Content:  []byte(content),
	}

	result, err := parser.Parse(context.Background(), req, nil)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if result.FileType != FileTypeMDX {
		t.Errorf("FileType = %s, want %s", result.FileType, FileTypeMDX)
	}

	if result.FileName != "test.mdx" {
		t.Errorf("FileName = %s, want test.mdx", result.FileName)
	}

	if len(result.Text) == 0 {
		t.Error("Text should not be empty")
	}
}

func TestMDXParserRemoveJSX(t *testing.T) {
	content := `<Component prop="value">content</Component>`
	result := extractMDXContent(content)
	if strings.Contains(result, "Component") {
		t.Errorf("JSX component not removed: %s", result)
	}
}

func TestMDXParserRemoveSelfClosingJSX(t *testing.T) {
	content := `<Component prop="value" />`
	result := extractMDXContent(content)
	if strings.Contains(result, "Component") {
		t.Errorf("Self-closing JSX not removed: %s", result)
	}
}

func TestMDXParserRemoveCodeBlocks(t *testing.T) {
	content := "```javascript\nconst x = 1;\n```\nSome text"
	result := extractMDXContent(content)
	if strings.Contains(result, "const") {
		t.Errorf("Code block not removed: %s", result)
	}
	if !strings.Contains(result, "Some text") {
		t.Errorf("Text content removed: %s", result)
	}
}

func TestMDXParserRemoveInlineCode(t *testing.T) {
	content := "This is `inline code` in text"
	result := extractMDXContent(content)
	if strings.Contains(result, "`") {
		t.Errorf("Inline code not removed: %s", result)
	}
}

func TestMDXParserRemoveLinks(t *testing.T) {
	content := "[Click here](https://example.com)"
	result := extractMDXContent(content)
	if strings.Contains(result, "http") {
		t.Errorf("Link URL not removed: %s", result)
	}
	if !strings.Contains(result, "Click here") {
		t.Errorf("Link text removed: %s", result)
	}
}

func TestMDXParserRemoveImages(t *testing.T) {
	content := "![alt text](image.png)"
	result := extractMDXContent(content)
	if strings.Contains(result, "image.png") {
		t.Errorf("Image URL not removed: %s", result)
	}
}

func TestMDXParserRemoveBold(t *testing.T) {
	content := "This is **bold** text"
	result := extractMDXContent(content)
	if strings.Contains(result, "**") {
		t.Errorf("Bold markers not removed: %s", result)
	}
	if !strings.Contains(result, "bold") {
		t.Errorf("Bold text removed: %s", result)
	}
}

func TestMDXParserRemoveItalic(t *testing.T) {
	content := "This is *italic* text"
	result := extractMDXContent(content)
	if strings.Contains(result, "*") {
		t.Errorf("Italic markers not removed: %s", result)
	}
	if !strings.Contains(result, "italic") {
		t.Errorf("Italic text removed: %s", result)
	}
}

func TestMDXParserRemoveHeadings(t *testing.T) {
	content := "# Heading 1\n## Heading 2\nSome text"
	result := extractMDXContent(content)
	if strings.Contains(result, "#") {
		t.Errorf("Heading markers not removed: %s", result)
	}
	if !strings.Contains(result, "Heading 1") {
		t.Errorf("Heading text removed: %s", result)
	}
}

func TestMDXParserRemoveFrontmatter(t *testing.T) {
	content := `---
title: Test
---
# Content`
	result := extractMDXContent(content)
	if strings.Contains(result, "title:") {
		t.Errorf("Frontmatter not removed: %s", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("Content removed: %s", result)
	}
}

func TestMDXParserEmpty(t *testing.T) {
	parser := &MDXParser{}
	req := &ParseRequest{
		FileType: FileTypeMDX,
		FileName: "empty.mdx",
		Content:  []byte(""),
	}

	_, err := parser.Parse(context.Background(), req, nil)
	if err != ErrEmptyInput {
		t.Errorf("Error = %v, want ErrEmptyInput", err)
	}
}

func TestMarkdownParserProvider(t *testing.T) {
	parser := NewMarkdownParser()
	if parser.Provider() != FileTypeMD {
		t.Errorf("Provider() = %s, want %s", parser.Provider(), FileTypeMD)
	}
}

func TestMarkdownParserSupportedTypes(t *testing.T) {
	parser := NewMarkdownParser()
	types := parser.SupportedTypes()
	if len(types) != 1 || types[0] != FileTypeMD {
		t.Errorf("SupportedTypes() = %v, want [%s]", types, FileTypeMD)
	}
}

func TestMarkdownParserBasic(t *testing.T) {
	parser := NewMarkdownParser()
	content := `# Heading 1
Some paragraph text.

## Heading 2
More text.`

	req := &ParseRequest{
		FileType: FileTypeMD,
		FileName: "test.md",
		Content:  []byte(content),
	}

	result, err := parser.Parse(context.Background(), req, nil)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if result.FileType != FileTypeMD {
		t.Errorf("FileType = %s, want %s", result.FileType, FileTypeMD)
	}

	if len(result.Sections) == 0 {
		t.Error("Sections should not be empty")
	}
}

func TestMarkdownParserSections(t *testing.T) {
	content := `# Heading 1
Content 1

## Heading 2
Content 2`

	sections := extractMarkdownSections(content)
	if len(sections) < 2 {
		t.Errorf("Expected at least 2 sections, got %d", len(sections))
	}
}
