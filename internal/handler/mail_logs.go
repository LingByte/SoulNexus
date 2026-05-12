// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// handleListMailLogs lists paginated mail logs for the current user (or admin can filter by user_id).
//
// 查询参数：page, pageSize/size, status, provider, channel_name, user_id (admin only)
func (h *Handlers) handleListMailLogs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	page, pageSize := h.parsePagination(c)
	offset := (page - 1) * pageSize

	q := h.db.Model(&mail.MailLog{})
	// 普通用户只能看自己的；管理员可按 user_id 过滤，否则看全部
	if !models.UserHasAdminAccess(h.db, user.ID) {
		q = q.Where("user_id = ?", user.ID)
	} else if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
		var uid uint
		_, _ = fmt.Sscanf(uidStr, "%d", &uid)
		if uid > 0 {
			q = q.Where("user_id = ?", uid)
		}
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" && s != "all" {
		q = q.Where("status = ?", s)
	}
	if s := strings.TrimSpace(c.Query("provider")); s != "" {
		q = q.Where("provider = ?", s)
	}
	if s := strings.TrimSpace(c.Query("channel_name")); s != "" {
		q = q.Where("channel_name = ?", s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	var list []mail.MailLog
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	totalPage := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPage++
	}
	response.Success(c, "success", gin.H{
		"list": list, "total": total, "page": page, "pageSize": pageSize, "size": pageSize, "totalPage": totalPage,
	})
}

// handleGetMailLogDetail returns a single log row.
func (h *Handlers) handleGetMailLogDetail(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	var row mail.MailLog
	q := h.db.Where("id = ?", id)
	if !models.UserHasAdminAccess(h.db, user.ID) {
		q = q.Where("user_id = ?", user.ID)
	}
	if err := q.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Mail log not found", nil)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", row)
}

// handleGetMailLogStats returns per-status counts for the current user (or all if admin & ?scope=all).
func (h *Handlers) handleGetMailLogStats(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	uid := user.ID
	if models.UserHasAdminAccess(h.db, user.ID) && strings.EqualFold(strings.TrimSpace(c.Query("scope")), "all") {
		uid = 0 // 0 means "all users" — interpreted by GetMailLogStats below
	}
	stats, err := getMailLogStats(h.db, uid)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", stats)
}

func getMailLogStats(db *gorm.DB, userID uint) (map[string]int64, error) {
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	q := db.Model(&mail.MailLog{}).Select("status, count(*) as cnt").Group("status")
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{"total": 0}
	for _, r := range rows {
		out[r.Status] = r.Cnt
		out["total"] += r.Cnt
	}
	return out, nil
}
