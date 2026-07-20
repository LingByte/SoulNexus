package handlers

import (
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/inbox"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerNotificationRoutes(r *humax.Group) {
	notif := r.Group("notification")
	// Tenant inbox only — platform admins use admin notification center routes.
	notif.Use(middleware.RequireTenantInboxUser())
	{
		notif.GET("unread-count", h.handleUnReadNotificationCount)
		notif.GET("", h.handleListNotifications)
		notif.POST("readAll", h.handleAllNotifications)
		notif.PUT("/read/:id", h.handleMarkNotificationAsRead)
		notif.DELETE("/:id", h.handleDeleteNotification)
		notif.POST("/batch-delete", h.handleBatchDeleteNotifications)
		notif.GET("/all-ids", h.handleAllNotificationIds)
	}
	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	notificationCenter := admin.Group("admin/notifications")
	{
		notificationCenter.GET("", h.handleAdminListInternalNotifications)
		notificationCenter.POST("/send", h.handleAdminSendInternalNotification)
		notificationCenter.GET("/:id", h.handleAdminGetInternalNotification)
		notificationCenter.DELETE("/:id", h.handleAdminDeleteInternalNotification)
	}
}

func (h *Handlers) handleUnReadNotificationCount(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	count, err := inbox.NewService(h.db).UnreadCount(userID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, count)
}

func (h *Handlers) handleListNotifications(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	page, size := ginutil.QueryPage(c, 200)
	filterBy := c.Query("filter")
	title := c.Query("title")
	content := c.Query("content")
	start, _ := timeutil.ParseQueryTime(c.Query("start_time"))
	end, _ := timeutil.ParseQueryTime(c.Query("end_time"))
	result, err := inbox.NewService(h.db).ListPage(userID, page, size, filterBy, title, content, start, end)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	payload := utils.PagePayload(inbox.MessagesToPublic(result.List), result.Total, page, size)
	payload["totalUnread"] = result.TotalUnread
	payload["totalRead"] = result.TotalRead
	response.SuccessI18n(c, i18n.KeySuccess, payload)
}

func (h *Handlers) handleAllNotifications(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	if err := inbox.NewService(h.db).MarkAllRead(userID); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyNotificationAllMarked, nil)
}

func (h *Handlers) handleMarkNotificationAsRead(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	notificationID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := inbox.NewService(h.db).MarkRead(userID, notificationID); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) handleDeleteNotification(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	notificationID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := inbox.NewService(h.db).Delete(userID, notificationID); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) handleBatchDeleteNotifications(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	var request struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	deletedCount, err := inbox.NewService(h.db).BatchDelete(userID, request.IDs)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"deletedCount": deletedCount})
}

func (h *Handlers) handleAllNotificationIds(c *gin.Context) {
	userID, ok := ginutil.RequireAuthUser(c)
	if !ok {
		return
	}
	filterBy := c.Query("filter")
	title := c.Query("title")
	content := c.Query("content")
	start, _ := timeutil.ParseQueryTime(c.Query("start_time"))
	end, _ := timeutil.ParseQueryTime(c.Query("end_time"))
	ids, err := inbox.NewService(h.db).AllIDs(userID, filterBy, title, content, start, end)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, ids)
}

func (h *Handlers) handleAdminListInternalNotifications(c *gin.Context) {
	page, pageSize := ginutil.QueryPage(c, 200)
	q := h.db.Model(&inbox.Message{})
	if uid := c.Query("user_id"); uid != "" {
		q = q.Where("user_id = ?", uid)
	}
	list, total, err := utils.FindPage[inbox.Message](q, page, pageSize, "id DESC", utils.MaxPageSize200)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	ginutil.PageSuccess(c, list, total, page, pageSize)
}

func (h *Handlers) handleAdminGetInternalNotification(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row inbox.Message
	if err := h.db.First(&row, id).Error; err != nil {
		response.AbortWithStatus(c, http.StatusNotFound)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) handleAdminDeleteInternalNotification(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	res := h.db.Delete(&inbox.Message{}, id)
	if res.Error != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.AbortWithStatus(c, http.StatusNotFound)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

func (h *Handlers) handleAdminSendInternalNotification(c *gin.Context) {
	var req struct {
		UserID  uint   `json:"userId" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	if err := inbox.NewService(h.db).Send(req.UserID, req.Title, req.Content); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}
