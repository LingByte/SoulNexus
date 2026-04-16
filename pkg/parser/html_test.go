package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLParser_Parse_Content(t *testing.T) {
	p := &HTMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeHTML, Content: []byte("<html><body><h1>Title</h1><p>Hello <b>World</b></p></body></html>")}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeHTML, res.FileType)
	assert.Contains(t, res.Text, "Title")
	assert.Contains(t, res.Text, "Hello")
	assert.Contains(t, res.Text, "World")
}
