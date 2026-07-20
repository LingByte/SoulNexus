package common

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"regexp"
	"strings"
)

const (
	CategoryMobile  = "mobile"
	CategoryDesktop = "desktop"
	CategoryTablet  = "tablet"
)

var (
	reAndroidMobile = regexp.MustCompile(`(?i)android.*mobile`)
	reAndroid       = regexp.MustCompile(`(?i)android`)
	reIOSMobile     = regexp.MustCompile(`(?i)(iphone|ipod)`)
	reIPad          = regexp.MustCompile(`(?i)ipad`)
	reMobileKeyword = regexp.MustCompile(`(?i)(mobile|phone)`)
)

// CategoryFromUserAgent class --mobile, desktop, or tablet.
func CategoryFromUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return CategoryDesktop
	}
	if reIPad.MatchString(ua) {
		return CategoryTablet
	}
	if reIOSMobile.MatchString(ua) || reAndroidMobile.MatchString(ua) {
		return CategoryMobile
	}
	if reAndroid.MatchString(ua) && !reMobileKeyword.MatchString(ua) {
		return CategoryTablet
	}
	if reMobileKeyword.MatchString(ua) {
		return CategoryMobile
	}
	return CategoryDesktop
}

// LoginLimitCategory maps tablet to mobile for concurrent session rules.
func LoginLimitCategory(ua string) string {
	c := CategoryFromUserAgent(ua)
	if c == CategoryTablet {
		return CategoryMobile
	}
	return c
}

// DisplayNameFromUserAgent builds a short label such as "Chrome · Windows".
func DisplayNameFromUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "Unknown device"
	}
	browser := detectBrowser(ua)
	osName := detectOS(ua)
	if browser != "" && osName != "" {
		return browser + " · " + osName
	}
	if browser != "" {
		return browser
	}
	if osName != "" {
		return osName
	}
	if len(ua) > 48 {
		return ua[:48] + "…"
	}
	return ua
}

func detectBrowser(ua string) string {
	checks := []struct {
		re   *regexp.Regexp
		name string
	}{
		{regexp.MustCompile(`(?i)Edg/`), "Edge"},
		{regexp.MustCompile(`(?i)OPR/|Opera`), "Opera"},
		{regexp.MustCompile(`(?i)Chrome/`), "Chrome"},
		{regexp.MustCompile(`(?i)Firefox/`), "Firefox"},
		{regexp.MustCompile(`(?i)Safari/`), "Safari"},
		{regexp.MustCompile(`(?i)MicroMessenger`), "WeChat"},
	}
	for _, c := range checks {
		if c.re.MatchString(ua) {
			return c.name
		}
	}
	return ""
}

func detectOS(ua string) string {
	checks := []struct {
		re   *regexp.Regexp
		name string
	}{
		{regexp.MustCompile(`(?i)Windows NT`), "Windows"},
		{regexp.MustCompile(`(?i)Mac OS X|Macintosh`), "macOS"},
		{regexp.MustCompile(`(?i)iPhone|iPad|iPod`), "iOS"},
		{regexp.MustCompile(`(?i)Android`), "Android"},
		{regexp.MustCompile(`(?i)Linux`), "Linux"},
	}
	for _, c := range checks {
		if c.re.MatchString(ua) {
			return c.name
		}
	}
	return ""
}
