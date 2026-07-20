package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerPlatformMailLogAdminRoutes 注册平台管理员邮件日志相关路由
// 路由前缀：admin/mail-logs
// 需要：平台管理员权限
func (h *Handlers) registerPlatformMailLogAdminRoutes(r *humax.Group) {
	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	mailLogsAdmin := admin.Group("admin/mail-logs")
	{
		mailLogsAdmin.GET("", h.handleListMailLogs)
		mailLogsAdmin.GET("/:id", h.handleGetMailLogDetail)
		mailLogsAdmin.GET("/stats/summary", h.handleGetMailLogStats)
	}
}

// handleListMailLogs 获取邮件日志分页列表
// GET /admin/mail-logs
// 鉴权：普通用户只能查询自己的日志，平台管理员可查询所有用户
// Query: page, pageSize, user_id（管理员可选）, status, provider, channel_name
// 响应：分页包装的邮件日志列表
func (h *Handlers) handleListMailLogs(c *gin.Context) {
	userID := middleware.AuthUserID(c)
	if userID == 0 && middleware.AuthPlatformAdminID(c) == 0 {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	page, pageSize := ginutil.QueryPage(c, 200)

	q := h.db.Model(&mail.MailLog{})
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

	list, total, err := utils.FindPage[mail.MailLog](q, page, pageSize, "id DESC", utils.MaxPageSize200)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	ginutil.PageSuccess(c, list, total, page, pageSize)
}

// handleGetMailLogDetail 获取单条邮件日志详情
// GET /admin/mail-logs/:id
// 鉴权：普通用户只能查看自己的日志，平台管理员可查看所有
// 响应：MailLog 完整对象；日志不存在返回错误
func (h *Handlers) handleGetMailLogDetail(c *gin.Context) {
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
	var row mail.MailLog
	q := h.db.Where("id = ?", id)
	if middleware.AuthPlatformAdminID(c) == 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Mail log not found", nil)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// handleGetMailLogStats 获取邮件日志统计摘要
// GET /admin/mail-logs/stats/summary
// 鉴权：普通用户只能查看自己的统计，平台管理员可通过 scope=all 查看全局统计
// 响应：按状态分组的计数统计，包含 total 总计
func (h *Handlers) handleGetMailLogStats(c *gin.Context) {
	userID := middleware.AuthUserID(c)
	if userID == 0 && middleware.AuthPlatformAdminID(c) == 0 {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	if middleware.AuthPlatformAdminID(c) > 0 && strings.EqualFold(strings.TrimSpace(c.Query("scope")), "all") {
		userID = 0
	}
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	q := h.db.Model(&mail.MailLog{}).Select("status, count(*) as cnt").Group("status")
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	out := map[string]int64{"total": 0}
	for _, r := range rows {
		out[r.Status] = r.Cnt
		out["total"] += r.Cnt
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}
