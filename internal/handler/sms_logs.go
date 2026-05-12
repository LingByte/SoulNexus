// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) handleListSMSLogs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	page, pageSize := h.parsePagination(c)
	offset := (page - 1) * pageSize

	orgID := notificationChannelOrgID(c)
	q := h.db.Model(&sms.SMSLog{}).Where("org_id = ?", orgID)
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
	if s := strings.TrimSpace(c.Query("to_phone")); s != "" {
		q = q.Where("to_phone = ?", s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	var list []sms.SMSLog
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

func (h *Handlers) handleGetSMSLogDetail(c *gin.Context) {
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
	orgID := notificationChannelOrgID(c)
	q := h.db.Where("org_id = ? AND id = ?", orgID, id)
	if !models.UserHasAdminAccess(h.db, user.ID) {
		q = q.Where("user_id = ?", user.ID)
	}
	var row sms.SMSLog
	if err := q.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "SMS log not found", nil)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", row)
}

type smsSendBody struct {
	To       string            `json:"to" binding:"required"`
	Content  string            `json:"content"`
	Template string            `json:"template"`
	Data     map[string]string `json:"data"`
}

// handleAdminSendSMS 管理端调试用：通过当前 OrgID 启用渠道发送一条短信。
func (h *Handlers) handleAdminSendSMS(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	var req smsSendBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	orgID := notificationChannelOrgID(c)
	chans, err := listeners.EnabledSMSChannels(h.db, orgID)
	if err != nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "未配置可用短信渠道", gin.H{"error": err.Error()})
		return
	}
	sender, err := sms.NewMultiSender(chans, h.db, c.ClientIP(),
		sms.WithSMSLogOrgID(orgID), sms.WithSMSLogUserID(user.ID))
	if err != nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "短信服务不可用", gin.H{"error": err.Error()})
		return
	}
	msg := sms.Message{
		Content:  strings.TrimSpace(req.Content),
		Template: strings.TrimSpace(req.Template),
		Data:     req.Data,
	}
	sendReq := sms.SendRequest{
		To:      []sms.PhoneNumber{{Number: strings.TrimSpace(req.To)}},
		Message: msg,
	}
	if err := sender.Send(context.Background(), sendReq); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "已发送", gin.H{"to": req.To})
}
