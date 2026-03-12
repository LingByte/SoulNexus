package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"regexp"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
)

// generateSlug 生成插件标识符
func generateSlug(name string) string {
	// 转换为小写
	slug := strings.ToLower(name)

	// 替换空格和特殊字符为连字符
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// 移除开头和结尾的连字符
	slug = strings.Trim(slug, "-")

	// 如果为空，使用时间戳
	if slug == "" {
		slug = "plugin-" + strings.ReplaceAll(time.Now().Format("2006-01-02-15-04-05"), "-", "")
	}

	return slug
}

// getUserID 从上下文获取用户ID
func getUserID(c *gin.Context) uint {
	// 首先尝试从CurrentUser获取
	if user := models.CurrentUser(c); user != nil {
		return user.ID
	}

	// 备用方法：直接从context获取
	if userObj, exists := c.Get(constants.UserField); exists && userObj != nil {
		if user, ok := userObj.(*models.User); ok {
			return user.ID
		}
	}

	return 0
}
