package gateway

import (
	"testing"
)

func TestMergeDialogQueryParams(t *testing.T) {
	out, err := MergeDialogQueryParams("ws://dialog/ws", "key", "secret", "agent1")
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected merged URL")
	}
}

func TestRedactDialogDialURL(t *testing.T) {
	u := RedactDialogDialURL("ws://x?apiKey=abc&apiSecret=def&payload=big")
	if u == "" {
		t.Fatal("expected redacted URL")
	}
}
