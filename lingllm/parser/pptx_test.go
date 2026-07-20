package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPPTXParser_Parse_Content(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w1, err := zw.Create("ppt/slides/slide1.xml")
	assert.NoError(t, err)
	_, err = w1.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Slide1</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`))
	assert.NoError(t, err)

	w2, err := zw.Create("ppt/slides/slide2.xml")
	assert.NoError(t, err)
	_, err = w2.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Slide2</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`))
	assert.NoError(t, err)

	assert.NoError(t, zw.Close())

	p := &PPTXParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypePPTX, Content: buf.Bytes()}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypePPTX, res.FileType)
	assert.Contains(t, res.Text, "Slide1")
	assert.Contains(t, res.Text, "Slide2")
	assert.GreaterOrEqual(t, len(res.Sections), 2)
}
