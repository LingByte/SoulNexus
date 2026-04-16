package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEMLParser_Parse_Content(t *testing.T) {
	eml := "Subject: Hello\r\nFrom: a@example.com\r\nTo: b@example.com\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nThis is the body.\r\n"
	p := &EMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeEML, Content: []byte(eml)}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeEML, res.FileType)
	assert.Contains(t, res.Text, "Subject: Hello")
	assert.Contains(t, res.Text, "This is the body")
}
