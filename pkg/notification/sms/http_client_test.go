package sms

import (
	"context"
	"strings"
	"testing"
)

func TestIs2xx(t *testing.T) {
	t.Parallel()
	if !is2xx(200) || !is2xx(299) {
		t.Error("expected 2xx to pass")
	}
	if is2xx(199) || is2xx(300) {
		t.Error("expected non-2xx to fail")
	}
}

func TestTruncateRaw(t *testing.T) {
	t.Parallel()
	if got := truncateRaw("  hello  ", 10); got != "hello" {
		t.Errorf("trim = %q", got)
	}
	long := strings.Repeat("a", 20)
	got := truncateRaw(long, 10)
	if !strings.HasPrefix(got, strings.Repeat("a", 10)) || !strings.HasSuffix(got, "…") {
		t.Errorf("truncate = %q", got)
	}
	if truncateRaw("x", 0) != "x" {
		t.Error("max<=0 should return full string")
	}
}

func TestJsonString(t *testing.T) {
	t.Parallel()
	got := jsonString(map[string]string{"a": "b"})
	if got != `{"a":"b"}` {
		t.Errorf("jsonString = %q", got)
	}
	if jsonString(make(chan int)) != "" {
		t.Error("invalid value should return empty")
	}
}

func TestCtxOrBackground(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	if ctxOrBackground(ctx) != ctx {
		t.Error("non-nil ctx should be preserved")
	}
	if ctxOrBackground(nil) == nil {
		t.Error("nil ctx should return background")
	}
}

func TestFirstRecipient(t *testing.T) {
	t.Parallel()
	req := SendRequest{To: []PhoneNumber{{Number: " 13800138000 "}}}
	got, err := firstRecipient(req)
	if err != nil || got != "13800138000" {
		t.Errorf("got %q err %v", got, err)
	}
	_, err = firstRecipient(SendRequest{})
	if err != ErrInvalidArgument {
		t.Errorf("empty recipients err = %v", err)
	}
	_, err = firstRecipient(SendRequest{To: []PhoneNumber{{Number: " "}}})
	if err != ErrInvalidArgument {
		t.Errorf("blank number err = %v", err)
	}
}

func TestNormalizeContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		content, sig, want string
	}{
		{"", "Sign", ""},
		{"plain", "MySign", "【MySign】plain"},
		{"【已有】内容", "Sign", "【已有】内容"},
		{"plain", "", "plain"},
	}
	for _, tt := range tests {
		if got := normalizeContent(tt.content, tt.sig); got != tt.want {
			t.Errorf("normalizeContent(%q,%q) = %q, want %q", tt.content, tt.sig, got, tt.want)
		}
	}
}
