package sms

import (
	"testing"
)

func TestRandHex(t *testing.T) {
	t.Parallel()
	if got := randHex(0); got != "" {
		t.Errorf("randHex(0) = %q, want empty", got)
	}
	if got := randHex(-1); got != "" {
		t.Errorf("randHex(-1) = %q, want empty", got)
	}
	a := randHex(8)
	b := randHex(8)
	if len(a) != 16 {
		t.Errorf("randHex(8) length = %d, want 16 hex chars", len(a))
	}
	if a == b {
		t.Error("expected different random values")
	}
}
