package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONParser_Parse_Content(t *testing.T) {
	p := &JSONParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeJSON, Content: []byte(`{"a":1,"b":{"c":2}}`)}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeJSON, res.FileType)
	assert.Contains(t, res.Text, "\"a\": 1")
	assert.Contains(t, res.Text, "\"c\": 2")
}

func TestYAMLParser_Parse_Content(t *testing.T) {
	p := &YAMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeYAML, Content: []byte("a: 1\nb:\n  c: 2\n")}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeYAML, res.FileType)
	assert.Contains(t, res.Text, "a: 1")
	assert.Contains(t, res.Text, "c: 2")
}
