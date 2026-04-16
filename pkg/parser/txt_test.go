package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTXTParser_Parse_Content(t *testing.T) {
	p := &TXTParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeTXT, FileName: "a.txt", Content: []byte("hello\nworld\n")}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeTXT, res.FileType)
	assert.Contains(t, res.Text, "hello")
	assert.Contains(t, res.Text, "world")
	assert.GreaterOrEqual(t, len(res.Sections), 1)
}

func TestTXTParser_Parse_Reader_Normalize(t *testing.T) {
	p := &TXTParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeTXT, FileName: "b.txt", Reader: bytes.NewReader([]byte("hello\n\nworld\t!"))}, &ParseOptions{PreserveLineBreaks: false})
	assert.NoError(t, err)
	assert.Equal(t, "hello world !", strings.TrimSpace(res.Text))
}

func TestTXTParser_Parse_Truncate(t *testing.T) {
	p := &TXTParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeTXT, Content: []byte("abcdefghijklmnopqrstuvwxyz")}, &ParseOptions{MaxTextLength: 10})
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(res.Text), 10)
}
