package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/gin-gonic/gin"
)

// LocaleMiddleware resolves locale from ?lang= and Accept-Language, stores on context.
func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := i18n.LocaleZhCN
		if h := c.GetHeader("Accept-Language"); h != "" {
			locale = i18n.ParseAcceptLanguage(h)
		}
		i18n.SetLocaleOnGin(c, locale)
		c.Next()
	}
}
