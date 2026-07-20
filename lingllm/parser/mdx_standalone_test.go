package parser

import (
	"strings"
	"testing"
)

// Standalone tests for MDX content extraction that don't require full parser build

func TestExtractMDXContentRemoveJSX(t *testing.T) {
	content := `<Component prop="value">content</Component>`
	result := extractMDXContent(content)
	if strings.Contains(result, "Component") {
		t.Errorf("JSX component not removed: %s", result)
	}
}

func TestExtractMDXContentRemoveSelfClosingJSX(t *testing.T) {
	content := `<Component prop="value" />`
	result := extractMDXContent(content)
	if strings.Contains(result, "Component") {
		t.Errorf("Self-closing JSX not removed: %s", result)
	}
}

func TestExtractMDXContentRemoveCodeBlocks(t *testing.T) {
	content := "```javascript\nconst x = 1;\n```\nSome text"
	result := extractMDXContent(content)
	if strings.Contains(result, "const") {
		t.Errorf("Code block not removed: %s", result)
	}
	if !strings.Contains(result, "Some text") {
		t.Errorf("Text content removed: %s", result)
	}
}

func TestExtractMDXContentRemoveInlineCode(t *testing.T) {
	content := "This is `inline code` in text"
	result := extractMDXContent(content)
	if strings.Contains(result, "`") {
		t.Errorf("Inline code not removed: %s", result)
	}
}

func TestExtractMDXContentRemoveLinks(t *testing.T) {
	content := "[Click here](https://example.com)"
	result := extractMDXContent(content)
	if strings.Contains(result, "http") {
		t.Errorf("Link URL not removed: %s", result)
	}
	if !strings.Contains(result, "Click here") {
		t.Errorf("Link text removed: %s", result)
	}
}

func TestExtractMDXContentRemoveImages(t *testing.T) {
	content := "![alt text](image.png)"
	result := extractMDXContent(content)
	if strings.Contains(result, "image.png") {
		t.Errorf("Image URL not removed: %s", result)
	}
}

func TestExtractMDXContentRemoveBold(t *testing.T) {
	content := "This is **bold** text"
	result := extractMDXContent(content)
	if strings.Contains(result, "**") {
		t.Errorf("Bold markers not removed: %s", result)
	}
	if !strings.Contains(result, "bold") {
		t.Errorf("Bold text removed: %s", result)
	}
}

func TestExtractMDXContentRemoveItalic(t *testing.T) {
	content := "This is *italic* text"
	result := extractMDXContent(content)
	if strings.Contains(result, "*") {
		t.Errorf("Italic markers not removed: %s", result)
	}
	if !strings.Contains(result, "italic") {
		t.Errorf("Italic text removed: %s", result)
	}
}

func TestExtractMDXContentRemoveHeadings(t *testing.T) {
	content := "# Heading 1\n## Heading 2\nSome text"
	result := extractMDXContent(content)
	if strings.Contains(result, "#") {
		t.Errorf("Heading markers not removed: %s", result)
	}
	if !strings.Contains(result, "Heading 1") {
		t.Errorf("Heading text removed: %s", result)
	}
}

func TestExtractMDXContentRemoveFrontmatter(t *testing.T) {
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

func TestExtractMarkdownSectionsBasic(t *testing.T) {
	content := `# Heading 1
Content 1

## Heading 2
Content 2`

	sections := extractMarkdownSections(content)
	if len(sections) < 2 {
		t.Errorf("Expected at least 2 sections, got %d", len(sections))
	}
}

func TestExtractMarkdownSectionsEmpty(t *testing.T) {
	content := ""
	sections := extractMarkdownSections(content)
	if len(sections) == 0 {
		t.Error("Should have at least one section for empty content")
	}
}

func TestExtractMarkdownSectionsNoHeadings(t *testing.T) {
	content := "Just some text without headings"
	sections := extractMarkdownSections(content)
	if len(sections) == 0 {
		t.Error("Should have at least one section")
	}
}
