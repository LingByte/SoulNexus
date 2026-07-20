// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package middleware

import (
	"runtime/debug"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler is a Gin middleware that recovers from panics and converts
// them to structured AppError responses. It also handles any errors stored
// in the gin.Context via c.Error().
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.ErrorCtx(c.Request.Context(), "panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(stack)),
				)

				var err error
				switch e := r.(type) {
				case error:
					err = e
				case string:
					err = response.New(response.CodeInternal, e)
				default:
					err = response.New(response.CodeInternal, "internal server error")
				}
				response.Render(c, err)
			}
		}()

		c.Next()

		if len(c.Errors) > 0 && !c.Writer.Written() {
			response.Render(c, c.Errors.Last().Err)
		}
	}
}

// WriteError is a convenience helper to attach an error to the context and abort.
func WriteError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	c.Error(err)
	c.Abort()
}
