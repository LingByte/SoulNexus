package version

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Version 版本号
var Version = "1.4.5"

// GitCommit Git提交哈希
var GitCommit = "unknown"

// BuildTime 构建时间
var BuildTime = "unknown"

// GoVersion Go版本
var GoVersion = "unknown"

// GetVersion 获取版本信息
func GetVersion() string {
	return Version
}

// GetVersionInfo 获取完整版本信息
func GetVersionInfo() string {
	return Version + " (commit: " + GitCommit + ", built at: " + BuildTime + ", go: " + GoVersion + ")"
}

// GetGitCommit 获取Git提交哈希
func GetGitCommit() string {
	return GitCommit
}

// GetBuildTime 获取构建时间
func GetBuildTime() string {
	return BuildTime
}

// GetGoVersion 获取Go版本
func GetGoVersion() string {
	return GoVersion
}
