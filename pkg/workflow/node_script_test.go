package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"testing"
)

func TestWrapScriptWithLibraries(t *testing.T) {
	script := `
func Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	result := StringLib.ToUpper("hello")
	return map[string]interface{}{"output": result}, nil
}
`

	wrapped := wrapScriptWithLibraries(script, nil)

	// 检查是否包含必要的内容
	checks := map[string]string{
		"package main":                            "package declaration",
		"type stringLib struct{}":                 "stringLib type",
		"type mathLib struct{}":                   "mathLib type",
		"type timeLib struct{}":                   "timeLib type",
		"type cryptoLib struct{}":                 "cryptoLib type",
		"type regexLib struct{}":                  "regexLib type",
		"type validationLib struct{}":             "validationLib type",
		"type arrayLib struct{}":                  "arrayLib type",
		"var StringLib = &stringLib{}":            "StringLib initialization",
		"func log(args ...interface{})":           "log function",
		"func Run(inputs map[string]interface{})": "Run function",
	}

	for check, desc := range checks {
		if !strings.Contains(wrapped, check) {
			t.Errorf("wrapped script should contain %s: %s", desc, check)
		}
	}

	// 检查是否只有一个 package main
	packageCount := strings.Count(wrapped, "package main")
	if packageCount != 1 {
		t.Errorf("wrapped script should have exactly 1 'package main', got %d", packageCount)
	}

	// 检查是否没有外部库导入
	if strings.Contains(wrapped, "github.com/LingByte/SoulNexus/pkg/workflow/libs") {
		t.Error("wrapped script should NOT contain external libs import")
	}

	t.Logf("Wrapped script length: %d bytes", len(wrapped))
}
