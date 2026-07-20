package utils

import (
	"net/http"
	"testing"
)

func TestSafePathUnderBase(t *testing.T) {
	base := t.TempDir()
	child := base + "/scripts/run.py"
	if _, err := SafePathUnderBase(base, child); err != nil {
		t.Fatalf("expected allowed path: %v", err)
	}
	if _, err := SafePathUnderBase(base, base+"/../outside"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

func TestValidateStdioCommand(t *testing.T) {
	if err := ValidateStdioCommand("npx"); err != nil {
		t.Fatalf("npx should be allowed: %v", err)
	}
	if err := ValidateStdioCommand("bash"); err == nil {
		t.Fatal("bash should be rejected")
	}
}

func TestValidateStdioArgsRejectsInjection(t *testing.T) {
	err := ValidateStdioArgs([]string{"safe", "foo; rm -rf /"})
	if err == nil {
		t.Fatal("expected injection to be rejected")
	}
}

func TestSanitizeForLog(t *testing.T) {
	got := SanitizeForLog("hello\nworld\r\t!")
	if got != "hello world  !" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestApplyCustomHeadersSkipsReserved(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com", nil)
	ApplyCustomHeaders(req, map[string]string{
		"Authorization": "Bearer secret",
		"Content-Type":  "text/plain",
		"X-Custom":      "ok",
	})
	if req.Header.Get("Authorization") != "" {
		t.Fatal("authorization should not be overridden")
	}
	if req.Header.Get("X-Custom") != "ok" {
		t.Fatal("custom header missing")
	}
}
