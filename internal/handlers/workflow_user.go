package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// workflowUser is a minimal stand-in for SoulNexus auth.User in workflow handlers.
type workflowUser struct {
	ID    uint
	Email string
}

// workflowCurrentUser returns the authenticated tenant user for workflow handlers.
func workflowCurrentUser(c *gin.Context) *workflowUser {
	if id := middleware.AuthUserID(c); id != 0 {
		email := ""
		if v, ok := c.Get(middleware.CtxAuthEmail); ok {
			if s, ok := v.(string); ok {
				email = s
			}
		}
		if email == "" {
			email = middleware.AuditOperator(c)
		}
		return &workflowUser{ID: id, Email: email}
	}
	return nil
}
