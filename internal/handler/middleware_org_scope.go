// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 多租户上下文中间件：解析请求头 X-Org-ID（或 query org_id），校验当前用户成员身份，
// 把"激活组织 ID"塞入 gin.Context；下游 handler 用 OrgIDFromContext 读。
//
// 注意：此中间件仅设置上下文，不强制要求所有请求都带 org id；个人账号场景 orgID = 0。

package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gin-gonic/gin"
)

const ginCtxOrgID = "lingbyte.org_id"

// OrgScopeMiddleware 解析 X-Org-ID / ?org_id=；
//   - 未指定 → orgID = 0（个人/全局视图）
//   - 已指定 → 必须是当前用户加入的组织，否则 403
//
// 该中间件由 *Handlers 提供，能拿到 db；不要用 free function。
func (h *Handlers) OrgScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("X-Org-ID"))
		if raw == "" {
			raw = strings.TrimSpace(c.Query("org_id"))
		}
		if raw == "" {
			c.Set(ginCtxOrgID, uint(0))
			c.Next()
			return
		}
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || v == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "invalid org id"}})
			c.Abort()
			return
		}
		orgID := uint(v)
		user := models.CurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "auth_error", "message": "authentication required for org scope"}})
			c.Abort()
			return
		}
		if !models.UserIsGroupMember(h.db, user.ID, orgID) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "forbidden", "message": "not a member of the target organization"}})
			c.Abort()
			return
		}
		c.Set(ginCtxOrgID, orgID)
		c.Next()
	}
}

// OrgIDFromContext 读出当前请求的激活组织 ID（0 = 个人 / 未指定）。
func OrgIDFromContext(c *gin.Context) uint {
	if v, ok := c.Get(ginCtxOrgID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}
