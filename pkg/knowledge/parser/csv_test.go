package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSVParser_Parse_Content(t *testing.T) {
	p := &CSVParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{FileType: FileTypeCSV, Content: []byte("name,age\nAlice,18\n")}, &ParseOptions{PreserveLineBreaks: true})
	assert.NoError(t, err)
	assert.Equal(t, FileTypeCSV, res.FileType)
	assert.Contains(t, res.Text, "Alice")
	assert.Contains(t, res.Text, "18")
}
