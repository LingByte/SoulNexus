package parser

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
)

func extractHTMLText(r io.Reader) (string, error) {
	n, err := html.Parse(r)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(text)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String(), nil
}

func extractHTMLTextFromBytes(data []byte) (string, error) {
	return extractHTMLText(bytes.NewReader(data))
}
