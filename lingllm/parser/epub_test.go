package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestEPUB(t *testing.T) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	mw, err := zw.Create("mimetype")
	require.NoError(t, err)
	_, err = mw.Write([]byte("application/epub+zip"))
	require.NoError(t, err)

	cw, err := zw.Create("META-INF/container.xml")
	require.NoError(t, err)
	_, err = cw.Write([]byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))
	require.NoError(t, err)

	opf, err := zw.Create("OEBPS/content.opf")
	require.NoError(t, err)
	_, err = opf.Write([]byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <manifest>
    <item id="c1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="c1"/>
  </spine>
</package>`))
	require.NoError(t, err)

	ch, err := zw.Create("OEBPS/chapter1.xhtml")
	require.NoError(t, err)
	_, err = ch.Write([]byte(`<!DOCTYPE html><html><body><h1>Chapter</h1><p>EPUB body text</p></body></html>`))
	require.NoError(t, err)

	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestEPUBParser_Parse(t *testing.T) {
	data := buildTestEPUB(t)
	p := &EPUBParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeEPUB,
		FileName: "book.epub",
		Content:  data,
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeEPUB, res.FileType)
	assert.Contains(t, res.Text, "EPUB body text")
	assert.GreaterOrEqual(t, len(res.Sections), 1)
}

func TestDetectFileType_EPUB_MHTML_Audio(t *testing.T) {
	assert.Equal(t, FileTypeEPUB, DetectFileType(&ParseRequest{FileName: "a.epub"}))
	assert.Equal(t, FileTypeMHTML, DetectFileType(&ParseRequest{FileName: "a.mhtml"}))
	assert.Equal(t, FileTypeMHT, DetectFileType(&ParseRequest{FileName: "a.mht"}))
	assert.Equal(t, FileTypeWAV, DetectFileType(&ParseRequest{FileName: "a.wav"}))
	assert.Equal(t, FileTypeMP3, DetectFileType(&ParseRequest{FileName: "a.mp3"}))
}
