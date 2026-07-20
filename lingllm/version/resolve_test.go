package version

import (
	"strings"
	"testing"
)

func TestResolvedDefaultsNotUnknown(t *testing.T) {
	if strings.TrimSpace(GetGoVersion()) == "" || GetGoVersion() == "unknown" {
		t.Fatalf("GoVersion=%q", GetGoVersion())
	}
	if strings.TrimSpace(GetGitCommit()) == "" || GetGitCommit() == "unknown" {
		t.Fatalf("GitCommit=%q", GetGitCommit())
	}
	if strings.TrimSpace(GetBuildTime()) == "" || GetBuildTime() == "unknown" {
		t.Fatalf("BuildTime=%q", GetBuildTime())
	}
	t.Log(GetVersionInfo())
}
