package handlers

import (
	"fmt"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// handleGetEmailLogs gets paginated email logs for current user
func (h *Handlers) handleGetEmailLogs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	page := c.DefaultQuery("page", "1")
	size := c.DefaultQuery("size", "10")
	status := c.DefaultQuery("status", "all") // all, sent, failed, pending

	var pageInt, sizeInt int
	_, _ = fmt.Sscanf(page, "%d", &pageInt)
	_, _ = fmt.Sscanf(size, "%d", &sizeInt)

	if pageInt < 1 {
		pageInt = 1
	}
	if sizeInt < 1 || sizeInt > 100 {
		sizeInt = 10
	}

	logs, total, err := notification.GetMailLogsWithStatus(h.db, user.ID, pageInt, sizeInt, status)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", gin.H{
		"list":  logs,
		"total": total,
		"page":  pageInt,
		"size":  sizeInt,
	})
}

// handleGetEmailLogDetail gets email log detail
func (h *Handlers) handleGetEmailLogDetail(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	var logID uint
	_, err := fmt.Sscanf(c.Param("id"), "%d", &logID)
	if err != nil {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}

	log, err := notification.GetMailLogByID(h.db, user.ID, logID)
	if err != nil {
		response.Fail(c, "Email log not found", nil)
		return
	}

	response.Success(c, "success", log)
}

// handleGetEmailStats gets email statistics for current user
func (h *Handlers) handleGetEmailStats(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	stats, err := notification.GetMailLogStats(h.db, user.ID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", stats)
}
