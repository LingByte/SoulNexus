package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// generateSlug 生成插件标识符
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "plugin-" + strings.ReplaceAll(time.Now().Format("2006-01-02-15-04-05"), "-", "")
	}
	return slug
}

// ensureUniqueWorkflowSlug generates a URL-safe slug from name and guarantees uniqueness.
func ensureUniqueWorkflowSlug(db *gorm.DB, name string) string {
	base := generateSlug(name)
	candidate := base
	for i := 0; i < 100; i++ {
		var count int64
		db.Model(&models.WorkflowDefinition{}).Where("slug = ?", candidate).Count(&count)
		if count == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i+1)
	}
	return base + "-" + time.Now().Format("150405")
}

// getUserID 从上下文获取用户ID
func getUserID(c *gin.Context) uint {
	if user := workflowCurrentUser(c); user != nil {
		return user.ID
	}
	return middleware.AuthUserID(c)
}
