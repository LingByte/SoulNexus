package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InjectDB 注入数据库实例到 Gin 上下文
func InjectDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(constants.DbField, db)
		c.Next()
	}
}
