package common_test

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils/common"
)

func TestTruncateRunes(t *testing.T) {
	if got := common.TruncateRunes("hello", 10); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := common.TruncateRunes("一二三四五", 3); got != "一二三…" {
		t.Fatalf("got %q", got)
	}
	if got := common.TruncateRunes("abc", 0); got != "abc" {
		t.Fatalf("got %q", got)
	}
}
