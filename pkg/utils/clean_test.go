package utils

import (
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
