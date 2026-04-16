package parser

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestXLSXParser_Parse_Content(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// Default sheet is Sheet1
	_ = f.SetCellValue("Sheet1", "A1", "Name")
	_ = f.SetCellValue("Sheet1", "B1", "Age")
	_ = f.SetCellValue("Sheet1", "A2", "Alice")
	_ = f.SetCellValue("Sheet1", "B2", 18)

	_, _ = f.NewSheet("Data")
	_ = f.SetCellValue("Data", "A1", "Hello")

	buf, err := f.WriteToBuffer()
	assert.NoError(t, err)

	p := &XLSXParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeXLSX, FileName: "t.xlsx", Content: buf.Bytes()}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeXLSX, res.FileType)
	assert.GreaterOrEqual(t, len(res.Sections), 1)
	assert.Contains(t, res.Text, "Alice")
	assert.Contains(t, res.Text, "Hello")

	// Ensure sheets are present
	found := false
	for _, s := range res.Sections {
		if s.Title == "Sheet1" {
			found = true
			assert.Contains(t, s.Text, "Name")
		}
	}
	assert.True(t, found)

	// Reader input
	res2, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeXLSX, Reader: bytes.NewReader(buf.Bytes())}, nil)
	assert.NoError(t, err)
	assert.Contains(t, res2.Text, "Alice")
}
