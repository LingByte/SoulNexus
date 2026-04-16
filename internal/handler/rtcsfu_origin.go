// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"net/http"
	"net/url"
	"strings"
)

// BuildRTCSFUSignalOriginChecker returns a CheckOrigin function for the SFU WebSocket.
// allowCSV is a comma-separated list of allowed Origin prefixes or full origins (e.g. https://app.example.com).
// When allowCSV is empty: in production mode only same-host Origins are allowed; in non-production, all origins are allowed.
func BuildRTCSFUSignalOriginChecker(mode, allowCSV string) func(r *http.Request) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	allowed := splitCSV(allowCSV)
	return func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		ou, err := url.Parse(origin)
		if err != nil || ou.Host == "" {
			return false
		}
		if len(allowed) > 0 {
			for _, a := range allowed {
				a = strings.TrimSpace(a)
				if a == "" {
					continue
				}
				if strings.HasPrefix(origin, a) || strings.EqualFold(ou.Host, a) {
					return true
				}
			}
			return false
		}
		if mode == "production" {
			return strings.EqualFold(ou.Host, r.Host)
		}
		return true
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
