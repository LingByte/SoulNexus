// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseUintParam 从 gin URL 参数读取 uint id；解析失败或为 0 时返回 ok=false。
// 之前定义在 knowledge_base.go 中，KB 模块删除后迁移到通用工具文件。
func parseUintParam(c *gin.Context, key string) (uint, bool) {
	raw := c.Param(key)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return uint(v), true
}
