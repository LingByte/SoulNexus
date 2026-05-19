package utils

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

func ParsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 20
	if raw := strings.TrimSpace(c.Query("pageSize")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			pageSize = parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			pageSize = parsed
		}
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
