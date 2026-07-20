package version

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	v := GetVersion()
	if v == "" {
		t.Error("GetVersion() should not return empty string")
	}
}

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()
	if info == "" {
		t.Error("GetVersionInfo() should not return empty string")
	}
}

func TestGetGitCommit(t *testing.T) {
	commit := GetGitCommit()
	if commit == "" {
		t.Error("GetGitCommit() should not return empty string")
	}
}

func TestGetBuildTime(t *testing.T) {
	time := GetBuildTime()
	if time == "" {
		t.Error("GetBuildTime() should not return empty string")
	}
}

func TestGetGoVersion(t *testing.T) {
	version := GetGoVersion()
	if version == "" {
		t.Error("GetGoVersion() should not return empty string")
	}
}
