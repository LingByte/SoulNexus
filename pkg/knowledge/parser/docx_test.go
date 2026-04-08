package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDOCXParser_Parse_Content(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w, err := zw.Create("word/document.xml")
	assert.NoError(t, err)

	// Minimal DOCX document.xml with a few w:t elements
	_, err = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Hello</w:t></w:r></w:p>
    <w:p><w:r><w:t>World</w:t></w:r></w:p>
  </w:body>
</w:document>`))
	assert.NoError(t, err)

	assert.NoError(t, zw.Close())

	p := &DOCXParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeDOCX, Content: buf.Bytes()}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeDOCX, res.FileType)
	assert.Contains(t, res.Text, "Hello")
	assert.Contains(t, res.Text, "World")
	assert.GreaterOrEqual(t, len(res.Sections), 1)
}
