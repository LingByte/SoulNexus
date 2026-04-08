package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFileType_ByExt(t *testing.T) {
	assert.Equal(t, FileTypeTXT, DetectFileType(&ParseRequest{FileName: "a.TXT"}))
	assert.Equal(t, FileTypeYML, DetectFileType(&ParseRequest{FileName: "a.yml"}))
	assert.Equal(t, FileTypeDOCX, DetectFileType(&ParseRequest{FileName: "a.docx"}))
}

func TestRouter_Parse_RejectsDoc(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{FileName: "a.doc", Content: []byte("x")}, &ParseOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "convert to .docx")
}

func TestRouter_Parse_Unsupported(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{FileName: "a.bin", Content: []byte("x")}, &ParseOptions{})
	assert.ErrorIs(t, err, ErrUnsupportedFileType)
}
