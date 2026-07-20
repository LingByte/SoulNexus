package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTextFromWordDocument_UTF16(t *testing.T) {
	// UTF-16LE for "Hello DOC"
	data := []byte{
		0x00, 0x01, 0x00, 0x02, // binary prefix noise
		'H', 0, 'e', 0, 'l', 0, 'l', 0, 'o', 0, ' ', 0, 'D', 0, 'O', 0, 'C', 0,
		0, 0,
	}
	text := extractTextFromWordDocument(data)
	assert.Contains(t, text, "Hello DOC")
}

func TestExtractTextFromWordDocument_ASCII(t *testing.T) {
	data := []byte("prefix\x00\x01\x02Legacy ASCII document text here\x00\xff")
	text := extractTextFromWordDocument(data)
	assert.Contains(t, text, "Legacy ASCII document text here")
}

func TestMostlyBinaryNoise(t *testing.T) {
	assert.True(t, mostlyBinaryNoise("!@#$%^&*()"))
	assert.False(t, mostlyBinaryNoise("hello world 123"))
}

func TestDOCParser_InvalidOLE(t *testing.T) {
	p := &DOCParser{}
	_, err := p.Parse(t.Context(), &ParseRequest{
		FileType: FileTypeDOC,
		FileName: "bad.doc",
		Content:  []byte("not-an-ole-file"),
	}, nil)
	assert.Error(t, err)
}
