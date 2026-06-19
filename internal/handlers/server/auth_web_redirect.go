package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/gin-gonic/gin"
)

func webAppOrigin() string {
	if v := strings.TrimSpace(os.Getenv("WEB_APP_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if config.GlobalConfig != nil {
		if v := strings.TrimSpace(config.GlobalConfig.Server.URL); v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	return "http://localhost:5173"
}

func redirectWebAuthFailure(c *gin.Context, reason string) {
	q := url.Values{}
	q.Set("auth_error", reason)
	target := webAppOrigin() + "/?" + q.Encode()
	c.Redirect(http.StatusFound, target)
}

func redirectWebAuthSuccess(c *gin.Context, redirectURL, token, refreshToken string) {
	path := strings.TrimSpace(redirectURL)
	if path == "" {
		path = "/assistants"
	}
	u, err := url.Parse(path)
	if err != nil {
		redirectWebAuthFailure(c, "invalid_redirect")
		return
	}
	if !u.IsAbs() {
		base := webAppOrigin()
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		u, _ = url.Parse(base + path)
	}
	q := u.Query()
	q.Set("auth_token", token)
	if refreshToken != "" {
		q.Set("refresh_token", refreshToken)
	}
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}
