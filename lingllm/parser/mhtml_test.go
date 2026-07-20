package parser

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMHTMLParser_Parse(t *testing.T) {
	body := strings.Join([]string{
		"From: saved@example.com",
		"Subject: Saved Page",
		"MIME-Version: 1.0",
		`Content-Type: multipart/related; boundary="BOUNDARY"`,
		"",
		"--BOUNDARY",
		"Content-Type: text/html; charset=utf-8",
		"Content-Transfer-Encoding: 7bit",
		"",
		"<html><body><p>MHTML paragraph</p></body></html>",
		"--BOUNDARY--",
	}, "\r\n")

	p := &MHTMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeMHTML,
		FileName: "page.mhtml",
		Content:  []byte(body),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Contains(t, res.Text, "MHTML paragraph")
}

func TestExtractMHTMLHTML_RawHTML(t *testing.T) {
	html, err := extractMHTMLHTML([]byte("<html><body>Hello MHTML</body></html>"))
	require.NoError(t, err)
	assert.Contains(t, string(html), "Hello MHTML")
}
