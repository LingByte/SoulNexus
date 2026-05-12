// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package utils — Emoji 与 4 字节 UTF-8 字符过滤工具。
//
// 背景：MySQL 默认 utf8 (utf8mb3) 只支持 BMP（≤U+FFFF）字符；当遇到 emoji
// 或 CJK 扩展（4 字节 UTF-8）时插入会报错：
//
//	Error 3988 (HY000): Conversion from collation utf8mb4_unicode_ci into
//	utf8mb3_general_ci impossible for parameter
//
// 治本方案是把表/列改为 utf8mb4（已在 bootstrap.seeds 中处理）。
// 此处的工具用于个别场景的兜底降级：当落库失败时，可去掉 4 字节字符再
// 写入一次，至少保留可读文本内容。
package utils

import (
	"strings"
	"unicode/utf8"
)

// ContainsFourByteRune 判断字符串是否包含 4 字节 UTF-8 字符（典型为 emoji 或
// 部分 CJK 扩展），这类字符在 utf8mb3 列下无法写入。
func ContainsFourByteRune(s string) bool {
	for _, r := range s {
		if utf8.RuneLen(r) >= 4 {
			return true
		}
	}
	return false
}

// StripFourByteRunes 去除所有 4 字节 UTF-8 字符，例如 emoji。
// 非 4 字节字符（含普通中文、英文、标点）保持原样。
//
// 示例：
//
//	StripFourByteRunes("Hello ✉️ World 🌟") -> "Hello  World "
//	StripFourByteRunes("您好")             -> "您好"
func StripFourByteRunes(s string) string {
	if !ContainsFourByteRune(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if utf8.RuneLen(r) >= 4 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ReplaceFourByteRunes 把 4 字节字符替换为指定占位符（默认空字符串）。
// 主要给可视化场景使用，例如把 emoji 替换为 "[emoji]"。
func ReplaceFourByteRunes(s, placeholder string) string {
	if !ContainsFourByteRune(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if utf8.RuneLen(r) >= 4 {
			b.WriteString(placeholder)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
