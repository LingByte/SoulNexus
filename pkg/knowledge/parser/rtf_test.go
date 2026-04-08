package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRTFParser_Parse_Content(t *testing.T) {
	rtf := `{\rtf1\ansi\deff0 {\fonttbl {\f0 Arial;}}\f0\fs20 Hello\par World}`
	p := &RTFParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeRTF, Content: []byte(rtf)}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeRTF, res.FileType)
	assert.Contains(t, res.Text, "Hello")
	assert.Contains(t, res.Text, "World")
}
