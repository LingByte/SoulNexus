package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
)

const ginLocaleKey = "locale"

// Locale is a BCP 47-style language tag used for validation and API messages.
type Locale string

func normalizeLocale(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "en"
	}
	if idx := strings.Index(s, ","); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, ";"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	switch strings.ToLower(strings.ReplaceAll(s, "_", "-")) {
	case "zh", "zh-cn":
		return "zh-CN"
	case "zh-tw":
		return "zh-TW"
	default:
		return s
	}
}
