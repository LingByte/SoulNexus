package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerPlatformSMSAdminRoutes(r *humax.Group) {
	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	smsLogsAdmin := admin.Group("admin")
	{
		smsLogsAdmin.GET("/sms-logs", h.handleListSMSLogs)
		smsLogsAdmin.GET("/sms-logs/:id", h.handleGetSMSLogDetail)
		smsLogsAdmin.POST("/sms/send", h.handleAdminSendSMS)
	}
}

func (h *Handlers) handleListSMSLogs(c *gin.Context) {
	userID := middleware.AuthUserID(c)
	if userID == 0 && middleware.AuthPlatformAdminID(c) == 0 {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	page, pageSize := ginutil.QueryPage(c, 200)

	q := h.db.Model(&sms.SMSLog{})
	if middleware.AuthPlatformAdminID(c) == 0 {
		q = q.Where("user_id = ?", userID)
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

	list, total, err := utils.FindPage[sms.SMSLog](q, page, pageSize, "id DESC", utils.MaxPageSize200)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	ginutil.PageSuccess(c, list, total, page, pageSize)
}

func (h *Handlers) handleGetSMSLogDetail(c *gin.Context) {
	userID := middleware.AuthUserID(c)
	if userID == 0 && middleware.AuthPlatformAdminID(c) == 0 {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	q := h.db.Where("id = ?", id)
	if middleware.AuthPlatformAdminID(c) == 0 {
		q = q.Where("user_id = ?", userID)
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
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

type smsSendBody struct {
	To       string            `json:"to" binding:"required"`
	Content  string            `json:"content"`
	Template string            `json:"template"`
	Data     map[string]string `json:"data"`
}

func (h *Handlers) handleAdminSendSMS(c *gin.Context) {
	userID := middleware.AuthUserID(c)
	var req smsSendBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	chans, err := listeners.EnabledSMSChannels(h.db)
	if err != nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "未配置可用短信渠道", gin.H{"error": err.Error()})
		return
	}
	sender, err := sms.NewMultiSender(chans, h.db, c.ClientIP(),
		sms.WithSMSLogUserID(userID))
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
	response.SuccessI18n(c, i18n.KeySent, gin.H{"to": req.To})
}
