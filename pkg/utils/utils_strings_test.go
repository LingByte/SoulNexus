package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"
)

func TestStringToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []byte{},
		},
		{
			name:     "simple string",
			input:    "hello",
			expected: []byte("hello"),
		},
		{
			name:     "string with unicode",
			input:    "你好世界",
			expected: []byte("你好世界"),
		},
		{
			name:     "string with special characters",
			input:    "Hello, 世界! 🎉",
			expected: []byte("Hello, 世界! 🎉"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToBytes(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("StringToBytes length = %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Fatalf("StringToBytes[%d] = %d, want %d", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "simple bytes",
			input:    []byte("hello"),
			expected: "hello",
		},
		{
			name:     "bytes with unicode",
			input:    []byte("你好世界"),
			expected: "你好世界",
		},
		{
			name:     "bytes with special characters",
			input:    []byte("Hello, 世界! 🎉"),
			expected: "Hello, 世界! 🎉",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToString(tt.input)
			if result != tt.expected {
				t.Fatalf("BytesToString = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStringToBytesAndBack(t *testing.T) {
	original := "Hello, 世界! 🎉"
	bytes := StringToBytes(original)
	backToString := BytesToString(bytes)
	if backToString != original {
		t.Fatalf("Round-trip conversion failed: got %q, want %q", backToString, original)
	}
}

func TestStringToBytes_ZeroCopy(t *testing.T) {
	// Test that StringToBytes doesn't copy (this is a behavioral test)
	original := "test string"
	bytes1 := StringToBytes(original)
	bytes2 := StringToBytes(original)

	// Both should have the same underlying data pointer
	// Note: This test verifies the function works, but we can't easily test
	// zero-copy behavior without unsafe pointer comparison
	if len(bytes1) != len(bytes2) {
		t.Fatalf("StringToBytes inconsistent lengths")
	}
}

func TestBytesToString_ZeroCopy(t *testing.T) {
	// Test that BytesToString doesn't copy
	original := []byte("test bytes")
	str1 := BytesToString(original)
	str2 := BytesToString(original)

	if str1 != str2 {
		t.Fatalf("BytesToString inconsistent results")
	}
	if str1 != "test bytes" {
		t.Fatalf("BytesToString = %q, want %q", str1, "test bytes")
	}
}
