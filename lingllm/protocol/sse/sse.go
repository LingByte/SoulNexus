package sse

import (
	"io"
	"strings"
)

// ReadLine reads a single line from an SSE stream (without trailing newline).
func ReadLine(r io.Reader, byteCounter *int) (string, error) {
	var line strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if byteCounter != nil {
				*byteCounter += n
			}
			if buf[0] == '\n' {
				return line.String(), nil
			}
			line.WriteByte(buf[0])
		}
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), io.EOF
			}
			return "", err
		}
	}
}

// DataPayload extracts the JSON payload from an SSE data line.
func DataPayload(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "" || payload == "[DONE]" {
		return payload, payload == "[DONE]"
	}
	return payload, false
}
