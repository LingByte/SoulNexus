package models

import (
	"strings"
	"testing"
)

func TestNormalizeDialogSkillCode_Chinese(t *testing.T) {
	got := NormalizeDialogSkillCode("基础能力")
	if got != "基础能力" {
		t.Fatalf("got %q want 基础能力", got)
	}
	got = EnsureDialogSkillCode("", "基础能力")
	if got != "基础能力" {
		t.Fatalf("Ensure from name got %q", got)
	}
	got = EnsureDialogSkillCode("", "")
	if got == "" || !strings.HasPrefix(got, "skill-") {
		t.Fatalf("generated code=%q", got)
	}
}

func TestSkillToolSuffix_Pinyin(t *testing.T) {
	suf := SkillToolSuffix("基础能力")
	if suf == "" || suf == "基础能力" {
		t.Fatalf("expected ascii/pinyin suffix, got %q", suf)
	}
}
