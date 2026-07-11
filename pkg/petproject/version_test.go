package petproject

import (
	"strings"
	"testing"
)

func TestBumpSemverPatch(t *testing.T) {
	tests := map[string]string{
		"":        "1.0.0",
		"1.0.0":   "1.0.1",
		"2.3.9":   "2.3.10",
		"invalid": "1.0.1",
	}
	for in, want := range tests {
		if got := BumpSemverPatch(in); got != want {
			t.Errorf("BumpSemverPatch(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestApplyVersionBump(t *testing.T) {
	files := map[string]string{
		SoulpetYamlFile: "specVersion: 1\nname: Test\nkind: sprite\nversion: \"1.0.0\"\n",
	}
	next, ver := ApplyVersionBump(files, "fix idle fps", true)
	if ver != "1.0.1" {
		t.Fatalf("version = %q", ver)
	}
	if !strings.Contains(next[SoulpetYamlFile], "1.0.1") {
		t.Fatalf("yaml not updated: %s", next[SoulpetYamlFile])
	}
	if !strings.Contains(next[ChangelogFile], "fix idle fps") {
		t.Fatalf("changelog missing: %s", next[ChangelogFile])
	}
}
