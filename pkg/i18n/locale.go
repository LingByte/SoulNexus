package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	LocaleZhCN = "zh-CN"
	LocaleZhTW = "zh-TW"
	LocaleEnUS = "en-US"
	LocaleJaJP = "ja-JP"
)

const ctxLocaleKey = "i18n.locale"

// ResolveLocale maps known locale strings to internal constants.
// Only the 4 values agreed with the frontend are recognized; anything else falls back to zh-CN.
func ResolveLocale(raw string) string {
	switch strings.TrimSpace(raw) {
	case LocaleZhCN:
		return LocaleZhCN
	case LocaleZhTW:
		return LocaleZhTW
	case LocaleEnUS:
		return LocaleEnUS
	case LocaleJaJP:
		return LocaleJaJP
	default:
		return LocaleZhCN
	}
}

// LocaleFromGin reads locale from context (set by LocaleMiddleware).
func LocaleFromGin(c *gin.Context) string {
	if c == nil {
		return LocaleZhCN
	}
	if v, ok := c.Get(ctxLocaleKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return LocaleZhCN
}

// SetLocaleOnGin stores locale on gin context.
func SetLocaleOnGin(c *gin.Context, locale string) {
	if c != nil {
		c.Set(ctxLocaleKey, ResolveLocale(locale))
	}
}

// ParseAcceptLanguage picks the first known locale from Accept-Language header.
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return LocaleZhCN
	}
	parts := strings.Split(header, ",")
	for _, part := range parts {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if tag == "" {
			continue
		}
		if loc := ResolveLocale(tag); loc != LocaleZhCN {
			return loc
		}
	}
	return LocaleZhCN
}
