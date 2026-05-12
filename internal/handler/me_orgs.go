// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// /api/me/orgs：当前用户已加入的组织列表，前端用于切换"激活组织"。
//
// 与 group_tenancy.go 配合：
//   - personal org：每个用户都会自动有一个 GroupTypePersonal
//   - team org：用户被邀请进入的协作组织
//
// 返回字段中 is_personal=true 的就是个人空间；前端默认选中。

package handlers

import (
	"errors"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

type meOrgItem struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Avatar     string `json:"avatar,omitempty"`
	Role       string `json:"role"`
	IsPersonal bool   `json:"is_personal"`
}

// handleMeListOrgs GET /api/me/orgs
//
// 返回当前用户加入的全部组织（包含个人空间）。如果个人空间不存在自动创建。
func (h *Handlers) handleMeListOrgs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	// 确保 personal 存在
	if _, err := models.EnsurePersonalGroupForUser(h.db, user.ID); err != nil {
		response.Fail(c, "ensure personal org failed", err)
		return
	}
	type row struct {
		ID     uint
		Name   string
		Type   string
		Avatar string
		Role   string
	}
	var rows []row
	err := h.db.Table("group_members AS gm").
		Select("g.id AS id, g.name AS name, g.type AS type, g.avatar AS avatar, gm.role AS role").
		Joins("JOIN `groups` AS g ON g.id = gm.group_id").
		Where("gm.user_id = ?", user.ID).
		Where("g.is_archived = ?", false).
		Order("g.type ASC, g.id ASC").
		Scan(&rows).Error
	if err != nil {
		response.Fail(c, "list orgs failed", err)
		return
	}
	out := make([]meOrgItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, meOrgItem{
			ID:         r.ID,
			Name:       r.Name,
			Type:       r.Type,
			Avatar:     r.Avatar,
			Role:       r.Role,
			IsPersonal: r.Type == models.GroupTypePersonal,
		})
	}
	response.Success(c, "orgs fetched", gin.H{"items": out})
}
