package sse

import (
	"io"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	line, err := ReadLine(strings.NewReader("hello\nworld"), nil)
	if err != nil || line != "hello" {
		t.Fatalf("unexpected line: %q err=%v", line, err)
	}
}

func TestReadLineEOFWithPartial(t *testing.T) {
	line, err := ReadLine(strings.NewReader("partial"), nil)
	if line != "partial" || err != io.EOF {
		t.Fatalf("unexpected: %q err=%v", line, err)
	}
}

func TestDataPayload(t *testing.T) {
	payload, done := DataPayload("data: {\"x\":1}")
	if done || payload != `{"x":1}` {
		t.Fatalf("unexpected payload: %q done=%v", payload, done)
	}
	_, done = DataPayload("data: [DONE]")
	if !done {
		t.Fatal("expected done")
	}
	if _, ok := DataPayload("event: ping"); ok {
		t.Fatal("expected false for event line")
	}
}
