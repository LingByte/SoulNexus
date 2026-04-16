package parser

import (
	"bytes"
	"context"
	"testing"

	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/assert"
)

func TestPDFParser_Parse_Content(t *testing.T) {
	pdfDoc := gofpdf.New("P", "mm", "A4", "")
	pdfDoc.AddPage()
	pdfDoc.SetFont("Arial", "", 14)
	pdfDoc.Cell(40, 10, "Hello PDF")

	var buf bytes.Buffer
	assert.NoError(t, pdfDoc.Output(&buf))

	p := &PDFParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypePDF, FileName: "t.pdf", Content: buf.Bytes()}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypePDF, res.FileType)
	assert.Contains(t, res.Text, "Hello")
}
