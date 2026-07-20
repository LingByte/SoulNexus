package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanText_UTF8AndLower(t *testing.T) {
	in := "Hello\xffWorld iPhone"
	out := CleanText(in, &Options{Lowercase: true})
	assert.Contains(t, out, "hello")
	assert.Contains(t, out, "iphone")
	assert.NotContains(t, out, "\ufffd")
}

func TestCleanText_StripMarkdown(t *testing.T) {
	in := "# Title\n\nThis is **bold** and a [link](http://a.com).\n\n```go\nfmt.Println(1)\n```"
	out := CleanText(in, &Options{StripMarkdown: true})
	assert.Contains(t, out, "Title")
	assert.Contains(t, out, "This is bold")
	assert.NotContains(t, out, "fmt.Println")
	assert.NotContains(t, out, "**")
}

func TestCleanText_DedupAndBoilerplate(t *testing.T) {
	in := "Header\nA\nB\nHeader\nC\nD\nHeader\nE\nF\n"
	out := CleanText(in, &Options{DropBoilerplateLines: true, DedupLines: true})
	assert.NotContains(t, out, "Header")
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "F")
}

func TestCleanText_StripSymbols(t *testing.T) {
	in := "Hello@World#123!@#$%"
	out := CleanText(in, &Options{StripSymbols: true})
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "World")
	assert.Contains(t, out, "123")
	assert.NotContains(t, out, "@")
	assert.NotContains(t, out, "#")
}

func TestCleanText_NilOptions(t *testing.T) {
	in := "Hello World"
	out := CleanText(in, nil)
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "World")
}

func TestCleanText_ControlCharacters(t *testing.T) {
	in := "Hello\x00World\x01Test"
	out := CleanText(in, nil)
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "World")
	assert.Contains(t, out, "Test")
	assert.NotContains(t, out, "\x00")
	assert.NotContains(t, out, "\x01")
}

func TestCleanText_WhitespaceNormalization(t *testing.T) {
	in := "Hello\r\nWorld\rTest\n\n\nMultiple"
	out := CleanText(in, nil)
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "World")
	assert.Contains(t, out, "Test")
	assert.Contains(t, out, "Multiple")
	// Should normalize line endings
	assert.NotContains(t, out, "\r")
}

func TestCleanText_TabsAndSpaces(t *testing.T) {
	in := "Hello\t\tWorld   Test"
	out := CleanText(in, nil)
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "World")
	assert.Contains(t, out, "Test")
}

func TestCleanText_BOM(t *testing.T) {
	in := "\uFEFFHello World"
	out := CleanText(in, nil)
	assert.Contains(t, out, "Hello")
	assert.NotContains(t, out, "\uFEFF")
}

func TestCleanText_MarkdownElements(t *testing.T) {
	in := "# Heading\n> Quote\n- List item\n**bold** and *italic*"
	out := CleanText(in, &Options{StripMarkdown: true})
	assert.Contains(t, out, "Heading")
	assert.Contains(t, out, "Quote")
	assert.Contains(t, out, "List item")
	assert.Contains(t, out, "bold")
	assert.Contains(t, out, "italic")
	assert.NotContains(t, out, "#")
	assert.NotContains(t, out, ">")
	assert.NotContains(t, out, "-")
	assert.NotContains(t, out, "**")
}

func TestCleanText_InlineCode(t *testing.T) {
	in := "Use `fmt.Println()` to print"
	out := CleanText(in, &Options{StripMarkdown: true})
	assert.Contains(t, out, "Use")
	assert.Contains(t, out, "to print")
	assert.NotContains(t, out, "`")
}

func TestCleanText_Links(t *testing.T) {
	in := "Check [this link](https://example.com) and ![image](img.png)"
	out := CleanText(in, &Options{StripMarkdown: true})
	assert.Contains(t, out, "this link")
	assert.NotContains(t, out, "https://")
	assert.NotContains(t, out, "img.png")
}

func TestCleanText_ConsecutiveDuplicates(t *testing.T) {
	in := "Line1\nLine1\nLine2\nLine2\nLine2\nLine3"
	out := CleanText(in, &Options{DedupLines: true})
	lines := strings.Split(out, "\n")
	// Should remove consecutive duplicates
	for i := 0; i < len(lines)-1; i++ {
		assert.NotEqual(t, lines[i], lines[i+1])
	}
}

func TestCleanText_EmptyLines(t *testing.T) {
	in := "Line1\n\n\n\nLine2\n\n\nLine3"
	out := CleanText(in, &Options{DedupLines: true})
	// Should normalize multiple empty lines to single empty line
	assert.NotContains(t, out, "\n\n\n\n")
}

func TestCleanText_BoilerplateThreshold(t *testing.T) {
	// Small document (< 6 lines) uses threshold of 2
	in := "A\nB\nB\nC"
	out := CleanText(in, &Options{DropBoilerplateLines: true})
	assert.Contains(t, out, "A")
	assert.NotContains(t, out, "B")
	assert.Contains(t, out, "C")
}

func TestCleanText_Combined(t *testing.T) {
	in := "# Title\n\nHello\r\nWorld\n\n\nHello\n**bold**\n\n```code```"
	out := CleanText(in, &Options{
		Lowercase:     true,
		StripMarkdown: true,
		StripSymbols:  false,
		DedupLines:    true,
	})
	assert.Contains(t, out, "title")
	assert.Contains(t, out, "hello")
	assert.Contains(t, out, "world")
	assert.Contains(t, out, "bold")
	assert.NotContains(t, out, "#")
	assert.NotContains(t, out, "**")
}

func TestSanitizeForSpeech(t *testing.T) {
	out := SanitizeForSpeech("**Sure**, I can hear you! 😊")
	assert.Equal(t, "Sure, I can hear you!", out)
	assert.Empty(t, SanitizeForSpeech("---"))
	assert.Empty(t, SanitizeForSpeech("😊"))
	assert.True(t, HasSpeakableContent("What's your name?"))
	assert.False(t, HasSpeakableContent("..."))
	assert.False(t, IsCloudTTSAcceptable("，"))
	assert.True(t, IsCloudTTSAcceptable("你好。"))
	assert.Empty(t, SanitizeForSpeech("<speak></speak>"))
}
