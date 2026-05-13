package dialog

import "testing"

func TestAppendTagAfterNameAddr(t *testing.T) {
	in := `"Bob" <sip:bob@biloxi.com>`
	got := AppendTagAfterNameAddr(in, "abc")
	if TagFromHeader(got) != "abc" {
		t.Fatalf("tag: %q", got)
	}
	if got != `"Bob" <sip:bob@biloxi.com>;tag=abc` {
		t.Fatalf("full: %q", got)
	}
	if AppendTagAfterNameAddr(got, "x") != got {
		t.Fatal("expected no double tag")
	}
}
